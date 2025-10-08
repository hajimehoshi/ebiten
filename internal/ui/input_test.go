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

package ui_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

func TestMultipleKeyReleases(t *testing.T) {
	const baseTick = 100
	var inputState ui.InputState

	// Press 'A'.
	inputState.SetKeyPressed(ui.KeyA, ui.NewInputTimeFromTick(baseTick))
	if got, want := inputState.IsKeyPressed(ui.KeyA, baseTick), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsKeyJustPressed(ui.KeyA, baseTick), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsKeyJustReleased(ui.KeyA, baseTick), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Check again in the next tick
	if got, want := inputState.IsKeyPressed(ui.KeyA, baseTick+1), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsKeyJustPressed(ui.KeyA, baseTick+1), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsKeyJustReleased(ui.KeyA, baseTick+1), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Release 'A'.
	inputState.SetKeyReleased(ui.KeyA, ui.NewInputTimeFromTick(baseTick+2))
	if got, want := inputState.IsKeyPressed(ui.KeyA, baseTick+2), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsKeyJustPressed(ui.KeyA, baseTick+2), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsKeyJustReleased(ui.KeyA, baseTick+2), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Release 'A' again in the next tick. This should be no-op (#3326).
	inputState.SetKeyReleased(ui.KeyA, ui.NewInputTimeFromTick(baseTick+3))
	if got, want := inputState.IsKeyPressed(ui.KeyA, baseTick+3), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsKeyJustPressed(ui.KeyA, baseTick+3), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsKeyJustReleased(ui.KeyA, baseTick+3), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestMultipleMouseButtonReleases(t *testing.T) {
	const baseTick = 100
	var inputState ui.InputState

	// Press a button.
	inputState.SetMouseButtonPressed(ui.MouseButton0, ui.NewInputTimeFromTick(baseTick))
	if got, want := inputState.IsMouseButtonPressed(ui.MouseButton0, baseTick), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsMouseButtonJustPressed(ui.MouseButton0, baseTick), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsMouseButtonJustReleased(ui.MouseButton0, baseTick), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Check again in the next tick.
	if got, want := inputState.IsMouseButtonPressed(ui.MouseButton0, baseTick+1), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsMouseButtonJustPressed(ui.MouseButton0, baseTick+1), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsMouseButtonJustReleased(ui.MouseButton0, baseTick+1), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Release the button.
	inputState.SetMouseButtonReleased(ui.MouseButton0, ui.NewInputTimeFromTick(baseTick+2))
	if got, want := inputState.IsMouseButtonPressed(ui.MouseButton0, baseTick+2), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsMouseButtonJustPressed(ui.MouseButton0, baseTick+2), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsMouseButtonJustReleased(ui.MouseButton0, baseTick+2), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Release the button again in the next tick. This should be no-op (#3326).
	inputState.SetMouseButtonReleased(ui.MouseButton0, ui.NewInputTimeFromTick(baseTick+3))
	if got, want := inputState.IsMouseButtonPressed(ui.MouseButton0, baseTick+3), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsMouseButtonJustPressed(ui.MouseButton0, baseTick+3), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := inputState.IsMouseButtonJustReleased(ui.MouseButton0, baseTick+3), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}
