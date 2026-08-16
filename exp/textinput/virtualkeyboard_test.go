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

package textinput_test

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
)

func TestVirtualKeyboardStateFixedWithinTick(t *testing.T) {
	defer textinput.SetVirtualKeyboardSourcesForTest(nil, nil)

	var tick int64 = 1
	var reads int
	visible := true
	textinput.SetVirtualKeyboardSourcesForTest(
		func() (bool, image.Rectangle, bool) {
			reads++
			return visible, image.Rectangle{}, false
		},
		func() int64 {
			return tick
		},
	)

	if got, want := textinput.IsVirtualKeyboardVisible(), true; got != want {
		t.Errorf("IsVirtualKeyboardVisible() = %t, want %t", got, want)
	}
	_ = textinput.VirtualKeyboardVisibleBounds()
	if got, want := reads, 1; got != want {
		t.Errorf("reads = %d, want %d", got, want)
	}

	// The state does not change within the same tick.
	visible = false
	if got, want := textinput.IsVirtualKeyboardVisible(), true; got != want {
		t.Errorf("IsVirtualKeyboardVisible() = %t, want %t", got, want)
	}
	if got, want := reads, 1; got != want {
		t.Errorf("reads = %d, want %d", got, want)
	}

	tick++
	if got, want := textinput.IsVirtualKeyboardVisible(), false; got != want {
		t.Errorf("IsVirtualKeyboardVisible() = %t, want %t", got, want)
	}
	if got, want := reads, 2; got != want {
		t.Errorf("reads = %d, want %d", got, want)
	}
}

func TestVirtualKeyboardVisibleBoundsWithoutLayout(t *testing.T) {
	defer textinput.SetVirtualKeyboardSourcesForTest(nil, nil)

	// Without a laid-out game screen, the bounds are empty whether or not a region is reported.
	for _, ok := range []bool{false, true} {
		textinput.SetVirtualKeyboardSourcesForTest(
			func() (bool, image.Rectangle, bool) {
				return false, image.Rect(0, 0, 100, 100), ok
			},
			func() int64 {
				return 1
			},
		)
		if got := textinput.VirtualKeyboardVisibleBounds(); !got.Empty() {
			t.Errorf("VirtualKeyboardVisibleBounds() = %v, want empty (ok: %t)", got, ok)
		}
	}
}
