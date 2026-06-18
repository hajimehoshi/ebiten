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
	"runtime"
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
