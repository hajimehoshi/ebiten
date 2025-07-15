// Copyright 2023 The Ebitengine Authors
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

package vector_test

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	t "github.com/hajimehoshi/ebiten/v2/internal/testing"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func TestMain(m *testing.M) {
	t.MainWithRunLoop(m)
}

// Issue #2589
func TestLine0(t *testing.T) {
	dst := ebiten.NewImage(16, 16)
	vector.StrokeLine(dst, 0, 0, 0, 0, 2, color.White, true)
	if got, want := dst.At(0, 0), (color.RGBA{}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

// Issue #3270
func TestStrokeRectAntiAlias(t *testing.T) {
	dst := ebiten.NewImage(16, 16)
	vector.StrokeRect(dst, 0, 0, 16, 16, 2, color.White, true)
	if got, want := dst.At(5, 5), (color.RGBA{}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}
