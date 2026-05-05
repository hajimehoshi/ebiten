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

package textinput

import (
	"image"
)

// Composer drives an IME loop for a single text-input target. It owns
// the per-tick session lifecycle and dispatches composition and commit
// events through caller-registered callbacks.
//
// See examples/textinput in the Ebitengine repository for a complete
// usage example.
type Composer struct {
	// OnNewSession is called when Composer needs to start a new session
	// (initial use, after a commit, or after a platform-side teardown).
	// It must return the caret bounds and the text immediately around the
	// caret so that the IME can position the candidate window and use the
	// surrounding text for prediction. Required.
	OnNewSession func() *SessionOptions

	// OnComposition is called when the IME's preedit changes, including
	// when the session ends and the preedit should be cleared (the
	// Composition argument is then the zero value). Optional.
	OnComposition func(c Composition)

	// OnCommit is called when the IME commits. The caller applies the
	// committed text and replacement range to its own buffer. Optional.
	OnCommit func(c Commit)

	s *session
}

// SessionOptions describes the IME's view of the caret. It is returned
// by Composer.OnNewSession when Composer needs to start a new session.
type SessionOptions struct {
	// CaretBounds is the rectangle of the caret in logical pixels.
	// The platform IME may use it to position the candidate window.
	CaretBounds image.Rectangle

	// TextBeforeCaret is text immediately before the caret, typically the
	// portion of the current line that lies before the caret. The platform
	// IME may use it for prediction and reconversion.
	TextBeforeCaret string

	// TextAfterCaret is text immediately after the caret, typically the
	// portion of the current line that lies after the caret. The platform
	// IME may use it for prediction and reconversion.
	TextAfterCaret string
}

// Composition describes the IME's current preedit text. It is passed to
// Composer.OnComposition.
type Composition struct {
	// Text is the preedit text. Empty when no composition is in progress.
	Text string

	// SelectionStartInBytes is the start position of the IME-side selection
	// within Text, in bytes.
	SelectionStartInBytes int

	// SelectionEndInBytes is the end position of the IME-side selection
	// within Text, in bytes.
	SelectionEndInBytes int
}

// Commit describes the data the IME recorded when committing. It is passed
// to Composer.OnCommit.
type Commit struct {
	// Text is the text the IME committed.
	Text string

	// ReplacementStartInBytes is the start position of the byte range that
	// Text replaces. The position is a byte offset treating
	// TextBeforeCaret followed by TextAfterCaret as a single buffer — not
	// within Text and not within the caller's full text. To translate to a
	// caller-side absolute offset, add the byte offset where TextBeforeCaret
	// began in the caller's text.
	//
	// Equal to ReplacementEndInBytes when the IME does not request a
	// replacement (the typical case).
	ReplacementStartInBytes int

	// ReplacementEndInBytes is the end position of the byte range that Text
	// replaces. See ReplacementStartInBytes for the coordinate system.
	ReplacementEndInBytes int
}

// Update processes IME events for one tick and dispatches them through
// the registered callbacks. It returns true if the IME consumed input
// during this tick — the caller should skip its own key handlers when
// handled is true to avoid double-processing keys (e.g. the Enter that
// committed a composition).
//
// Update may invoke OnCommit and OnComposition multiple times in one call
// if the platform queued multiple compositions between ticks.
func (c *Composer) Update() (handled bool, err error) {
	for {
		if c.s == nil {
			var opts *SessionOptions
			if c.OnNewSession != nil {
				opts = c.OnNewSession()
			}
			var sess *session
			sess, err = startSession(opts)
			if err != nil {
				return false, err
			}
			if sess == nil {
				break
			}
			c.s = sess
		}

		if err = c.s.Update(); err != nil {
			c.s = nil
			c.dispatchComposition(Composition{})
			return handled, err
		}

		if c.s.IsCommitted() {
			if c.OnCommit != nil {
				c.OnCommit(c.s.Commit())
			}
			c.dispatchComposition(Composition{})
			c.s = nil
			handled = true
			continue
		}

		if c.s.IsClosed() {
			c.s = nil
			c.dispatchComposition(Composition{})
			break
		}

		// Active session: composing or idle.
		c.dispatchComposition(c.s.Composition())
		if c.s.IsCompositing() {
			handled = true
		}
		break
	}
	return handled, nil
}

// Cancel ends the current session if any, firing OnComposition with an
// empty composition so the caller can clear its preedit overlay. Call
// this when the field loses focus or when the caller wants to abandon
// any in-progress composition.
func (c *Composer) Cancel() {
	if c.s == nil {
		return
	}
	c.s.Cancel()
	c.s = nil
	c.dispatchComposition(Composition{})
}

func (c *Composer) dispatchComposition(comp Composition) {
	if c.OnComposition != nil {
		c.OnComposition(comp)
	}
}
