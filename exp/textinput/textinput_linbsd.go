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

// queueCarryTicks is how long after a session ends the states queued since
// still count as belonging to the next session.
const queueCarryTicks = 2

type textInputImpl struct {
	events *textInputEvents

	// registerOnce installs the input method handlers, which needs the window
	// and so cannot happen before the first session.
	registerOnce sync.Once

	// discardNeeded reports whether the input method can still hold a
	// composition for an abandoned target, to be discarded when the next
	// session starts.
	discardNeeded atomic.Bool

	// lastEndTick is the tick the last session ended at, or 0 before the first
	// one ends. It is written when the input method commits, on the main
	// thread, and read when a session starts, on the game thread.
	lastEndTick atomic.Int64

	// queuedTick is the tick the queued states were reported in. It is written
	// on the main thread as they are reported, and read on the game thread
	// when a session starts.
	queuedTick atomic.Int64

	spotX, spotY int
	spotSet      bool
}

func (t *textInputImpl) markIMEDiscardNeeded() {
	t.discardNeeded.Store(true)
}

func (t *textInputImpl) Start(bounds image.Rectangle, _, _ string) (<-chan textInputState, func()) {
	t.registerOnce.Do(func() {
		ebiten.RunOnMainThread(func() {
			ui.Get().SetX11TextInputHandlersOnMainThread(t.sendComposition, t.sendCommit)
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
	if discarded || !t.queueBelongsToNextSession() {
		t.events.clearQueue()
	}
	ch, _ := t.events.start()
	return ch, func() {
		t.end()
	}
}

// end ends the session and records when it ended.
func (t *textInputImpl) end() {
	t.lastEndTick.Store(ebiten.Tick())
	t.events.end()
}

// withinQueueCarry reports whether a session starting at tick follows one that
// ended at lastEndTick closely enough to take over its queued states.
// lastEndTick is 0 when no session has ended yet.
//
// The input method reports text whether or not a session is open, and a commit
// ends the session, so the text typed while the application starts the next one
// is queued. That text belongs to the next session: typing faster than one
// character per tick would otherwise lose everything after the first. The two
// are told apart by when the last session ended, as an application driving text
// input starts the next session as soon as it observes the commit.
func withinQueueCarry(lastEndTick, tick int64) bool {
	if lastEndTick == 0 {
		return false
	}
	return tick-lastEndTick <= queueCarryTicks
}

// queueBelongsToNextSession reports whether the queued states are for the
// session about to start.
//
// queueBelongsToNextSession is called from both threads.
func (t *textInputImpl) queueBelongsToNextSession() bool {
	return queuedStatesBelong(t.lastEndTick.Load(), t.queuedTick.Load(), ebiten.Tick())
}

// queuedStatesBelong reports whether states reported at queuedTick are for a
// session starting at tick, which is so in two cases: the previous session
// ended within the carry window, or they were reported in the tick the session
// starts in.
//
// The events of a tick are processed before the game updates, so text typed in
// the same tick a session starts in was reported before the session existed.
// It is what the application is starting the session for.
func queuedStatesBelong(lastEndTick, queuedTick, tick int64) bool {
	return withinQueueCarry(lastEndTick, tick) || queuedTick == tick
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

// send reports a state to the session and reports whether the session took it.
// A state no session can claim is dropped: queueing it would grow the queue for
// as long as the application takes keyboard input without opening a session,
// and would reach whichever session eventually starts.
//
// send is called from the main thread.
func (t *textInputImpl) send(state textInputState) bool {
	if !t.events.isOpen() {
		// States left from an earlier tick are for a session that never
		// started. Dropping them as the tick turns over bounds what is held to
		// a single tick of typing, however long the application goes without
		// opening a session.
		if !t.queueBelongsToNextSession() {
			t.events.clearQueue()
		}
		t.queuedTick.Store(ebiten.Tick())
	}
	return t.events.send(state)
}

// sendComposition reports a composition update from the input method.
//
// sendComposition is called from the main thread.
func (t *textInputImpl) sendComposition(text string, selStartInBytes, selEndInBytes int) {
	t.send(textInputState{
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
	// session, or dropped, ends nothing: treating it as an ending would push
	// the carry deadline forward for as long as the application takes keyboard
	// input, and stale text would reach the next session started.
	if !t.send(textInputState{
		Text:                    text,
		ReplacementStartInBytes: noReplacement,
		ReplacementEndInBytes:   noReplacement,
		CommitKind:              commitRegular,
	}) {
		return
	}
	t.end()
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
