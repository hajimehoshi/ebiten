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
	"github.com/hajimehoshi/ebiten/internal/testflock"
)

func TestMain(m *testing.M) {
	testflock.Lock()
	defer testflock.Unlock()

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
	img.ReplacePixels(pix, 0, 0, w, h)
}

func TestRestore(t *testing.T) {
	img0 := NewImage(1, 1, false)
	defer img0.Dispose()

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
		imgs[i+1].DrawImage(imgs[i], 0, 0, 1, 1, nil, nil, opengl.CompositeModeCopy, graphics.FilterNearest)
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

func TestRestoreChain2(t *testing.T) {
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

	clr0 := color.RGBA{0xff, 0x00, 0x00, 0xff}
	fill(imgs[0], clr0.R, clr0.G, clr0.B, clr0.A)
	clr7 := color.RGBA{0x00, 0xff, 0x00, 0xff}
	fill(imgs[7], clr7.R, clr7.G, clr7.B, clr7.A)
	clr8 := color.RGBA{0x00, 0x00, 0xff, 0xff}
	fill(imgs[8], clr8.R, clr8.G, clr8.B, clr8.A)

	imgs[8].DrawImage(imgs[7], 0, 0, 1, 1, nil, nil, opengl.CompositeModeCopy, graphics.FilterNearest)
	imgs[9].DrawImage(imgs[8], 0, 0, 1, 1, nil, nil, opengl.CompositeModeCopy, graphics.FilterNearest)
	for i := 0; i < 7; i++ {
		imgs[i+1].DrawImage(imgs[i], 0, 0, 1, 1, nil, nil, opengl.CompositeModeCopy, graphics.FilterNearest)
	}

	if err := ResolveStaleImages(); err != nil {
		t.Fatal(err)
	}
	if err := Restore(); err != nil {
		t.Fatal(err)
	}
	for i, img := range imgs {
		want := clr0
		if i == 8 || i == 9 {
			want = clr7
		}
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
	img2.DrawImage(img1, 0, 0, 1, 1, nil, nil, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	img3.DrawImage(img2, 0, 0, 1, 1, nil, nil, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	fill(img0, clr1.R, clr1.G, clr1.B, clr1.A)
	img1.DrawImage(img0, 0, 0, 1, 1, nil, nil, opengl.CompositeModeSourceOver, graphics.FilterNearest)
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
	img0 := newImageFromImage(base)
	img1 := newImageFromImage(base)
	img2 := newImageFromImage(base)
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
	img3.DrawImage(img0, 0, 0, 4, 1, (*affine.GeoM)(nil).Translate(0, 0), nil, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	img3.DrawImage(img1, 0, 0, 4, 1, (*affine.GeoM)(nil).Translate(1, 0), nil, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	img4.DrawImage(img1, 0, 0, 4, 1, (*affine.GeoM)(nil).Translate(1, 0), nil, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	img4.DrawImage(img2, 0, 0, 4, 1, (*affine.GeoM)(nil).Translate(2, 0), nil, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	img5.DrawImage(img3, 0, 0, 4, 1, (*affine.GeoM)(nil).Translate(0, 0), nil, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	img6.DrawImage(img3, 0, 0, 4, 1, (*affine.GeoM)(nil).Translate(0, 0), nil, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	img6.DrawImage(img4, 0, 0, 4, 1, (*affine.GeoM)(nil).Translate(1, 0), nil, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	img7.DrawImage(img2, 0, 0, 4, 1, (*affine.GeoM)(nil).Translate(0, 0), nil, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	img7.DrawImage(img3, 0, 0, 4, 1, (*affine.GeoM)(nil).Translate(2, 0), nil, opengl.CompositeModeSourceOver, graphics.FilterNearest)
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

func newImageFromImage(rgba *image.RGBA) *Image {
	s := rgba.Bounds().Size()
	img := NewImage(s.X, s.Y, false)
	img.ReplacePixels(rgba.Pix, 0, 0, s.X, s.Y)
	return img
}

func TestRestoreRecursive(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 4, 1))
	base.Pix[0] = 0xff
	base.Pix[1] = 0xff
	base.Pix[2] = 0xff
	base.Pix[3] = 0xff

	img0 := newImageFromImage(base)
	img1 := NewImage(4, 1, false)
	fill(img1, 0, 0, 0, 0)
	defer func() {
		img1.Dispose()
		img0.Dispose()
	}()
	img1.DrawImage(img0, 0, 0, 4, 1, (*affine.GeoM)(nil).Translate(1, 0), nil, opengl.CompositeModeSourceOver, graphics.FilterNearest)
	img0.DrawImage(img1, 0, 0, 4, 1, (*affine.GeoM)(nil).Translate(1, 0), nil, opengl.CompositeModeSourceOver, graphics.FilterNearest)
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

func TestReplacePixels(t *testing.T) {
	const (
		w = 17
		h = 31
	)

	img := NewImage(17, 31, false)
	defer img.Dispose()

	pix := make([]byte, 4*4*4)
	for i := range pix {
		pix[i] = 0xff
	}
	img.ReplacePixels(pix, 5, 7, 4, 4)
	// Check the region (5, 7)-(9, 11). Outside state is indeterministic.
	for j := 7; j < 11; j++ {
		for i := 5; i < 9; i++ {
			got, err := img.At(i, j)
			if err != nil {
				t.Fatal(err)
			}
			want := color.RGBA{0xff, 0xff, 0xff, 0xff}
			if got != want {
				t.Errorf("img.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
	if err := ResolveStaleImages(); err != nil {
		t.Fatal(err)
	}
	if err := Restore(); err != nil {
		t.Fatal(err)
	}
	for j := 7; j < 11; j++ {
		for i := 5; i < 9; i++ {
			got, err := img.At(i, j)
			if err != nil {
				t.Fatal(err)
			}
			want := color.RGBA{0xff, 0xff, 0xff, 0xff}
			if got != want {
				t.Errorf("img.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestDrawImageAndReplacePixels(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 1, 1))
	base.Pix[0] = 0xff
	base.Pix[1] = 0xff
	base.Pix[2] = 0xff
	base.Pix[3] = 0xff
	img0 := newImageFromImage(base)
	defer img0.Dispose()
	img1 := NewImage(2, 1, false)
	defer img1.Dispose()
	img1.DrawImage(img0, 0, 0, 1, 1, nil, nil, opengl.CompositeModeCopy, graphics.FilterNearest)
	img1.ReplacePixels([]byte{0xff, 0xff, 0xff, 0xff}, 1, 0, 1, 1)

	if err := ResolveStaleImages(); err != nil {
		t.Fatal(err)
	}
	if err := Restore(); err != nil {
		t.Fatal(err)
	}
	got, err := img1.At(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := color.RGBA{0xff, 0xff, 0xff, 0xff}
	if !sameColors(got, want, 1) {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestDispose(t *testing.T) {
	base0 := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img0 := newImageFromImage(base0)
	defer img0.Dispose()

	base1 := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img1 := newImageFromImage(base1)

	base2 := image.NewRGBA(image.Rect(0, 0, 1, 1))
	base2.Pix[0] = 0xff
	base2.Pix[1] = 0xff
	base2.Pix[2] = 0xff
	base2.Pix[3] = 0xff
	img2 := newImageFromImage(base2)
	defer img2.Dispose()

	img1.DrawImage(img2, 0, 0, 1, 1, nil, nil, opengl.CompositeModeCopy, graphics.FilterNearest)
	img0.DrawImage(img1, 0, 0, 1, 1, nil, nil, opengl.CompositeModeCopy, graphics.FilterNearest)
	img1.Dispose()

	if err := ResolveStaleImages(); err != nil {
		t.Fatal(err)
	}
	if err := Restore(); err != nil {
		t.Fatal(err)
	}
	got, err := img0.At(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := color.RGBA{0xff, 0xff, 0xff, 0xff}
	if !sameColors(got, want, 1) {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestDoubleResolve(t *testing.T) {
	base0 := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img0 := newImageFromImage(base0)

	base := image.NewRGBA(image.Rect(0, 0, 1, 1))
	base.Pix[0] = 0xff
	base.Pix[1] = 0x00
	base.Pix[2] = 0x00
	base.Pix[3] = 0xff
	img1 := newImageFromImage(base)

	img0.DrawImage(img1, 0, 0, 1, 1, nil, nil, opengl.CompositeModeCopy, graphics.FilterNearest)
	img0.ReplacePixels([]uint8{0x00, 0xff, 0x00, 0xff}, 1, 1, 1, 1)
	// Now img0 is stale.
	if err := ResolveStaleImages(); err != nil {
		t.Fatal(err)
	}

	img0.ReplacePixels([]uint8{0x00, 0x00, 0xff, 0xff}, 1, 0, 1, 1)
	if err := ResolveStaleImages(); err != nil {
		t.Fatal(err)
	}

	if err := Restore(); err != nil {
		t.Fatal(err)
	}

	wantImg := image.NewRGBA(image.Rect(0, 0, 2, 2))
	wantImg.Set(0, 0, color.RGBA{0xff, 0x00, 0x00, 0xff})
	wantImg.Set(1, 0, color.RGBA{0x00, 0x00, 0xff, 0xff})
	wantImg.Set(1, 1, color.RGBA{0x00, 0xff, 0x00, 0xff})
	for j := 0; j < 2; j++ {
		for i := 0; i < 2; i++ {
			got, err := img0.At(i, j)
			if err != nil {
				t.Fatal(err)
			}
			want := wantImg.At(i, j).(color.RGBA)
			if !sameColors(got, want, 1) {
				t.Errorf("got: %v, want: %v", got, want)
			}
		}
	}
}

func TestClear(t *testing.T) {
	pix := make([]uint8, 4*4*4)
	for i := range pix {
		pix[i] = 0xff
	}

	img := NewImage(4, 4, false)
	img.ReplacePixels(pix, 0, 0, 4, 4)
	// This doesn't make the image stale. Its base pixels are available.
	img.ReplacePixels(nil, 1, 1, 2, 2)

	cases := []struct {
		Index int
		Want  color.RGBA
	}{
		{
			Index: 0,
			Want:  color.RGBA{0xff, 0xff, 0xff, 0xff},
		},
		{
			Index: 3,
			Want:  color.RGBA{0xff, 0xff, 0xff, 0xff},
		},
		{
			Index: 4,
			Want:  color.RGBA{0xff, 0xff, 0xff, 0xff},
		},
		{
			Index: 5,
			Want:  color.RGBA{0, 0, 0, 0},
		},
		{
			Index: 7,
			Want:  color.RGBA{0xff, 0xff, 0xff, 0xff},
		},
		{
			Index: 8,
			Want:  color.RGBA{0xff, 0xff, 0xff, 0xff},
		},
		{
			Index: 10,
			Want:  color.RGBA{0, 0, 0, 0},
		},
		{
			Index: 11,
			Want:  color.RGBA{0xff, 0xff, 0xff, 0xff},
		},
		{
			Index: 12,
			Want:  color.RGBA{0xff, 0xff, 0xff, 0xff},
		},
		{
			Index: 15,
			Want:  color.RGBA{0xff, 0xff, 0xff, 0xff},
		},
	}
	for _, c := range cases {
		got := byteSliceToColor(img.BasePixelsForTesting(), c.Index)
		want := c.Want
		if got != want {
			t.Errorf("base pixel [%d]: got %v, want %v", c.Index, got, want)
		}
	}
}

// TODO: How about volatile/screen images?
