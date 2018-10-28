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
	"github.com/hajimehoshi/ebiten/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/internal/graphicsutil"
	"github.com/hajimehoshi/ebiten/internal/opengl"
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

const bigSize = 2049

func TestEnsureNotShared(t *testing.T) {
	// Create img1 and img2 with this size so that the next images are allocated
	// with non-upper-left location.
	img1 := NewImage(bigSize, 100)
	defer img1.Dispose()
	// Ensure img1's region is allocated.
	img1.ReplacePixels(make([]byte, 4*bigSize*100))

	img2 := NewImage(100, bigSize)
	defer img2.Dispose()
	img2.ReplacePixels(make([]byte, 4*100*bigSize))

	const size = 32

	img3 := NewImage(size/2, size/2)
	defer img3.Dispose()
	img3.ReplacePixels(make([]byte, (size/2)*(size/2)*4))

	img4 := NewImage(size, size)
	defer img4.Dispose()

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
	vs := img3.QuadVertices(0, 0, size/2, size/2, 1, 0, 0, 1, size/4, size/4, 1, 1, 1, 1)
	is := graphicsutil.QuadIndices()
	img4.DrawImage(img3, vs, is, nil, opengl.CompositeModeCopy, graphicscommand.FilterNearest)
	want := false
	if got := img4.IsSharedForTesting(); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	for j := 0; j < size; j++ {
		for i := 0; i < size; i++ {
			got := img4.At(i, j)
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
	img4.DrawImage(img3, vs, is, nil, opengl.CompositeModeCopy, graphicscommand.FilterNearest)
}

func Disabled_TestReshared(t *testing.T) {
	const size = 16

	img0 := NewImage(size, size)
	defer img0.Dispose()
	img0.ReplacePixels(make([]byte, 4*size*size))

	img1 := NewImage(size, size)
	defer img1.Dispose()
	img1.ReplacePixels(make([]byte, 4*size*size))
	want := true
	if got := img1.IsSharedForTesting(); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	img2 := NewImage(size, size)
	defer img2.Dispose()
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

	img3 := NewVolatileImage(size, size)
	defer img3.Dispose()
	img1.ReplacePixels(make([]byte, 4*size*size))
	want = false
	if got := img3.IsSharedForTesting(); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Use img1 as a render target.
	vs := img2.QuadVertices(0, 0, size, size, 1, 0, 0, 1, 0, 0, 1, 1, 1, 1)
	is := graphicsutil.QuadIndices()
	img1.DrawImage(img2, vs, is, nil, opengl.CompositeModeCopy, graphicscommand.FilterNearest)
	want = false
	if got := img1.IsSharedForTesting(); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	// Use img1 as a render source.
	for i := 0; i < MaxCountForShare-1; i++ {
		img0.DrawImage(img1, vs, is, nil, opengl.CompositeModeCopy, graphicscommand.FilterNearest)
		want := false
		if got := img1.IsSharedForTesting(); got != want {
			t.Errorf("got: %v, want: %v", got, want)
		}
	}

	for j := 0; j < size; j++ {
		for i := 0; i < size; i++ {
			want := color.RGBA{byte(i + j), byte(i + j), byte(i + j), byte(i + j)}
			got := img1.At(i, j)
			if got != want {
				t.Errorf("got: %v, want: %v", got, want)
			}
		}
	}

	img0.DrawImage(img1, vs, is, nil, opengl.CompositeModeCopy, graphicscommand.FilterNearest)
	want = true
	if got := img1.IsSharedForTesting(); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	for j := 0; j < size; j++ {
		for i := 0; i < size; i++ {
			want := color.RGBA{byte(i + j), byte(i + j), byte(i + j), byte(i + j)}
			got := img1.At(i, j)
			if got != want {
				t.Errorf("got: %v, want: %v", got, want)
			}
		}
	}

	// Use img3 as a render source. img3 never uses a shared texture.
	for i := 0; i < MaxCountForShare*2; i++ {
		img0.DrawImage(img3, vs, is, nil, opengl.CompositeModeCopy, graphicscommand.FilterNearest)
		want := false
		if got := img3.IsSharedForTesting(); got != want {
			t.Errorf("got: %v, want: %v", got, want)
		}
	}

	runtime.GC()
}
