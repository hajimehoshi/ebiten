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

// End-to-end ReadPixels round-trip: the guest has no GPU, so when it reads an offscreen image back
// mid-tick the host renders the image on demand and returns the pixels. The guest paints the screen
// with what it read, so a correct round-trip is provable from the rendered screen.

package vmhost_test

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/internal/vmprotocol"
)

func TestReadPixelsRoundTrip(t *testing.T) {
	guest := startGuest(t, "./testdata/readpixels", activateByEnv, "unix")

	// These calls run inside the host's run loop (the test runs from the game's Update), so issuing
	// ebiten draws and reading pixels back is allowed.
	const w, h = 64, 48

	// The guest renders at the host's device scale factor, so the target screen is physical-sized.
	scale := ebiten.Monitor().DeviceScaleFactor()
	pw, ph := int(w*scale), int(h*scale)
	outsideScreen := ebiten.NewImage(pw, ph)

	if err := guest.SetOutsideScreen(outsideScreen); err != nil {
		t.Fatal(err)
	}

	// The round-trip happens inside this tick's Update; the host serves it on demand.
	tickAndFrame(t, guest)

	// The guest filled its offscreen with (0x12, 0x34, 0x56, 0xff), read it back through the host, and
	// painted the screen with exactly those bytes. Recover them from the screen.
	pixels := make([]byte, 4*pw*ph)
	outsideScreen.ReadPixels(pixels)
	i := 4 * ((ph/2)*pw + pw/2)
	r, g8, b, a := pixels[i], pixels[i+1], pixels[i+2], pixels[i+3]
	t.Logf("screen center = (%d, %d, %d, %d)", r, g8, b, a)
	if r != 0x12 || g8 != 0x34 || b != 0x56 || a != 0xff {
		t.Errorf("screen center = (%d, %d, %d, %d); want (0x12, 0x34, 0x56, 0xff) — ReadPixels did not round-trip",
			r, g8, b, a)
	}
}

func TestReadPixelsIsServedWhileWaitingForTheTick(t *testing.T) {
	guest := startGuest(t, "./testdata/readpixels", activateByEnv, "unix")

	const w, h = 64, 48
	scale := ebiten.Monitor().DeviceScaleFactor()
	outsideScreen := ebiten.NewImage(int(w*scale), int(h*scale))
	if err := guest.SetOutsideScreen(outsideScreen); err != nil {
		t.Fatal(err)
	}

	guest.AdvanceTicks(1)
	waited := make(chan bool, 1)
	go func() {
		waited <- guest.WaitTicks()
	}()

	const budget = 30 * time.Second
	select {
	case ok := <-waited:
		if !ok {
			t.Fatalf("advancing the guest failed: %v", guest.Err())
		}
	case <-time.After(budget):
		t.Fatalf("WaitTicks did not return within %v: the guest's read-back was never served", budget)
	}
}

func TestReadPixelsAsyncGuestExit(t *testing.T) {
	guest := startGuest(t, filepath.Join("..", "..", "internal", "processtest", "testdata", "readpixelsasync.go"), activateByEnv, "unix")
	screen := ebiten.NewImage(16, 16)
	defer screen.Deallocate()
	if err := guest.SetOutsideScreen(screen); err != nil {
		t.Fatal(err)
	}
	flush := ebiten.NewImage(1, 1)
	defer flush.Deallocate()
	pixels := make([]byte, 4)
	deadline := time.Now().Add(30 * time.Second)
	for range 100 {
		if !time.Now().Before(deadline) {
			break
		}
		guest.AdvanceTicks(1)
		if !guest.WaitTicks() {
			if err := guest.Err(); !errors.Is(err, ebiten.Termination) {
				t.Errorf("guest failed: %v", err)
			}
			return
		}
		// Submit and poll the host's read-backs while this test occupies its Update.
		flush.Fill(color.White)
		flush.ReadPixels(pixels)
	}
	t.Error("the guest did not complete its asynchronous read-back")
}

func TestReadPixelsAsyncPendingAndDispose(t *testing.T) {
	answers := make(chan vmprotocol.HostMessage, 8)
	var id uint64
	var tick int
	guest := newFakeGuestSession(t, func(fg *fakeGuest) {
		query := func(msg *vmprotocol.GuestMessage) vmprotocol.HostMessage {
			fg.send(msg)
			answer, err := fg.receive()
			if err != nil {
				t.Error(err)
			}
			answers <- answer
			return answer
		}
		switch tick {
		case 0:
			fg.sendCommands(vmprotocol.GraphicsCommand{
				Kind:    vmprotocol.GraphicsCommandKindNewImage,
				ImageID: 1,
				Width:   2,
				Height:  1,
			}, vmprotocol.GraphicsCommand{
				Kind:    vmprotocol.GraphicsCommandKindWritePixels,
				ImageID: 1,
				Regions: []image.Rectangle{image.Rect(0, 0, 2, 1)},
				Pixels:  [][]byte{{1, 2, 3, 255, 4, 5, 6, 255}},
			})
			start := &vmprotocol.GuestMessage{
				Kind:        vmprotocol.GuestMessageKindQueryReadPixelsAsync,
				ReadImageID: 1,
				ReadRegions: []image.Rectangle{image.Rect(0, 0, 1, 1), image.Rect(1, 0, 2, 1)},
			}
			id = query(start).ReadbackID
			abandoned := query(start).ReadbackID
			query(&vmprotocol.GuestMessage{
				Kind:       vmprotocol.GuestMessageKindQueryPollReadPixels,
				ReadbackID: id,
			})
			fg.send(&vmprotocol.GuestMessage{
				Kind:       vmprotocol.GuestMessageKindDisposeReadPixels,
				ReadbackID: abandoned,
			})
			query(&vmprotocol.GuestMessage{
				Kind:       vmprotocol.GuestMessageKindQueryPollReadPixels,
				ReadbackID: abandoned,
			})
			fg.sendCommands(vmprotocol.GraphicsCommand{
				Kind:    vmprotocol.GraphicsCommandKindDisposeImage,
				ImageID: 1,
			})
		default:
			query(&vmprotocol.GuestMessage{
				Kind:       vmprotocol.GuestMessageKindQueryPollReadPixels,
				ReadbackID: id,
			})
		}
		tick++
	})
	setOutsideScreen(t, guest)
	guest.AdvanceTicks(1)
	if !guest.WaitTicks() {
		t.Fatal(guest.Err())
	}
	first, second := <-answers, <-answers
	if first.Err != "" || second.Err != "" || first.ReadbackID == second.ReadbackID {
		t.Fatalf("starting independent read-backs failed: %+v, %+v", first, second)
	}
	if answer := <-answers; answer.Err != "" || answer.ReadbackDone || len(answer.Pixels) != 0 {
		t.Errorf("poll before host submission = %+v; want pending", answer)
	}
	if answer := <-answers; answer.Err == "" {
		t.Error("poll of a released read-back succeeded")
	}
	flush := ebiten.NewImage(1, 1)
	defer flush.Deallocate()
	pixels := make([]byte, 4)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		flush.Fill(color.White)
		flush.ReadPixels(pixels)
		guest.AdvanceTicks(1)
		if !guest.WaitTicks() {
			t.Fatal(guest.Err())
		}
		answer := <-answers
		if answer.Err != "" {
			t.Fatal(answer.Err)
		}
		if !answer.ReadbackDone {
			continue
		}
		want := [][]byte{{1, 2, 3, 255}, {4, 5, 6, 255}}
		if len(answer.Pixels) != len(want) {
			t.Fatalf("got %d regions, want %d", len(answer.Pixels), len(want))
		}
		for j := range want {
			if !bytes.Equal(answer.Pixels[j], want[j]) {
				t.Errorf("region %d = %v, want %v", j, answer.Pixels[j], want[j])
			}
		}
		return
	}
	t.Error("the read-back did not complete after host submission")
}
