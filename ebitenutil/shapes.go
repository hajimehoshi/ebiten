// Copyright 2017 The Ebiten Authors
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

package ebitenutil

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var (
	whiteImage    = ebiten.NewImage(3, 3)
	whiteSubImage = whiteImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
)

func init() {
	whiteImage.Fill(color.White)
}

// DrawLine draws a line segment on the given destination dst.
//
// DrawLine is intended to be used mainly for debugging or prototyping purpose.
//
// Deprecated: as of v2.5. Use vector.StrokeLine instead.
func DrawLine(dst *ebiten.Image, x1, y1, x2, y2 float64, clr color.Color) {
	vector.StrokeLine(dst, float32(x1), float32(y1), float32(x2), float32(y2), 1, clr)
}

// DrawRect draws a rectangle on the given destination dst.
//
// DrawRect is intended to be used mainly for debugging or prototyping purpose.
//
// Deprecated: as of v2.5. Use vector.DrawFilledRect instead.
func DrawRect(dst *ebiten.Image, x, y, width, height float64, clr color.Color) {
	vector.DrawFilledRect(dst, float32(x), float32(y), float32(width), float32(height), clr)
}

// DrawCircle draws a circle on given destination dst.
//
// DrawCircle is intended to be used mainly for debugging or prototyping purpose.
//
// Deprecated: as of v2.5. Use vector.DrawFilledCircle instead.
func DrawCircle(dst *ebiten.Image, cx, cy, r float64, clr color.Color) {
	vector.DrawFilledCircle(dst, float32(cx), float32(cy), float32(r), clr)
}
