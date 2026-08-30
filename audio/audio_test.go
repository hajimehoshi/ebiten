// Copyright 2018 The Ebiten Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package audio_test

import (
	"bytes"
	"io"
	"math"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

var context *audio.Context

func setup() {
	context = audio.NewContext(44100)
}

func teardown() {
	audio.ResetContextForTesting()
	context = nil
}

// Issue #746
func TestGC(t *testing.T) {
	setup()
	defer teardown()

	p, _ := context.NewPlayer(bytes.NewReader(make([]byte, 4)))
	got := audio.PlayersCountForTesting()
	if want := 0; got != want {
		t.Errorf("PlayersCountForTesting(): got: %d, want: %d", got, want)
	}

	p.Play()
	got = audio.PlayersCountForTesting()
	if want := 1; got != want {
		t.Errorf("PlayersCountForTesting() after Play: got: %d, want: %d", got, want)
	}

	runtime.KeepAlive(p)
	p = nil
	runtime.GC()

	for range 10 {
		got = audio.PlayersCountForTesting()
		if want := 0; got == want {
			return
		}
		if err := audio.UpdateForTesting(); err != nil {
			t.Error(err)
		}
		// 200[ms] should be enough all the bytes are consumed.
		// TODO: This is a dirty hack. Would it be possible to use virtual time?
		time.Sleep(200 * time.Millisecond)
	}
	t.Errorf("time out")
}

type infiniteReader struct{}

func (i *infiniteReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

// Issue #853
func TestSameSourcePlayers(t *testing.T) {
	setup()
	defer teardown()

	src := &infiniteReader{}
	p0, err := context.NewPlayer(src)
	if err != nil {
		t.Fatal(err)
	}
	p1, err := context.NewPlayer(src)
	if err != nil {
		t.Fatal(err)
	}

	// As the player does not play yet, error doesn't happen.
	if err := audio.UpdateForTesting(); err != nil {
		t.Error(err)
	}

	p0.Play()
	p1.Play()

	for range 10 {
		if err := audio.UpdateForTesting(); err != nil {
			// An error is expected.
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Errorf("time out")
}

func TestPauseBeforeInit(t *testing.T) {
	setup()
	defer teardown()

	src := bytes.NewReader(make([]byte, 4))
	p, err := context.NewPlayer(src)
	if err != nil {
		t.Fatal(err)
	}

	p.Play()
	p.Pause()
	p.Play()

	if err := audio.UpdateForTesting(); err != nil {
		t.Error(err)
	}
}

type emptySource struct{}

func (emptySource) Read(buf []byte) (int, error) {
	return len(buf), nil
}

func TestNonSeekableSource(t *testing.T) {
	if runtime.GOOS == "js" {
		t.Skip("infinite steams in tests cannot be treated well on browsers")
	}

	setup()
	defer teardown()

	p, err := context.NewPlayer(emptySource{})
	if err != nil {
		t.Fatal(err)
	}

	p.Play()
	p.Pause()

	if err := audio.UpdateForTesting(); err != nil {
		t.Error(err)
	}
}

// Issue #3438
func TestDeferredDeviceCreation(t *testing.T) {
	setup()
	defer teardown()

	p, err := context.NewPlayer(bytes.NewReader(make([]byte, 4)))
	if err != nil {
		t.Fatal(err)
	}

	// Touching a player before the first update must not create the audio device.
	p.SetVolume(0.5)
	p.Play()

	if audio.ContextCreatedForTesting() {
		t.Errorf("the audio device must not be created before the first update")
	}

	// The state set before the device is created is recorded.
	if got, want := p.Volume(), 0.5; got != want {
		t.Errorf("Volume(): got: %v, want: %v", got, want)
	}
	if got, want := p.IsPlaying(), true; got != want {
		t.Errorf("IsPlaying(): got: %t, want: %t", got, want)
	}
	if got, want := audio.PlayersCountForTesting(), 1; got != want {
		t.Errorf("PlayersCountForTesting(): got: %d, want: %d", got, want)
	}

	// The first update creates the audio device.
	if err := audio.UpdateForTesting(); err != nil {
		t.Error(err)
	}
	if !audio.ContextCreatedForTesting() {
		t.Errorf("the audio device must be created after the first update")
	}

	// The pending player eventually starts playing and finishes, which is only possible
	// once the device is created and the player is materialized.
	for range 10 {
		if audio.PlayersCountForTesting() == 0 {
			return
		}
		if err := audio.UpdateForTesting(); err != nil {
			t.Error(err)
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Errorf("time out")
}

// Issue #3438
func TestSetPositionBeforeDeviceCreation(t *testing.T) {
	setup()
	defer teardown()

	// 44100 [Hz] * 8 [bytes/sample] is one second of 32bit float stereo audio.
	src := bytes.NewReader(make([]byte, 44100*8))
	p, err := context.NewPlayerF32(src)
	if err != nil {
		t.Fatal(err)
	}

	if err := p.SetPosition(500 * time.Millisecond); err != nil {
		t.Fatal(err)
	}

	// SetPosition must not create the audio device.
	if audio.ContextCreatedForTesting() {
		t.Errorf("the audio device must not be created before the first update")
	}

	// The position set before the device is created is recorded.
	if got, want := p.Position(), 500*time.Millisecond; got != want {
		t.Errorf("Position(): got: %v, want: %v", got, want)
	}
}

func TestSetPositionLongDuration(t *testing.T) {
	setup()
	defer teardown()

	p, err := context.NewPlayerF32(bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}

	// 10 milliseconds is exactly 441 samples at 44100 Hz, so the position
	// round-trips exactly while exercising both the whole-seconds and the
	// remainder terms of the conversion.
	const duration = 8*time.Hour + 10*time.Millisecond
	if err := p.SetPosition(duration); err != nil {
		t.Fatal(err)
	}
	if got, want := p.Position(), duration; got != want {
		t.Errorf("Position(): got: %v, want: %v", got, want)
	}
}

// Issue #3438
func TestRewindNonSeekableBeforeDeviceCreation(t *testing.T) {
	setup()
	defer teardown()

	p, err := context.NewPlayer(emptySource{})
	if err != nil {
		t.Fatal(err)
	}

	p.Play()

	// Rewinding to the start before the device is created must not panic even for a
	// non-seekable source, as the stream has not been read yet.
	if err := p.Rewind(); err != nil {
		t.Fatal(err)
	}

	p.Pause()

	if err := audio.UpdateForTesting(); err != nil {
		t.Error(err)
	}
}

type uncomparableSource []int

func (uncomparableSource) Read(buf []byte) (int, error) {
	return 0, io.EOF
}

// Issue #3039
func TestUncomparableSource(t *testing.T) {
	setup()
	defer teardown()

	p, err := context.NewPlayer(uncomparableSource{})
	if err != nil {
		t.Fatal(err)
	}

	p.Play()
	p.Pause()

	if err := audio.UpdateForTesting(); err != nil {
		t.Error(err)
	}
}

// countingSource is an infinite source counting the Read calls.
type countingSource struct {
	reads atomic.Int64
}

func (s *countingSource) Read(buf []byte) (int, error) {
	s.reads.Add(1)
	return len(buf), nil
}

// waitForRead waits until src's read count differs from reads and returns the new count.
func waitForRead(t *testing.T, src *countingSource, reads int64) int64 {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		if got := src.reads.Load(); got != reads {
			return got
		}
		if time.Now().After(deadline) {
			t.Fatal("the source was not read")
		}
		time.Sleep(time.Millisecond)
	}
}

// Issue #3510
func TestPauseAndStopReading(t *testing.T) {
	if runtime.GOOS == "js" {
		t.Skip("infinite steams in tests cannot be treated well on browsers")
	}

	setup()
	defer teardown()

	src := &countingSource{}
	p, err := context.NewPlayerF32(src)
	if err != nil {
		t.Fatal(err)
	}

	p.Play()

	// The first update creates the audio device and starts the pending player.
	if err := audio.UpdateForTesting(); err != nil {
		t.Error(err)
	}

	waitForRead(t, src, 0)

	p.PauseAndStopReading()

	if p.IsPlaying() {
		t.Error("IsPlaying() after PauseAndStopReading: got: true, want: false")
	}
	if got, want := audio.PlayersCountForTesting(), 0; got != want {
		t.Errorf("PlayersCountForTesting() after PauseAndStopReading: got: %d, want: %d", got, want)
	}

	// After PauseAndStopReading, the source must not be read.
	reads := src.reads.Load()
	time.Sleep(100 * time.Millisecond)
	if got := src.reads.Load(); got != reads {
		t.Errorf("the source was read after PauseAndStopReading: got: %d reads, want: %d", got, reads)
	}

	// The player is reusable after PauseAndStopReading.
	p.Play()
	waitForRead(t, src, reads)
}

// Issue #3510
func TestPauseAndStopReadingWhilePaused(t *testing.T) {
	if runtime.GOOS == "js" {
		t.Skip("infinite steams in tests cannot be treated well on browsers")
	}

	setup()
	defer teardown()

	src := &countingSource{}
	p, err := context.NewPlayerF32(src)
	if err != nil {
		t.Fatal(err)
	}

	p.Play()

	// The first update creates the audio device and starts the pending player.
	if err := audio.UpdateForTesting(); err != nil {
		t.Error(err)
	}

	reads := waitForRead(t, src, 0)

	// A paused player keeps reading the source to fill its buffer.
	p.Pause()
	waitForRead(t, src, reads)

	// PauseAndStopReading must stop the reads even though the player is already paused.
	p.PauseAndStopReading()
	reads = src.reads.Load()
	time.Sleep(100 * time.Millisecond)
	if got := src.reads.Load(); got != reads {
		t.Errorf("the source was read after PauseAndStopReading: got: %d reads, want: %d", got, reads)
	}
}

// Issue #3510
func TestPauseAndStopReadingBeforeInit(t *testing.T) {
	setup()
	defer teardown()

	p, err := context.NewPlayer(bytes.NewReader(make([]byte, 4)))
	if err != nil {
		t.Fatal(err)
	}

	// PauseAndStopReading before the device is created cancels the pending play.
	p.Play()
	p.PauseAndStopReading()

	if p.IsPlaying() {
		t.Error("IsPlaying() after PauseAndStopReading: got: true, want: false")
	}
	if got, want := audio.PlayersCountForTesting(), 0; got != want {
		t.Errorf("PlayersCountForTesting() after PauseAndStopReading: got: %d, want: %d", got, want)
	}

	if err := audio.UpdateForTesting(); err != nil {
		t.Error(err)
	}
}

// Issue #3359
func TestSetVolumeInvalidValue(t *testing.T) {
	setup()
	defer teardown()

	p, err := context.NewPlayer(&infiniteReader{})
	if err != nil {
		t.Fatal(err)
	}

	volumes := []struct {
		in   float64
		want float64
	}{
		{in: 0.5, want: 0.5},
		{in: 2, want: 2},
		{in: math.MaxFloat32, want: math.MaxFloat32},
		{in: -1, want: 0},
		{in: math.MaxFloat32 * 2, want: 0},
		{in: math.Inf(1), want: 0},
		{in: math.Inf(-1), want: 0},
		{in: math.NaN(), want: 0},
	}

	// The volume is kept by the player itself before the audio device is created.
	for _, v := range volumes {
		p.SetVolume(v.in)
		if got, want := p.Volume(), v.want; got != want {
			t.Errorf("Volume() after SetVolume(%v) before the device creation: got: %v, want: %v", v.in, got, want)
		}
	}

	p.Play()
	if err := audio.UpdateForTesting(); err != nil {
		t.Fatal(err)
	}
	if !audio.ContextCreatedForTesting() {
		t.Fatal("the audio device must be created after the first update")
	}

	// The volume is kept by the underlying player after the audio device is created.
	for _, v := range volumes {
		p.SetVolume(v.in)
		if got, want := p.Volume(), v.want; got != want {
			t.Errorf("Volume() after SetVolume(%v) after the device creation: got: %v, want: %v", v.in, got, want)
		}
	}
}

func TestPositionNotGrowingAfterFinished(t *testing.T) {
	setup()
	defer teardown()

	// 44100 [Hz] * 8 [bytes/sample] is one second of 32bit float stereo audio.
	src := bytes.NewReader(make([]byte, 44100*8*4))
	p, err := context.NewPlayerF32(src)
	if err != nil {
		t.Fatal(err)
	}

	p.Play()
	// Wait until the player finishes its source. The deadline is generous because time.Sleep has
	// a several-millisecond floor on browsers, which slows draining the source down.
	var finished bool
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if err := audio.UpdateForTesting(); err != nil {
			t.Fatal(err)
		}
		if !p.IsPlaying() {
			finished = true
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !finished {
		t.Fatal("time out: the player did not finish")
	}

	end := p.Position()
	if got, want := end, 4*time.Second; got < want || got > want+100*time.Millisecond {
		t.Errorf("Position() after the player finished: got: %v, want: %v or a bit larger", got, want)
	}

	// Play on a finished player does nothing, so the position must not grow anymore.
	for range 20 {
		p.Play()
		if err := audio.UpdateForTesting(); err != nil {
			t.Fatal(err)
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := p.Position(); got != end {
		t.Errorf("position grew after the player finished: %v -> %v", end, got)
	}
}
