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

// The session runs the guest on a background goroutine: AdvanceTicks and AdvanceFrame queue work without
// blocking, PendingTicks reports the tick backlog, WaitTicks/WaitFrame block for completion, and
// CompositeFrame presents the rendered frame.

package vmhost_test

import (
	"runtime"
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
	guest.AdvanceFrame()
	if guest.CompositeFrame() {
		t.Error("CompositeFrame before the first tick reported a composited frame")
	}

	// Queue several ticks without blocking, then drain them. PendingTicks reports the backlog.
	const ticks = 5
	guest.AdvanceTicks(ticks)
	if !guest.WaitTicks() {
		t.Fatalf("WaitTicks failed: %v", guest.Err())
	}
	if n := guest.PendingTicks(); n != 0 {
		t.Errorf("PendingTicks after WaitTicks = %d; want 0", n)
	}

	// A frame now composites the drawn state. AdvanceFrame requests it, WaitFrame blocks for it, and
	// CompositeFrame presents it. The red tile's interior at (10+8, 10+8) device-independent pixels,
	// scaled to physical pixels, must carry the guest's red.
	guest.AdvanceFrame()
	if !guest.WaitFrame() {
		t.Fatalf("WaitFrame failed: %v", guest.Err())
	}
	if !guest.CompositeFrame() {
		t.Fatalf("CompositeFrame failed: %v", guest.Err())
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

// TestWaitFrameResolvesToLatestRequest checks that when a completed frame is awaiting compositing and a
// newer frame is requested, WaitFrame drops the stale frame and resolves to the latest request rather
// than returning the stale frame.
func TestWaitFrameResolvesToLatestRequest(t *testing.T) {
	guest := startGuest(t, "./testdata/guest", activateByEnv, "unix")

	const w, h = 320, 240
	scale := ebiten.Monitor().DeviceScaleFactor()
	pw, ph := int(w*scale), int(h*scale)
	outsideScreen := ebiten.NewImage(pw, ph)
	if err := guest.SetOutsideScreen(outsideScreen); err != nil {
		t.Fatal(err)
	}

	// Render a frame with the sprite at A and leave it completed but uncomposited: the single mirror now
	// holds a stale frame.
	const ax, ay = 40, 40
	guest.MoveCursor(ax, ay)
	guest.AdvanceTicks(1)
	guest.AdvanceFrame()
	if !guest.WaitFrame() {
		t.Fatalf("WaitFrame for the first frame failed: %v", guest.Err())
	}

	// Request a newer frame with the sprite at B without compositing the stale one first. WaitFrame must
	// drop the stale frame and resolve to this latest request, not return the stale frame at A.
	const bx, by = 200, 150
	guest.MoveCursor(bx, by)
	guest.AdvanceTicks(1)
	guest.AdvanceFrame()
	if !guest.WaitFrame() {
		t.Fatalf("WaitFrame for the latest frame failed: %v", guest.Err())
	}
	if !guest.CompositeFrame() {
		t.Fatalf("CompositeFrame failed: %v", guest.Err())
	}

	pixels := make([]byte, 4*pw*ph)
	outsideScreen.ReadPixels(pixels)
	at := func(x, y int) (r, g, b uint8) {
		i := 4 * (y*pw + x)
		return pixels[i], pixels[i+1], pixels[i+2]
	}

	// The latest frame placed the sprite at B: B's center is near-white, and A's center fell back to the
	// guest's bluish background (the stale frame at A was dropped, not composited).
	bcx, bcy := int((bx+8)*scale), int((by+8)*scale)
	acx, acy := int((ax+8)*scale), int((ay+8)*scale)
	if r, g, b := at(bcx, bcy); !(r > 0xc0 && g > 0xc0 && b > 0xc0) {
		t.Errorf("sprite not at the latest position B: (%d, %d, %d); want near-white", r, g, b)
	}
	if r, g, b := at(acx, acy); !(b > r && b > 0x40) {
		t.Errorf("stale frame A was composited: A's center = (%d, %d, %d); want bluish background", r, g, b)
	}
}

// TestWaitFrameUnderConcurrentAdvanceFrame stresses WaitFrame against AdvanceFrame called from another
// goroutine. Because WaitFrame waits for the request outstanding at its entry rather than chasing every
// later one, each call still returns despite the steady stream of newer requests; this also exercises the
// concurrent path under the race detector. It cannot pin which frame a given WaitFrame resolves to (that
// depends on timing), so it asserts that the wait completes, not a pixel.
func TestWaitFrameUnderConcurrentAdvanceFrame(t *testing.T) {
	guest := startGuest(t, "./testdata/guest", activateByEnv, "unix")

	scale := ebiten.Monitor().DeviceScaleFactor()
	outsideScreen := ebiten.NewImage(int(64*scale), int(64*scale))
	if err := guest.SetOutsideScreen(outsideScreen); err != nil {
		t.Fatal(err)
	}

	// Request frames continuously from a background goroutine for the whole capture.
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			default:
				guest.AdvanceFrame()
				runtime.Gosched()
			}
		}
	}()

	for range 20 {
		guest.AdvanceTicks(1)
		guest.AdvanceFrame()
		if !guest.WaitFrame() {
			t.Fatalf("WaitFrame did not return under concurrent AdvanceFrame: %v", guest.Err())
		}
		if !guest.CompositeFrame() {
			t.Fatalf("CompositeFrame failed under concurrent AdvanceFrame: %v", guest.Err())
		}
	}

	close(stop)
	<-done

	// The background goroutine may have left frames owed. Drain them so the session is idle when the test's
	// cleanup closes the guest: an idle guest sees a clean EOF, whereas closing mid-frame would interrupt
	// the guest's write and exit it non-zero.
	for guest.WaitFrame() {
		guest.CompositeFrame()
	}
}

// TestAdvanceTicks checks that AdvanceTicks(n) queues n ticks like n successive AdvanceTick calls, and
// that a non-positive count queues nothing.
func TestAdvanceTicks(t *testing.T) {
	guest := startGuest(t, "./testdata/atlas", activateByEnv, "unix")

	const w, h = 128, 128
	scale := ebiten.Monitor().DeviceScaleFactor()
	pw, ph := int(w*scale), int(h*scale)
	outsideScreen := ebiten.NewImage(pw, ph)
	if err := guest.SetOutsideScreen(outsideScreen); err != nil {
		t.Fatal(err)
	}

	// A zero count queues nothing, so the backlog stays empty.
	guest.AdvanceTicks(0)
	if n := guest.PendingTicks(); n != 0 {
		t.Errorf("PendingTicks after AdvanceTicks(0) = %d; want 0", n)
	}

	// A negative count is a programming error and panics.
	func() {
		defer func() {
			if recover() == nil {
				t.Error("AdvanceTicks(-1) did not panic")
			}
		}()
		guest.AdvanceTicks(-1)
	}()

	// AdvanceTicks(n) drives n Updates; WaitTicks drains the whole backlog.
	guest.AdvanceTicks(5)
	if !guest.WaitTicks() {
		t.Fatalf("WaitTicks failed: %v", guest.Err())
	}
	if n := guest.PendingTicks(); n != 0 {
		t.Errorf("PendingTicks after WaitTicks = %d; want 0", n)
	}

	// The guest ran: a frame now composites its drawn state.
	guest.AdvanceFrame()
	if !guest.WaitFrame() {
		t.Fatalf("WaitFrame failed: %v", guest.Err())
	}
	if !guest.CompositeFrame() {
		t.Fatalf("CompositeFrame failed: %v", guest.Err())
	}
}
