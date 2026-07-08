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

package ebiten_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestPreferredColorMode(t *testing.T) {
	// The preferred color mode must be kept even on a platform without a window (#3480).
	defer ebiten.SetPreferredColorMode(ebiten.ColorModeUnknown)

	if got, want := ebiten.PreferredColorMode(), ebiten.ColorModeUnknown; got != want {
		t.Errorf("ebiten.PreferredColorMode(): got: %d, want: %d", got, want)
	}

	for _, mode := range []ebiten.ColorMode{ebiten.ColorModeDark, ebiten.ColorModeLight, ebiten.ColorModeUnknown} {
		ebiten.SetPreferredColorMode(mode)
		if got, want := ebiten.PreferredColorMode(), mode; got != want {
			t.Errorf("ebiten.PreferredColorMode() after ebiten.SetPreferredColorMode(%d): got: %d, want: %d", mode, got, want)
		}
	}
}
