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

package ebiten

import (
	"image"
	"image/color"
	"image/draw"
)

// imageToBytes gets RGBA bytes from img.
//
// Basically imageToBytes just calls draw.Draw.
// If img is a paletted image, an optimized copying method is used.
//
// If img is *image.RGBA and its length is same as 4*width*height, imageToBytes returns its Pix.
func imageToBytes(img image.Image) []byte {
	size := img.Bounds().Size()
	w, h := size.X, size.Y

	switch img := img.(type) {
	case *image.Paletted:
		bs := make([]byte, 4*w*h)

		b := img.Bounds()
		x0 := b.Min.X
		y0 := b.Min.Y
		x1 := b.Max.X
		y1 := b.Max.Y

		palette := make([]uint8, len(img.Palette)*4)
		for i, c := range img.Palette {
			// Create a temporary slice to reduce boundary checks.
			pl := palette[4*i : 4*i+4]
			rgba := color.RGBAModel.Convert(c).(color.RGBA)
			pl[0] = rgba.R
			pl[1] = rgba.G
			pl[2] = rgba.B
			pl[3] = rgba.A
		}
		// Even img is a subimage of another image, Pix starts with 0-th index.
		idx0 := 0
		idx1 := 0
		d := img.Stride - (x1 - x0)
		for j := 0; j < y1-y0; j++ {
			for i := 0; i < x1-x0; i++ {
				p := int(img.Pix[idx0])
				copy(bs[idx1:idx1+4], palette[4*p:4*p+4])
				idx0++
				idx1 += 4
			}
			idx0 += d
		}
		return bs
	case *image.RGBA:
		if len(img.Pix) == 4*w*h {
			return img.Pix
		}
		return imageToBytesSlow(img)
	default:
		return imageToBytesSlow(img)
	}
}

func imageToBytesSlow(img image.Image) []byte {
	size := img.Bounds().Size()
	w, h := size.X, size.Y
	bs := make([]byte, 4*w*h)

	dstImg := &image.RGBA{
		Pix:    bs,
		Stride: 4 * w,
		Rect:   image.Rect(0, 0, w, h),
	}
	draw.Draw(dstImg, image.Rect(0, 0, w, h), img, img.Bounds().Min, draw.Src)
	return bs
}
