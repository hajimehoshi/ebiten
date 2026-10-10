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

package gamepad

import (
	"syscall/js"
	"time"
)

type (
	Gamepads   = gamepads
	JSGamepads = nativeGamepadsImpl
)

// Init starts watching for gamepads, as the gamepad list does once.
func (n *nativeGamepadsImpl) Init(gamepads *gamepads) error {
	return n.init(gamepads)
}

// RemoveEventListener removes the gamepadconnected listener that Init adds and releases its function.
func (n *nativeGamepadsImpl) RemoveEventListener() {
	if n.onGamepadConnected.IsUndefined() {
		return
	}
	js.Global().Call("removeEventListener", "gamepadconnected", n.onGamepadConnected)
	n.onGamepadConnected.Release()
	n.onGamepadConnected = js.Func{}
}

// Update polls navigator.getGamepads when it is due and applies the result to gamepads.
func (n *nativeGamepadsImpl) Update(gamepads *gamepads) error {
	return n.update(gamepads)
}

// SetLastPoll sets the time of the last poll to t.
func (n *nativeGamepadsImpl) SetLastPoll(t time.Time) {
	n.lastPoll = t
}

// AppendGamepadIDs appends the IDs of the gamepads registered in g.
func (g *gamepads) AppendGamepadIDs(ids []ID) []ID {
	return g.appendGamepadIDs(ids)
}

// Get returns the gamepad registered in g with the given ID, or nil.
func (g *gamepads) Get(id ID) *Gamepad {
	return g.get(id)
}
