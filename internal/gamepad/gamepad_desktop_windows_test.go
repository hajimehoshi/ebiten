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

package gamepad_test

import (
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
)

func TestDesktopGamepadVibrateAfterClose(t *testing.T) {
	r := &gamepad.RumblerForTest{}
	g, close := gamepad.NewDesktopGamepadForTest(r)
	if !g.IsVibrationAvailable() {
		t.Error("IsVibrationAvailable before close: got false, want true")
	}
	g.Vibrate(time.Second, 1, 0.5)
	if r.VibrateCount != 1 {
		t.Errorf("Vibrate calls before close: got %d, want 1", r.VibrateCount)
	}

	close()
	if g.IsVibrationAvailable() {
		t.Error("IsVibrationAvailable after close: got true, want false")
	}
	g.Vibrate(time.Second, 1, 0.5)
	g.Vibrate(0, 0, 0)
	if r.VibrateCount != 1 {
		t.Errorf("Vibrate calls after close: got %d, want 1", r.VibrateCount)
	}

	close()
	if r.CloseCount != 1 {
		t.Errorf("Close calls: got %d, want 1", r.CloseCount)
	}
}
