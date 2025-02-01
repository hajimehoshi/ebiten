// Copyright 2022 The Ebitengine Authors
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

package colorm_test

import (
	"fmt"
	"image/color"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	t "github.com/hajimehoshi/ebiten/v2/internal/testing"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// sameColors compares c1 and c2 and returns a boolean value indicating
// if the two colors are (almost) same.
//
// Pixels read from GPU might include errors (#492), and
// sameColors considers such errors as delta.
func sameColors(c1, c2 color.RGBA, delta int) bool {
	return abs(int(c1.R)-int(c2.R)) <= delta &&
		abs(int(c1.G)-int(c2.G)) <= delta &&
		abs(int(c1.B)-int(c2.B)) <= delta &&
		abs(int(c1.A)-int(c2.A)) <= delta
}

func TestMain(m *testing.M) {
	ui.SetPanicOnErrorOnReadingPixelsForTesting(true)
	t.MainWithRunLoop(m)
}

func TestDrawTrianglesWithColorM(t *testing.T) {
	const w, h = 16, 16
	dst0 := ebiten.NewImage(w, h)
	src := ebiten.NewImage(w, h)
	src.Fill(color.White)

	vs0 := []ebiten.Vertex{
		{
			DstX:   0,
			DstY:   0,
			SrcX:   0,
			SrcY:   0,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
		{
			DstX:   w,
			DstY:   0,
			SrcX:   w,
			SrcY:   0,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
		{
			DstX:   0,
			DstY:   h,
			SrcX:   0,
			SrcY:   h,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
		{
			DstX:   w,
			DstY:   h,
			SrcX:   w,
			SrcY:   h,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
	}

	var cm colorm.ColorM
	cm.Scale(0.2, 0.4, 0.6, 0.8)
	op := &colorm.DrawTrianglesOptions{}
	is := []uint16{0, 1, 2, 1, 2, 3}
	colorm.DrawTriangles(dst0, vs0, is, src, cm, op)

	for _, format := range []ebiten.ColorScaleMode{
		ebiten.ColorScaleModeStraightAlpha,
		ebiten.ColorScaleModePremultipliedAlpha,
	} {
		format := format
		t.Run(fmt.Sprintf("format%d", format), func(t *testing.T) {
			var cr, cg, cb, ca float32
			switch format {
			case ebiten.ColorScaleModeStraightAlpha:
				// The values are the same as ColorM.Scale
				cr = 0.2
				cg = 0.4
				cb = 0.6
				ca = 0.8
			case ebiten.ColorScaleModePremultipliedAlpha:
				cr = 0.2 * 0.8
				cg = 0.4 * 0.8
				cb = 0.6 * 0.8
				ca = 0.8
			}
			vs1 := []ebiten.Vertex{
				{
					DstX:   0,
					DstY:   0,
					SrcX:   0,
					SrcY:   0,
					ColorR: cr,
					ColorG: cg,
					ColorB: cb,
					ColorA: ca,
				},
				{
					DstX:   w,
					DstY:   0,
					SrcX:   w,
					SrcY:   0,
					ColorR: cr,
					ColorG: cg,
					ColorB: cb,
					ColorA: ca,
				},
				{
					DstX:   0,
					DstY:   h,
					SrcX:   0,
					SrcY:   h,
					ColorR: cr,
					ColorG: cg,
					ColorB: cb,
					ColorA: ca,
				},
				{
					DstX:   w,
					DstY:   h,
					SrcX:   w,
					SrcY:   h,
					ColorR: cr,
					ColorG: cg,
					ColorB: cb,
					ColorA: ca,
				},
			}

			dst1 := ebiten.NewImage(w, h)
			op := &ebiten.DrawTrianglesOptions{}
			op.ColorScaleMode = format
			dst1.DrawTriangles(vs1, is, src, op)

			for j := 0; j < h; j++ {
				for i := 0; i < w; i++ {
					got := dst0.At(i, j)
					want := dst1.At(i, j)
					if got != want {
						t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
					}
				}
			}
		})
	}
}

func TestColorMAndScale(t *testing.T) {
	const w, h = 16, 16
	src := ebiten.NewImage(w, h)

	src.Fill(color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x80})
	vs := []ebiten.Vertex{
		{
			SrcX:   0,
			SrcY:   0,
			DstX:   0,
			DstY:   0,
			ColorR: 0.5,
			ColorG: 0.25,
			ColorB: 0.5,
			ColorA: 0.75,
		},
		{
			SrcX:   w,
			SrcY:   0,
			DstX:   w,
			DstY:   0,
			ColorR: 0.5,
			ColorG: 0.25,
			ColorB: 0.5,
			ColorA: 0.75,
		},
		{
			SrcX:   0,
			SrcY:   h,
			DstX:   0,
			DstY:   h,
			ColorR: 0.5,
			ColorG: 0.25,
			ColorB: 0.5,
			ColorA: 0.75,
		},
		{
			SrcX:   w,
			SrcY:   h,
			DstX:   w,
			DstY:   h,
			ColorR: 0.5,
			ColorG: 0.25,
			ColorB: 0.5,
			ColorA: 0.75,
		},
	}
	is := []uint16{0, 1, 2, 1, 2, 3}

	for _, format := range []ebiten.ColorScaleMode{
		ebiten.ColorScaleModeStraightAlpha,
		ebiten.ColorScaleModePremultipliedAlpha,
	} {
		format := format
		t.Run(fmt.Sprintf("format%d", format), func(t *testing.T) {
			dst := ebiten.NewImage(w, h)

			var cm colorm.ColorM
			cm.Translate(0.25, 0.25, 0.25, 0)
			op := &colorm.DrawTrianglesOptions{}
			op.ColorScaleMode = format
			colorm.DrawTriangles(dst, vs, is, src, cm, op)

			got := dst.At(0, 0).(color.RGBA)
			alphaBeforeScale := 0.5
			var want color.RGBA
			switch format {
			case ebiten.ColorScaleModeStraightAlpha:
				want = color.RGBA{
					R: byte(math.Floor(0xff * (0.5/alphaBeforeScale + 0.25) * alphaBeforeScale * 0.5 * 0.75)),
					G: byte(math.Floor(0xff * (0.5/alphaBeforeScale + 0.25) * alphaBeforeScale * 0.25 * 0.75)),
					B: byte(math.Floor(0xff * (0.5/alphaBeforeScale + 0.25) * alphaBeforeScale * 0.5 * 0.75)),
					A: byte(math.Floor(0xff * alphaBeforeScale * 0.75)),
				}
			case ebiten.ColorScaleModePremultipliedAlpha:
				want = color.RGBA{
					R: byte(math.Floor(0xff * (0.5/alphaBeforeScale + 0.25) * alphaBeforeScale * 0.5)),
					G: byte(math.Floor(0xff * (0.5/alphaBeforeScale + 0.25) * alphaBeforeScale * 0.25)),
					B: byte(math.Floor(0xff * (0.5/alphaBeforeScale + 0.25) * alphaBeforeScale * 0.5)),
					A: byte(math.Floor(0xff * alphaBeforeScale * 0.75)),
				}
			}
			if !sameColors(got, want, 2) {
				t.Errorf("got: %v, want: %v", got, want)
			}
		})
	}
}

// Issue #1213
func TestColorMCopy(t *testing.T) {
	const w, h = 16, 16
	dst := ebiten.NewImage(w, h)
	src := ebiten.NewImage(w, h)

	for k := 0; k < 256; k++ {
		var cm colorm.ColorM
		cm.Translate(1, 1, 1, float64(k)/0xff)
		op := &colorm.DrawImageOptions{}
		op.Blend = ebiten.BlendCopy
		colorm.DrawImage(dst, src, cm, op)

		for j := 0; j < h; j++ {
			for i := 0; i < w; i++ {
				got := dst.At(i, j).(color.RGBA)
				want := color.RGBA{R: byte(k), G: byte(k), B: byte(k), A: byte(k)}
				if !sameColors(got, want, 1) {
					t.Fatalf("dst.At(%d, %d), k: %d: got %v, want %v", i, j, k, got, want)
				}
			}
		}
	}
}

func TestColorScale(t *testing.T) {
	dst := ebiten.NewImage(1, 1)
	src := ebiten.NewImage(1, 1)
	src.Fill(color.White)

	op := &colorm.DrawImageOptions{}
	op.ColorScale.Scale(0.25, 0.5, 0.5, 0.5)
	var cm colorm.ColorM
	cm.Scale(1, 0.5, 1, 0.5)
	colorm.DrawImage(dst, src, cm, op)
	if got, want := dst.At(0, 0).(color.RGBA), (color.RGBA{0x20, 0x20, 0x40, 0x40}); !sameColors(got, want, 1) {
		t.Errorf("got: %v, want: %v", got, want)
	}
}
