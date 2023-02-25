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
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/restorable"
	etesting "github.com/hajimehoshi/ebiten/v2/internal/testing"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

func TestMain(m *testing.M) {
	restorable.EnableRestoringForTesting()
	etesting.MainWithRunLoop(m)
}

func pixelsToColor(p *restorable.Pixels, i, j, imageWidth, imageHeight int) color.RGBA {
	var pix [4]byte
	p.ReadPixels(pix[:], i, j, 1, 1, imageWidth, imageHeight)
	return color.RGBA{R: pix[0], G: pix[1], B: pix[2], A: pix[3]}
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

func TestRestore(t *testing.T) {
	img0 := restorable.NewImage(1, 1, restorable.ImageTypeRegular)
	defer img0.Dispose()

	clr0 := color.RGBA{A: 0xff}
	img0.WritePixels([]byte{clr0.R, clr0.G, clr0.B, clr0.A}, 0, 0, 1, 1)
	if err := restorable.ResolveStaleImages(ui.GraphicsDriverForTesting(), false); err != nil {
		t.Fatal(err)
	}
	if err := restorable.RestoreIfNeeded(ui.GraphicsDriverForTesting()); err != nil {
		t.Fatal(err)
	}
	want := clr0
	got := pixelsToColor(img0.BasePixelsForTesting(), 0, 0, 1, 1)
	if !sameColors(got, want, 1) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestRestoreWithoutDraw(t *testing.T) {
	img0 := restorable.NewImage(1024, 1024, restorable.ImageTypeRegular)
	defer img0.Dispose()

	// If there is no drawing command on img0, img0 is cleared when restored.

	if err := restorable.ResolveStaleImages(ui.GraphicsDriverForTesting(), false); err != nil {
		t.Fatal(err)
	}
	if err := restorable.RestoreIfNeeded(ui.GraphicsDriverForTesting()); err != nil {
		t.Fatal(err)
	}

	for j := 0; j < 1024; j++ {
		for i := 0; i < 1024; i++ {
			want := color.RGBA{}
			got := pixelsToColor(img0.BasePixelsForTesting(), i, j, 1024, 1024)
			if !sameColors(got, want, 0) {
				t.Errorf("got %v, want %v", got, want)
			}
		}
	}
}

func quadVertices(srcImage *restorable.Image, sw, sh, x, y int) []float32 {
	dx0 := float32(x)
	dy0 := float32(y)
	dx1 := float32(x + sw)
	dy1 := float32(y + sh)
	sx0 := float32(0)
	sy0 := float32(0)
	sx1 := float32(sw)
	sy1 := float32(sh)
	iswf := float32(1)
	ishf := float32(1)
	if srcImage != nil {
		isw, ish := srcImage.InternalSize()
		iswf, ishf = float32(isw), float32(ish)
	}
	return []float32{
		dx0, dy0, sx0 / iswf, sy0 / ishf, 1, 1, 1, 1,
		dx1, dy0, sx1 / iswf, sy0 / ishf, 1, 1, 1, 1,
		dx0, dy1, sx0 / iswf, sy1 / ishf, 1, 1, 1, 1,
		dx1, dy1, sx1 / iswf, sy1 / ishf, 1, 1, 1, 1,
	}
}

func TestRestoreChain(t *testing.T) {
	const num = 10
	imgs := []*restorable.Image{}
	for i := 0; i < num; i++ {
		img := restorable.NewImage(1, 1, restorable.ImageTypeRegular)
		imgs = append(imgs, img)
	}
	defer func() {
		for _, img := range imgs {
			img.Dispose()
		}
	}()
	clr := color.RGBA{A: 0xff}
	imgs[0].WritePixels([]byte{clr.R, clr.G, clr.B, clr.A}, 0, 0, 1, 1)
	for i := 0; i < num-1; i++ {
		vs := quadVertices(imgs[i], 1, 1, 0, 0)
		is := graphics.QuadIndices()
		dr := graphicsdriver.Region{
			X:      0,
			Y:      0,
			Width:  1,
			Height: 1,
		}
		imgs[i+1].DrawTriangles([graphics.ShaderImageCount]*restorable.Image{imgs[i]}, [graphics.ShaderImageCount - 1][2]float32{}, vs, is, graphicsdriver.BlendCopy, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	}
	if err := restorable.ResolveStaleImages(ui.GraphicsDriverForTesting(), false); err != nil {
		t.Fatal(err)
	}
	if err := restorable.RestoreIfNeeded(ui.GraphicsDriverForTesting()); err != nil {
		t.Fatal(err)
	}
	want := clr
	for i, img := range imgs {
		got := pixelsToColor(img.BasePixelsForTesting(), 0, 0, 1, 1)
		if !sameColors(got, want, 1) {
			t.Errorf("%d: got %v, want %v", i, got, want)
		}
	}
}

func TestRestoreChain2(t *testing.T) {
	const (
		num = 10
		w   = 1
		h   = 1
	)
	imgs := []*restorable.Image{}
	for i := 0; i < num; i++ {
		img := restorable.NewImage(w, h, restorable.ImageTypeRegular)
		imgs = append(imgs, img)
	}
	defer func() {
		for _, img := range imgs {
			img.Dispose()
		}
	}()

	clr0 := color.RGBA{R: 0xff, A: 0xff}
	imgs[0].WritePixels([]byte{clr0.R, clr0.G, clr0.B, clr0.A}, 0, 0, w, h)
	clr7 := color.RGBA{G: 0xff, A: 0xff}
	imgs[7].WritePixels([]byte{clr7.R, clr7.G, clr7.B, clr7.A}, 0, 0, w, h)
	clr8 := color.RGBA{B: 0xff, A: 0xff}
	imgs[8].WritePixels([]byte{clr8.R, clr8.G, clr8.B, clr8.A}, 0, 0, w, h)

	is := graphics.QuadIndices()
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  w,
		Height: h,
	}
	imgs[8].DrawTriangles([graphics.ShaderImageCount]*restorable.Image{imgs[7]}, [graphics.ShaderImageCount - 1][2]float32{}, quadVertices(imgs[7], w, h, 0, 0), is, graphicsdriver.BlendCopy, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	imgs[9].DrawTriangles([graphics.ShaderImageCount]*restorable.Image{imgs[8]}, [graphics.ShaderImageCount - 1][2]float32{}, quadVertices(imgs[8], w, h, 0, 0), is, graphicsdriver.BlendCopy, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	for i := 0; i < 7; i++ {
		imgs[i+1].DrawTriangles([graphics.ShaderImageCount]*restorable.Image{imgs[i]}, [graphics.ShaderImageCount - 1][2]float32{}, quadVertices(imgs[i], w, h, 0, 0), is, graphicsdriver.BlendCopy, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	}

	if err := restorable.ResolveStaleImages(ui.GraphicsDriverForTesting(), false); err != nil {
		t.Fatal(err)
	}
	if err := restorable.RestoreIfNeeded(ui.GraphicsDriverForTesting()); err != nil {
		t.Fatal(err)
	}
	for i, img := range imgs {
		want := clr0
		if i == 8 || i == 9 {
			want = clr7
		}
		got := pixelsToColor(img.BasePixelsForTesting(), 0, 0, w, h)
		if !sameColors(got, want, 1) {
			t.Errorf("%d: got %v, want %v", i, got, want)
		}
	}
}

func TestRestoreOverrideSource(t *testing.T) {
	const (
		w = 1
		h = 1
	)
	img0 := restorable.NewImage(w, h, restorable.ImageTypeRegular)
	img1 := restorable.NewImage(w, h, restorable.ImageTypeRegular)
	img2 := restorable.NewImage(w, h, restorable.ImageTypeRegular)
	img3 := restorable.NewImage(w, h, restorable.ImageTypeRegular)
	defer func() {
		img3.Dispose()
		img2.Dispose()
		img1.Dispose()
		img0.Dispose()
	}()
	clr0 := color.RGBA{A: 0xff}
	clr1 := color.RGBA{B: 0x01, A: 0xff}
	img1.WritePixels([]byte{clr0.R, clr0.G, clr0.B, clr0.A}, 0, 0, w, h)
	is := graphics.QuadIndices()
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  w,
		Height: h,
	}
	img2.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{img1}, [graphics.ShaderImageCount - 1][2]float32{}, quadVertices(img1, w, h, 0, 0), is, graphicsdriver.BlendSourceOver, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	img3.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{img2}, [graphics.ShaderImageCount - 1][2]float32{}, quadVertices(img2, w, h, 0, 0), is, graphicsdriver.BlendSourceOver, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	img0.WritePixels([]byte{clr1.R, clr1.G, clr1.B, clr1.A}, 0, 0, w, h)
	img1.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{img0}, [graphics.ShaderImageCount - 1][2]float32{}, quadVertices(img0, w, h, 0, 0), is, graphicsdriver.BlendSourceOver, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	if err := restorable.ResolveStaleImages(ui.GraphicsDriverForTesting(), false); err != nil {
		t.Fatal(err)
	}
	if err := restorable.RestoreIfNeeded(ui.GraphicsDriverForTesting()); err != nil {
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
			pixelsToColor(img0.BasePixelsForTesting(), 0, 0, w, h),
		},
		{
			"1",
			clr1,
			pixelsToColor(img1.BasePixelsForTesting(), 0, 0, w, h),
		},
		{
			"2",
			clr0,
			pixelsToColor(img2.BasePixelsForTesting(), 0, 0, w, h),
		},
		{
			"3",
			clr0,
			pixelsToColor(img3.BasePixelsForTesting(), 0, 0, w, h),
		},
	}
	for _, c := range testCases {
		if !sameColors(c.got, c.want, 1) {
			t.Errorf("%s: got %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestRestoreComplexGraph(t *testing.T) {
	const (
		w = 4
		h = 1
	)
	// 0 -> 3
	// 1 -> 3
	// 1 -> 4
	// 2 -> 4
	// 2 -> 7
	// 3 -> 5
	// 3 -> 6
	// 3 -> 7
	// 4 -> 6
	base := image.NewRGBA(image.Rect(0, 0, w, h))
	base.Pix[0] = 0xff
	base.Pix[1] = 0xff
	base.Pix[2] = 0xff
	base.Pix[3] = 0xff
	img0 := newImageFromImage(base)
	img1 := newImageFromImage(base)
	img2 := newImageFromImage(base)
	img3 := restorable.NewImage(w, h, restorable.ImageTypeRegular)
	img4 := restorable.NewImage(w, h, restorable.ImageTypeRegular)
	img5 := restorable.NewImage(w, h, restorable.ImageTypeRegular)
	img6 := restorable.NewImage(w, h, restorable.ImageTypeRegular)
	img7 := restorable.NewImage(w, h, restorable.ImageTypeRegular)
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
	is := graphics.QuadIndices()
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  w,
		Height: h,
	}
	var offsets [graphics.ShaderImageCount - 1][2]float32
	vs := quadVertices(img0, w, h, 0, 0)
	img3.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{img0}, offsets, vs, is, graphicsdriver.BlendSourceOver, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	vs = quadVertices(img1, w, h, 1, 0)
	img3.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{img1}, offsets, vs, is, graphicsdriver.BlendSourceOver, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	vs = quadVertices(img1, w, h, 1, 0)
	img4.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{img1}, offsets, vs, is, graphicsdriver.BlendSourceOver, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	vs = quadVertices(img2, w, h, 2, 0)
	img4.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{img2}, offsets, vs, is, graphicsdriver.BlendSourceOver, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	vs = quadVertices(img3, w, h, 0, 0)
	img5.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{img3}, offsets, vs, is, graphicsdriver.BlendSourceOver, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	vs = quadVertices(img3, w, h, 0, 0)
	img6.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{img3}, offsets, vs, is, graphicsdriver.BlendSourceOver, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	vs = quadVertices(img4, w, h, 1, 0)
	img6.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{img4}, offsets, vs, is, graphicsdriver.BlendSourceOver, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	vs = quadVertices(img2, w, h, 0, 0)
	img7.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{img2}, offsets, vs, is, graphicsdriver.BlendSourceOver, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	vs = quadVertices(img3, w, h, 2, 0)
	img7.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{img3}, offsets, vs, is, graphicsdriver.BlendSourceOver, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	if err := restorable.ResolveStaleImages(ui.GraphicsDriverForTesting(), false); err != nil {
		t.Fatal(err)
	}
	if err := restorable.RestoreIfNeeded(ui.GraphicsDriverForTesting()); err != nil {
		t.Fatal(err)
	}
	testCases := []struct {
		name  string
		out   string
		image *restorable.Image
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
				want = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
			}
			got := pixelsToColor(c.image.BasePixelsForTesting(), i, 0, w, h)
			if !sameColors(got, want, 1) {
				t.Errorf("%s[%d]: got %v, want %v", c.name, i, got, want)
			}
		}
	}
}

func newImageFromImage(rgba *image.RGBA) *restorable.Image {
	s := rgba.Bounds().Size()
	img := restorable.NewImage(s.X, s.Y, restorable.ImageTypeRegular)
	img.WritePixels(rgba.Pix, 0, 0, s.X, s.Y)
	return img
}

func TestRestoreRecursive(t *testing.T) {
	const (
		w = 4
		h = 1
	)
	base := image.NewRGBA(image.Rect(0, 0, w, h))
	base.Pix[0] = 0xff
	base.Pix[1] = 0xff
	base.Pix[2] = 0xff
	base.Pix[3] = 0xff

	img0 := newImageFromImage(base)
	img1 := restorable.NewImage(w, h, restorable.ImageTypeRegular)
	defer func() {
		img1.Dispose()
		img0.Dispose()
	}()
	is := graphics.QuadIndices()
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  w,
		Height: h,
	}
	img1.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{img0}, [graphics.ShaderImageCount - 1][2]float32{}, quadVertices(img0, w, h, 1, 0), is, graphicsdriver.BlendSourceOver, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	img0.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{img1}, [graphics.ShaderImageCount - 1][2]float32{}, quadVertices(img1, w, h, 1, 0), is, graphicsdriver.BlendSourceOver, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	if err := restorable.ResolveStaleImages(ui.GraphicsDriverForTesting(), false); err != nil {
		t.Fatal(err)
	}
	if err := restorable.RestoreIfNeeded(ui.GraphicsDriverForTesting()); err != nil {
		t.Fatal(err)
	}
	testCases := []struct {
		name  string
		out   string
		image *restorable.Image
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
				want = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
			}
			got := pixelsToColor(c.image.BasePixelsForTesting(), i, 0, w, h)
			if !sameColors(got, want, 1) {
				t.Errorf("%s[%d]: got %v, want %v", c.name, i, got, want)
			}
		}
	}
}

func TestWritePixels(t *testing.T) {
	img := restorable.NewImage(17, 31, restorable.ImageTypeRegular)
	defer img.Dispose()

	pix := make([]byte, 4*4*4)
	for i := range pix {
		pix[i] = 0xff
	}
	img.WritePixels(pix, 5, 7, 4, 4)
	// Check the region (5, 7)-(9, 11). Outside state is indeterminate.
	for i := range pix {
		pix[i] = 0
	}
	if err := img.ReadPixels(ui.GraphicsDriverForTesting(), pix, 5, 7, 4, 4); err != nil {
		t.Fatal(err)
	}
	for j := 7; j < 11; j++ {
		for i := 5; i < 9; i++ {
			idx := 4 * ((j-7)*4 + i - 5)
			got := color.RGBA{R: pix[idx], G: pix[idx+1], B: pix[idx+2], A: pix[idx+3]}
			want := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
			if got != want {
				t.Errorf("(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
	if err := restorable.ResolveStaleImages(ui.GraphicsDriverForTesting(), false); err != nil {
		t.Fatal(err)
	}
	if err := restorable.RestoreIfNeeded(ui.GraphicsDriverForTesting()); err != nil {
		t.Fatal(err)
	}
	if err := img.ReadPixels(ui.GraphicsDriverForTesting(), pix, 5, 7, 4, 4); err != nil {
		t.Fatal(err)
	}
	for j := 7; j < 11; j++ {
		for i := 5; i < 9; i++ {
			idx := 4 * ((j-7)*4 + i - 5)
			got := color.RGBA{R: pix[idx], G: pix[idx+1], B: pix[idx+2], A: pix[idx+3]}
			want := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
			if got != want {
				t.Errorf("(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestDrawTrianglesAndWritePixels(t *testing.T) {
	base := image.NewRGBA(image.Rect(0, 0, 1, 1))
	base.Pix[0] = 0xff
	base.Pix[1] = 0
	base.Pix[2] = 0
	base.Pix[3] = 0xff
	img0 := newImageFromImage(base)
	defer img0.Dispose()
	img1 := restorable.NewImage(2, 1, restorable.ImageTypeRegular)
	defer img1.Dispose()

	vs := quadVertices(img0, 1, 1, 0, 0)
	is := graphics.QuadIndices()
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  2,
		Height: 1,
	}
	img1.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{img0}, [graphics.ShaderImageCount - 1][2]float32{}, vs, is, graphicsdriver.BlendCopy, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	img1.WritePixels([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, 0, 0, 2, 1)

	if err := restorable.ResolveStaleImages(ui.GraphicsDriverForTesting(), false); err != nil {
		t.Fatal(err)
	}
	if err := restorable.RestoreIfNeeded(ui.GraphicsDriverForTesting()); err != nil {
		t.Fatal(err)
	}
	var pix [4]byte
	if err := img1.ReadPixels(ui.GraphicsDriverForTesting(), pix[:], 0, 0, 1, 1); err != nil {
		t.Fatal(err)
	}
	got := color.RGBA{R: pix[0], G: pix[1], B: pix[2], A: pix[3]}
	want := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
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

	is := graphics.QuadIndices()
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  1,
		Height: 1,
	}
	img1.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{img2}, [graphics.ShaderImageCount - 1][2]float32{}, quadVertices(img2, 1, 1, 0, 0), is, graphicsdriver.BlendCopy, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	img0.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{img1}, [graphics.ShaderImageCount - 1][2]float32{}, quadVertices(img1, 1, 1, 0, 0), is, graphicsdriver.BlendCopy, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	img1.Dispose()

	if err := restorable.ResolveStaleImages(ui.GraphicsDriverForTesting(), false); err != nil {
		t.Fatal(err)
	}
	if err := restorable.RestoreIfNeeded(ui.GraphicsDriverForTesting()); err != nil {
		t.Fatal(err)
	}
	var pix [4]byte
	if err := img0.ReadPixels(ui.GraphicsDriverForTesting(), pix[:], 0, 0, 1, 1); err != nil {
		t.Fatal(err)
	}
	got := color.RGBA{R: pix[0], G: pix[1], B: pix[2], A: pix[3]}
	want := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	if !sameColors(got, want, 1) {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestWritePixelsPart(t *testing.T) {
	pix := make([]uint8, 4*2*2)
	for i := range pix {
		pix[i] = 0xff
	}

	img := restorable.NewImage(4, 4, restorable.ImageTypeRegular)
	// This doesn't make the image stale. Its base pixels are available.
	img.WritePixels(pix, 1, 1, 2, 2)

	cases := []struct {
		i    int
		j    int
		want color.RGBA
	}{
		{
			i:    0,
			j:    0,
			want: color.RGBA{},
		},
		{
			i:    3,
			j:    0,
			want: color.RGBA{},
		},
		{
			i:    0,
			j:    1,
			want: color.RGBA{},
		},
		{
			i:    1,
			j:    1,
			want: color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
		},
		{
			i:    3,
			j:    1,
			want: color.RGBA{},
		},
		{
			i:    0,
			j:    2,
			want: color.RGBA{},
		},
		{
			i:    2,
			j:    2,
			want: color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
		},
		{
			i:    3,
			j:    2,
			want: color.RGBA{},
		},
		{
			i:    0,
			j:    3,
			want: color.RGBA{},
		},
		{
			i:    3,
			j:    3,
			want: color.RGBA{},
		},
	}
	for _, c := range cases {
		got := pixelsToColor(img.BasePixelsForTesting(), c.i, c.j, 4, 4)
		want := c.want
		if got != want {
			t.Errorf("base pixel (%d, %d): got %v, want %v", c.i, c.j, got, want)
		}
	}
}

func TestWritePixelsOnly(t *testing.T) {
	const w, h = 128, 128
	img0 := restorable.NewImage(w, h, restorable.ImageTypeRegular)
	defer img0.Dispose()
	img1 := restorable.NewImage(1, 1, restorable.ImageTypeRegular)
	defer img1.Dispose()

	for i := 0; i < w*h; i += 5 {
		img0.WritePixels([]byte{1, 2, 3, 4}, i%w, i/w, 1, 1)
	}

	vs := quadVertices(img0, 1, 1, 0, 0)
	is := graphics.QuadIndices()
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  1,
		Height: 1,
	}
	img1.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{img0}, [graphics.ShaderImageCount - 1][2]float32{}, vs, is, graphicsdriver.BlendCopy, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	img0.WritePixels([]byte{5, 6, 7, 8}, 0, 0, 1, 1)

	// BasePixelsForTesting is available without GPU accessing.
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			idx := j*w + i
			var want color.RGBA
			switch {
			case idx == 0:
				want = color.RGBA{R: 5, G: 6, B: 7, A: 8}
			case idx%5 == 0:
				want = color.RGBA{R: 1, G: 2, B: 3, A: 4}
			}
			got := pixelsToColor(img0.BasePixelsForTesting(), i, j, w, h)
			if !sameColors(got, want, 0) {
				t.Errorf("got %v, want %v", got, want)
			}
		}
	}

	if err := restorable.ResolveStaleImages(ui.GraphicsDriverForTesting(), false); err != nil {
		t.Fatal(err)
	}
	if err := restorable.RestoreIfNeeded(ui.GraphicsDriverForTesting()); err != nil {
		t.Fatal(err)
	}
	want := color.RGBA{R: 1, G: 2, B: 3, A: 4}
	got := pixelsToColor(img1.BasePixelsForTesting(), 0, 0, w, h)
	if !sameColors(got, want, 0) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TODO: How about volatile/screen images?

// Issue #793
func TestReadPixelsFromVolatileImage(t *testing.T) {
	const w, h = 16, 16
	dst := restorable.NewImage(w, h, restorable.ImageTypeVolatile)
	src := restorable.NewImage(w, h, restorable.ImageTypeRegular)

	// First, make sure that dst has pixels
	dst.WritePixels(make([]byte, 4*w*h), 0, 0, w, h)

	// Second, draw src to dst. If the implementation is correct, dst becomes stale.
	pix := make([]byte, 4*w*h)
	for i := range pix {
		pix[i] = 0xff
	}
	src.WritePixels(pix, 0, 0, w, h)
	vs := quadVertices(src, 1, 1, 0, 0)
	is := graphics.QuadIndices()
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  w,
		Height: h,
	}
	dst.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{src}, [graphics.ShaderImageCount - 1][2]float32{}, vs, is, graphicsdriver.BlendCopy, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)

	// Read the pixels. If the implementation is correct, dst tries to read its pixels from GPU due to being
	// stale.
	want := byte(0xff)

	var result [4]byte
	if err := dst.ReadPixels(ui.GraphicsDriverForTesting(), result[:], 0, 0, 1, 1); err != nil {
		t.Fatal(err)
	}
	got := result[0]
	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestAllowWritePixelsAfterDrawTriangles(t *testing.T) {
	const w, h = 16, 16
	src := restorable.NewImage(w, h, restorable.ImageTypeRegular)
	dst := restorable.NewImage(w, h, restorable.ImageTypeRegular)

	vs := quadVertices(src, w, h, 0, 0)
	is := graphics.QuadIndices()
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  w,
		Height: h,
	}
	dst.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{src}, [graphics.ShaderImageCount - 1][2]float32{}, vs, is, graphicsdriver.BlendSourceOver, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	dst.WritePixels(make([]byte, 4*w*h), 0, 0, w, h)
	// WritePixels for a whole image doesn't panic.
}

func TestAllowWritePixelsForPartAfterDrawTriangles(t *testing.T) {
	const w, h = 16, 16
	src := restorable.NewImage(w, h, restorable.ImageTypeRegular)
	dst := restorable.NewImage(w, h, restorable.ImageTypeRegular)

	pix := make([]byte, 4*w*h)
	for i := range pix {
		pix[i] = 0xff
	}
	src.WritePixels(pix, 0, 0, w, h)

	vs := quadVertices(src, w, h, 0, 0)
	is := graphics.QuadIndices()
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  w,
		Height: h,
	}
	dst.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{src}, [graphics.ShaderImageCount - 1][2]float32{}, vs, is, graphicsdriver.BlendSourceOver, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	dst.WritePixels(make([]byte, 4*2*2), 0, 0, 2, 2)
	// WritePixels for a part of image doesn't panic.

	if err := restorable.ResolveStaleImages(ui.GraphicsDriverForTesting(), false); err != nil {
		t.Fatal(err)
	}
	if err := restorable.RestoreIfNeeded(ui.GraphicsDriverForTesting()); err != nil {
		t.Fatal(err)
	}

	result := make([]byte, 4*w*h)
	if err := dst.ReadPixels(ui.GraphicsDriverForTesting(), result, 0, 0, w, h); err != nil {
		t.Fatal(err)
	}
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := color.RGBA{R: result[4*(j*w+i)], G: result[4*(j*w+i)+1], B: result[4*(j*w+i)+2], A: result[4*(j*w+i)+3]}
			var want color.RGBA
			if i >= 2 || j >= 2 {
				want = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
			}
			if got != want {
				t.Errorf("color at (%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestExtend(t *testing.T) {
	pixAt := func(i, j int) byte {
		return byte(17*i + 13*j + 0x40)
	}

	const w, h = 16, 16
	orig := restorable.NewImage(w, h, restorable.ImageTypeRegular)
	pix := make([]byte, 4*w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			idx := j*w + i
			v := pixAt(i, j)
			pix[4*idx] = v
			pix[4*idx+1] = v
			pix[4*idx+2] = v
			pix[4*idx+3] = v
		}
	}

	orig.WritePixels(pix, 0, 0, w, h)
	extended := orig.Extend(w*2, h*2) // After this, orig is already disposed.

	result := make([]byte, 4*(w*2)*(h*2))
	if err := extended.ReadPixels(ui.GraphicsDriverForTesting(), result, 0, 0, w*2, h*2); err != nil {
		t.Fatal(err)
	}
	for j := 0; j < h*2; j++ {
		for i := 0; i < w*2; i++ {
			got := result[4*(j*(w*2)+i)]
			want := byte(0)
			if i < w && j < h {
				want = pixAt(i, j)
			}
			if got != want {
				t.Errorf("extended.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestClearPixels(t *testing.T) {
	const w, h = 16, 16
	img := restorable.NewImage(w, h, restorable.ImageTypeRegular)
	img.WritePixels(make([]byte, 4*4*4), 0, 0, 4, 4)
	img.WritePixels(make([]byte, 4*4*4), 4, 0, 4, 4)
	img.ClearPixels(0, 0, 4, 4)
	img.ClearPixels(4, 0, 4, 4)

	// After clearing, the regions will be available again.
	img.WritePixels(make([]byte, 4*8*4), 0, 0, 8, 4)
}

func TestMutateSlices(t *testing.T) {
	const w, h = 16, 16
	dst := restorable.NewImage(w, h, restorable.ImageTypeRegular)
	src := restorable.NewImage(w, h, restorable.ImageTypeRegular)
	pix := make([]byte, 4*w*h)
	for i := 0; i < w*h; i++ {
		pix[4*i] = byte(i)
		pix[4*i+1] = byte(i)
		pix[4*i+2] = byte(i)
		pix[4*i+3] = 0xff
	}
	src.WritePixels(pix, 0, 0, w, h)

	vs := quadVertices(src, w, h, 0, 0)
	is := make([]uint16, len(graphics.QuadIndices()))
	copy(is, graphics.QuadIndices())
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  w,
		Height: h,
	}
	dst.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{src}, [graphics.ShaderImageCount - 1][2]float32{}, vs, is, graphicsdriver.BlendSourceOver, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
	for i := range vs {
		vs[i] = 0
	}
	for i := range is {
		is[i] = 0
	}
	if err := restorable.ResolveStaleImages(ui.GraphicsDriverForTesting(), false); err != nil {
		t.Fatal(err)
	}
	if err := restorable.RestoreIfNeeded(ui.GraphicsDriverForTesting()); err != nil {
		t.Fatal(err)
	}

	srcPix := make([]byte, 4*w*h)
	if err := src.ReadPixels(ui.GraphicsDriverForTesting(), srcPix, 0, 0, w, h); err != nil {
		t.Fatal(err)
	}
	dstPix := make([]byte, 4*w*h)
	if err := dst.ReadPixels(ui.GraphicsDriverForTesting(), dstPix, 0, 0, w, h); err != nil {
		t.Fatal(err)
	}

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			idx := 4 * (j*w + i)
			want := color.RGBA{R: srcPix[idx], G: srcPix[idx+1], B: srcPix[idx+2], A: srcPix[idx+3]}
			got := color.RGBA{R: dstPix[idx], G: dstPix[idx+1], B: dstPix[idx+2], A: dstPix[idx+3]}
			if !sameColors(got, want, 1) {
				t.Errorf("(%d, %d): got %v, want %v", i, j, got, want)
			}
		}
	}
}

func TestOverlappedPixels(t *testing.T) {
	dst := restorable.NewImage(3, 3, restorable.ImageTypeRegular)

	pix0 := make([]byte, 4*2*2)
	for j := 0; j < 2; j++ {
		for i := 0; i < 2; i++ {
			idx := 4 * (j*2 + i)
			pix0[idx] = 0xff
			pix0[idx+1] = 0
			pix0[idx+2] = 0
			pix0[idx+3] = 0xff
		}
	}
	dst.WritePixels(pix0, 0, 0, 2, 2)

	pix1 := make([]byte, 4*2*2)
	for j := 0; j < 2; j++ {
		for i := 0; i < 2; i++ {
			idx := 4 * (j*2 + i)
			pix1[idx] = 0
			pix1[idx+1] = 0xff
			pix1[idx+2] = 0
			pix1[idx+3] = 0xff
		}
	}
	dst.WritePixels(pix1, 1, 1, 2, 2)

	wantColors := []color.RGBA{
		{0xff, 0, 0, 0xff},
		{0xff, 0, 0, 0xff},
		{0, 0, 0, 0},

		{0xff, 0, 0, 0xff},
		{0, 0xff, 0, 0xff},
		{0, 0xff, 0, 0xff},

		{0, 0, 0, 0},
		{0, 0xff, 0, 0xff},
		{0, 0xff, 0, 0xff},
	}

	result := make([]byte, 4*3*3)
	if err := dst.ReadPixels(ui.GraphicsDriverForTesting(), result, 0, 0, 3, 3); err != nil {
		t.Fatal(err)
	}
	for j := 0; j < 3; j++ {
		for i := 0; i < 3; i++ {
			idx := 4 * (j*3 + i)
			got := color.RGBA{R: result[idx], G: result[idx+1], B: result[idx+2], A: result[idx+3]}
			want := wantColors[3*j+i]
			if got != want {
				t.Errorf("color at (%d, %d): got %v, want: %v", i, j, got, want)
			}
		}
	}

	dst.WritePixels(nil, 1, 0, 2, 2)

	wantColors = []color.RGBA{
		{0xff, 0, 0, 0xff},
		{0, 0, 0, 0},
		{0, 0, 0, 0},

		{0xff, 0, 0, 0xff},
		{0, 0, 0, 0},
		{0, 0, 0, 0},

		{0, 0, 0, 0},
		{0, 0xff, 0, 0xff},
		{0, 0xff, 0, 0xff},
	}
	if err := dst.ReadPixels(ui.GraphicsDriverForTesting(), result, 0, 0, 3, 3); err != nil {
		t.Fatal(err)
	}
	for j := 0; j < 3; j++ {
		for i := 0; i < 3; i++ {
			idx := 4 * (j*3 + i)
			got := color.RGBA{R: result[idx], G: result[idx+1], B: result[idx+2], A: result[idx+3]}
			want := wantColors[3*j+i]
			if got != want {
				t.Errorf("color at (%d, %d): got %v, want: %v", i, j, got, want)
			}
		}
	}

	pix2 := make([]byte, 4*2*2)
	for j := 0; j < 2; j++ {
		for i := 0; i < 2; i++ {
			idx := 4 * (j*2 + i)
			pix2[idx] = 0
			pix2[idx+1] = 0
			pix2[idx+2] = 0xff
			pix2[idx+3] = 0xff
		}
	}
	dst.WritePixels(pix2, 1, 1, 2, 2)

	wantColors = []color.RGBA{
		{0xff, 0, 0, 0xff},
		{0, 0, 0, 0},
		{0, 0, 0, 0},

		{0xff, 0, 0, 0xff},
		{0, 0, 0xff, 0xff},
		{0, 0, 0xff, 0xff},

		{0, 0, 0, 0},
		{0, 0, 0xff, 0xff},
		{0, 0, 0xff, 0xff},
	}
	if err := dst.ReadPixels(ui.GraphicsDriverForTesting(), result, 0, 0, 3, 3); err != nil {
		t.Fatal(err)
	}
	for j := 0; j < 3; j++ {
		for i := 0; i < 3; i++ {
			idx := 4 * (j*3 + i)
			got := color.RGBA{R: result[idx], G: result[idx+1], B: result[idx+2], A: result[idx+3]}
			want := wantColors[3*j+i]
			if got != want {
				t.Errorf("color at (%d, %d): got %v, want: %v", i, j, got, want)
			}
		}
	}

	if err := restorable.ResolveStaleImages(ui.GraphicsDriverForTesting(), false); err != nil {
		t.Fatal(err)
	}
	if err := restorable.RestoreIfNeeded(ui.GraphicsDriverForTesting()); err != nil {
		t.Fatal(err)
	}

	if err := dst.ReadPixels(ui.GraphicsDriverForTesting(), result, 0, 0, 3, 3); err != nil {
		t.Fatal(err)
	}
	for j := 0; j < 3; j++ {
		for i := 0; i < 3; i++ {
			idx := 4 * (j*3 + i)
			got := color.RGBA{R: result[idx], G: result[idx+1], B: result[idx+2], A: result[idx+3]}
			want := wantColors[3*j+i]
			if got != want {
				t.Errorf("color at (%d, %d): got %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #2324
func TestDrawTrianglesAndReadPixels(t *testing.T) {
	const w, h = 1, 1
	src := restorable.NewImage(w, h, restorable.ImageTypeRegular)
	dst := restorable.NewImage(w, h, restorable.ImageTypeRegular)

	src.WritePixels([]byte{0x80, 0x80, 0x80, 0x80}, 0, 0, 1, 1)

	vs := quadVertices(src, w, h, 0, 0)
	is := graphics.QuadIndices()
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  w,
		Height: h,
	}
	dst.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{src}, [graphics.ShaderImageCount - 1][2]float32{}, vs, is, graphicsdriver.BlendSourceOver, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)

	pix := make([]byte, 4*w*h)
	if err := dst.ReadPixels(ui.GraphicsDriverForTesting(), pix, 0, 0, w, h); err != nil {
		t.Fatal(err)
	}
	if got, want := (color.RGBA{R: pix[0], G: pix[1], B: pix[2], A: pix[3]}), (color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x80}); !sameColors(got, want, 1) {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestWritePixelsAndDrawTriangles(t *testing.T) {
	src := restorable.NewImage(1, 1, restorable.ImageTypeRegular)
	dst := restorable.NewImage(2, 1, restorable.ImageTypeRegular)

	src.WritePixels([]byte{0x80, 0x80, 0x80, 0x80}, 0, 0, 1, 1)

	// Call WritePixels first.
	dst.WritePixels([]byte{0x40, 0x40, 0x40, 0x40}, 0, 0, 1, 1)

	// Call DrawTriangles at a different region second.
	vs := quadVertices(src, 1, 1, 1, 0)
	is := graphics.QuadIndices()
	dr := graphicsdriver.Region{
		X:      1,
		Y:      0,
		Width:  1,
		Height: 1,
	}
	dst.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{src}, [graphics.ShaderImageCount - 1][2]float32{}, vs, is, graphicsdriver.BlendSourceOver, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)

	// Get the pixels.
	pix := make([]byte, 4*2*1)
	if err := dst.ReadPixels(ui.GraphicsDriverForTesting(), pix, 0, 0, 2, 1); err != nil {
		t.Fatal(err)
	}
	if got, want := (color.RGBA{R: pix[0], G: pix[1], B: pix[2], A: pix[3]}), (color.RGBA{R: 0x40, G: 0x40, B: 0x40, A: 0x40}); !sameColors(got, want, 1) {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := (color.RGBA{R: pix[4], G: pix[5], B: pix[6], A: pix[7]}), (color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x80}); !sameColors(got, want, 1) {
		t.Errorf("got: %v, want: %v", got, want)
	}
}
