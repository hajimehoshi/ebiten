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
	. "github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	t "github.com/hajimehoshi/ebiten/v2/internal/testing"
)

const (
	minImageSizeForTesting = 1024
	maxImageSizeForTesting = 4096
)

func TestMain(m *testing.M) {
	SetImageSizeForTesting(minImageSizeForTesting, maxImageSizeForTesting)
	defer ResetImageSizeForTesting()
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
	img1 := NewImage(bigSize, 100)
	defer img1.MarkDisposed()
	// Ensure img1's region is allocated.
	img1.ReplacePixels(make([]byte, 4*bigSize*100))

	img2 := NewImage(100, bigSize)
	defer img2.MarkDisposed()
	img2.ReplacePixels(make([]byte, 4*100*bigSize))

	const size = 32

	img3 := NewImage(size/2, size/2)
	defer img3.MarkDisposed()
	img3.ReplacePixels(make([]byte, (size/2)*(size/2)*4))

	img4 := NewImage(size, size)
	defer img4.MarkDisposed()

	pix := make([]byte, size*size*4)
	for j := 0; j < size; j++ {
		for i := 0; i < size; i++ {
			pix[4*(i+j*size)] = byte(i + j)
			pix[4*(i+j*size)+1] = byte(i + j)
			pix[4*(i+j*size)+2] = byte(i + j)
			pix[4*(i+j*size)+3] = byte(i + j)
		}
	}
	img4.ReplacePixels(pix)

	const (
		dx0 = size / 4
		dy0 = size / 4
		dx1 = size * 3 / 4
		dy1 = size * 3 / 4
	)
	// img4.EnsureIsolated() should be called.
	vs := quadVertices(size/2, size/2, size/4, size/4, 1)
	is := graphics.QuadIndices()
	dr := driver.Region{
		X:      0,
		Y:      0,
		Width:  size,
		Height: size,
	}
	img4.DrawTriangles([graphics.ShaderImageNum]*Image{img3}, vs, is, affine.ColorMIdentity{}, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressUnsafe, dr, driver.Region{}, [graphics.ShaderImageNum - 1][2]float32{}, nil, nil, false)
	want := false
	if got := img4.IsOnAtlasForTesting(); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	pix, err := img4.Pixels(0, 0, size, size)
	if err != nil {
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
	img4.DrawTriangles([graphics.ShaderImageNum]*Image{img3}, vs, is, affine.ColorMIdentity{}, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressUnsafe, dr, driver.Region{}, [graphics.ShaderImageNum - 1][2]float32{}, nil, nil, false)
}

func TestReputOnAtlas(t *testing.T) {
	const size = 16

	img0 := NewImage(size, size)
	defer img0.MarkDisposed()
	img0.ReplacePixels(make([]byte, 4*size*size))

	img1 := NewImage(size, size)
	defer img1.MarkDisposed()
	img1.ReplacePixels(make([]byte, 4*size*size))
	if got, want := img1.IsOnAtlasForTesting(), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	img2 := NewImage(size, size)
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
	img2.ReplacePixels(pix)

	img3 := NewImage(size, size)
	img3.SetVolatile(true)
	defer img3.MarkDisposed()
	img1.ReplacePixels(make([]byte, 4*size*size))
	if got, want := img3.IsOnAtlasForTesting(), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Use img1 as a render target.
	vs := quadVertices(size, size, 0, 0, 1)
	is := graphics.QuadIndices()
	dr := driver.Region{
		X:      0,
		Y:      0,
		Width:  size,
		Height: size,
	}
	img1.DrawTriangles([graphics.ShaderImageNum]*Image{img2}, vs, is, affine.ColorMIdentity{}, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressUnsafe, dr, driver.Region{}, [graphics.ShaderImageNum - 1][2]float32{}, nil, nil, false)
	if got, want := img1.IsOnAtlasForTesting(), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Use img1 as a render source.
	// Use the doubled count since img1 was on a texture atlas and became an isolated image once.
	// Then, img1 requires longer time to recover to be on a textur atlas again.
	for i := 0; i < BaseCountToPutOnAtlas*2; i++ {
		if err := PutImagesOnAtlasForTesting(); err != nil {
			t.Fatal(err)
		}
		img0.DrawTriangles([graphics.ShaderImageNum]*Image{img1}, vs, is, affine.ColorMIdentity{}, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressUnsafe, dr, driver.Region{}, [graphics.ShaderImageNum - 1][2]float32{}, nil, nil, false)
		if got, want := img1.IsOnAtlasForTesting(), false; got != want {
			t.Errorf("got: %v, want: %v", got, want)
		}
	}
	if err := PutImagesOnAtlasForTesting(); err != nil {
		t.Fatal(err)
	}

	pix, err := img1.Pixels(0, 0, size, size)
	if err != nil {
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
	img0.DrawTriangles([graphics.ShaderImageNum]*Image{img1}, vs, is, affine.ColorMIdentity{}, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressUnsafe, dr, driver.Region{}, [graphics.ShaderImageNum - 1][2]float32{}, nil, nil, false)
	if got, want := img1.IsOnAtlasForTesting(), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	pix, err = img1.Pixels(0, 0, size, size)
	if err != nil {
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
	img1.DrawTriangles([graphics.ShaderImageNum]*Image{img2}, vs, is, affine.ColorMIdentity{}, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressUnsafe, dr, driver.Region{}, [graphics.ShaderImageNum - 1][2]float32{}, nil, nil, false)
	if got, want := img1.IsOnAtlasForTesting(), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Use img1 as a render source, but call ReplacePixels.
	// Now use 4x count as img1 became an isolated image again.
	for i := 0; i < BaseCountToPutOnAtlas*4; i++ {
		if err := PutImagesOnAtlasForTesting(); err != nil {
			t.Fatal(err)
		}
		img1.ReplacePixels(make([]byte, 4*size*size))
		img0.DrawTriangles([graphics.ShaderImageNum]*Image{img1}, vs, is, affine.ColorMIdentity{}, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressUnsafe, dr, driver.Region{}, [graphics.ShaderImageNum - 1][2]float32{}, nil, nil, false)
		if got, want := img1.IsOnAtlasForTesting(), false; got != want {
			t.Errorf("got: %v, want: %v", got, want)
		}
	}
	if err := PutImagesOnAtlasForTesting(); err != nil {
		t.Fatal(err)
	}

	// img1 is not on an atlas due to ReplacePixels.
	img0.DrawTriangles([graphics.ShaderImageNum]*Image{img1}, vs, is, affine.ColorMIdentity{}, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressUnsafe, dr, driver.Region{}, [graphics.ShaderImageNum - 1][2]float32{}, nil, nil, false)
	if got, want := img1.IsOnAtlasForTesting(), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Use img3 as a render source. As img3 is volatile, img3 is never on an atlas.
	for i := 0; i < BaseCountToPutOnAtlas*2; i++ {
		if err := PutImagesOnAtlasForTesting(); err != nil {
			t.Fatal(err)
		}
		img0.DrawTriangles([graphics.ShaderImageNum]*Image{img3}, vs, is, affine.ColorMIdentity{}, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressUnsafe, dr, driver.Region{}, [graphics.ShaderImageNum - 1][2]float32{}, nil, nil, false)
		if got, want := img3.IsOnAtlasForTesting(), false; got != want {
			t.Errorf("got: %v, want: %v", got, want)
		}
	}

	runtime.GC()
}

func TestExtend(t *testing.T) {
	const w0, h0 = 100, 100
	img0 := NewImage(w0, h0)
	defer img0.MarkDisposed()

	p0 := make([]byte, 4*w0*h0)
	for i := 0; i < w0*h0; i++ {
		p0[4*i] = byte(i)
		p0[4*i+1] = byte(i)
		p0[4*i+2] = byte(i)
		p0[4*i+3] = byte(i)
	}
	img0.ReplacePixels(p0)

	const w1, h1 = minImageSizeForTesting + 1, 100
	img1 := NewImage(w1, h1)
	defer img1.MarkDisposed()

	p1 := make([]byte, 4*w1*h1)
	for i := 0; i < w1*h1; i++ {
		p1[4*i] = byte(i)
		p1[4*i+1] = byte(i)
		p1[4*i+2] = byte(i)
		p1[4*i+3] = byte(i)
	}
	// Ensure to allocate
	img1.ReplacePixels(p1)

	pix0, err := img0.Pixels(0, 0, w0, h0)
	if err != nil {
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

	pix1, err := img1.Pixels(0, 0, w1, h1)
	if err != nil {
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

func TestReplacePixelsAfterDrawTriangles(t *testing.T) {
	const w, h = 256, 256
	src := NewImage(w, h)
	defer src.MarkDisposed()
	dst := NewImage(w, h)
	defer dst.MarkDisposed()

	pix := make([]byte, 4*w*h)
	for i := 0; i < w*h; i++ {
		pix[4*i] = byte(i)
		pix[4*i+1] = byte(i)
		pix[4*i+2] = byte(i)
		pix[4*i+3] = byte(i)
	}
	src.ReplacePixels(pix)

	vs := quadVertices(w, h, 0, 0, 1)
	is := graphics.QuadIndices()
	dr := driver.Region{
		X:      0,
		Y:      0,
		Width:  w,
		Height: h,
	}
	dst.DrawTriangles([graphics.ShaderImageNum]*Image{src}, vs, is, affine.ColorMIdentity{}, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressUnsafe, dr, driver.Region{}, [graphics.ShaderImageNum - 1][2]float32{}, nil, nil, false)
	dst.ReplacePixels(pix)

	pix, err := dst.Pixels(0, 0, w, h)
	if err != nil {
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
	src := NewImage(w, h)
	defer src.MarkDisposed()
	dst := NewImage(w, h)
	defer dst.MarkDisposed()

	pix := make([]byte, 4*w*h)
	for i := 0; i < w*h; i++ {
		pix[4*i] = 0xff
		pix[4*i+1] = 0xff
		pix[4*i+2] = 0xff
		pix[4*i+3] = 0xff
	}
	src.ReplacePixels(pix)

	vs := quadVertices(w, h, 0, 0, 1)
	is := graphics.QuadIndices()
	dr := driver.Region{
		X:      0,
		Y:      0,
		Width:  w,
		Height: h,
	}
	dst.DrawTriangles([graphics.ShaderImageNum]*Image{src}, vs, is, affine.ColorMIdentity{}, driver.CompositeModeSourceOver, driver.FilterNearest, driver.AddressUnsafe, dr, driver.Region{}, [graphics.ShaderImageNum - 1][2]float32{}, nil, nil, false)

	pix, err := dst.Pixels(0, 0, w, h)
	if err != nil {
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
	src := NewImage(w, h)
	defer src.MarkDisposed()

	const dstW, dstH = 256, 256
	dst := NewImage(dstW, dstH)
	defer dst.MarkDisposed()

	pix := make([]byte, 4*w*h)
	for i := 0; i < w*h; i++ {
		pix[4*i] = 0xff
		pix[4*i+1] = 0xff
		pix[4*i+2] = 0xff
		pix[4*i+3] = 0xff
	}
	src.ReplacePixels(pix)

	const scale = 120
	vs := quadVertices(w, h, 0, 0, scale)
	is := graphics.QuadIndices()
	dr := driver.Region{
		X:      0,
		Y:      0,
		Width:  dstW,
		Height: dstH,
	}
	dst.DrawTriangles([graphics.ShaderImageNum]*Image{src}, vs, is, affine.ColorMIdentity{}, driver.CompositeModeSourceOver, driver.FilterNearest, driver.AddressUnsafe, dr, driver.Region{}, [graphics.ShaderImageNum - 1][2]float32{}, nil, nil, false)

	pix, err := dst.Pixels(0, 0, dstW, dstH)
	if err != nil {
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
	// This tests restorable.Image.ClearPixels is called but ReplacePixels is not called.

	img0 := NewImage(16, 16)
	img0.EnsureIsolatedForTesting()
	defer img0.MarkDisposed()

	img1 := NewImage(16, 16)
	img1.EnsureIsolatedForTesting()
	defer img1.MarkDisposed()

	// img0 and img1 should share the same backend in 99.9999% possibility.
}

// Issue #1028
func TestExtendWithBigImage(t *testing.T) {
	img0 := NewImage(1, 1)
	defer img0.MarkDisposed()

	img0.ReplacePixels(make([]byte, 4*1*1))

	img1 := NewImage(minImageSizeForTesting+1, minImageSizeForTesting+1)
	defer img1.MarkDisposed()

	img1.ReplacePixels(make([]byte, 4*(minImageSizeForTesting+1)*(minImageSizeForTesting+1)))
}

// Issue #1217
func TestMaxImageSize(t *testing.T) {
	// This tests that a too-big image is allocated correctly.
	s := maxImageSizeForTesting - 2*PaddingSize
	img := NewImage(s, s)
	defer img.MarkDisposed()
	img.ReplacePixels(make([]byte, 4*s*s))
}

// Issue #1217 (disabled)
func Disable_TestMinImageSize(t *testing.T) {
	// The backend cannot be reset. If this is necessary, sync the state with the images (#1756).
	// ResetBackendsForTesting()

	// This tests that extending a backend works correctly.
	// Though the image size is minimum size of the backend, extending the backend happens due to the paddings.
	s := minImageSizeForTesting
	img := NewImage(s, s)
	defer img.MarkDisposed()
	img.ReplacePixels(make([]byte, 4*s*s))
}

// Issue #1421
func TestDisposedAndReputOnAtlas(t *testing.T) {
	const size = 16

	src := NewImage(size, size)
	defer src.MarkDisposed()
	src2 := NewImage(size, size)
	defer src2.MarkDisposed()
	dst := NewImage(size, size)
	defer dst.MarkDisposed()

	// Use src as a render target so that src is not on an atlas.
	vs := quadVertices(size, size, 0, 0, 1)
	is := graphics.QuadIndices()
	dr := driver.Region{
		X:      0,
		Y:      0,
		Width:  size,
		Height: size,
	}
	src.DrawTriangles([graphics.ShaderImageNum]*Image{src2}, vs, is, affine.ColorMIdentity{}, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressUnsafe, dr, driver.Region{}, [graphics.ShaderImageNum - 1][2]float32{}, nil, nil, false)
	if got, want := src.IsOnAtlasForTesting(), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Use src as a render source.
	for i := 0; i < BaseCountToPutOnAtlas/2; i++ {
		if err := PutImagesOnAtlasForTesting(); err != nil {
			t.Fatal(err)
		}
		dst.DrawTriangles([graphics.ShaderImageNum]*Image{src}, vs, is, affine.ColorMIdentity{}, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressUnsafe, dr, driver.Region{}, [graphics.ShaderImageNum - 1][2]float32{}, nil, nil, false)
		if got, want := src.IsOnAtlasForTesting(), false; got != want {
			t.Errorf("got: %v, want: %v", got, want)
		}
	}

	// Before PutImaegsOnAtlasForTesting, dispose the image.
	src.MarkDisposed()

	// Force to dispose the image.
	ResolveDeferredForTesting()

	// Confirm that PutImagesOnAtlasForTesting doesn't panic.
	if err := PutImagesOnAtlasForTesting(); err != nil {
		t.Fatal(err)
	}
}

// Issue #1456
func TestImageIsNotReputOnAtlasWithoutUsingAsSource(t *testing.T) {
	const size = 16

	src := NewImage(size, size)
	defer src.MarkDisposed()
	src2 := NewImage(size, size)
	defer src2.MarkDisposed()
	dst := NewImage(size, size)
	defer dst.MarkDisposed()

	// Use src as a render target so that src is not on an atlas.
	vs := quadVertices(size, size, 0, 0, 1)
	is := graphics.QuadIndices()
	dr := driver.Region{
		X:      0,
		Y:      0,
		Width:  size,
		Height: size,
	}

	// Use src2 as a rendering target, and make src2 an independent image.
	src2.DrawTriangles([graphics.ShaderImageNum]*Image{src}, vs, is, affine.ColorMIdentity{}, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressUnsafe, dr, driver.Region{}, [graphics.ShaderImageNum - 1][2]float32{}, nil, nil, false)
	if got, want := src2.IsOnAtlasForTesting(), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Update the count without using src2 as a rendering source.
	// This should not affect whether src2 is on an atlas or not.
	for i := 0; i < BaseCountToPutOnAtlas; i++ {
		if err := PutImagesOnAtlasForTesting(); err != nil {
			t.Fatal(err)
		}
		if got, want := src2.IsOnAtlasForTesting(), false; got != want {
			t.Errorf("got: %v, want: %v", got, want)
		}
	}

	// Update the count with using src2 as a rendering source.
	for i := 0; i < BaseCountToPutOnAtlas; i++ {
		if err := PutImagesOnAtlasForTesting(); err != nil {
			t.Fatal(err)
		}
		dst.DrawTriangles([graphics.ShaderImageNum]*Image{src2}, vs, is, affine.ColorMIdentity{}, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressUnsafe, dr, driver.Region{}, [graphics.ShaderImageNum - 1][2]float32{}, nil, nil, false)
		if got, want := src2.IsOnAtlasForTesting(), false; got != want {
			t.Errorf("got: %v, want: %v", got, want)
		}
	}

	if err := PutImagesOnAtlasForTesting(); err != nil {
		t.Fatal(err)
	}
	if got, want := src2.IsOnAtlasForTesting(), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

// TODO: Add tests to extend image on an atlas out of the main loop
