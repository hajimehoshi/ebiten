// Copyright 2018 The Ebiten Authors
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

package atlas_test

import (
	"image/color"
	"runtime"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/affine"
	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	t "github.com/hajimehoshi/ebiten/v2/internal/testing"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

const (
	minImageSizeForTesting = 1024
	maxImageSizeForTesting = 4096
)

func TestMain(m *testing.M) {
	atlas.SetImageSizeForTesting(minImageSizeForTesting, maxImageSizeForTesting)
	defer atlas.ResetImageSizeForTesting()
	t.MainWithRunLoop(m)
}

func quadVertices(sw, sh, x, y int, scalex float32) []float32 {
	dx0 := float32(x)
	dy0 := float32(y)
	dx1 := float32(x) + float32(sw)*scalex
	dy1 := float32(y) + float32(sh)
	sx0 := float32(0)
	sy0 := float32(0)
	sx1 := float32(sw)
	sy1 := float32(sh)
	return []float32{
		dx0, dy0, sx0, sy0, 1, 1, 1, 1,
		dx1, dy0, sx1, sy0, 1, 1, 1, 1,
		dx0, dy1, sx0, sy1, 1, 1, 1, 1,
		dx1, dy1, sx1, sy1, 1, 1, 1, 1,
	}
}

const bigSize = 2049

func TestEnsureIsolated(t *testing.T) {
	// Create img1 and img2 with this size so that the next images are allocated
	// with non-upper-left location.
	img1 := atlas.NewImage(bigSize, 100, atlas.ImageTypeRegular)
	defer img1.MarkDisposed()
	// Ensure img1's region is allocated.
	img1.WritePixels(make([]byte, 4*bigSize*100), 0, 0, bigSize, 100)

	img2 := atlas.NewImage(100, bigSize, atlas.ImageTypeRegular)
	defer img2.MarkDisposed()
	img2.WritePixels(make([]byte, 4*100*bigSize), 0, 0, 100, bigSize)

	const size = 32

	img3 := atlas.NewImage(size/2, size/2, atlas.ImageTypeRegular)
	defer img3.MarkDisposed()
	img3.WritePixels(make([]byte, (size/2)*(size/2)*4), 0, 0, size/2, size/2)

	img4 := atlas.NewImage(size, size, atlas.ImageTypeRegular)
	defer img4.MarkDisposed()

	img5 := atlas.NewImage(size/2, size/2, atlas.ImageTypeRegular)
	defer img3.MarkDisposed()

	pix := make([]byte, size*size*4)
	for j := 0; j < size; j++ {
		for i := 0; i < size; i++ {
			pix[4*(i+j*size)] = byte(i + j)
			pix[4*(i+j*size)+1] = byte(i + j)
			pix[4*(i+j*size)+2] = byte(i + j)
			pix[4*(i+j*size)+3] = byte(i + j)
		}
	}
	img4.WritePixels(pix, 0, 0, size, size)

	const (
		dx0 = size / 4
		dy0 = size / 4
		dx1 = size * 3 / 4
		dy1 = size * 3 / 4
	)
	// img4.EnsureIsolated() should be called.
	vs := quadVertices(size/2, size/2, size/4, size/4, 1)
	is := graphics.QuadIndices()
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  size,
		Height: size,
	}
	img4.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{img3}, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeCopy, graphicsdriver.FilterNearest, graphicsdriver.AddressUnsafe, dr, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, nil, nil, false)
	if got, want := img4.IsOnAtlasForTesting(), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Make img3 isolated before getting pixels.
	vs = quadVertices(0, 0, size/2, size/2, 1)
	dr = graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  size / 2,
		Height: size / 2,
	}
	img3.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{img5}, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeCopy, graphicsdriver.FilterNearest, graphicsdriver.AddressUnsafe, dr, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, nil, nil, false)
	if got, want := img3.IsOnAtlasForTesting(), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	pix = make([]byte, 4*size*size)
	if err := img4.ReadPixels(ui.GraphicsDriverForTesting(), pix); err != nil {
		t.Fatal(err)
	}
	for j := 0; j < size; j++ {
		for i := 0; i < size; i++ {
			r := pix[4*(size*j+i)]
			g := pix[4*(size*j+i)+1]
			b := pix[4*(size*j+i)+2]
			a := pix[4*(size*j+i)+3]
			got := color.RGBA{r, g, b, a}
			var want color.RGBA
			if i < dx0 || dx1 <= i || j < dy0 || dy1 <= j {
				c := byte(i + j)
				want = color.RGBA{c, c, c, c}
			}
			if got != want {
				t.Errorf("at(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	// Check further drawing doesn't cause panic.
	// This bug was fixed by 03dcd948.
	img4.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{img3}, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeCopy, graphicsdriver.FilterNearest, graphicsdriver.AddressUnsafe, dr, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, nil, nil, false)
}

func TestReputOnAtlas(t *testing.T) {
	const size = 16

	img0 := atlas.NewImage(size, size, atlas.ImageTypeRegular)
	defer img0.MarkDisposed()
	img0.WritePixels(make([]byte, 4*size*size), 0, 0, size, size)

	img1 := atlas.NewImage(size, size, atlas.ImageTypeRegular)
	defer img1.MarkDisposed()
	img1.WritePixels(make([]byte, 4*size*size), 0, 0, size, size)
	if got, want := img1.IsOnAtlasForTesting(), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	img2 := atlas.NewImage(size, size, atlas.ImageTypeRegular)
	defer img2.MarkDisposed()
	pix := make([]byte, 4*size*size)
	for j := 0; j < size; j++ {
		for i := 0; i < size; i++ {
			pix[4*(i+j*size)] = byte(i + j)
			pix[4*(i+j*size)+1] = byte(i + j)
			pix[4*(i+j*size)+2] = byte(i + j)
			pix[4*(i+j*size)+3] = byte(i + j)
		}
	}
	img2.WritePixels(pix, 0, 0, size, size)

	// Create a volatile image. This should always be isolated.
	img3 := atlas.NewImage(size, size, atlas.ImageTypeVolatile)
	defer img3.MarkDisposed()
	img1.WritePixels(make([]byte, 4*size*size), 0, 0, size, size)
	if got, want := img3.IsOnAtlasForTesting(), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Use img1 as a render target.
	vs := quadVertices(size, size, 0, 0, 1)
	is := graphics.QuadIndices()
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  size,
		Height: size,
	}
	img1.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{img2}, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeCopy, graphicsdriver.FilterNearest, graphicsdriver.AddressUnsafe, dr, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, nil, nil, false)
	if got, want := img1.IsOnAtlasForTesting(), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Use img1 as a render source.
	// Use the doubled count since img1 was on a texture atlas and became an isolated image once.
	// Then, img1 requires longer time to recover to be on a textur atlas again.
	for i := 0; i < atlas.BaseCountToPutOnAtlas*2; i++ {
		if err := atlas.PutImagesOnAtlasForTesting(ui.GraphicsDriverForTesting()); err != nil {
			t.Fatal(err)
		}
		img0.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{img1}, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeCopy, graphicsdriver.FilterNearest, graphicsdriver.AddressUnsafe, dr, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, nil, nil, false)
		if got, want := img1.IsOnAtlasForTesting(), false; got != want {
			t.Errorf("got: %v, want: %v", got, want)
		}
	}
	if err := atlas.PutImagesOnAtlasForTesting(ui.GraphicsDriverForTesting()); err != nil {
		t.Fatal(err)
	}

	pix = make([]byte, 4*size*size)
	if err := img1.ReadPixels(ui.GraphicsDriverForTesting(), pix); err != nil {
		t.Fatal(err)
	}
	for j := 0; j < size; j++ {
		for i := 0; i < size; i++ {
			want := color.RGBA{byte(i + j), byte(i + j), byte(i + j), byte(i + j)}
			r := pix[4*(size*j+i)]
			g := pix[4*(size*j+i)+1]
			b := pix[4*(size*j+i)+2]
			a := pix[4*(size*j+i)+3]
			got := color.RGBA{r, g, b, a}
			if got != want {
				t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	// img1 is on an atlas again.
	img0.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{img1}, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeCopy, graphicsdriver.FilterNearest, graphicsdriver.AddressUnsafe, dr, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, nil, nil, false)
	if got, want := img1.IsOnAtlasForTesting(), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	pix = make([]byte, 4*size*size)
	if err := img1.ReadPixels(ui.GraphicsDriverForTesting(), pix); err != nil {
		t.Fatal(err)
	}
	for j := 0; j < size; j++ {
		for i := 0; i < size; i++ {
			want := color.RGBA{byte(i + j), byte(i + j), byte(i + j), byte(i + j)}
			r := pix[4*(size*j+i)]
			g := pix[4*(size*j+i)+1]
			b := pix[4*(size*j+i)+2]
			a := pix[4*(size*j+i)+3]
			got := color.RGBA{r, g, b, a}
			if got != want {
				t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	// Use img1 as a render target again.
	img1.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{img2}, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeCopy, graphicsdriver.FilterNearest, graphicsdriver.AddressUnsafe, dr, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, nil, nil, false)
	if got, want := img1.IsOnAtlasForTesting(), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Use img1 as a render source, but call WritePixels.
	// Now use 4x count as img1 became an isolated image again.
	for i := 0; i < atlas.BaseCountToPutOnAtlas*4; i++ {
		if err := atlas.PutImagesOnAtlasForTesting(ui.GraphicsDriverForTesting()); err != nil {
			t.Fatal(err)
		}
		img1.WritePixels(make([]byte, 4*size*size), 0, 0, size, size)
		img0.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{img1}, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeCopy, graphicsdriver.FilterNearest, graphicsdriver.AddressUnsafe, dr, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, nil, nil, false)
		if got, want := img1.IsOnAtlasForTesting(), false; got != want {
			t.Errorf("got: %v, want: %v", got, want)
		}
	}
	if err := atlas.PutImagesOnAtlasForTesting(ui.GraphicsDriverForTesting()); err != nil {
		t.Fatal(err)
	}

	// img1 is not on an atlas due to WritePixels.
	img0.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{img1}, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeCopy, graphicsdriver.FilterNearest, graphicsdriver.AddressUnsafe, dr, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, nil, nil, false)
	if got, want := img1.IsOnAtlasForTesting(), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Use img3 as a render source. As img3 is volatile, img3 is never on an atlas.
	for i := 0; i < atlas.BaseCountToPutOnAtlas*2; i++ {
		if err := atlas.PutImagesOnAtlasForTesting(ui.GraphicsDriverForTesting()); err != nil {
			t.Fatal(err)
		}
		img0.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{img3}, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeCopy, graphicsdriver.FilterNearest, graphicsdriver.AddressUnsafe, dr, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, nil, nil, false)
		if got, want := img3.IsOnAtlasForTesting(), false; got != want {
			t.Errorf("got: %v, want: %v", got, want)
		}
	}

	runtime.GC()
}

func TestExtend(t *testing.T) {
	const w0, h0 = 100, 100
	img0 := atlas.NewImage(w0, h0, atlas.ImageTypeRegular)
	defer img0.MarkDisposed()

	p0 := make([]byte, 4*w0*h0)
	for i := 0; i < w0*h0; i++ {
		p0[4*i] = byte(i)
		p0[4*i+1] = byte(i)
		p0[4*i+2] = byte(i)
		p0[4*i+3] = byte(i)
	}
	img0.WritePixels(p0, 0, 0, w0, h0)

	const w1, h1 = minImageSizeForTesting + 1, 100
	img1 := atlas.NewImage(w1, h1, atlas.ImageTypeRegular)
	defer img1.MarkDisposed()

	p1 := make([]byte, 4*w1*h1)
	for i := 0; i < w1*h1; i++ {
		p1[4*i] = byte(i)
		p1[4*i+1] = byte(i)
		p1[4*i+2] = byte(i)
		p1[4*i+3] = byte(i)
	}
	// Ensure to allocate
	img1.WritePixels(p1, 0, 0, w1, h1)

	pix0 := make([]byte, 4*w0*h0)
	if err := img0.ReadPixels(ui.GraphicsDriverForTesting(), pix0); err != nil {
		t.Fatal(err)
	}
	for j := 0; j < h0; j++ {
		for i := 0; i < w0; i++ {
			r := pix0[4*(w0*j+i)]
			g := pix0[4*(w0*j+i)+1]
			b := pix0[4*(w0*j+i)+2]
			a := pix0[4*(w0*j+i)+3]
			got := color.RGBA{r, g, b, a}
			c := byte(i + w0*j)
			want := color.RGBA{c, c, c, c}
			if got != want {
				t.Errorf("at(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	pix1 := make([]byte, 4*w1*h1)
	if err := img1.ReadPixels(ui.GraphicsDriverForTesting(), pix1); err != nil {
		t.Fatal(err)
	}
	for j := 0; j < h1; j++ {
		for i := 0; i < w1; i++ {
			r := pix1[4*(w1*j+i)]
			g := pix1[4*(w1*j+i)+1]
			b := pix1[4*(w1*j+i)+2]
			a := pix1[4*(w1*j+i)+3]
			got := color.RGBA{r, g, b, a}
			c := byte(i + w1*j)
			want := color.RGBA{c, c, c, c}
			if got != want {
				t.Errorf("at(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestWritePixelsAfterDrawTriangles(t *testing.T) {
	const w, h = 256, 256
	src := atlas.NewImage(w, h, atlas.ImageTypeRegular)
	defer src.MarkDisposed()
	dst := atlas.NewImage(w, h, atlas.ImageTypeRegular)
	defer dst.MarkDisposed()

	pix := make([]byte, 4*w*h)
	for i := 0; i < w*h; i++ {
		pix[4*i] = byte(i)
		pix[4*i+1] = byte(i)
		pix[4*i+2] = byte(i)
		pix[4*i+3] = byte(i)
	}
	src.WritePixels(pix, 0, 0, w, h)

	vs := quadVertices(w, h, 0, 0, 1)
	is := graphics.QuadIndices()
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  w,
		Height: h,
	}
	dst.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{src}, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeCopy, graphicsdriver.FilterNearest, graphicsdriver.AddressUnsafe, dr, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, nil, nil, false)
	dst.WritePixels(pix, 0, 0, w, h)

	pix = make([]byte, 4*w*h)
	if err := dst.ReadPixels(ui.GraphicsDriverForTesting(), pix); err != nil {
		t.Fatal(err)
	}
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			r := pix[4*(w*j+i)]
			g := pix[4*(w*j+i)+1]
			b := pix[4*(w*j+i)+2]
			a := pix[4*(w*j+i)+3]
			got := color.RGBA{r, g, b, a}
			c := byte(i + w*j)
			want := color.RGBA{c, c, c, c}
			if got != want {
				t.Errorf("at(%d, %d): got %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #887
func TestSmallImages(t *testing.T) {
	const w, h = 4, 8
	src := atlas.NewImage(w, h, atlas.ImageTypeRegular)
	defer src.MarkDisposed()
	dst := atlas.NewImage(w, h, atlas.ImageTypeRegular)
	defer dst.MarkDisposed()

	pix := make([]byte, 4*w*h)
	for i := 0; i < w*h; i++ {
		pix[4*i] = 0xff
		pix[4*i+1] = 0xff
		pix[4*i+2] = 0xff
		pix[4*i+3] = 0xff
	}
	src.WritePixels(pix, 0, 0, w, h)

	vs := quadVertices(w, h, 0, 0, 1)
	is := graphics.QuadIndices()
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  w,
		Height: h,
	}
	dst.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{src}, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeSourceOver, graphicsdriver.FilterNearest, graphicsdriver.AddressUnsafe, dr, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, nil, nil, false)

	pix = make([]byte, 4*w*h)
	if err := dst.ReadPixels(ui.GraphicsDriverForTesting(), pix); err != nil {
		t.Fatal(err)
	}
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			r := pix[4*(w*j+i)]
			a := pix[4*(w*j+i)+3]
			if got, want := r, byte(0xff); got != want {
				t.Errorf("at(%d, %d) red: got: %d, want: %d", i, j, got, want)
			}
			if got, want := a, byte(0xff); got != want {
				t.Errorf("at(%d, %d) alpha: got: %d, want: %d", i, j, got, want)
			}
		}
	}
}

// Issue #887
func TestLongImages(t *testing.T) {
	const w, h = 1, 6
	src := atlas.NewImage(w, h, atlas.ImageTypeRegular)
	defer src.MarkDisposed()

	const dstW, dstH = 256, 256
	dst := atlas.NewImage(dstW, dstH, atlas.ImageTypeRegular)
	defer dst.MarkDisposed()

	pix := make([]byte, 4*w*h)
	for i := 0; i < w*h; i++ {
		pix[4*i] = 0xff
		pix[4*i+1] = 0xff
		pix[4*i+2] = 0xff
		pix[4*i+3] = 0xff
	}
	src.WritePixels(pix, 0, 0, w, h)

	const scale = 120
	vs := quadVertices(w, h, 0, 0, scale)
	is := graphics.QuadIndices()
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  dstW,
		Height: dstH,
	}
	dst.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{src}, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeSourceOver, graphicsdriver.FilterNearest, graphicsdriver.AddressUnsafe, dr, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, nil, nil, false)

	pix = make([]byte, 4*dstW*dstH)
	if err := dst.ReadPixels(ui.GraphicsDriverForTesting(), pix); err != nil {
		t.Fatal(err)
	}
	for j := 0; j < h; j++ {
		for i := 0; i < w*scale; i++ {
			r := pix[4*(dstW*j+i)]
			a := pix[4*(dstW*j+i)+3]
			if got, want := r, byte(0xff); got != want {
				t.Errorf("at(%d, %d) red: got: %d, want: %d", i, j, got, want)
			}
			if got, want := a, byte(0xff); got != want {
				t.Errorf("at(%d, %d) alpha: got: %d, want: %d", i, j, got, want)
			}
		}
	}
}

func TestDisposeImmediately(t *testing.T) {
	// This tests restorable.Image.ClearPixels is called but WritePixels is not called.

	img0 := atlas.NewImage(16, 16, atlas.ImageTypeRegular)
	img0.EnsureIsolatedForTesting()
	defer img0.MarkDisposed()

	img1 := atlas.NewImage(16, 16, atlas.ImageTypeRegular)
	img1.EnsureIsolatedForTesting()
	defer img1.MarkDisposed()

	// img0 and img1 should share the same backend in 99.9999% possibility.
}

// Issue #1028
func TestExtendWithBigImage(t *testing.T) {
	img0 := atlas.NewImage(1, 1, atlas.ImageTypeRegular)
	defer img0.MarkDisposed()

	img0.WritePixels(make([]byte, 4*1*1), 0, 0, 1, 1)

	img1 := atlas.NewImage(minImageSizeForTesting+1, minImageSizeForTesting+1, atlas.ImageTypeRegular)
	defer img1.MarkDisposed()

	img1.WritePixels(make([]byte, 4*(minImageSizeForTesting+1)*(minImageSizeForTesting+1)), 0, 0, minImageSizeForTesting+1, minImageSizeForTesting+1)
}

// Issue #1217
func TestMaxImageSize(t *testing.T) {
	img0 := atlas.NewImage(1, 1, atlas.ImageTypeRegular)
	defer img0.MarkDisposed()
	paddingSize := img0.PaddingSizeForTesting()

	// This tests that a too-big image is allocated correctly.
	s := maxImageSizeForTesting - 2*paddingSize
	img1 := atlas.NewImage(s, s, atlas.ImageTypeRegular)
	defer img1.MarkDisposed()
	img1.WritePixels(make([]byte, 4*s*s), 0, 0, s, s)
}

// Issue #1217 (disabled)
func Disable_TestMinImageSize(t *testing.T) {
	// The backend cannot be reset. If this is necessary, sync the state with the images (#1756).
	// ResetBackendsForTesting()

	// This tests that extending a backend works correctly.
	// Though the image size is minimum size of the backend, extending the backend happens due to the paddings.
	s := minImageSizeForTesting
	img := atlas.NewImage(s, s, atlas.ImageTypeRegular)
	defer img.MarkDisposed()
	img.WritePixels(make([]byte, 4*s*s), 0, 0, s, s)
}

func TestMaxImageSizeJust(t *testing.T) {
	s := maxImageSizeForTesting
	// An unmanged image never belongs to an atlas and doesn't have its paddings.
	// TODO: Should we allow such this size for ImageTypeRegular?
	img := atlas.NewImage(s, s, atlas.ImageTypeUnmanaged)
	defer img.MarkDisposed()
	img.WritePixels(make([]byte, 4*s*s), 0, 0, s, s)
}

func TestMaxImageSizeExceeded(t *testing.T) {
	// This tests that a too-big image is allocated correctly.
	s := maxImageSizeForTesting
	img := atlas.NewImage(s+1, s, atlas.ImageTypeRegular)
	defer img.MarkDisposed()

	defer func() {
		if err := recover(); err == nil {
			t.Errorf("WritePixels must panic but not")
		}
	}()

	img.WritePixels(make([]byte, 4*(s+1)*s), 0, 0, s+1, s)
}

// Issue #1421
func TestDisposedAndReputOnAtlas(t *testing.T) {
	const size = 16

	src := atlas.NewImage(size, size, atlas.ImageTypeRegular)
	defer src.MarkDisposed()
	src2 := atlas.NewImage(size, size, atlas.ImageTypeRegular)
	defer src2.MarkDisposed()
	dst := atlas.NewImage(size, size, atlas.ImageTypeRegular)
	defer dst.MarkDisposed()

	// Use src as a render target so that src is not on an atlas.
	vs := quadVertices(size, size, 0, 0, 1)
	is := graphics.QuadIndices()
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  size,
		Height: size,
	}
	src.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{src2}, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeCopy, graphicsdriver.FilterNearest, graphicsdriver.AddressUnsafe, dr, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, nil, nil, false)
	if got, want := src.IsOnAtlasForTesting(), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Use src as a render source.
	for i := 0; i < atlas.BaseCountToPutOnAtlas/2; i++ {
		if err := atlas.PutImagesOnAtlasForTesting(ui.GraphicsDriverForTesting()); err != nil {
			t.Fatal(err)
		}
		dst.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{src}, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeCopy, graphicsdriver.FilterNearest, graphicsdriver.AddressUnsafe, dr, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, nil, nil, false)
		if got, want := src.IsOnAtlasForTesting(), false; got != want {
			t.Errorf("got: %v, want: %v", got, want)
		}
	}

	// Before PutImaegsOnAtlasForTesting, dispose the image.
	src.MarkDisposed()

	// Force to dispose the image.
	atlas.ResolveDeferredForTesting()

	// Confirm that PutImagesOnAtlasForTesting doesn't panic.
	if err := atlas.PutImagesOnAtlasForTesting(ui.GraphicsDriverForTesting()); err != nil {
		t.Fatal(err)
	}
}

// Issue #1456
func TestImageIsNotReputOnAtlasWithoutUsingAsSource(t *testing.T) {
	const size = 16

	src := atlas.NewImage(size, size, atlas.ImageTypeRegular)
	defer src.MarkDisposed()
	src2 := atlas.NewImage(size, size, atlas.ImageTypeRegular)
	defer src2.MarkDisposed()
	dst := atlas.NewImage(size, size, atlas.ImageTypeRegular)
	defer dst.MarkDisposed()

	// Use src as a render target so that src is not on an atlas.
	vs := quadVertices(size, size, 0, 0, 1)
	is := graphics.QuadIndices()
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  size,
		Height: size,
	}

	// Use src2 as a rendering target, and make src2 an independent image.
	src2.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{src}, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeCopy, graphicsdriver.FilterNearest, graphicsdriver.AddressUnsafe, dr, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, nil, nil, false)
	if got, want := src2.IsOnAtlasForTesting(), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Update the count without using src2 as a rendering source.
	// This should not affect whether src2 is on an atlas or not.
	for i := 0; i < atlas.BaseCountToPutOnAtlas; i++ {
		if err := atlas.PutImagesOnAtlasForTesting(ui.GraphicsDriverForTesting()); err != nil {
			t.Fatal(err)
		}
		if got, want := src2.IsOnAtlasForTesting(), false; got != want {
			t.Errorf("got: %v, want: %v", got, want)
		}
	}

	// Update the count with using src2 as a rendering source.
	for i := 0; i < atlas.BaseCountToPutOnAtlas; i++ {
		if err := atlas.PutImagesOnAtlasForTesting(ui.GraphicsDriverForTesting()); err != nil {
			t.Fatal(err)
		}
		dst.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{src2}, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeCopy, graphicsdriver.FilterNearest, graphicsdriver.AddressUnsafe, dr, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, nil, nil, false)
		if got, want := src2.IsOnAtlasForTesting(), false; got != want {
			t.Errorf("got: %v, want: %v", got, want)
		}
	}

	if err := atlas.PutImagesOnAtlasForTesting(ui.GraphicsDriverForTesting()); err != nil {
		t.Fatal(err)
	}
	if got, want := src2.IsOnAtlasForTesting(), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestImageWritePixelsModify(t *testing.T) {
	for _, typ := range []atlas.ImageType{atlas.ImageTypeRegular, atlas.ImageTypeVolatile, atlas.ImageTypeUnmanaged} {
		const size = 16
		img := atlas.NewImage(size, size, typ)
		defer img.MarkDisposed()
		pix := make([]byte, 4*size*size)
		for j := 0; j < size; j++ {
			for i := 0; i < size; i++ {
				pix[4*(i+j*size)] = byte(i + j)
				pix[4*(i+j*size)+1] = byte(i + j)
				pix[4*(i+j*size)+2] = byte(i + j)
				pix[4*(i+j*size)+3] = byte(i + j)
			}
		}
		img.WritePixels(pix, 0, 0, size, size)

		// Modify pix after WritePixels.
		for j := 0; j < size; j++ {
			for i := 0; i < size; i++ {
				pix[4*(i+j*size)] = 0
				pix[4*(i+j*size)+1] = 0
				pix[4*(i+j*size)+2] = 0
				pix[4*(i+j*size)+3] = 0
			}
		}

		// Check the pixels are the original ones.
		pix = make([]byte, 4*size*size)
		if err := img.ReadPixels(ui.GraphicsDriverForTesting(), pix); err != nil {
			t.Fatal(err)
		}
		for j := 0; j < size; j++ {
			for i := 0; i < size; i++ {
				want := color.RGBA{byte(i + j), byte(i + j), byte(i + j), byte(i + j)}
				r := pix[4*(size*j+i)]
				g := pix[4*(size*j+i)+1]
				b := pix[4*(size*j+i)+2]
				a := pix[4*(size*j+i)+3]
				got := color.RGBA{r, g, b, a}
				if got != want {
					t.Errorf("Type: %d, At(%d, %d): got: %v, want: %v", typ, i, j, got, want)
				}
			}
		}
	}
}

// TODO: Add tests to extend image on an atlas out of the main loop
