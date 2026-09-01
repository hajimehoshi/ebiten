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
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// The X11 backend prefers the XIM on-the-spot input style, in which the input
// method reports its composition and the application draws the preedit inline,
// as on Windows and macOS. Input methods offering only over-the-spot draw the
// preedit themselves and report no composition, so only commits arrive.
// Committed text reaches the application the same way in both styles.

type textInputImpl struct {
	events *textInputEvents

	// registerOnce installs the input method handlers, which needs the window
	// and so cannot happen before the first session.
	registerOnce sync.Once

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
	t.registerOnce.Do(func() {
		ebiten.RunOnMainThread(func() {
			ui.Get().SetX11TextInputHandlersOnMainThread(t.sendComposition, t.sendCommit, t.events.isOpen)
		})
		t.seedFromInputChars()
	})

	// Discarding an abandoned composition can make the input method report text
	// one last time, so it has to happen before this session can observe
	// anything. It runs on the main thread, which is where the input method
	// reports, so it has finished reporting once this returns.
	discarded := t.updateIMEState(bounds)

	// The states queued for an abandoned composition describe a target this
	// session knows nothing about.
	if discarded {
		t.events.clearQueue()
	}
	return t.events.start()
}

// seedFromInputChars reports the text of the current tick as a commit. Until
// the handlers are registered the input method reports nowhere, so the input
// characters are the only record of what the tick already delivered.
//
// seedFromInputChars is called from the game thread.
func (t *textInputImpl) seedFromInputChars() {
	rs := ebiten.AppendInputChars(nil)
	if len(rs) == 0 {
		return
	}
	t.sendCommit(string(rs))
}

// sendComposition reports a composition update from the input method.
//
// sendComposition is called from the main thread.
func (t *textInputImpl) sendComposition(text string, selStartInBytes, selEndInBytes int) {
	t.events.send(textInputState{
		Text:                             text,
		CompositionSelectionStartInBytes: selStartInBytes,
		CompositionSelectionEndInBytes:   selEndInBytes,
		ReplacementStartInBytes:          noReplacement,
		ReplacementEndInBytes:            noReplacement,
		CommitKind:                       commitNone,
	})
}

// sendCommit reports committed text.
//
// sendCommit is called from the main thread.
func (t *textInputImpl) sendCommit(text string) {
	// A session ends at the commit it takes. One that was queued for a later
	// session, or dropped, ends nothing.
	if !t.events.send(textInputState{
		Text:                    text,
		ReplacementStartInBytes: noReplacement,
		ReplacementEndInBytes:   noReplacement,
		CommitKind:              commitRegular,
	}) {
		return
	}
	t.events.end()
}

// updateIMEState places the input method's candidate window at the caret, and
// discards an abandoned composition. It syncs with the main thread only when
// there is something to update.
func (t *textInputImpl) updateIMEState(bounds image.Rectangle) (discarded bool) {
	bounds = caretBoundsInClientNativePixels(bounds)
	x, y := bounds.Min.X, bounds.Max.Y
	discard := t.discardNeeded.CompareAndSwap(true, false)
	if !discard && t.spotSet && x == t.spotX && y == t.spotY {
		return false
	}
	t.spotX, t.spotY = x, y
	t.spotSet = true
	ebiten.RunOnMainThread(func() {
		if discard {
			ui.Get().ResetX11InputContextOnMainThread()
		}
		ic := ui.Get().X11InputContextOnMainThread()
		if ic == 0 {
			return
		}
		setIMESpotLocation(ic, x, y)
	})
	return discard
}
