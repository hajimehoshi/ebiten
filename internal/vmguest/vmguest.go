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

	"github.com/hajimehoshi/ebiten/v2/internal/vmprotocol"
)

var (
	postTickHooksMu sync.Mutex
	postTickHooks   []func(vmprotocol.GuestMessageEncoder) error
)

// AppendPostTickHook registers a hook run after each guest tick, before the tick's concluding message,
// so a guest subsystem can forward its own messages for that tick. Hooks are registered at
// initialization, before the guest serves.
func AppendPostTickHook(hook func(vmprotocol.GuestMessageEncoder) error) {
	postTickHooksMu.Lock()
	defer postTickHooksMu.Unlock()
	postTickHooks = append(postTickHooks, hook)
}

// RunPostTickHooks runs the registered post-tick hooks with the given encoder. The guest UI backend
// calls it after a successful tick.
func RunPostTickHooks(enc vmprotocol.GuestMessageEncoder) error {
	postTickHooksMu.Lock()
	hooks := postTickHooks
	postTickHooksMu.Unlock()
	for _, hook := range hooks {
		if err := hook(enc); err != nil {
			return err
		}
	}
	return nil
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
