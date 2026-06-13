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

// The session runs the guest on a background goroutine: AdvanceTick queues ticks without blocking,
// PendingTicks reports the backlog, and WaitTick/WaitFrame block for completion.

package vmhost_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestAsyncTickQueue(t *testing.T) {
	guest := startGuest(t, "./testdata/atlas", activateByEnv, "unix")

	const w, h = 128, 128
	scale := ebiten.Monitor().DeviceScaleFactor()
	pw, ph := int(w*scale), int(h*scale)
	outsideScreen := ebiten.NewImage(pw, ph)
	if err := guest.SetOutsideScreen(outsideScreen); err != nil {
		t.Fatal(err)
	}

	// A frame requested before the first tick composites nothing: the guest draws no frame before its
	// first Update.
	if guest.AdvanceFrame() {
		t.Error("AdvanceFrame before the first tick reported a composited frame")
	}

	// Queue several ticks without blocking, then drain them. PendingTicks reports the backlog.
	const ticks = 5
	for range ticks {
		guest.AdvanceTick()
	}
	if !guest.WaitTick() {
		t.Fatalf("WaitTick failed: %v", guest.Err())
	}
	if n := guest.PendingTicks(); n != 0 {
		t.Errorf("PendingTicks after WaitTick = %d; want 0", n)
	}

	// A frame now composites the drawn state. AdvanceFrame requests it and WaitFrame blocks for it. The
	// red tile's interior at (10+8, 10+8) device-independent pixels, scaled to physical pixels, must
	// carry the guest's red.
	guest.AdvanceFrame()
	if !guest.WaitFrame() {
		t.Fatalf("WaitFrame failed: %v", guest.Err())
	}
	pixels := make([]byte, 4*pw*ph)
	outsideScreen.ReadPixels(pixels)
	x := int(float64(10+8) * scale)
	y := int(float64(10+8) * scale)
	i := 4 * (y*pw + x)
	if r, g, b := pixels[i], pixels[i+1], pixels[i+2]; r != 0xff || g != 0x00 || b != 0x00 {
		t.Errorf("red tile after async driving = (%d, %d, %d); want (255, 0, 0)", r, g, b)
	}
}
