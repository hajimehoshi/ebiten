// Copyright 2026 The Ebitengine Authors
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
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// The position must not grow after the player finished its source, even if Play is called again.
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
	// Wait until the player finishes its source.
	finished := false
	for i := 0; i < 200; i++ {
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

	// The context does not update a finished player anymore, so play and update a few more times
	// to let the position settle with the exhausted source.
	for i := 0; i < 5; i++ {
		p.Play()
		if err := audio.UpdateForTesting(); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	end := p.Position()
	if got, want := end, 4*time.Second; got < want || got > want+100*time.Millisecond {
		t.Errorf("Position() after the player finished: got: %v, want: %v or a bit larger", got, want)
	}

	// Play on a finished player does nothing, so the position must not grow anymore.
	for i := 0; i < 20; i++ {
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

// The position must not grow after the player finished, even when the underlying player keeps
// playing its buffered data after its source is exhausted, as real players do. In that case the
// consumed samples reach their final value before the player stops playing, and the stopwatch must
// be stopped in the update then to freeze the position.
func TestPositionNotGrowingAfterFinishedWithBufferedSource(t *testing.T) {
	setup()
	defer teardown()

	// Consume 1/7 of the source per update so that the buffered data is drained a few ticks after
	// the source is exhausted.
	audio.SetBufferedDrainForTesting(44100 * 8 / 7)
	defer audio.SetBufferedDrainForTesting(0)

	// 44100 [Hz] * 8 [bytes/sample] is one second of 32bit float stereo audio.
	src := bytes.NewReader(make([]byte, 44100*8*2))
	p, err := context.NewPlayerF32(src)
	if err != nil {
		t.Fatal(err)
	}

	p.Play()
	// Wait until the player drains both its source and its buffer.
	finished := false
	for i := 0; i < 500; i++ {
		if err := audio.UpdateForTesting(); err != nil {
			t.Fatal(err)
		}
		if !p.IsPlaying() {
			finished = true
			break
		}
		time.Sleep(time.Millisecond)
	}
	if !finished {
		t.Fatal("time out: the player did not finish")
	}

	end := p.Position()
	if got, want := end, 2*time.Second; got < want || got > want+100*time.Millisecond {
		t.Errorf("Position() after the player finished: got: %v, want: %v or a bit larger", got, want)
	}

	// Play on a finished player does nothing, so the position must not grow anymore.
	for i := 0; i < 20; i++ {
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
