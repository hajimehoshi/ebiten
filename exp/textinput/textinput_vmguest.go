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
	"errors"
	"image"
	"slices"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
	"github.com/hajimehoshi/ebiten/v2/internal/vmguest"
	"github.com/hajimehoshi/ebiten/v2/internal/vmprotocol"
)

// vmGuestTextInput is the text-input backend for a process running as a VM guest. A guest is
// headless, so a session is forwarded to the host, which serves it with text-input states (#3484).
type vmGuestTextInput struct {
	events *textInputEvents

	// mu guards the fields below. Start and the session-end callback run on the game goroutine; the
	// host-message handlers and the post-tick flush run on the guest's serve goroutine.
	mu sync.Mutex

	// currentID is the session the host serves now; 0 means none.
	currentID int64

	// nextID is the last assigned session ID. IDs increase and are unique within the process.
	nextID int64

	// pending holds the session start/end messages to forward to the host after the current tick.
	pending []vmprotocol.GuestMessage
}

var theVMGuestTextInput = vmGuestTextInput{events: &theTextInput.events}

func init() {
	vmguest.AppendPostTickHook(theVMGuestTextInput.flushPending)
	vmguest.RegisterTextInputHandlers(theVMGuestTextInput.handleState, theVMGuestTextInput.handleEnd)
}

func (v *vmGuestTextInput) Start(bounds image.Rectangle, textBeforeCaret, textAfterCaret string) (<-chan textInputState, func()) {
	// The host reads the caret bounds in the guest's client-area device-independent pixels, which mean
	// the same lengths on the host regardless of the platform's native pixel convention.
	cMinX, cMinY := ui.Get().LogicalPositionToClientPositionInDIPs(float64(bounds.Min.X), float64(bounds.Min.Y))
	cMaxX, cMaxY := ui.Get().LogicalPositionToClientPositionInDIPs(float64(bounds.Max.X), float64(bounds.Max.Y))

	v.mu.Lock()
	defer v.mu.Unlock()

	v.nextID++
	id := v.nextID
	v.currentID = id
	v.pending = append(v.pending, vmprotocol.GuestMessage{
		Kind:                     vmprotocol.GuestMessageKindTextInput,
		TextInputID:              id,
		TextInputBounds:          image.Rect(int(cMinX), int(cMinY), int(cMaxX), int(cMaxY)),
		TextInputTextBeforeCaret: textBeforeCaret,
		TextInputTextAfterCaret:  textAfterCaret,
	})

	v.events.end()
	ch, _ := v.events.start()
	return ch, func() {
		v.endSession(id)
	}
}

// markIMEDiscardNeeded is a no-op: a cancelled session always reaches the host as
// GuestMessageKindTextInputEnd, which carries the discard.
func (v *vmGuestTextInput) markIMEDiscardNeeded() {
}

// endSession releases the session: the game observed a commit or cancelled. The host is told to
// stop serving it.
func (v *vmGuestTextInput) endSession(id int64) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.pending = append(v.pending, vmprotocol.GuestMessage{
		Kind:        vmprotocol.GuestMessageKindTextInputEnd,
		TextInputID: id,
	})
	if v.currentID == id {
		v.currentID = 0
		v.events.end()
	}
}

// handleState delivers a host text-input state to the session's event channel. A state for a
// released session is dropped.
func (v *vmGuestTextInput) handleState(id int64, state vmprotocol.TextInputState) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if id != v.currentID {
		return
	}

	var kind commitKind
	switch state.CommitKind {
	case vmprotocol.TextInputCommitKindRegular:
		kind = commitRegular
	case vmprotocol.TextInputCommitKindWithPassthroughKey:
		kind = commitWithPassthroughKey
	default:
		kind = commitNone
	}

	replStart, replEnd := state.ReplacementStartInBytes, state.ReplacementEndInBytes
	if replStart == vmprotocol.TextInputNoReplacement || replEnd == vmprotocol.TextInputNoReplacement {
		replStart, replEnd = noReplacement, noReplacement
	}

	var err error
	if state.Err != "" {
		err = errors.New("textinput: the host failed to serve text inputting: " + state.Err)
	}

	v.events.send(textInputState{
		Text:                             state.Text,
		CompositionSelectionStartInBytes: state.CompositionSelectionStartInBytes,
		CompositionSelectionEndInBytes:   state.CompositionSelectionEndInBytes,
		ReplacementStartInBytes:          replStart,
		ReplacementEndInBytes:            replEnd,
		CommitKind:                       kind,
		Error:                            err,
	})
	if kind.committed() || err != nil {
		v.events.end()
	}
}

// handleEnd ends the session from the host side, closing its event channel.
func (v *vmGuestTextInput) handleEnd(id int64) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.currentID != id {
		return
	}
	v.currentID = 0
	v.events.end()
}

// flushPending forwards the queued session start/end messages, in order.
func (v *vmGuestTextInput) flushPending(enc vmprotocol.GuestMessageEncoder, tick int) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	for i := range v.pending {
		if err := enc.EncodeGuestMessage(&v.pending[i]); err != nil {
			return err
		}
	}
	v.pending = slices.Delete(v.pending, 0, len(v.pending))
	return nil
}
