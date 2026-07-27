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

//go:build (freebsd || (linux && !android) || netbsd) && !nintendosdk && !playstation5

package textinput

import (
	"image"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// The X11 backend uses the XIM over-the-spot input style: the input method
// draws the preedit itself at the spot location, kept at the caret, and the
// application receives only the committed text, which arrives as input
// characters. Composition states are never reported.
//
// TODO: Implement the on-the-spot input style (XIMPreeditCallbacks) so that
// composition states are reported and the application can draw the preedit
// inline, as on Windows and macOS. This requires the input context to be
// created with preedit callbacks in internal/glfw.

type textInputImpl struct {
	events *textInputEvents

	rs       []rune
	lastTick int64

	// discardNeeded reports whether the input method can still hold a
	// composition for an abandoned target, to be discarded when the next
	// session starts.
	discardNeeded atomic.Bool

	spotX, spotY int
	spotSet      bool
}

func (t *textInputImpl) markIMEDiscardNeeded() {
	t.discardNeeded.Store(true)
}

func (t *textInputImpl) Start(bounds image.Rectangle, _, _ string) (<-chan textInputState, func()) {
	// AppendInputChars is updated only when the tick is updated.
	// If the tick is not updated, return nil immediately.
	tick := ebiten.Tick()
	if t.lastTick == tick {
		return nil, nil
	}
	defer func() {
		t.lastTick = tick
	}()

	t.updateIMEState(bounds)

	ch, _ := t.events.start()

	t.rs = ebiten.AppendInputChars(t.rs[:0])
	if len(t.rs) == 0 {
		return nil, nil
	}
	t.events.send(textInputState{
		Text:                    string(t.rs),
		ReplacementStartInBytes: noReplacement,
		ReplacementEndInBytes:   noReplacement,
		CommitKind:              commitRegular,
	})
	t.events.end()
	return ch, func() {}
}

// updateIMEState places the input method's preedit and candidate windows at
// the caret, and discards an abandoned composition. It syncs with the main
// thread only when there is something to update.
func (t *textInputImpl) updateIMEState(bounds image.Rectangle) {
	bounds = caretBoundsInClientNativePixels(bounds)
	x, y := bounds.Min.X, bounds.Max.Y
	discard := t.discardNeeded.CompareAndSwap(true, false)
	if !discard && t.spotSet && x == t.spotX && y == t.spotY {
		return
	}
	t.spotX, t.spotY = x, y
	t.spotSet = true
	ebiten.RunOnMainThread(func() {
		ic := ui.Get().X11InputContextOnMainThread()
		if ic == 0 {
			return
		}
		if discard {
			discardIMEComposition(ic)
		}
		setIMESpotLocation(ic, x, y)
	})
}
