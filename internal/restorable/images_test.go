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
	"image/color"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/opengl"
	. "github.com/hajimehoshi/ebiten/internal/restorable"
)

func TestMain(m *testing.M) {
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

func uint8SliceToColor(b []uint8) color.RGBA {
	return color.RGBA{b[0], b[1], b[2], b[3]}
}

func TestRestore(t *testing.T) {
	img0 := NewImage(1, 1, opengl.Nearest, false)
	defer img0.Dispose()
	clr0 := color.RGBA{0x00, 0x00, 0x00, 0xff}
	img0.Fill(clr0)
	if err := ResolveStalePixels(); err != nil {
		t.Fatal(err)
	}
	if err := Restore(); err != nil {
		t.Fatal(err)
	}
	want := clr0
	got := uint8SliceToColor(img0.BasePixelsForTesting())
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func vertices() []float32 {
	const a, b, c, d, tx, ty = 1, 0, 0, 1, 0, 0
	return []float32{
		0, 0, 0, 0, a, b, c, d, tx, ty,
		0, 1, 0, 1, a, b, c, d, tx, ty,
		1, 0, 1, 0, a, b, c, d, tx, ty,
		1, 1, 1, 1, a, b, c, d, tx, ty,
	}
}

func TestRestoreReversedOrder(t *testing.T) {
	img0 := NewImage(1, 1, opengl.Nearest, false)
	img1 := NewImage(1, 1, opengl.Nearest, false)
	img2 := NewImage(1, 1, opengl.Nearest, false)
	img3 := NewImage(1, 1, opengl.Nearest, false)
	println(img0, img1, img2, img3)
	defer func() {
		img3.Dispose()
		img2.Dispose()
		img1.Dispose()
		img0.Dispose()
	}()
	clr0 := color.RGBA{0x00, 0x00, 0x00, 0xff}
	clr1 := color.RGBA{0x00, 0x00, 0x01, 0xff}
	img1.Fill(clr0)
	img2.DrawImage(img1, vertices(), &affine.ColorM{}, opengl.CompositeModeSourceOver)
	img3.DrawImage(img2, vertices(), &affine.ColorM{}, opengl.CompositeModeSourceOver)
	img0.Fill(clr1)
	img1.DrawImage(img0, vertices(), &affine.ColorM{}, opengl.CompositeModeSourceOver)
	if err := ResolveStalePixels(); err != nil {
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
			uint8SliceToColor(img0.BasePixelsForTesting()),
		},
		{
			"1",
			clr1,
			uint8SliceToColor(img1.BasePixelsForTesting()),
		},
		{
			"2",
			clr0,
			uint8SliceToColor(img2.BasePixelsForTesting()),
		},
		{
			"3",
			clr0,
			uint8SliceToColor(img3.BasePixelsForTesting()),
		},
	}
	for _, c := range testCases {
		if c.got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, c.got, c.want)
		}
	}
}
