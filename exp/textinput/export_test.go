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

const QueueCarryTicks = queueCarryTicks

func WithinQueueCarry(lastEndTick, tick int64) bool {
	return withinQueueCarry(lastEndTick, tick)
}

func QueuedStatesBelong(lastEndTick, queuedTick, tick int64) bool {
	return queuedStatesBelong(lastEndTick, queuedTick, tick)
}

type TextInputEvents = textInputEvents

// TextInputState re-exports the internal state record so white-box tests can
// build the events they send.
type TextInputState = textInputState

// CommitKind and its constants re-export the internal commit-kind enum.
type CommitKind = commitKind

const (
	CommitNone               = commitNone
	CommitRegular            = commitRegular
	CommitWithPassthroughKey = commitWithPassthroughKey
)

// SetTick replaces the tick source these events date queue ownership from.
func (s *TextInputEvents) SetTick(tick func() int64) {
	s.m.Lock()
	defer s.m.Unlock()
	s.tick = tick
}

// QueuedStateCount reports how many states are held for a session to take.
func (s *TextInputEvents) QueuedStateCount() int {
	s.m.Lock()
	defer s.m.Unlock()
	return len(s.queuedStates)
}

func (s *TextInputEvents) Start() {
	s.start()
}

func (s *TextInputEvents) End() {
	s.end()
}

func (s *TextInputEvents) ClearQueue() {
	s.clearQueue()
}

func (s *TextInputEvents) Send(state TextInputState) bool {
	return s.send(state)
}

// StartSessionCommit starts a session on a freshly opened channel, as the
// platform start() does (flushing any queued states), pumps one Update, and
// reports the committed text and whether a commit arrived.
func (s *TextInputEvents) StartSessionCommit() (string, bool) {
	ch, end := s.start()
	sess := &session{ch: ch, end: end, events: s}
	_ = sess.Update()
	if !sess.IsCommitted() {
		return "", false
	}
	return sess.Commit().text, true
}

// StartSessionCompositing starts a session on a freshly opened channel, as the
// platform start() does (flushing any queued states), pumps one Update, and
// reports whether the session observed a live composition.
func (s *TextInputEvents) StartSessionCompositing() bool {
	ch, end := s.start()
	sess := &session{ch: ch, end: end, events: s}
	_ = sess.Update()
	return sess.IsCompositing()
}

// DiffSender drives the buffer differ for a session with the given surrounding
// text, and collects the states it sends.
type DiffSender struct {
	events  textInputEvents
	sender  diffSender
	session *session
	ch      <-chan textInputState
}

// NewDiffSender returns a differ for a session seeded with the surrounding text,
// as a platform backend does on start.
func NewDiffSender(textBeforeCaret, textAfterCaret string) *DiffSender {
	d := &DiffSender{}
	d.events.tick = func() int64 {
		return 0
	}
	d.sender.events = &d.events
	ch, end := d.events.start()
	d.ch = ch
	d.session = &session{
		ch:              ch,
		end:             end,
		events:          &d.events,
		textBeforeCaret: textBeforeCaret,
		textAfterCaret:  textAfterCaret,
	}
	d.sender.reset(textBeforeCaret + textAfterCaret)
	return d
}

// TrySend hands the differ a snapshot of the platform buffer.
func (d *DiffSender) TrySend(value string, selStartInUTF16, selEndInUTF16 int, caretAtPreeditEnd bool, kind CommitKind) {
	d.sender.trySend(d.session, value, selStartInUTF16, selEndInUTF16, caretAtPreeditEnd, kind)
}

// StartNextSession starts a session seeded with the surrounding text the
// application holds after applying the previous session's commit, as a platform
// backend does when the application opens the next one. The states queued behind
// that commit are taken over.
func (d *DiffSender) StartNextSession(textBeforeCaret, textAfterCaret string) {
	d.sender.reset(textBeforeCaret + textAfterCaret)
	ch, end := d.events.start()
	d.ch = ch
	d.session = &session{
		ch:              ch,
		end:             end,
		events:          &d.events,
		textBeforeCaret: textBeforeCaret,
		textAfterCaret:  textAfterCaret,
	}
}

// Update drains the session's states, as a tick does.
func (d *DiffSender) Update() error {
	return d.session.Update()
}

// EndByUser ends the events as the platform does for the user's dismissal.
func (d *DiffSender) EndByUser() {
	d.events.endByUser()
}

// End ends the events as the platform does for its own teardown.
func (d *DiffSender) End() {
	d.events.end()
}

// SessionClosedByUser reports whether the session recorded the user's ending.
func (d *DiffSender) SessionClosedByUser() bool {
	return d.session.IsClosedByUser()
}

// SessionClosed reports whether the session is closed.
func (d *DiffSender) SessionClosed() bool {
	return d.session.IsClosed()
}

// Commit returns the commit the session observed, or nil if it observed none.
func (d *DiffSender) Commit() *Commit {
	if !d.session.IsCommitted() {
		return nil
	}
	return d.session.Commit()
}

// Composition returns the preedit the session last observed.
func (d *DiffSender) Composition() *Composition {
	c := d.session.Composition()
	return &c
}

// Drain returns the states sent to the session since the last call.
func (d *DiffSender) Drain() []TextInputState {
	var states []TextInputState
	for {
		select {
		case state, ok := <-d.ch:
			if !ok {
				return states
			}
			states = append(states, state)
		default:
			return states
		}
	}
}

// PendingStateCount reports how many states are held for the next session.
func (d *DiffSender) PendingStateCount() int {
	return d.events.QueuedStateCount()
}

// PlatformStateHandler drives handlePlatformState with its own events and
// sender, mirroring a platform backend whose channel is open.
type PlatformStateHandler struct {
	events        textInputEvents
	sender        diffSender
	legacyCleared bool
	ch            <-chan textInputState
}

// NewPlatformStateHandler returns a handler whose diff baseline is value, as a
// platform backend after seeding.
func NewPlatformStateHandler(value string) *PlatformStateHandler {
	h := &PlatformStateHandler{}
	h.events.tick = func() int64 {
		return 0
	}
	h.sender.events = &h.events
	ch, _ := h.events.start()
	h.ch = ch
	h.sender.reset(value)
	return h
}

// Handle reports a platform state with the caret pinned to the preedit's end,
// and returns whether the platform buffer must be cleared.
func (h *PlatformStateHandler) Handle(value string, selStartInUTF16, selEndInUTF16 int, kind CommitKind, fieldFocused bool) bool {
	return handlePlatformState(&h.events, &h.sender, &h.legacyCleared, value, selStartInUTF16, selEndInUTF16, true, kind, fieldFocused)
}

// RegisterSession installs an active session with the given surrounding text.
func (h *PlatformStateHandler) RegisterSession(textBeforeCaret, textAfterCaret string) {
	h.events.setActiveSession(&session{
		ch:              h.ch,
		end:             h.events.end,
		events:          &h.events,
		textBeforeCaret: textBeforeCaret,
		textAfterCaret:  textAfterCaret,
	})
}

// Drain returns the states delivered since the last call.
func (h *PlatformStateHandler) Drain() []TextInputState {
	var states []TextInputState
	for {
		select {
		case state, ok := <-h.ch:
			if !ok {
				return states
			}
			states = append(states, state)
		default:
			return states
		}
	}
}

// IsOpen reports whether the channel is open.
func (h *PlatformStateHandler) IsOpen() bool {
	return h.events.isOpen()
}

// LegacyCleared reports whether the legacy path cleared the buffer.
func (h *PlatformStateHandler) LegacyCleared() bool {
	return h.legacyCleared
}

// SeedGate re-exports the internal reseeding arbitration.
type SeedGate = seedGate

// Start re-exports seedGate.start.
func (g *SeedGate) Start(value string) int {
	return g.start(value)
}

// Abandon re-exports seedGate.abandon.
func (g *SeedGate) Abandon() {
	g.abandon()
}

// Admit re-exports seedGate.admit.
func (g *SeedGate) Admit(generation int) (resetBaseline bool, ok bool) {
	return g.admit(generation)
}

// PendingValue returns the pending seed.
func (g *SeedGate) PendingValue() string {
	return g.pendingValue
}
