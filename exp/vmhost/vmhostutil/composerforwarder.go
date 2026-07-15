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
	"slices"

	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
)

// pendingCommit is a commit the host's IME produced while no guest session existed, queued for the
// guest's next session.
type pendingCommit struct {
	text string

	// textBeforeCaret and textAfterCaret are the surrounding text the commit was composed against;
	// the guest's next session must report the same for the replacement range to apply as is.
	textBeforeCaret string
	textAfterCaret  string

	// replacementStartInBytes and replacementEndInBytes locate the replaced region in the joined
	// textBeforeCaret + textAfterCaret buffer, as in [vmhost.GuestTextInputCommitOptions].
	replacementStartInBytes int
	replacementEndInBytes   int

	passthroughKey bool
}

// ComposerForwarder serves at most one guest text-input session with the host's own IME: it starts
// a host text-input session at the guest's caret and forwards its compositions and its commit to
// the guest until either side ends the session. The host session takes the platform IME over from
// any text inputting the host itself runs, so forward a session only while the guest is what the
// user is typing into.
//
// A ComposerForwarder is a buffer between the host's IME and the guest: the user's typing is
// captured continuously and delivered once the guest can receive it. [ComposerForwarder.Reset]
// discards the input not yet delivered.
//
// The zero value is ready to use. A ComposerForwarder is not safe for concurrent use: call it from
// the host's game loop, like the [textinput.Composer] it drives.
//
// A ComposerForwarder is built only on Ebitengine's public API, so a host can also write its own
// variant.
type ComposerForwarder struct {
	// target is the guest session being served, nil when idle or bridging. bounds is the host-side
	// caret rectangle.
	target *vmhost.GuestTextInput
	bounds image.Rectangle

	composer          textinput.Composer
	composerCallbacks bool

	// lastText, lastSelStart, and lastSelEnd are the composition last observed, so only changes are
	// sent: the composer reports the composition every update, and resending identical states would
	// make the guest treat the IME as consuming input continuously.
	lastText     string
	lastSelStart int
	lastSelEnd   int

	// bridging reports that a commit ended the guest session and the guest's next session has not
	// been forwarded yet. While bridging, target is nil but the host session stays alive.
	bridging bool

	// predictedBefore and predictedAfter hold the surrounding text the guest is expected to report
	// once it restarts text inputting after a commit.
	predictedBefore string
	predictedAfter  string

	// pendingCommits queues the commits produced while bridging, delivered one per arriving guest
	// session (each commit ends the session it is delivered to).
	pendingCommits []pendingCommit

	// bridgeUpdates counts the Update calls since the bridge last made progress, to bound how long
	// the host session outlives a guest that never restarts text inputting.
	bridgeUpdates int
}

// Forward makes t the forwarded session, superseding the current one and ending its host session.
// caretBounds is in the host's logical pixels.
func (f *ComposerForwarder) Forward(t *vmhost.GuestTextInput, caretBounds image.Rectangle) {
	f.initComposerCallbacks()
	if f.target == t {
		f.bounds = caretBounds
		return
	}
	if f.bridging {
		f.resolveBridge(t, caretBounds)
		return
	}
	f.reset()
	f.target = t
	f.bounds = caretBounds
}

// resolveBridge advances a bridge with the guest's next session: a queued commit is delivered to
// it (which ends it, keeping the bridge up for the session after), or it becomes the served target
// with the live host session carried over.
func (f *ComposerForwarder) resolveBridge(t *vmhost.GuestTextInput, caretBounds image.Rectangle) {
	f.bounds = caretBounds

	// A session the guest has already released serves nothing; keep bridging for the next one. The
	// timeout is not reset: this session made no progress, and the timeout must still bound a bridge
	// fed nothing but released sessions.
	if t.IsClosed() {
		return
	}
	f.bridgeUpdates = 0

	if len(f.pendingCommits) > 0 {
		// Only one commit can be delivered: a commit ends the session it is delivered to, so the
		// rest of the queue waits for the guest's next session. Should the guest never start one,
		// the bridge timeout in Update discards the queue.
		pc := f.pendingCommits[0]
		f.pendingCommits = slices.Delete(f.pendingCommits, 0, 1)
		opts := &vmhost.GuestTextInputCommitOptions{PassthroughKey: pc.passthroughKey}
		if t.TextBeforeCaret() == pc.textBeforeCaret && t.TextAfterCaret() == pc.textAfterCaret {
			// The guest resolves the range against the surrounding text this session reported, which
			// is the very buffer the range was computed in, so the range denotes the intended content.
			opts.ReplaceSurroundingText = true
			opts.ReplacementStartInBytes = pc.replacementStartInBytes
			opts.ReplacementEndInBytes = pc.replacementEndInBytes
		} else {
			// The session's surrounding text is not what the commit was composed against (the guest
			// edited its text on its own): insert at the caret rather than replace a range that no
			// longer exists, and restart the host session on the real text. The prediction is
			// re-anchored to it; later queued commits then also take this insertion path.
			f.composer.Cancel()
			f.predictedBefore = t.TextBeforeCaret() + pc.text
			f.predictedAfter = t.TextAfterCaret()
		}
		t.CommitWithOptions(pc.text, opts)
		return
	}

	// No commit is owed: serve this session.
	f.bridging = false
	if t.TextBeforeCaret() != f.predictedBefore || t.TextAfterCaret() != f.predictedAfter {
		// The prediction missed (the guest edited its text on its own): restart the host session on
		// the real surrounding text. An in-flight composition is discarded, as when a session is
		// superseded.
		f.reset()
		f.target = t
		f.bounds = caretBounds
		return
	}
	f.predictedBefore, f.predictedAfter = "", ""
	f.target = t
	// A composition produced during the bridge reached no guest session; deliver it so the guest
	// renders the preedit.
	if f.lastText != "" || f.lastSelStart != 0 || f.lastSelEnd != 0 {
		t.SetComposition(f.lastText, f.lastSelStart, f.lastSelEnd)
	}
}

// Update advances the forwarding: it pumps the host's text inputting and forwards what it produced.
// Call Update once per tick (Update) while the forwarder is in use.
//
// Update reports whether the host's IME consumed input during this tick, like
// [textinput.Composer.Update]: the host should skip its own key handlers when handled is true.
// Input forwarded to the guest is not affected: keep forwarding it regardless, as the guest skips
// what its own session consumed.
func (f *ComposerForwarder) Update() (handled bool) {
	f.initComposerCallbacks()

	if f.bridging {
		// A guest normally restarts text inputting within a few ticks of a commit; one that still
		// has not after this many Update calls (one second at the default tick rate) stopped
		// instead, so the bridge ends, discarding the queued commits — they have no destination.
		const bridgeTimeoutUpdates = 60
		f.bridgeUpdates++
		if f.bridgeUpdates > bridgeTimeoutUpdates {
			f.reset()
			return false
		}
		handled, err := f.composer.Update()
		if err != nil {
			// No guest session exists to deliver the error to; the guest's next session starts
			// afresh.
			f.reset()
		}
		return handled
	}

	t := f.target
	if t == nil {
		return false
	}

	// The guest released the session (its game cancelled text inputting), or its guest session ended.
	if t.IsClosed() {
		f.reset()
		return false
	}

	handled, err := f.composer.Update()
	if err != nil {
		// The error's consumer is the guest session: it ends with the error, as a session on a native
		// IME would. A commit during this Update may have ended the session and started a bridge; the
		// error ends that too.
		if t := f.target; t != nil && !t.IsClosed() {
			t.EndWithError(err)
		}
		f.reset()
	}
	return handled
}

// Reset returns the forwarder to its idle state, as if freshly created: the guest session being
// served, if any, is forgotten, and the host's text-input session ends, discarding an in-progress
// composition and any commits not yet delivered. Call it when the guest goes away or the host
// takes the IME back for its own text inputting; a later [ComposerForwarder.Forward] starts
// afresh.
func (f *ComposerForwarder) Reset() {
	f.reset()
}

// initComposerCallbacks registers the composer callbacks once.
func (f *ComposerForwarder) initComposerCallbacks() {
	if f.composerCallbacks {
		return
	}
	f.composerCallbacks = true
	f.composer.OnNewSession = f.onNewSession
	f.composer.OnComposition = f.onComposition
	f.composer.OnCommit = f.onCommit
}

// reset releases the host session, if any, and forgets the target and any bridge. The host's
// pending composition and queued commits are discarded, not committed: the target is gone or being
// replaced.
func (f *ComposerForwarder) reset() {
	f.composer.Cancel()
	f.target = nil
	f.lastText, f.lastSelStart, f.lastSelEnd = "", 0, 0
	f.bridging = false
	f.predictedBefore, f.predictedAfter = "", ""
	f.pendingCommits = slices.Delete(f.pendingCommits, 0, len(f.pendingCommits))
	f.bridgeUpdates = 0
}

// onNewSession is the composer's OnNewSession callback: the host session mirrors the guest's caret.
// While bridging, it mirrors the predicted post-commit state, so the host session survives the
// round trip to the guest's next session.
func (f *ComposerForwarder) onNewSession() *textinput.SessionOptions {
	if f.bridging {
		return &textinput.SessionOptions{
			CaretBounds:     f.bounds,
			TextBeforeCaret: f.predictedBefore,
			TextAfterCaret:  f.predictedAfter,
		}
	}
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

// onComposition is the composer's OnComposition callback, forwarding composition changes.
// While bridging, the change is only recorded: resolveBridge delivers the latest composition once the
// guest's next session arrives.
func (f *ComposerForwarder) onComposition(c *textinput.Composition) {
	t := f.target
	if !f.bridging && (t == nil || t.IsClosed()) {
		return
	}
	text := c.Text()
	selStart, selEnd := c.SelectionRangeInBytes()
	if text == f.lastText && selStart == f.lastSelStart && selEnd == f.lastSelEnd {
		return
	}
	f.lastText, f.lastSelStart, f.lastSelEnd = text, selStart, selEnd
	if t != nil && !t.IsClosed() {
		t.SetComposition(text, selStart, selEnd)
	}
}

// onCommit is the composer's OnCommit callback, forwarding the commit. Delivering a commit
// closes the target, so a bridge follows it until the guest's next session is forwarded; a commit
// produced while already bridging is queued for that session instead.
func (f *ComposerForwarder) onCommit(c *textinput.Commit) {
	before, after := c.SurroundingText()

	if f.bridging {
		f.pendingCommits = append(f.pendingCommits, pendingCommit{
			text:                    c.Text(),
			textBeforeCaret:         f.predictedBefore,
			textAfterCaret:          f.predictedAfter,
			replacementStartInBytes: len(before),
			replacementEndInBytes:   len(f.predictedBefore) + len(f.predictedAfter) - len(after),
			passthroughKey:          c.HasPassthroughKey(),
		})
		f.predictedBefore = before + c.Text()
		f.predictedAfter = after
		return
	}

	t := f.target
	if t == nil || t.IsClosed() {
		return
	}
	// Recover the replacement range in the joined surrounding-text buffer from what remains around
	// the replacement.
	joinedLen := len(t.TextBeforeCaret()) + len(t.TextAfterCaret())
	t.CommitWithOptions(c.Text(), &vmhost.GuestTextInputCommitOptions{
		ReplaceSurroundingText:  true,
		ReplacementStartInBytes: len(before),
		ReplacementEndInBytes:   joinedLen - len(after),
		PassthroughKey:          c.HasPassthroughKey(),
	})
	f.target = nil
	f.bridging = true
	f.predictedBefore = before + c.Text()
	f.predictedAfter = after
	f.bridgeUpdates = 0
}
