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
// A session can span multiple ticks. While one is in progress, do not
// edit the target buffer except in response to [Composer.OnCommit]: a
// commit is applied relative to the surrounding text captured by
// [Composer.OnNewSession], so editing underneath a live session can leave
// that position stale. Call [Composer.Confirm] before a caller-driven edit.
//
// See examples/textinput in the Ebitengine repository for a complete
// usage example.
type Composer struct {
	// OnNewSession is called when Composer needs to start a new session
	// (initial use, after a commit, or after a platform-side teardown).
	// It must return the caret bounds and the text immediately around the
	// caret so that the IME can position the candidate window and use the
	// surrounding text for prediction, or nil to start no session for now
	// (Update asks again later). Required.
	OnNewSession func() *SessionOptions

	// OnComposition is called when the IME's preedit changes, including
	// when the session ends and the preedit should be cleared (the
	// Composition is then the zero value). Optional.
	OnComposition func(c *Composition)

	// OnCommit is called when the IME commits. The caller applies the
	// committed text to its own buffer. Optional.
	OnCommit func(c *Commit)

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
	text     string
	selStart int
	selEnd   int
}

// Text returns the preedit text. Empty when no composition is in progress.
func (c *Composition) Text() string {
	return c.text
}

// SelectionRangeInBytes returns the IME-side selection within Text as a
// half-open byte range [start, end).
func (c *Composition) SelectionRangeInBytes() (start, end int) {
	return c.selStart, c.selEnd
}

// Commit describes the data the IME recorded when committing. It is passed
// to [Composer.OnCommit].
//
// In the typical case the caller inserts Text at the current caret. When
// the IME requested a replacement of part of the surrounding text — see
// [Commit.IsSurroundingTextReplaced] and [Commit.SurroundingText] — the
// caller replaces the original [SessionOptions.TextBeforeCaret] +
// [SessionOptions.TextAfterCaret] slice with prefix + Text + suffix, where
// prefix and suffix come from SurroundingText, and places the caret at
// the boundary after Text.
type Commit struct {
	text            string
	textBeforeCaret string
	textAfterCaret  string
	replStart       int
	replEnd         int
	passthroughKey  bool
}

// Text returns the text the IME committed.
func (c *Commit) Text() string {
	return c.text
}

// HasPassthroughKey reports whether the commit's triggering key press also
// passes through to the game's own key input, in which case the caller
// usually still acts on that key.
func (c *Commit) HasPassthroughKey() bool {
	return c.passthroughKey
}

// IsSurroundingTextReplaced reports whether the IME requested a
// replacement of the before-caret or after-caret region. Both are false in
// the typical case (the IME just inserted Text at the caret); the caller
// can simply insert Text at its current caret without consulting
// [Commit.SurroundingText].
func (c *Commit) IsSurroundingTextReplaced() (before, after bool) {
	preLen := len(c.textBeforeCaret)
	return c.replStart != preLen, c.replEnd != preLen
}

// SurroundingText returns the prefix and suffix that remain after the
// IME's replacement region is removed from the joined
// [SessionOptions.TextBeforeCaret] + [SessionOptions.TextAfterCaret]
// slice. The two strings equal the original SessionOptions values when
// [Commit.IsSurroundingTextReplaced] returns (false, false).
func (c *Commit) SurroundingText() (before, after string) {
	preLen := len(c.textBeforeCaret)
	if c.replStart <= preLen {
		before = c.textBeforeCaret[:c.replStart]
	} else {
		before = c.textBeforeCaret + c.textAfterCaret[:c.replStart-preLen]
	}
	if c.replEnd >= preLen {
		after = c.textAfterCaret[c.replEnd-preLen:]
	} else {
		after = c.textBeforeCaret[c.replEnd:] + c.textAfterCaret
	}
	return
}

// Update processes IME events for one tick and dispatches them through
// the registered callbacks. It returns true if the IME consumed input
// during this tick — the caller should skip its own key handlers when
// handled is true to avoid double-processing keys.
//
// A commit delivered with a key press that the game also receives leaves
// handled false, so the caller still acts on that key.
//
// Update may invoke OnCommit and OnComposition multiple times in one call
// if the platform queued multiple compositions between ticks.
//
// Update should be called every tick (Update) during editing. It is safe to
// call more often than once per tick.
func (c *Composer) Update() (handled bool, err error) {
	for {
		if c.s == nil {
			var opts *SessionOptions
			if c.OnNewSession != nil {
				opts = c.OnNewSession()
				if opts == nil {
					break
				}
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
			c.dispatchEmptyComposition()
			return handled, err
		}

		if c.s.IsCommitted() {
			if c.OnCommit != nil {
				c.OnCommit(c.s.Commit())
			}
			c.dispatchEmptyComposition()
			// A commit whose key passes through to the game leaves handled false.
			if !c.s.IsCommittedWithPassthroughKey() {
				handled = true
			}
			c.s = nil
			continue
		}

		if c.s.IsClosed() {
			c.s = nil
			c.dispatchEmptyComposition()
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

// Confirm ends the current session if any. Any in-progress composition is
// committed through OnCommit as if the IME had committed it, then
// OnComposition is fired with an empty composition so the caller can clear
// its preedit overlay.
func (c *Composer) Confirm() {
	c.end(true)
}

// Cancel ends the current session if any, discarding an in-progress
// composition: unlike [Composer.Confirm], nothing is committed.
// OnComposition is fired with an empty composition so the caller can clear
// its preedit overlay.
func (c *Composer) Cancel() {
	c.end(false)
}

// end ends the current session if any, committing an in-progress
// composition through OnCommit only when commit is true.
func (c *Composer) end(commit bool) {
	if c.s == nil {
		return
	}
	if commit && c.OnCommit != nil && c.s.loadComposition().text != "" {
		c.OnCommit(c.s.compositionAsCommit())
	}
	c.s.Cancel()
	c.s = nil
	c.dispatchEmptyComposition()
}

// Finish ends the current session if any.
//
// Deprecated: Use [Composer.Confirm] instead.
func (c *Composer) Finish() {
	c.Confirm()
}

func (c *Composer) dispatchComposition(comp Composition) {
	if c.OnComposition != nil {
		c.OnComposition(&comp)
	}
}

func (c *Composer) dispatchEmptyComposition() {
	if c.OnComposition != nil {
		c.OnComposition(&Composition{})
	}
}
