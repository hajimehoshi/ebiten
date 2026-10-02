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
	"encoding/binary"
	"image"
	"image/color"
	"image/draw"
)

// imageToBytes gets RGBA bytes from img.
// premultipliedAlpha specifies whether the returned bytes are in premultiplied alpha format or not.
//
// Basically imageToBytes just calls draw.Draw.
// If img is a paletted image, an optimized copying method is used.
//
// imageToBytes might return img.Pix directly without copying when possible.
func imageToBytes(img image.Image, premultipliedAlpha bool) []byte {
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

		palette := make([]uint32, len(img.Palette))
		if premultipliedAlpha {
			for i, c := range img.Palette {
				rgba := color.RGBAModel.Convert(c).(color.RGBA)
				palette[i] = uint32(rgba.R) | uint32(rgba.G)<<8 | uint32(rgba.B)<<16 | uint32(rgba.A)<<24
			}
		} else {
			for i, c := range img.Palette {
				nrgba := color.NRGBAModel.Convert(c).(color.NRGBA)
				palette[i] = uint32(nrgba.R) | uint32(nrgba.G)<<8 | uint32(nrgba.B)<<16 | uint32(nrgba.A)<<24
			}
		}
		// Even when img is a subimage of another image, Pix starts with the 0-th index.
		var srcIdx, dstIdx int
		d := img.Stride - (x1 - x0)
		for range y1 - y0 {
			for range x1 - x0 {
				p := int(img.Pix[srcIdx])
				binary.LittleEndian.PutUint32(bs[dstIdx:dstIdx+4], palette[p])
				srcIdx++
				dstIdx += 4
			}
			srcIdx += d
		}
		return bs
	case *image.RGBA:
		if premultipliedAlpha && len(img.Pix) == 4*w*h {
			return img.Pix
		}
	case *image.NRGBA:
		if !premultipliedAlpha && len(img.Pix) == 4*w*h {
			return img.Pix
		}
	}
	return imageToBytesSlow(img, premultipliedAlpha)
}

func imageToBytesSlow(img image.Image, premultipliedAlpha bool) []byte {
	size := img.Bounds().Size()
	w, h := size.X, size.Y
	bs := make([]byte, 4*w*h)

	var dstImg draw.Image
	if premultipliedAlpha {
		dstImg = &image.RGBA{
			Pix:    bs,
			Stride: 4 * w,
			Rect:   image.Rect(0, 0, w, h),
		}
	} else {
		dstImg = &image.NRGBA{
			Pix:    bs,
			Stride: 4 * w,
			Rect:   image.Rect(0, 0, w, h),
		}
	}
	draw.Draw(dstImg, image.Rect(0, 0, w, h), img, img.Bounds().Min, draw.Src)
	return bs
}
