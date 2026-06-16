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
// an infinite 0.25 source at volume 0.5. The host pulls each on demand as its own stream (never
// mixed), so each is asserted byte-exactly — including that the volume is reported but not applied.

package vmhost_test

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vmhost"
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
	guest := startGuest(t, "./testdata/audio", activateByEnv, "unix")

	// AdvanceTick requires a screen even though only audio is read back.
	scale := ebiten.Monitor().DeviceScaleFactor()
	outsideScreen := ebiten.NewImage(int(64*scale), int(64*scale))
	if err := guest.SetOutsideScreen(outsideScreen); err != nil {
		t.Fatal(err)
	}

	// Tick past the point where both players start (ramp at tick 3, flat at tick 4) so the host has
	// learned them through the control plane.
	for tick := 1; tick <= 6; tick++ {
		guest.AdvanceTick()
		if !guest.WaitTick() {
			t.Fatalf("waiting for tick %d failed: %v", tick, guest.Err())
		}
	}

	players := guest.AppendAudioStreams(nil)
	if len(players) != 2 {
		t.Fatalf("got %d audio players; want 2 (the guest's players must not be mixed)", len(players))
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

	// The guest closes the flat player at tick 8. The host must drop it even though its infinite source
	// never reaches EOF: its Read ends and it disappears from the stream list. The ramp, by contrast,
	// reached EOF but was never closed, so it must persist (it could be sought back and replayed).
	for tick := 7; tick <= 8; tick++ {
		guest.AdvanceTick()
		if !guest.WaitTick() {
			t.Fatalf("waiting for tick %d failed: %v", tick, guest.Err())
		}
	}
	if _, eof := readComps(t, flat, 1); !eof {
		t.Error("the closed flat stream did not reach EOF on the host")
	}
	var sawFlat, sawRamp bool
	for _, p := range guest.AppendAudioStreams(nil) {
		switch p {
		case flat:
			sawFlat = true
		case ramp:
			sawRamp = true
		}
	}
	if sawFlat {
		t.Error("the closed stream is still returned by AppendAudioStreams")
	}
	if !sawRamp {
		t.Error("the ramp stream was dropped at EOF; it must persist until closed")
	}
	// The finished ramp reports not playing, matching audio.Player at its end.
	if ramp.IsPlaying() {
		t.Error("the finished ramp stream still reports playing")
	}
}
