// Copyright 2024 The Ebitengine Authors
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

package ebiten_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestScreenSizeInFullscreen(t *testing.T) {
	// Just call ScreenSizeInFullscreen. There was a crash bug on browsers (#2975).
	w, h := ebiten.ScreenSizeInFullscreen()
	if w <= 0 {
		t.Errorf("w must be positive but not: %d", w)
	}
	if h <= 0 {
		t.Errorf("h must be positive but not: %d", h)
	}
}
