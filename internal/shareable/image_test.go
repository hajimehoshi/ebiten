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

package shareable_test

import (
	"errors"
	"image/color"
	"os"
	"runtime"
	"testing"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	. "github.com/hajimehoshi/ebiten/internal/shareable"
	"github.com/hajimehoshi/ebiten/internal/testflock"
)

func TestMain(m *testing.M) {
	testflock.Lock()
	defer testflock.Unlock()

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
		dx0, dy0, sx0, sy0, sx0, sy0, sx1, sy1, 1, 1, 1, 1,
		dx1, dy0, sx1, sy0, sx0, sy0, sx1, sy1, 1, 1, 1, 1,
		dx0, dy1, sx0, sy1, sx0, sy0, sx1, sy1, 1, 1, 1, 1,
		dx1, dy1, sx1, sy1, sx0, sy0, sx1, sy1, 1, 1, 1, 1,
	}
}

const bigSize = 2049

func TestEnsureNotShared(t *testing.T) {
	// Create img1 and img2 with this size so that the next images are allocated
	// with non-upper-left location.
	img1 := NewImage(bigSize, 100, false)
	defer img1.MarkDisposed()
	// Ensure img1's region is allocated.
	img1.ReplacePixels(make([]byte, 4*bigSize*100))

	img2 := NewImage(100, bigSize, false)
	defer img2.MarkDisposed()
	img2.ReplacePixels(make([]byte, 4*100*bigSize))

	const size = 32

	img3 := NewImage(size/2, size/2, false)
	defer img3.MarkDisposed()
	img3.ReplacePixels(make([]byte, (size/2)*(size/2)*4))

	img4 := NewImage(size, size, false)
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
	// img4.ensureNotShared() should be called.
	vs := quadVertices(size/2, size/2, size/4, size/4, 1)
	is := graphics.QuadIndices()
	img4.DrawTriangles(img3, vs, is, nil, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressClampToZero)
	want := false
	if got := img4.IsSharedForTesting(); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	for j := 0; j < size; j++ {
		for i := 0; i < size; i++ {
			r, g, b, a, err := img4.At(i, j)
			if err != nil {
				t.Fatal(err)
			}
			got := color.RGBA{r, g, b, a}
			var want color.RGBA
			if i < dx0 || dx1 <= i || j < dy0 || dy1 <= j {
				c := byte(i + j)
				want = color.RGBA{c, c, c, c}
			}
			if got != want {
				t.Errorf("img4.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	// Check further drawing doesn't cause panic.
	// This bug was fixed by 03dcd948.
	img4.DrawTriangles(img3, vs, is, nil, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressClampToZero)
}

func TestReshared(t *testing.T) {
	const size = 16

	img0 := NewImage(size, size, false)
	defer img0.MarkDisposed()
	img0.ReplacePixels(make([]byte, 4*size*size))

	img1 := NewImage(size, size, false)
	defer img1.MarkDisposed()
	img1.ReplacePixels(make([]byte, 4*size*size))
	if got, want := img1.IsSharedForTesting(), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	img2 := NewImage(size, size, false)
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

	img3 := NewImage(size, size, true /* volatile */)
	defer img3.MarkDisposed()
	img1.ReplacePixels(make([]byte, 4*size*size))
	if got, want := img3.IsSharedForTesting(), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Use img1 as a render target.
	vs := quadVertices(size, size, 0, 0, 1)
	is := graphics.QuadIndices()
	img1.DrawTriangles(img2, vs, is, nil, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressClampToZero)
	if got, want := img1.IsSharedForTesting(), false; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Use img1 as a render source.
	for i := 0; i < MaxCountForShare; i++ {
		if err := MakeImagesSharedForTesting(); err != nil {
			t.Fatal(err)
		}
		img0.DrawTriangles(img1, vs, is, nil, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressClampToZero)
		if got, want := img1.IsSharedForTesting(), false; got != want {
			t.Errorf("got: %v, want: %v", got, want)
		}
	}
	if err := MakeImagesSharedForTesting(); err != nil {
		t.Fatal(err)
	}

	for j := 0; j < size; j++ {
		for i := 0; i < size; i++ {
			want := color.RGBA{byte(i + j), byte(i + j), byte(i + j), byte(i + j)}
			r, g, b, a, err := img1.At(i, j)
			if err != nil {
				t.Fatal(err)
			}
			got := color.RGBA{r, g, b, a}
			if got != want {
				t.Errorf("got: %v, want: %v", got, want)
			}
		}
	}

	img0.DrawTriangles(img1, vs, is, nil, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressClampToZero)
	if got, want := img1.IsSharedForTesting(), true; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	for j := 0; j < size; j++ {
		for i := 0; i < size; i++ {
			want := color.RGBA{byte(i + j), byte(i + j), byte(i + j), byte(i + j)}
			r, g, b, a, err := img1.At(i, j)
			if err != nil {
				t.Fatal(err)
			}
			got := color.RGBA{r, g, b, a}
			if got != want {
				t.Errorf("got: %v, want: %v", got, want)
			}
		}
	}

	// Use img3 as a render source. img3 never uses a shared texture.
	for i := 0; i < MaxCountForShare*2; i++ {
		if err := MakeImagesSharedForTesting(); err != nil {
			t.Fatal(err)
		}
		img0.DrawTriangles(img3, vs, is, nil, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressClampToZero)
		if got, want := img3.IsSharedForTesting(), false; got != want {
			t.Errorf("got: %v, want: %v", got, want)
		}
	}

	runtime.GC()
}

func TestExtend(t *testing.T) {
	const w0, h0 = 100, 100
	img0 := NewImage(w0, h0, false)
	defer img0.MarkDisposed()
	p0 := make([]byte, 4*w0*h0)
	for i := 0; i < w0*h0; i++ {
		p0[4*i] = byte(i)
		p0[4*i+1] = byte(i)
		p0[4*i+2] = byte(i)
		p0[4*i+3] = byte(i)
	}
	img0.ReplacePixels(p0)

	const w1, h1 = 1025, 100
	img1 := NewImage(w1, h1, false)
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

	for j := 0; j < h0; j++ {
		for i := 0; i < w0; i++ {
			r, g, b, a, err := img0.At(i, j)
			if err != nil {
				t.Fatal(err)
			}
			got := color.RGBA{r, g, b, a}
			c := byte(i + w0*j)
			want := color.RGBA{c, c, c, c}
			if got != want {
				t.Errorf("img0.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	for j := 0; j < h1; j++ {
		for i := 0; i < w1; i++ {
			r, g, b, a, err := img1.At(i, j)
			if err != nil {
				t.Fatal(err)
			}
			got := color.RGBA{r, g, b, a}
			c := byte(i + w1*j)
			want := color.RGBA{c, c, c, c}
			if got != want {
				t.Errorf("img1.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	img0.MarkDisposed()
	img1.MarkDisposed()
}

func TestReplacePixelsAfterDrawTriangles(t *testing.T) {
	const w, h = 256, 256
	src := NewImage(w, h, false)
	defer src.MarkDisposed()
	dst := NewImage(w, h, false)
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
	dst.DrawTriangles(src, vs, is, nil, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressClampToZero)
	dst.ReplacePixels(pix)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			r, g, b, a, err := dst.At(i, j)
			if err != nil {
				t.Fatal(err)
			}
			got := color.RGBA{r, g, b, a}
			c := byte(i + w*j)
			want := color.RGBA{c, c, c, c}
			if got != want {
				t.Errorf("dst.At(%d, %d): got %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #887
func TestSmallImages(t *testing.T) {
	const w, h = 4, 8
	src := NewImage(w, h, false)
	defer src.MarkDisposed()
	dst := NewImage(w, h, false)
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
	dst.DrawTriangles(src, vs, is, nil, driver.CompositeModeSourceOver, driver.FilterNearest, driver.AddressClampToZero)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			r, _, _, a, err := dst.At(i, j)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := r, byte(0xff); got != want {
				t.Errorf("At(%d, %d) red: got: %d, want: %d", i, j, got, want)
			}
			if got, want := a, byte(0xff); got != want {
				t.Errorf("At(%d, %d) alpha: got: %d, want: %d", i, j, got, want)
			}
		}
	}
}

// Issue #887
func TestLongImages(t *testing.T) {
	const w, h = 1, 6
	src := NewImage(w, h, false)
	defer src.MarkDisposed()
	dst := NewImage(256, 256, false)
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
	dst.DrawTriangles(src, vs, is, nil, driver.CompositeModeSourceOver, driver.FilterNearest, driver.AddressClampToZero)

	for j := 0; j < h; j++ {
		for i := 0; i < w*scale; i++ {
			r, _, _, a, err := dst.At(i, j)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := r, byte(0xff); got != want {
				t.Errorf("At(%d, %d) red: got: %d, want: %d", i, j, got, want)
			}
			if got, want := a, byte(0xff); got != want {
				t.Errorf("At(%d, %d) alpha: got: %d, want: %d", i, j, got, want)
			}
		}
	}
}

func TestDisposeImmediately(t *testing.T) {
	// This tests restorable.Image.ClearPixels is called but ReplacePixels is not called.

	img0 := NewImage(16, 16, false)
	img0.EnsureNotSharedForTesting()

	img1 := NewImage(16, 16, false)
	img1.EnsureNotSharedForTesting()

	// img0 and img1 should share the same backend in 99.9999% possibility.

	img0.MarkDisposed()
	img1.MarkDisposed()
}

// Issue #1028
func TestExtendWithBigImage(t *testing.T) {
	img0 := NewImage(1, 1, false)
	defer img0.MarkDisposed()

	img0.ReplacePixels(make([]byte, 4*1*1))

	img1 := NewImage(1025, 1025, false)
	defer img1.MarkDisposed()

	img1.ReplacePixels(make([]byte, 4*1025*1025))
}

// TODO: Add tests to extend shareable image out of the main loop
