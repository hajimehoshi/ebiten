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

package restorable_test

import (
	"errors"
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/opengl"
	. "github.com/hajimehoshi/ebiten/internal/restorable"
)

func TestMain(m *testing.M) {
	EnableRestoringForTesting()
	code := 0
	regularTermination := errors.New("regular termination")
	f := func(screen *ebiten.Image) error {
		code = m.Run()
		return regularTermination
	}
	if err := ebiten.Run(f, 320, 240, 1, "Test"); err != nil && err != regularTermination {
		panic(err)
	}
	os.Exit(code)
}

func byteSliceToColor(b []byte, index int) color.RGBA {
	i := index * 4
	return color.RGBA{b[i], b[i+1], b[i+2], b[i+3]}
}

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

func fill(img *Image, r, g, b, a uint8) {
	w, h := img.Size()
	pix := make([]uint8, w*h*4)
	for i := 0; i < w*h; i++ {
		pix[4*i] = r
		pix[4*i+1] = g
		pix[4*i+2] = b
		pix[4*i+3] = a
	}
	img.ReplacePixels(pix)
}

func TestRestore(t *testing.T) {
	img0 := NewImage(1, 1, false)
	// Clear images explicitly.
	// In this 'restorable' layer, reused texture might not be cleared.
	fill(img0, 0, 0, 0, 0)
	defer func() {
		img0.Dispose()
	}()
	clr0 := color.RGBA{0x00, 0x00, 0x00, 0xff}
	fill(img0, clr0.R, clr0.G, clr0.B, clr0.A)
	if err := ResolveStaleImages(); err != nil {
		t.Fatal(err)
	}
	if err := Restore(); err != nil {
		t.Fatal(err)
	}
	want := clr0
	got := byteSliceToColor(img0.BasePixelsForTesting(), 0)
	if !sameColors(got, want, 1) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func vertices(sw, sh int, x, y int) []float32 {
	swf := float32(sw)
	shf := float32(sh)
	tx := float32(x)
	ty := float32(y)

	// For the rule of values, see vertices.go.
	return []float32{
		0 + tx, 0 + ty, 0, 0, 1, 1,
		swf + tx, 0 + ty, 1, 0, 0, 1,
		0 + tx, shf + ty, 0, 1, 1, 0,
		swf + tx, shf + ty, 1, 1, 0, 0,
	}
}

func TestRestoreChain(t *testing.T) {
	const num = 10
	imgs := []*Image{}
	for i := 0; i < num; i++ {
		img := NewImage(1, 1, false)
		fill(img, 0, 0, 0, 0)
		imgs = append(imgs, img)
	}
	defer func() {
		for _, img := range imgs {
			img.Dispose()
		}
	}()
	clr := color.RGBA{0x00, 0x00, 0x00, 0xff}
	fill(imgs[0], clr.R, clr.G, clr.B, clr.A)
	for i := 0; i < num-1; i++ {
		imgs[i+1].DrawImage(imgs[i], vertices(1, 1, 0, 0), &affine.ColorM{}, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	}
	if err := ResolveStaleImages(); err != nil {
		t.Fatal(err)
	}
	if err := Restore(); err != nil {
		t.Fatal(err)
	}
	want := clr
	for i, img := range imgs {
		got := byteSliceToColor(img.BasePixelsForTesting(), 0)
		if !sameColors(got, want, 1) {
			t.Errorf("%d: got %v, want %v", i, got, want)
		}
	}
}

func TestRestoreOverrideSource(t *testing.T) {
	img0 := NewImage(1, 1, false)
	fill(img0, 0, 0, 0, 0)
	img1 := NewImage(1, 1, false)
	fill(img1, 0, 0, 0, 0)
	img2 := NewImage(1, 1, false)
	fill(img2, 0, 0, 0, 0)
	img3 := NewImage(1, 1, false)
	fill(img3, 0, 0, 0, 0)
	defer func() {
		img3.Dispose()
		img2.Dispose()
		img1.Dispose()
		img0.Dispose()
	}()
	clr0 := color.RGBA{0x00, 0x00, 0x00, 0xff}
	clr1 := color.RGBA{0x00, 0x00, 0x01, 0xff}
	fill(img1, clr0.R, clr0.G, clr0.B, clr0.A)
	img2.DrawImage(img1, vertices(1, 1, 0, 0), &affine.ColorM{}, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	img3.DrawImage(img2, vertices(1, 1, 0, 0), &affine.ColorM{}, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	fill(img0, clr1.R, clr1.G, clr1.B, clr1.A)
	img1.DrawImage(img0, vertices(1, 1, 0, 0), &affine.ColorM{}, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	if err := ResolveStaleImages(); err != nil {
		t.Fatal(err)
	}
	if err := Restore(); err != nil {
		t.Fatal(err)
	}
	testCases := []struct {
		name string
		want color.RGBA
		got  color.RGBA
	}{
		{
			"0",
			clr1,
			byteSliceToColor(img0.BasePixelsForTesting(), 0),
		},
		{
			"1",
			clr1,
			byteSliceToColor(img1.BasePixelsForTesting(), 0),
		},
		{
			"2",
			clr0,
			byteSliceToColor(img2.BasePixelsForTesting(), 0),
		},
		{
			"3",
			clr0,
			byteSliceToColor(img3.BasePixelsForTesting(), 0),
		},
	}
	for _, c := range testCases {
		if !sameColors(c.got, c.want, 1) {
			t.Errorf("%s: got %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestRestoreComplexGraph(t *testing.T) {
	// 0 -> 3
	// 1 -> 3
	// 1 -> 4
	// 2 -> 4
	// 2 -> 7
	// 3 -> 5
	// 3 -> 6
	// 3 -> 7
	// 4 -> 6
	base := image.NewRGBA(image.Rect(0, 0, 4, 1))
	base.Pix[0] = 0xff
	base.Pix[1] = 0xff
	base.Pix[2] = 0xff
	base.Pix[3] = 0xff
	img0 := NewImageFromImage(base)
	img1 := NewImageFromImage(base)
	img2 := NewImageFromImage(base)
	img3 := NewImage(4, 1, false)
	fill(img3, 0, 0, 0, 0)
	img4 := NewImage(4, 1, false)
	fill(img4, 0, 0, 0, 0)
	img5 := NewImage(4, 1, false)
	fill(img5, 0, 0, 0, 0)
	img6 := NewImage(4, 1, false)
	fill(img6, 0, 0, 0, 0)
	img7 := NewImage(4, 1, false)
	fill(img7, 0, 0, 0, 0)
	defer func() {
		img7.Dispose()
		img6.Dispose()
		img5.Dispose()
		img4.Dispose()
		img3.Dispose()
		img2.Dispose()
		img1.Dispose()
		img0.Dispose()
	}()
	img3.DrawImage(img0, vertices(4, 1, 0, 0), &affine.ColorM{}, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	img3.DrawImage(img1, vertices(4, 1, 1, 0), &affine.ColorM{}, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	img4.DrawImage(img1, vertices(4, 1, 1, 0), &affine.ColorM{}, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	img4.DrawImage(img2, vertices(4, 1, 2, 0), &affine.ColorM{}, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	img5.DrawImage(img3, vertices(4, 1, 0, 0), &affine.ColorM{}, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	img6.DrawImage(img3, vertices(4, 1, 0, 0), &affine.ColorM{}, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	img6.DrawImage(img4, vertices(4, 1, 1, 0), &affine.ColorM{}, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	img7.DrawImage(img2, vertices(4, 1, 0, 0), &affine.ColorM{}, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	img7.DrawImage(img3, vertices(4, 1, 2, 0), &affine.ColorM{}, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	if err := ResolveStaleImages(); err != nil {
		t.Fatal(err)
	}
	if err := Restore(); err != nil {
		t.Fatal(err)
	}
	testCases := []struct {
		name  string
		out   string
		image *Image
	}{
		{
			"0",
			"*---",
			img0,
		},
		{
			"1",
			"*---",
			img1,
		},
		{
			"2",
			"*---",
			img2,
		},
		{
			"3",
			"**--",
			img3,
		},
		{
			"4",
			"-**-",
			img4,
		},
		{
			"5",
			"**--",
			img5,
		},
		{
			"6",
			"****",
			img6,
		},
		{
			"7",
			"*-**",
			img7,
		},
	}
	for _, c := range testCases {
		for i := 0; i < 4; i++ {
			want := color.RGBA{}
			if c.out[i] == '*' {
				want = color.RGBA{0xff, 0xff, 0xff, 0xff}
			}
			got := byteSliceToColor(c.image.BasePixelsForTesting(), i)
			if !sameColors(got, want, 1) {
				t.Errorf("%s[%d]: got %v, want %v", c.name, i, got, want)
			}
		}
	}
}

func TestRestoreRecursive(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 4, 1))
	base.Pix[0] = 0xff
	base.Pix[1] = 0xff
	base.Pix[2] = 0xff
	base.Pix[3] = 0xff
	img0 := NewImageFromImage(base)
	img1 := NewImage(4, 1, false)
	fill(img1, 0, 0, 0, 0)
	defer func() {
		img1.Dispose()
		img0.Dispose()
	}()
	img1.DrawImage(img0, vertices(4, 1, 1, 0), &affine.ColorM{}, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	img0.DrawImage(img1, vertices(4, 1, 1, 0), &affine.ColorM{}, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	if err := ResolveStaleImages(); err != nil {
		t.Fatal(err)
	}
	if err := Restore(); err != nil {
		t.Fatal(err)
	}
	testCases := []struct {
		name  string
		out   string
		image *Image
	}{
		{
			"0",
			"*-*-",
			img0,
		},
		{
			"1",
			"-*--",
			img1,
		},
	}
	for _, c := range testCases {
		for i := 0; i < 4; i++ {
			want := color.RGBA{}
			if c.out[i] == '*' {
				want = color.RGBA{0xff, 0xff, 0xff, 0xff}
			}
			got := byteSliceToColor(c.image.BasePixelsForTesting(), i)
			if !sameColors(got, want, 1) {
				t.Errorf("%s[%d]: got %v, want %v", c.name, i, got, want)
			}
		}
	}
}

// TODO: How about volatile/screen images?
