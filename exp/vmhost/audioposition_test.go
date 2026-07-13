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

// The guest (see testdata/audioposition) asserts its own player position each tick: it must be
// exactly the samples the host has pulled, without the wall-clock smoothing (#2901) that would run
// ahead while the host pulls nothing and snap back once it starts.

package vmhost_test

import (
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
)

// audioPositionSampleRate is the fixture's sample rate.
const audioPositionSampleRate = 48000

func TestAudioPosition(t *testing.T) {
	// The handler runs on this goroutine, during AdvanceTicks and WaitTicks, so the collected streams
	// need no lock.
	var streams []*vmhost.GuestAudioStream

	guest := startGuestWithOptions(t, "./testdata/audioposition", activateByEnv, "unix", &vmhost.NewGuestSessionOptions{
		OnAudioStream: func(s *vmhost.GuestAudioStream) {
			streams = append(streams, s)
		},
	})

	// AdvanceTicks requires a screen even though only audio is exercised.
	scale := ebiten.Monitor().DeviceScaleFactor()
	outsideScreen := ebiten.NewImage(int(64*scale), int(64*scale))
	if err := guest.SetOutsideScreen(outsideScreen); err != nil {
		t.Fatal(err)
	}

	// Tick 1 starts the guest's player. A failed guest-side assertion surfaces as that tick's error.
	guest.AdvanceTicks(1)
	if !guest.WaitTicks() {
		t.Fatalf("waiting for tick 1 failed: %v", guest.Err())
	}
	if len(streams) != 1 {
		t.Fatalf("OnAudioStream reported %d streams; want 1", len(streams))
	}

	// Let real time pass without pulling anything, giving the guest's position updater (which runs on
	// its own wall-clock schedule) ample opportunity to misreport progress; the guest asserts through
	// tick 4 that the position stays 0.
	time.Sleep(500 * time.Millisecond)
	for tick := 2; tick <= 4; tick++ {
		guest.AdvanceTicks(1)
		if !guest.WaitTicks() {
			t.Fatalf("waiting for tick %d failed: %v", tick, guest.Err())
		}
	}

	// Pull exactly 100ms of samples (frames of 8 bytes), then let the guest's position updater observe
	// the consumption; the guest asserts at tick 5 that the position is exactly 100ms.
	buf := make([]byte, 8*audioPositionSampleRate/10)
	for read, empties := 0, 0; read < len(buf); {
		n, err := streams[0].Read(buf[read:])
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if n == 0 {
			if empties++; empties > 1000 {
				t.Fatal("the audio stream produced no samples")
			}
			continue
		}
		empties = 0
		read += n
	}
	time.Sleep(500 * time.Millisecond)
	guest.AdvanceTicks(1)
	if !guest.WaitTicks() {
		t.Fatalf("waiting for tick 5 failed: %v", guest.Err())
	}
}
