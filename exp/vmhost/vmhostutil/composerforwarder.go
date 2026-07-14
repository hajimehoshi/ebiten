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

// Package vmhostutil provides utilities for virtualization hosts.
//
// This package is experimental and the API might be changed in the future.
package vmhostutil

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
)

// ComposerForwarder serves at most one guest text-input session with the host's own IME: it starts
// a host text-input session at the guest's caret and forwards its compositions and its commit to
// the guest until either side ends the session. The host session takes the platform IME over from
// any text inputting the host itself runs, so forward a session only while the guest is what the
// user is typing into.
//
// The zero value is ready to use. A ComposerForwarder is not safe for concurrent use: call it from
// the host's game loop, like the [textinput.Composer] it drives.
//
// A ComposerForwarder is built only on Ebitengine's public API, so a host can also write its own
// variant.
type ComposerForwarder struct {
	// target is the guest session being served, nil when idle. bounds is the host-side caret
	// rectangle.
	target *vmhost.GuestTextInput
	bounds image.Rectangle

	composer          textinput.Composer
	composerCallbacks bool

	// lastText, lastSelStart, and lastSelEnd are the composition last forwarded, so only changes are
	// sent: the composer reports the composition every update, and resending identical states would
	// make the guest treat the IME as consuming input continuously.
	lastText     string
	lastSelStart int
	lastSelEnd   int
}

// Forward makes t the forwarded session, superseding the current one and ending its host session.
// caretBounds is in the host's logical pixels.
func (f *ComposerForwarder) Forward(t *vmhost.GuestTextInput, caretBounds image.Rectangle) {
	f.initComposerCallbacks()
	if f.target == t {
		f.bounds = caretBounds
		return
	}
	f.stop()
	f.target = t
	f.bounds = caretBounds
}

// Update advances the forwarding: it pumps the host's text inputting and forwards what it produced.
// Call Update once per tick (Update) while the forwarder is in use.
func (f *ComposerForwarder) Update() {
	f.initComposerCallbacks()

	t := f.target
	if t == nil {
		return
	}

	// The guest released the session (its game cancelled text inputting), or its guest session ended.
	if t.IsClosed() {
		f.stop()
		return
	}

	if _, err := f.composer.Update(); err != nil {
		t.EndWithError(err)
		f.stop()
	}
}

// initComposerCallbacks registers the composer callbacks once.
func (f *ComposerForwarder) initComposerCallbacks() {
	if f.composerCallbacks {
		return
	}
	f.composerCallbacks = true
	f.composer.OnNewSession = f.newSession
	f.composer.OnComposition = f.dispatchComposition
	f.composer.OnCommit = f.dispatchCommit
}

// stop releases the host session, if any, and forgets the target. The host's pending composition is
// discarded, not committed: the target is gone or being replaced.
func (f *ComposerForwarder) stop() {
	f.composer.Cancel()
	f.target = nil
	f.lastText, f.lastSelStart, f.lastSelEnd = "", 0, 0
}

// newSession is the composer's OnNewSession callback: the host session mirrors the guest's caret.
// Once the target is served (a commit closed it), no further session starts.
func (f *ComposerForwarder) newSession() *textinput.SessionOptions {
	t := f.target
	if t == nil || t.IsClosed() {
		return nil
	}
	return &textinput.SessionOptions{
		CaretBounds:     f.bounds,
		TextBeforeCaret: t.TextBeforeCaret(),
		TextAfterCaret:  t.TextAfterCaret(),
	}
}

// dispatchComposition is the composer's OnComposition callback, forwarding composition changes.
func (f *ComposerForwarder) dispatchComposition(c *textinput.Composition) {
	t := f.target
	if t == nil || t.IsClosed() {
		return
	}
	text := c.Text()
	selStart, selEnd := c.SelectionRangeInBytes()
	if text == f.lastText && selStart == f.lastSelStart && selEnd == f.lastSelEnd {
		return
	}
	t.SetComposition(text, selStart, selEnd)
	f.lastText, f.lastSelStart, f.lastSelEnd = text, selStart, selEnd
}

// dispatchCommit is the composer's OnCommit callback, forwarding the commit. It closes the target,
// so newSession starts no host session until the guest's next session is forwarded.
func (f *ComposerForwarder) dispatchCommit(c *textinput.Commit) {
	t := f.target
	if t == nil || t.IsClosed() {
		return
	}
	// Recover the replacement range in the joined surrounding-text buffer from what remains around
	// the replacement.
	before, after := c.SurroundingText()
	joinedLen := len(t.TextBeforeCaret()) + len(t.TextAfterCaret())
	t.CommitWithOptions(c.Text(), &vmhost.GuestTextInputCommitOptions{
		ReplaceSurroundingText:  true,
		ReplacementStartInBytes: len(before),
		ReplacementEndInBytes:   joinedLen - len(after),
		PassthroughKey:          c.HasPassthroughKey(),
	})
}
