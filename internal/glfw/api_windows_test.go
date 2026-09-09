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

package glfw_test

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

func TestPackPoint64(t *testing.T) {
	coordinates := []int32{math.MinInt32, -1920, -2, -1, 0, 1, 1080, math.MaxInt32}
	for _, x := range coordinates {
		for _, y := range coordinates {
			got := glfw.PackPoint64(x, y)
			if gotX, gotY := int32(got), int32(got>>32); gotX != x || gotY != y {
				t.Errorf("PackPoint64(%d, %d) = (%d, %d)", x, y, gotX, gotY)
			}
		}
	}
}
