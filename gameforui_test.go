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

package ebiten

import (
	"fmt"
	"image/color"
	"testing"
)

func TestGameForUIInvalidImageSizePreservesExistingImage(t *testing.T) {
	for _, screen := range []bool{false, true} {
		for _, size := range [][2]int{{0, 1}, {1, 0}, {-1, 1}, {1, -1}, {-1, -1}} {
			t.Run(fmt.Sprintf("screen=%t/%dx%d", screen, size[0], size[1]), func(t *testing.T) {
				existing := NewImage(1, 1)
				defer existing.Deallocate()
				want := color.RGBA{R: 255, A: 255}
				existing.Set(0, 0, want)
				g := &gameForUI{}
				if screen {
					g.screen = existing
				} else {
					g.offscreen = existing
				}
				func() {
					defer func() {
						if recover() == nil {
							t.Error("invalid image size must panic")
						}
					}()
					if screen {
						g.NewScreenImage(size[0], size[1])
					} else {
						g.NewOffscreenImage(size[0], size[1])
					}
				}()
				got := g.offscreen
				if screen {
					got = g.screen
				}
				if got != existing {
					t.Fatal("invalid size replaced the existing image")
				}
				if got := existing.At(0, 0); got != want {
					t.Errorf("existing image pixel = %v, want %v", got, want)
				}
			})
		}
	}
}
