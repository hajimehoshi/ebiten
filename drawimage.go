// Copyright 2014 Hajime Hoshi
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
	"image"
)

func DrawImage(target *Image, img *Image, x, y int) error {
	return DrawImageColor(target, img, x, y, ColorMatrixI())
}

func DrawImageGeometry(target *Image, img *Image, geo GeometryMatrix) error {
	return DrawImageGeometryColor(target, img, geo, ColorMatrixI())
}

func DrawImageColor(target *Image, img *Image, x, y int, color ColorMatrix) error {
	geo := TranslateGeometry(float64(x), float64(y))
	return DrawImageGeometryColor(target, img, geo, color)
}

func DrawImageGeometryColor(target *Image, img *Image, geo GeometryMatrix, color ColorMatrix) error {
	w, h := img.Size()
	dsts := []image.Rectangle{image.Rect(0, 0, w, h)}
	srcs := []image.Rectangle{image.Rect(0, 0, w, h)}
	return DrawImagePartsGeometryColor(target, dsts, img, srcs, geo, color)
}

func DrawImageParts(target *Image, dsts []image.Rectangle, img *Image, srcs []image.Rectangle, x, y int) error {
	return DrawImagePartsColor(target, dsts, img, srcs, x, y, ColorMatrixI())
}

func DrawImagePartsGeometry(target *Image, dsts []image.Rectangle, img *Image, srcs []image.Rectangle, geo GeometryMatrix) error {
	return DrawImagePartsGeometryColor(target, dsts, img, srcs, geo, ColorMatrixI())
}

func DrawImagePartsColor(target *Image, dsts []image.Rectangle, img *Image, srcs []image.Rectangle, x, y int, color ColorMatrix) error {
	geo := TranslateGeometry(float64(x), float64(y))
	return DrawImagePartsGeometryColor(target, dsts, img, srcs, geo, color)
}

func DrawImagePartsGeometryColor(target *Image, dsts []image.Rectangle, img *Image, srcs []image.Rectangle, geo GeometryMatrix, color ColorMatrix) error {
	return target.DrawImage(dsts, img, srcs, geo, color)
}
