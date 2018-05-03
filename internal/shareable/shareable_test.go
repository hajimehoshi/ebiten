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
	"testing"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/graphics"
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
	geom := (*affine.GeoM)(nil).Translate(size/4, size/4)
	img4.DrawImage(img3, 0, 0, size/2, size/2, geom, nil, opengl.CompositeModeCopy, graphics.FilterNearest)

	for j := 0; j < size; j++ {
		for i := 0; i < size; i++ {
			got, err := img4.At(i, j)
			if err != nil {
				t.Fatal(err)
			}
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
	img4.DrawImage(img3, 0, 0, size/2, size/2, geom, nil, opengl.CompositeModeCopy, graphics.FilterNearest)
}
