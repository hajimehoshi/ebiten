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
	"errors"
	"image/color"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	. "github.com/hajimehoshi/ebiten/internal/graphicscommand"
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

type testVertexPutter struct {
	w float32
	h float32
}

func (t *testVertexPutter) PutVertex(vs []float32, dx, dy, sx, sy float32, bx0, by0, bx1, by1 float32, cr, cg, cb, ca float32) {
	// The implementation is basically same as restorable.(*Image).PutVertex.
	vs[0] = dx
	vs[1] = dy
	vs[2] = sx / t.w
	vs[3] = sy / t.h
	vs[4] = bx0 / t.w
	vs[5] = by0 / t.h
	vs[6] = bx1 / t.w
	vs[7] = by1 / t.h
	vs[8] = cr
	vs[9] = cg
	vs[10] = cb
	vs[11] = ca
}

func TestClear(t *testing.T) {
	const w, h = 1024, 1024
	src := NewImage(w/2, h/2)
	dst := NewImage(w, h)

	vs := make([]float32, 4*graphics.VertexFloatNum)
	graphics.PutQuadVertices(vs, &testVertexPutter{w / 2, h / 2}, 0, 0, w/2, h/2, 1, 0, 0, 1, 0, 0, 1, 1, 1, 1)
	is := graphics.QuadIndices()
	dst.DrawTriangles(src, vs, is, nil, graphics.CompositeModeClear, graphics.FilterNearest, graphics.AddressClampToZero)

	pix := dst.Pixels()
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
	src := NewImage(16, 16)
	dst := NewImage(w, h)
	vs := make([]float32, 4*graphics.VertexFloatNum)
	graphics.PutQuadVertices(vs, &testVertexPutter{w / 2, h / 2}, 0, 0, w, h, 1, 0, 0, 1, 0, 0, 1, 1, 1, 1)
	is := graphics.QuadIndices()
	dst.DrawTriangles(clr, vs, is, nil, graphics.CompositeModeClear, graphics.FilterNearest, graphics.AddressClampToZero)
	dst.DrawTriangles(src, vs, is, nil, graphics.CompositeModeSourceOver, graphics.FilterNearest, graphics.AddressClampToZero)
	dst.ReplacePixels(make([]byte, 4), 0, 0, 1, 1)
}
