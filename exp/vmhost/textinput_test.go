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

// End-to-end text input: the guest's game runs an exp/textinput Composer, whose session reaches the
// host through the OnTextInput handler; the host answers with a composition and then a commit,
// and the guest's painted state proves each response arrived intact.

package vmhost_test

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
)

// The session parameters and responses mirror the guest program; keep the two in sync.
const (
	textInputTextBeforeCaret = "ab"
	textInputTextAfterCaret  = "cd"
	textInputComposition     = "こん"
	textInputCommit          = "こんにちは"
)

func TestTextInput(t *testing.T) {
	// The handler runs during AdvanceTicks and WaitTicks on this goroutine, so no lock is needed.
	var sessions []*vmhost.GuestTextInput
	guest := startGuestWithOptions(t, "./testdata/textinput", activateByEnv, "unix", &vmhost.NewGuestSessionOptions{
		OnTextInput: func(ti *vmhost.GuestTextInput) {
			sessions = append(sessions, ti)
		},
	})

	const w, h = 32, 32
	scale := ebiten.Monitor().DeviceScaleFactor()
	pw, ph := int(w*scale), int(h*scale)
	outsideScreen := ebiten.NewImage(pw, ph)
	if err := guest.SetOutsideScreen(outsideScreen); err != nil {
		t.Fatal(err)
	}

	// The first tick starts the guest's session; its start message rides the tick, so the handler has
	// run once WaitTicks returns.
	guest.AdvanceTicks(1)
	if !guest.WaitTicks() {
		t.Fatalf("waiting for the tick failed: %v", guest.Err())
	}
	got := sessions
	if len(got) != 1 {
		t.Fatalf("got %d text-input sessions; want 1", len(got))
	}
	ti := got[0]
	if got, want := ti.TextBeforeCaret(), textInputTextBeforeCaret; got != want {
		t.Errorf("TextBeforeCaret() = %q; want %q", got, want)
	}
	if got, want := ti.TextAfterCaret(), textInputTextAfterCaret; got != want {
		t.Errorf("TextAfterCaret() = %q; want %q", got, want)
	}
	// The caret rectangle set by the guest survives the round trip through native pixels, up to
	// rounding.
	if got, want := ti.CaretBounds(), image.Rect(4, 8, 6, 24); !rectsAlmostEqual(got, want, 2) {
		t.Errorf("CaretBounds() = %v; want about %v", got, want)
	}
	if ti.IsClosed() {
		t.Error("IsClosed() = true before any response; want false")
	}

	// A composition response reaches the guest at its next tick; the guest paints blue when it shows
	// the expected preedit.
	ti.SetComposition(textInputComposition, 0, len(textInputComposition))
	tickAndFrame(t, guest)
	if r, g, b := screenColor(outsideScreen, pw, ph); !(b > 0xc0 && r < 0x40 && g < 0x40) {
		t.Errorf("after the composition the guest painted (%d,%d,%d); want blue", r, g, b)
	}

	// A commit ends the session; the guest applies it at the caret and paints green. Its Composer then
	// starts the next session within the same tick, so the release and the new start ride together.
	ti.Commit(textInputCommit)
	tickAndFrame(t, guest)
	if r, g, b := screenColor(outsideScreen, pw, ph); !(g > 0xc0 && r < 0x40 && b < 0x40) {
		t.Errorf("after the commit the guest painted (%d,%d,%d); want green", r, g, b)
	}
	if !ti.IsClosed() {
		t.Error("IsClosed() = false after the commit; want true")
	}
	// Handlers run during AdvanceTicks and WaitTicks; tickAndFrame's AdvanceTicks dispatched before
	// the tick was processed, so deliver the restarted session's event now.
	if !guest.WaitTicks() {
		t.Fatalf("waiting for the tick failed: %v", guest.Err())
	}
	got = sessions
	if len(got) != 2 {
		t.Fatalf("got %d text-input sessions after the commit; want 2", len(got))
	}
	if got[1].IsClosed() {
		t.Error("the session restarted after the commit is already closed")
	}

	// Ending the follow-up session from the host side is observed by the guest as a teardown: its
	// Composer clears the preedit and retries, so the painted state stays green (the committed
	// buffer).
	got[1].End()
	tickAndFrame(t, guest)
	if r, g, b := screenColor(outsideScreen, pw, ph); !(g > 0xc0 && r < 0x40 && b < 0x40) {
		t.Errorf("after ending the follow-up session the guest painted (%d,%d,%d); want green (the committed buffer)", r, g, b)
	}

	// The guest's Composer restarts at the tick after a teardown (unlike after a commit, which
	// restarts within the same tick), so the third session arrives one tick later. A commit replacing
	// bytes [1, 3) of the surrounding "ab"+"cd" with "XY" yields the aXYd buffer, painted cyan.
	guest.AdvanceTicks(1)
	if !guest.WaitTicks() {
		t.Fatalf("waiting for the tick failed: %v", guest.Err())
	}
	got = sessions
	if len(got) != 3 {
		t.Fatalf("got %d text-input sessions after the host-side end; want 3", len(got))
	}
	got[2].CommitWithOptions("XY", &vmhost.GuestTextInputCommitOptions{
		ReplaceSurroundingText:  true,
		ReplacementStartInBytes: 1,
		ReplacementEndInBytes:   3,
	})
	tickAndFrame(t, guest)
	if r, g, b := screenColor(outsideScreen, pw, ph); !(g > 0xc0 && b > 0xc0 && r < 0x40) {
		t.Errorf("after the replacement commit the guest painted (%d,%d,%d); want cyan", r, g, b)
	}
}

// screenColor samples the composited screen's center pixel.
func screenColor(screen *ebiten.Image, pw, ph int) (r, g, b uint8) {
	pixels := make([]byte, 4*pw*ph)
	screen.ReadPixels(pixels)
	i := 4 * ((ph/2)*pw + pw/2)
	return pixels[i], pixels[i+1], pixels[i+2]
}

// rectsAlmostEqual reports whether the corresponding edges of a and b differ by at most tolerance
// pixels, absorbing the rounding of a device-scale round trip.
func rectsAlmostEqual(a, b image.Rectangle, tolerance int) bool {
	abs := func(v int) int {
		if v < 0 {
			return -v
		}
		return v
	}
	return abs(a.Min.X-b.Min.X) <= tolerance &&
		abs(a.Min.Y-b.Min.Y) <= tolerance &&
		abs(a.Max.X-b.Max.X) <= tolerance &&
		abs(a.Max.Y-b.Max.Y) <= tolerance
}
