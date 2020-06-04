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

package graphicscommand_test

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	. "github.com/hajimehoshi/ebiten/internal/graphicscommand"
	etesting "github.com/hajimehoshi/ebiten/internal/testing"
)

func TestMain(m *testing.M) {
	etesting.MainWithRunLoop(m)
}

func quadVertices(w, h float32) []float32 {
	return []float32{
		0, 0, 0, 0, 0, 0, w, h, 1, 1, 1, 1,
		w, 0, w, 0, 0, 0, w, h, 1, 1, 1, 1,
		0, w, 0, h, 0, 0, w, h, 1, 1, 1, 1,
		w, h, w, h, 0, 0, w, h, 1, 1, 1, 1,
	}
}

func TestClear(t *testing.T) {
	const w, h = 1024, 1024
	src := NewImage(w/2, h/2)
	dst := NewImage(w, h)

	vs := quadVertices(w/2, h/2)
	is := graphics.QuadIndices()
	dst.DrawTriangles(src, vs, is, nil, driver.CompositeModeClear, driver.FilterNearest, driver.AddressClampToZero, nil, nil)

	pix, err := dst.Pixels()
	if err != nil {
		t.Fatal(err)
	}
	for j := 0; j < h/2; j++ {
		for i := 0; i < w/2; i++ {
			idx := 4 * (i + w*j)
			got := color.RGBA{pix[idx], pix[idx+1], pix[idx+2], pix[idx+3]}
			want := color.RGBA{}
			if got != want {
				t.Errorf("dst.At(%d, %d) after DrawTriangles: got %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestReplacePixelsPartAfterDrawTriangles(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("ReplacePixels must panic but not")
		}
	}()
	const w, h = 32, 32
	clr := NewImage(w, h)
	src := NewImage(w/2, h/2)
	dst := NewImage(w, h)
	vs := quadVertices(w/2, h/2)
	is := graphics.QuadIndices()
	dst.DrawTriangles(clr, vs, is, nil, driver.CompositeModeClear, driver.FilterNearest, driver.AddressClampToZero, nil, nil)
	dst.DrawTriangles(src, vs, is, nil, driver.CompositeModeSourceOver, driver.FilterNearest, driver.AddressClampToZero, nil, nil)
	dst.ReplacePixels(make([]byte, 4), 0, 0, 1, 1)
}

func TestShader(t *testing.T) {
	if !IsShaderAvailable() {
		t.Skip("shader is not implemented on this environment")
	}

	const w, h = 16, 16
	clr := NewImage(w, h)
	dst := NewImage(w, h)
	vs := quadVertices(w, h)
	is := graphics.QuadIndices()
	dst.DrawTriangles(clr, vs, is, nil, driver.CompositeModeClear, driver.FilterNearest, driver.AddressClampToZero, nil, nil)

	ir := etesting.ShaderProgramFill(0xff, 0, 0, 0xff)
	s := NewShader(&ir)
	us := []interface{}{
		[]float32{w, h},
	}
	dst.DrawTriangles(nil, vs, is, nil, driver.CompositeModeSourceOver, 0, 0, s, us)

	pix, err := dst.Pixels()
	if err != nil {
		t.Fatal(err)
	}
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			idx := 4 * (i + w*j)
			got := color.RGBA{pix[idx], pix[idx+1], pix[idx+2], pix[idx+3]}
			want := color.RGBA{0xff, 0, 0, 0xff}
			if got != want {
				t.Errorf("dst.At(%d, %d) after DrawTriangles: got %v, want: %v", i, j, got, want)
			}
		}
	}
}
