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

// The guest plays a scripted pair of sources (see testdata/audio): a finite ramp at full volume and
// an infinite 0.25 source at volume 0.5. The host learns each new stream through the OnAudioStream
// handler and pulls it on demand as its own stream (never mixed), so each is asserted byte-exactly —
// including that the volume is reported but not applied.

package vmhost_test

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"slices"
	"sync"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
)

// readComps reads up to want components from the player, stopping early at io.EOF. A read returning no
// samples without ending is retried, but a bounded number of times so a stuck stream fails the test
// rather than hanging.
func readComps(t *testing.T, p *vmhost.GuestAudioStream, want int) (comps []float32, eof bool) {
	t.Helper()
	buf := make([]byte, 4096)
	for empties := 0; len(comps) < want; {
		n, err := p.Read(buf)
		for i := 0; i+4 <= n; i += 4 {
			comps = append(comps, math.Float32frombits(binary.LittleEndian.Uint32(buf[i:])))
		}
		if errors.Is(err, io.EOF) {
			return comps, true
		}
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if n == 0 {
			if empties++; empties > 1000 {
				t.Fatal("audio stream produced no samples")
			}
			continue
		}
		empties = 0
	}
	return comps, false
}

func TestAudioForwarding(t *testing.T) {
	// The handler runs on the session goroutine, so guard the collected streams with a mutex.
	var mu sync.Mutex
	var streams []*vmhost.GuestAudioStream
	snapshot := func() []*vmhost.GuestAudioStream {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(streams)
	}

	guest := startGuestWithOptions(t, "./testdata/audio", activateByEnv, "unix", &vmhost.NewGuestSessionOptions{
		// The handler must not read the stream (that would deadlock the session goroutine), so it only
		// records the handle; the test reads it below on its own goroutine.
		OnAudioStream: func(s *vmhost.GuestAudioStream) {
			mu.Lock()
			streams = append(streams, s)
			mu.Unlock()
		},
	})

	// AdvanceTicks requires a screen even though only audio is read back.
	scale := ebiten.Monitor().DeviceScaleFactor()
	outsideScreen := ebiten.NewImage(int(64*scale), int(64*scale))
	if err := guest.SetOutsideScreen(outsideScreen); err != nil {
		t.Fatal(err)
	}

	// Tick past the point where both players start (ramp at tick 3, flat at tick 4). WaitTick returns only
	// after the tick's messages are handled, so by tick 6 the handler has been called for each new stream.
	for tick := 1; tick <= 6; tick++ {
		guest.AdvanceTicks(1)
		if !guest.WaitTick() {
			t.Fatalf("waiting for tick %d failed: %v", tick, guest.Err())
		}
	}

	players := snapshot()
	if len(players) != 2 {
		t.Fatalf("OnAudioStream reported %d streams; want 2 (one per guest player, never mixed)", len(players))
	}

	// The host learns the guest's own sample rate (the fixture's 48000); it is not asked to match it.
	if rate := guest.AudioSampleRate(); rate != 48000 {
		t.Errorf("AudioSampleRate() = %d; want 48000", rate)
	}

	// Classify the two streams by their reported volume: the ramp is full volume, the flat source 0.5.
	var ramp, flat *vmhost.GuestAudioStream
	for _, p := range players {
		switch p.Volume() {
		case 1:
			ramp = p
		case 0.5:
			flat = p
		default:
			t.Fatalf("unexpected player volume %v", p.Volume())
		}
	}
	if ramp == nil || flat == nil {
		t.Fatalf("missing a stream: ramp=%v flat=%v", ramp != nil, flat != nil)
	}

	// Each stream is stamped with the guest tick it started on: the fixture plays the ramp during its 3rd
	// Update (ebiten.Tick() 2) and the flat source during its 4th (ebiten.Tick() 3). StartTick anchors the
	// stream to the guest's tick timeline.
	if ramp.StartTick() != 2 {
		t.Errorf("ramp.StartTick() = %d; want 2", ramp.StartTick())
	}
	if flat.StartTick() != 3 {
		t.Errorf("flat.StartTick() = %d; want 3", flat.StartTick())
	}

	// The ramp source is 2000 frames = 4000 components, value i+1 at index i, and it ends.
	rampComps, eof := readComps(t, ramp, 1<<30)
	if !eof {
		t.Error("the finite ramp stream did not reach EOF")
	}
	if len(rampComps) != 4000 {
		t.Errorf("ramp stream has %d components; want 4000", len(rampComps))
	}
	for i, v := range rampComps {
		if v != float32(i+1) {
			t.Fatalf("ramp component %d = %v; want %v", i, v, float32(i+1))
		}
	}

	// The flat source is infinite at volume 0.5, but the volume is NOT applied to the samples, so each
	// component is the raw 0.25.
	flatComps, _ := readComps(t, flat, 4000)
	if len(flatComps) < 4000 {
		t.Errorf("flat stream produced %d components; want at least 4000", len(flatComps))
	}
	for i, v := range flatComps {
		if v != 0.25 {
			t.Fatalf("flat component %d = %v; want 0.25 (volume must not be applied)", i, v)
		}
	}

	// The guest closes the flat player at tick 8. Its infinite source never reaches EOF on its own, so the
	// close is what ends the host's stream: a later Read reports EOF. Closing a stream is not a new-stream
	// event, so the handler must not fire again.
	for tick := 7; tick <= 8; tick++ {
		guest.AdvanceTicks(1)
		if !guest.WaitTick() {
			t.Fatalf("waiting for tick %d failed: %v", tick, guest.Err())
		}
	}
	if _, eof := readComps(t, flat, 1); !eof {
		t.Error("the closed flat stream did not reach EOF on the host")
	}
	if all := snapshot(); len(all) != 2 {
		t.Errorf("OnAudioStream reported %d streams after a close; want 2 (a close is not a new stream)", len(all))
	}
	// The finished ramp reports not playing, matching audio.Player at its end.
	if ramp.IsPlaying() {
		t.Error("the finished ramp stream still reports playing")
	}
	// IsClosed distinguishes the guest-closed flat from the ramp, which only reached EOF: a host tracking
	// streams keeps the latter (it could be replayed) but may drop the former.
	if !flat.IsClosed() {
		t.Error("the guest-closed flat stream does not report closed")
	}
	if ramp.IsClosed() {
		t.Error("the ramp stream reports closed, but it only reached EOF and was never closed")
	}
}
