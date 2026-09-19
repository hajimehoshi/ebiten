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
	"time"
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

// tickBoundaryGameStub is a Game stub to run readInputStateForTick in tests. When applyEvent
// is set, UpdateInputState simulates a browser input event arriving from a callback goroutine
// right after the snapshot for the current tick is taken, while the tick-boundary lock is
// still held.
type tickBoundaryGameStub struct {
	t *testing.T
	u *UserInterface
	e js.Value

	target *InputState

	applyEvent bool
	// eventDone receives the result of updateInputFromEvent once the simulated browser
	// callback returns.
	eventDone chan error
	// eventCompleted reports whether the result was already received from eventDone.
	eventCompleted bool
}

func (*tickBoundaryGameStub) NewOffscreenImage(width, height int) *Image { return nil }

func (*tickBoundaryGameStub) NewScreenImage(width, height int) *Image { return nil }

func (*tickBoundaryGameStub) Layout(outsideWidth, outsideHeight float64) (screenWidth, screenHeight float64) {
	return 0, 0
}

func (*tickBoundaryGameStub) Update() error { return nil }

func (*tickBoundaryGameStub) DrawOffscreen() error { return nil }

func (*tickBoundaryGameStub) DrawFinalScreen(scale, offsetX, offsetY float64) {}

func (g *tickBoundaryGameStub) UpdateInputState(fn func(*InputState)) {
	fn(g.target)
	if !g.applyEvent {
		return
	}

	go func() {
		g.eventDone <- g.u.updateInputFromEvent(g.e)
	}()

	// readInputStateForTick holds the tick-boundary lock while calling this method, so the
	// event goroutine must stay blocked until the input clock advances to the next tick.
	time.Sleep(10 * time.Millisecond)
	select {
	case err := <-g.eventDone:
		g.eventCompleted = true
		if err != nil {
			g.t.Errorf("updateInputFromEvent failed: %v", err)
		}
		g.t.Error("an event was recorded while the tick boundary was locked")
	default:
	}
}

func TestJSInputEventCannotCrossTickBoundary(t *testing.T) {
	var u UserInterface
	const currentTick = 11
	u.tick.Store(currentTick)

	e, getModifierState := newKeyDownEventForTesting()
	defer getModifierState.Release()

	g := &tickBoundaryGameStub{
		t:          t,
		u:          &u,
		e:          e,
		applyEvent: true,
		eventDone:  make(chan error, 1),
	}
	c := &context{game: g}

	var before InputState
	g.target = &before
	c.readInputStateForTick(&u)

	if !g.eventCompleted {
		if err := <-g.eventDone; err != nil {
			t.Fatalf("updateInputFromEvent failed: %v", err)
		}
	}
	if before.IsKeyJustPressed(KeyA, currentTick) {
		t.Error("a key press leaked into the snapshot taken before the event")
	}

	g.applyEvent = false
	var after InputState
	g.target = &after
	c.readInputStateForTick(&u)

	if !after.IsKeyJustPressed(KeyA, currentTick+1) {
		t.Error("the event was not recorded in the tick after the tick boundary")
	}
	if !slices.Equal(after.Runes, []rune{'a'}) {
		t.Errorf("Runes: got %q, want %q", after.Runes, []rune{'a'})
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
