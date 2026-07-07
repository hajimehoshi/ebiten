// Copyright 2025 The Ebitengine Authors
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

func ConvertUTF16CountToByteCount(text string, c int) int {
	return convertUTF16CountToByteCount(text, c)
}

func ConvertByteCountToUTF16Count(text string, c int) int {
	return convertByteCountToUTF16Count(text, c)
}

func FindLineBounds(text string, selStart, selEnd int) (int, int) {
	return findLineBounds(text, selStart, selEnd)
}

func ComputeReplacement(baseline, newText string, caretInBytes int) (string, int, int) {
	return computeReplacement(baseline, newText, caretInBytes)
}

type TextInputEvents = textInputEvents

// TextInputState re-exports the internal state record so white-box tests can
// build the events they send.
type TextInputState = textInputState

// CommitKind and its constants re-export the internal commit-kind enum.
type CommitKind = commitKind

const (
	CommitNone            = commitNone
	CommitWithoutKeyPress = commitWithoutKeyPress
	CommitWithKeyPress    = commitWithKeyPress
)

func (s *TextInputEvents) Start() {
	s.start()
}

func (s *TextInputEvents) End() {
	s.end()
}

func (s *TextInputEvents) ClearQueue() {
	s.clearQueue()
}

func (s *TextInputEvents) Send(state TextInputState) {
	s.send(state)
}

// StartSessionCompositing starts a session on a freshly opened channel, as the
// platform start() does (flushing any queued states), pumps one Update, and
// reports whether the session observed a live composition.
func (s *TextInputEvents) StartSessionCompositing() bool {
	ch, end := s.start()
	sess := &session{ch: ch, end: end}
	_ = sess.Update()
	return sess.IsCompositing()
}
