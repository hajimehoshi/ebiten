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

package ebitenmobileview

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

func TestCancelClearsAllTouches(t *testing.T) {
	// A two-finger gesture is cancelled. ACTION_CANCEL lifts every pointer that was down,
	// not only the action-index pointer, so no touch may be left as a ghost touch.
	touches := map[ui.TouchID]position{}
	if !updateTouchesAndroid(touches, actionDown, 0, 1, 2) {
		t.Fatalf("ACTION_DOWN was not handled")
	}
	if !updateTouchesAndroid(touches, actionPointerDown, 1, 3, 4) {
		t.Fatalf("ACTION_POINTER_DOWN was not handled")
	}
	if got, want := len(touches), 2; got != want {
		t.Fatalf("got: %d touches, want: %d", got, want)
	}

	if !updateTouchesAndroid(touches, actionCancel, 1, 3, 4) {
		t.Fatalf("ACTION_CANCEL was not handled")
	}
	if got, want := len(touches), 0; got != want {
		t.Errorf("after ACTION_CANCEL got: %d touches, want: %d", got, want)
	}
}

func TestPointerUpRemovesOnlyOneTouch(t *testing.T) {
	// ACTION_POINTER_UP lifts only the pointer it names, unlike ACTION_CANCEL.
	touches := map[ui.TouchID]position{}
	updateTouchesAndroid(touches, actionDown, 0, 1, 2)
	updateTouchesAndroid(touches, actionPointerDown, 1, 3, 4)
	if !updateTouchesAndroid(touches, actionPointerUp, 1, 3, 4) {
		t.Fatalf("ACTION_POINTER_UP was not handled")
	}
	if got, want := len(touches), 1; got != want {
		t.Fatalf("got: %d touches, want: %d", got, want)
	}
	if _, ok := touches[ui.TouchID(0)]; !ok {
		t.Errorf("the remaining touch was removed by ACTION_POINTER_UP")
	}
}

func TestUpdateTouchesAndroidUnknownAction(t *testing.T) {
	touches := map[ui.TouchID]position{}
	if updateTouchesAndroid(touches, 0xff, 0, 0, 0) {
		t.Errorf("an unknown action was reported as handled")
	}
}
