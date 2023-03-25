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

	for i := 0; i < 10; i++ {
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

// Issue #853
func TestSameSourcePlayers(t *testing.T) {
	setup()
	defer teardown()

	src := bytes.NewReader(make([]byte, 4))
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

	if err := audio.UpdateForTesting(); err == nil {
		t.Errorf("got: nil, want: an error")
	}
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
