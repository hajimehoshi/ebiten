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

// Package vmguest is the guest-side runtime coordination for virtualization: the registries through
// which guest subsystems (such as audio) participate in the guest's serve loop. It lets the UI backend
// run those subsystems without depending on them, and keeps this glue out of vmprotocol, which is only
// the wire definition.
package vmguest

import (
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/vmprotocol"
)

// guestState is whether this process runs as a VM guest. It stays
// guestStateUndetermined until a run entry selects its UI backend.
const (
	guestStateUndetermined int32 = iota
	guestStateGuest
	guestStateNotGuest
)

var guestState atomic.Int32

// MarkGuest records whether this process runs as a VM guest. A run entry
// calls it when selecting its UI backend, before the game runs. MarkGuest
// panics when the guest mode is already determined: a game can run only once
// in one process.
func MarkGuest(guest bool) {
	state := guestStateNotGuest
	if guest {
		state = guestStateGuest
	}
	if !guestState.CompareAndSwap(guestStateUndetermined, state) {
		panic("vmguest: MarkGuest is called more than once")
	}
}

// IsGuest reports whether this process runs as a VM guest. IsGuest panics when
// the guest mode is not determined yet: it must not be called before a run
// entry selects its UI backend.
func IsGuest() bool {
	switch guestState.Load() {
	case guestStateGuest:
		return true
	case guestStateNotGuest:
		return false
	default:
		panic("vmguest: IsGuest is called before the guest mode is determined")
	}
}

var (
	postTickHooksMu sync.Mutex
	postTickHooks   []func(vmprotocol.GuestMessageEncoder, int) error
)

// AppendPostTickHook registers a hook run after each guest tick, before the tick's concluding message,
// so a guest subsystem can forward its own messages for that tick. The hook receives the guest's
// ebiten.Tick() during that tick, to stamp the messages it forwards. Hooks are registered at
// initialization, before the guest serves.
func AppendPostTickHook(hook func(vmprotocol.GuestMessageEncoder, int) error) {
	postTickHooksMu.Lock()
	defer postTickHooksMu.Unlock()
	postTickHooks = append(postTickHooks, hook)
}

// RunPostTickHooks runs the registered post-tick hooks with the given encoder and the guest's
// ebiten.Tick() during the tick just run. The guest UI backend calls it after a successful tick.
func RunPostTickHooks(enc vmprotocol.GuestMessageEncoder, tick int) error {
	postTickHooksMu.Lock()
	hooks := postTickHooks
	postTickHooksMu.Unlock()
	for _, hook := range hooks {
		if err := hook(enc, tick); err != nil {
			return err
		}
	}
	return nil
}

var (
	textInputHandlersMu   sync.Mutex
	textInputStateHandler func(id int64, state vmprotocol.TextInputState)
	textInputEndHandler   func(id int64)
)

// RegisterTextInputHandlers registers the handlers answering HostMessageKindTextInputState and
// HostMessageKindEndTextInput. The text-input subsystem registers them at initialization, so the UI
// backend can deliver text-input messages without depending on it.
func RegisterTextInputHandlers(state func(id int64, state vmprotocol.TextInputState), end func(id int64)) {
	textInputHandlersMu.Lock()
	defer textInputHandlersMu.Unlock()
	textInputStateHandler = state
	textInputEndHandler = end
}

// RunTextInputStateHandler runs the registered text-input state handler. When none is registered the
// state is dropped, since a guest without the text-input subsystem starts no session.
func RunTextInputStateHandler(id int64, state vmprotocol.TextInputState) {
	textInputHandlersMu.Lock()
	handler := textInputStateHandler
	textInputHandlersMu.Unlock()
	if handler == nil {
		return
	}
	handler(id, state)
}

// RunTextInputEndHandler runs the registered text-input end handler. When none is registered the
// message is dropped.
func RunTextInputEndHandler(id int64) {
	textInputHandlersMu.Lock()
	handler := textInputEndHandler
	textInputHandlersMu.Unlock()
	if handler == nil {
		return
	}
	handler(id)
}

// AudioReadFunc reads player id's samples into buf, like io.Reader.Read, and reports whether the
// player's source has ended (an unknown player reads as ended). The samples are raw (the volume is
// reported separately, not applied).
type AudioReadFunc func(id int64, buf []byte) (n int, eof bool)

var (
	audioReadHandlerMu sync.Mutex
	audioReadHandler   AudioReadFunc
)

// RegisterAudioReadHandler registers the handler that answers HostMessageKindReadAudio. A guest
// subsystem (audio) registers it at initialization, so the UI backend can answer audio reads without
// depending on the audio packages.
func RegisterAudioReadHandler(handler AudioReadFunc) {
	audioReadHandlerMu.Lock()
	defer audioReadHandlerMu.Unlock()
	audioReadHandler = handler
}

// RunAudioReadHandler runs the registered audio-read handler. When none is registered it reports
// end-of-stream, since the host only asks for players the audio subsystem reported in the first place.
func RunAudioReadHandler(id int64, buf []byte) (n int, eof bool) {
	audioReadHandlerMu.Lock()
	handler := audioReadHandler
	audioReadHandlerMu.Unlock()
	if handler == nil {
		return 0, true
	}
	return handler(id, buf)
}
