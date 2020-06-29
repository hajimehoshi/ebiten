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
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/internal/colormcache"
)

var (
	emptyImage *ebiten.Image
)

func init() {
	emptyImage, _ = ebiten.NewImage(1, 1, ebiten.FilterDefault)
	_ = emptyImage.Fill(color.White)
}

// DrawLine draws a line segment on the given destination dst.
//
// DrawLine is intended to be used mainly for debugging or prototyping purpose.
//
// DrawLine is not concurrent-safe.
func DrawLine(dst *ebiten.Image, x1, y1, x2, y2 float64, clr color.Color) {
	ew, eh := emptyImage.Size()
	length := math.Hypot(x2-x1, y2-y1)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(length/float64(ew), 1/float64(eh))
	op.GeoM.Rotate(math.Atan2(y2-y1, x2-x1))
	op.GeoM.Translate(x1, y1)
	op.ColorM = colormcache.ColorToColorM(clr)
	// Filter must be 'nearest' filter (default).
	// Linear filtering would make edges blurred.
	_ = dst.DrawImage(emptyImage, op)
}

// DrawRect draws a rectangle on the given destination dst.
//
// DrawRect is intended to be used mainly for debugging or prototyping purpose.
//
// DrawRect is not concurrent-safe.
func DrawRect(dst *ebiten.Image, x, y, width, height float64, clr color.Color) {
	ew, eh := emptyImage.Size()

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(width/float64(ew), height/float64(eh))
	op.GeoM.Translate(x, y)
	op.ColorM = colormcache.ColorToColorM(clr)
	// Filter must be 'nearest' filter (default).
	// Linear filtering would make edges blurred.
	_ = dst.DrawImage(emptyImage, op)
}
