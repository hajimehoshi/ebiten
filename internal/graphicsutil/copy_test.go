// Copyright 2016 The Ebiten Authors
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

package graphicsutil_test

import (
	"bytes"
	"image"
	"image/color"
	"image/color/palette"
	"testing"

	. "github.com/hajimehoshi/ebiten/internal/graphicsutil"
)

func TestCopyImage(t *testing.T) {
	pal := make(color.Palette, 256)
	for i := range pal {
		pal[i] = color.White
	}
	p := make([]color.Color, 255)
	for i := range p {
		if i == 64 {
			p[i] = color.White
		} else {
			p[i] = color.Transparent
		}
	}
	bigPalette := color.Palette(p)
	cases := []struct {
		In  image.Image
		Out []uint8
	}{
		{
			In: &image.Paletted{
				Pix:    []uint8{0, 1, 1, 0},
				Stride: 2,
				Rect:   image.Rect(0, 0, 2, 2),
				Palette: color.Palette([]color.Color{
					color.Transparent, color.White,
				}),
			},
			Out: []uint8{0, 0, 0, 0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0, 0, 0, 0},
		},
		{
			In:  image.NewPaletted(image.Rect(0, 0, 240, 160), pal).SubImage(image.Rect(238, 158, 240, 160)),
			Out: []uint8{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		},
		{
			In: &image.RGBA{
				Pix:    []uint8{0, 0, 0, 0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0, 0, 0, 0},
				Stride: 8,
				Rect:   image.Rect(0, 0, 2, 2),
			},
			Out: []uint8{0, 0, 0, 0, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0, 0, 0, 0},
		},
		{
			In: &image.NRGBA{
				Pix:    []uint8{0, 0, 0, 0, 0xff, 0xff, 0xff, 0x80, 0x80, 0x80, 0x80, 0x80, 0, 0, 0, 0},
				Stride: 8,
				Rect:   image.Rect(0, 0, 2, 2),
			},
			Out: []uint8{0, 0, 0, 0, 0x80, 0x80, 0x80, 0x80, 0x40, 0x40, 0x40, 0x80, 0, 0, 0, 0},
		},
		{
			In: &image.Paletted{
				Pix:     []uint8{0, 64, 0, 0},
				Stride:  2,
				Rect:    image.Rect(0, 0, 2, 2),
				Palette: bigPalette,
			},
			Out: []uint8{0, 0, 0, 0, 0xff, 0xff, 0xff, 0xff, 0, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			In: (&image.Paletted{
				Pix:     []uint8{0, 64, 0, 0},
				Stride:  2,
				Rect:    image.Rect(0, 0, 2, 2),
				Palette: bigPalette,
			}).SubImage(image.Rect(1, 0, 2, 1)),
			Out: []uint8{0xff, 0xff, 0xff, 0xff},
		},
	}
	for i, c := range cases {
		got := CopyImage(c.In)
		want := c.Out
		if !bytes.Equal(got, want) {
			t.Errorf("Test %d: got: %v, want: %v", i, got, want)
		}
	}
}

func BenchmarkCopyImageRGBA(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, 4096, 4096))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CopyImage(img)
	}
}

func BenchmarkCopyImageNRGBA(b *testing.B) {
	img := image.NewNRGBA(image.Rect(0, 0, 4096, 4096))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CopyImage(img)
	}
}

func BenchmarkCopyImagePaletted(b *testing.B) {
	img := image.NewPaletted(image.Rect(0, 0, 4096, 4096), palette.Plan9)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CopyImage(img)
	}
}
