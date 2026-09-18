// Copyright 2026 The Ebitengine Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ui

import (
	"slices"
	"syscall/js"
	"testing"
)

func newKeyDownEventForTesting() (js.Value, js.Func) {
	e := js.Global().Get("Object").New()
	e.Set("type", "keydown")
	e.Set("key", "a")
	e.Set("code", "KeyA")
	e.Set("repeat", false)
	getModifierState := js.FuncOf(func(this js.Value, args []js.Value) any {
		return args[0].String() == "CapsLock"
	})
	e.Set("getModifierState", getModifierState)
	return e, getModifierState
}

func TestJSInputAppliesOneEventAtomically(t *testing.T) {
	var u UserInterface
	const tick = 7
	u.inputTime.Store(int64(NewInputTimeFromTick(tick)))
	e, getModifierState := newKeyDownEventForTesting()
	defer getModifierState.Release()
	if err := u.updateInputFromEvent(e); err != nil {
		t.Fatal(err)
	}

	var got InputState
	u.inputState.copyAndReset(&got)
	if !got.IsKeyJustPressed(KeyA, tick) {
		t.Error("the key press was not recorded in the event's tick")
	}
	if !slices.Equal(got.Runes, []rune{'a'}) {
		t.Errorf("Runes: got %q, want %q", got.Runes, []rune{'a'})
	}
	if got.CapsLock != LockKeyStateOn || got.NumLock != LockKeyStateOff {
		t.Errorf("lock keys: got (%v, %v), want (%v, %v)", got.CapsLock, got.NumLock, LockKeyStateOn, LockKeyStateOff)
	}
}

func TestJSInputEventCannotCrossTickBoundary(t *testing.T) {
	var u UserInterface
	const currentTick = 11
	u.tick.Store(currentTick)
	e, getModifierState := newKeyDownEventForTesting()
	defer getModifierState.Release()

	u.inputMu.Lock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := u.updateInputFromEvent(e); err != nil {
			t.Error(err)
		}
	}()

	var before InputState
	u.inputState.copyAndReset(&before)
	u.advanceInputTimeToNextTick()
	u.inputMu.Unlock()
	<-done

	if before.IsKeyPressed(KeyA, currentTick) {
		t.Fatal("the event entered the snapshot while the tick boundary was locked")
	}
	var after InputState
	u.inputState.copyAndReset(&after)
	if !after.IsKeyJustPressed(KeyA, currentTick+1) {
		t.Error("the event was not stamped for the tick after the snapshot")
	}
}

func TestJSScheduledFrameIsTakenOnce(t *testing.T) {
	var u UserInterface
	u.fpsMode.Store(int32(FPSModeVsyncOffMinimum))
	u.onceUpdateCalled.Store(true)

	u.ScheduleFrame()
	if !u.needsUpdate() {
		t.Fatal("a scheduled frame was not observed")
	}
	if u.needsUpdate() {
		t.Fatal("the same frame request was consumed more than once")
	}
	u.ScheduleFrame()
	if !u.needsUpdate() {
		t.Fatal("a later frame request was lost")
	}
}
