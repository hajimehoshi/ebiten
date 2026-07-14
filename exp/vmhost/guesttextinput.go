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

package vmhost

import (
	"image"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/vmprotocol"
)

// GuestTextInput is a text-input session the guest's game started. The host serves it as the guest's
// IME: it responds with composition updates and a commit, or ends it. A host receives one through the
// [NewGuestSessionOptions] OnTextInput handler.
//
// The methods are safe to call from any goroutine. Like the input injectors, a response is observed
// by the guest at its next tick; a response to a session the guest has already released is discarded.
type GuestTextInput struct {
	g  *GuestSession
	id int64

	bounds          image.Rectangle
	textBeforeCaret string
	textAfterCaret  string

	closed atomic.Bool
}

// CaretBounds returns the guest caret's rectangle, in the guest's outside-screen device-independent
// pixels, e.g. for placing a candidate window near it.
func (t *GuestTextInput) CaretBounds() image.Rectangle {
	return t.bounds
}

// TextBeforeCaret returns the text immediately before the guest's caret.
func (t *GuestTextInput) TextBeforeCaret() string {
	return t.textBeforeCaret
}

// TextAfterCaret returns the text immediately after the guest's caret.
func (t *GuestTextInput) TextAfterCaret() string {
	return t.textAfterCaret
}

// IsClosed reports whether the session has ended: the host committed or ended it, or the guest
// released it (its game cancelled text inputting).
func (t *GuestTextInput) IsClosed() bool {
	return t.closed.Load()
}

// markClosed records that the session ended.
func (t *GuestTextInput) markClosed() {
	t.closed.Store(true)
}

// SetComposition sets the composition (preedit) text and the selection within it, as byte offsets
// into text.
func (t *GuestTextInput) SetComposition(text string, selectionStartInBytes, selectionEndInBytes int) {
	msg := t.g.takeMessage()
	msg.Kind = vmprotocol.HostMessageKindTextInputState
	msg.TextInputID = t.id
	msg.TextInputState = vmprotocol.TextInputState{
		Text:                             text,
		CompositionSelectionStartInBytes: selectionStartInBytes,
		CompositionSelectionEndInBytes:   selectionEndInBytes,
		ReplacementStartInBytes:          vmprotocol.TextInputNoReplacement,
		ReplacementEndInBytes:            vmprotocol.TextInputNoReplacement,
	}
	t.g.postMessage(msg)
}

// Commit commits text: the guest inserts it at its caret, and the session ends.
func (t *GuestTextInput) Commit(text string) {
	t.CommitWithOptions(text, nil)
}

// GuestTextInputCommitOptions represents options for [GuestTextInput.CommitWithOptions]. The zero
// value commits like [GuestTextInput.Commit].
type GuestTextInputCommitOptions struct {
	// ReplaceSurroundingText makes the commit replace the byte range [ReplacementStartInBytes,
	// ReplacementEndInBytes) of the joined [GuestTextInput.TextBeforeCaret] +
	// [GuestTextInput.TextAfterCaret] buffer with the committed text, rather than inserting the text
	// at the caret.
	ReplaceSurroundingText  bool
	ReplacementStartInBytes int
	ReplacementEndInBytes   int

	// PassthroughKey marks the commit's triggering key press as also passing through to the guest's
	// game, so the guest's text-input handling leaves the key for the game to act on. The key press
	// itself is not injected: deliver it like any input, e.g. with [GuestSession.PressKey].
	PassthroughKey bool
}

// CommitWithOptions commits text like [GuestTextInput.Commit], with options. options can be nil,
// which is equivalent to Commit.
func (t *GuestTextInput) CommitWithOptions(text string, options *GuestTextInputCommitOptions) {
	replStart, replEnd := vmprotocol.TextInputNoReplacement, vmprotocol.TextInputNoReplacement
	var passthroughKey bool
	if options != nil {
		if options.ReplaceSurroundingText {
			replStart, replEnd = options.ReplacementStartInBytes, options.ReplacementEndInBytes
		}
		passthroughKey = options.PassthroughKey
	}
	t.sendCommitState(text, replStart, replEnd, passthroughKey)
}

// sendCommitState sends a committed state with an explicit replacement range and passthrough-key
// flag, ending the session.
func (t *GuestTextInput) sendCommitState(text string, replacementStartInBytes, replacementEndInBytes int, passthroughKey bool) {
	t.markClosed()
	kind := vmprotocol.TextInputCommitKindRegular
	if passthroughKey {
		kind = vmprotocol.TextInputCommitKindWithPassthroughKey
	}
	msg := t.g.takeMessage()
	msg.Kind = vmprotocol.HostMessageKindTextInputState
	msg.TextInputID = t.id
	msg.TextInputState = vmprotocol.TextInputState{
		Text:                    text,
		ReplacementStartInBytes: replacementStartInBytes,
		ReplacementEndInBytes:   replacementEndInBytes,
		CommitKind:              kind,
	}
	t.g.postMessage(msg)
}

// End ends the session without committing, like a platform IME tearing down text inputting. The
// guest discards the composition shown so far.
func (t *GuestTextInput) End() {
	t.markClosed()
	msg := t.g.takeMessage()
	msg.Kind = vmprotocol.HostMessageKindEndTextInput
	msg.TextInputID = t.id
	t.g.postMessage(msg)
}

// EndWithError ends the session, delivering err as a text-inputting error: the guest observes the
// failure, as when a platform IME fails. err must not be nil.
func (t *GuestTextInput) EndWithError(err error) {
	t.markClosed()
	msg := t.g.takeMessage()
	msg.Kind = vmprotocol.HostMessageKindTextInputState
	msg.TextInputID = t.id
	msg.TextInputState = vmprotocol.TextInputState{
		Err: err.Error(),
	}
	t.g.postMessage(msg)
}
