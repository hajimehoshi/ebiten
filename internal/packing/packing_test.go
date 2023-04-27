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

package packing_test

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/packing"
)

func TestPage(t *testing.T) {
	rect := func(x0, y0, x1, y1 int) *image.Rectangle {
		r := image.Rect(x0, y0, x1, y1)
		return &r
	}

	type Op struct {
		Width      int
		Height     int
		FreeNodeID int
	}

	cases := []struct {
		Name string
		In   []Op
		Out  []*image.Rectangle
	}{
		{
			Name: "alloc and random free",
			In: []Op{
				{100, 100, -1},
				{100, 100, -1},
				{100, 100, -1},
				{100, 100, -1},
				{100, 100, -1},
				{100, 100, -1},
				{0, 0, 1},
				{0, 0, 3},
				{0, 0, 5},
				{0, 0, 0},
				{0, 0, 2},
				{0, 0, 4},
				{1024, 1024, -1},
			},
			Out: []*image.Rectangle{
				rect(0, 0, 100, 100),
				rect(0, 100, 100, 200),
				rect(0, 200, 100, 300),
				rect(0, 300, 100, 400),
				rect(0, 400, 100, 500),
				rect(0, 500, 100, 600),
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				rect(0, 0, 1024, 1024),
			},
		},
		{
			Name: "alloc and free and empty",
			In: []Op{
				{31, 41, -1},
				{59, 26, -1},
				{53, 58, -1},
				{97, 93, -1},
				{28, 84, -1},
				{62, 64, -1},
				{0, 0, 0},
				{0, 0, 1},
				{0, 0, 2},
				{0, 0, 3},
				{0, 0, 4},
				{0, 0, 5},
			},
			Out: []*image.Rectangle{
				rect(0, 0, 31, 41),
				rect(31, 0, 31+59, 26),
				rect(31, 26, 31+53, 26+58),
				rect(31, 84, 31+97, 84+93),
				rect(0, 41, 28, 41+84),
				rect(31, 177, 31+62, 177+64),
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
			},
		},
		{
			Name: "random Alloc",
			In: []Op{
				{100, 200, -1},
				{1024, 1024, -1},
				{100, 200, -1},
				{50, 50, -1},
				{200, 200, -1},
				{1024, 1024, -1},
				{500, 500, -1},
				{600, 600, -1},
				{100, 100, -1},
				{0, 0, 2},
				{100, 200, -1},
			},
			Out: []*image.Rectangle{
				rect(0, 0, 100, 200),
				nil,
				rect(0, 200, 100, 400),
				rect(0, 400, 50, 450),
				rect(100, 0, 300, 200),
				nil,
				rect(100, 200, 600, 700),
				nil,
				rect(0, 450, 100, 550),
				nil,
				rect(0, 200, 100, 400),
			},
		},
		{
			Name: "fill squares",
			In: []Op{
				{256, 256, -1},
				{256, 256, -1},
				{256, 256, -1},
				{256, 256, -1},

				{256, 256, -1},
				{256, 256, -1},
				{256, 256, -1},
				{256, 256, -1},

				{256, 256, -1},
				{256, 256, -1},
				{256, 256, -1},
				{256, 256, -1},

				{256, 256, -1},
				{256, 256, -1},
				{256, 256, -1},
				{256, 256, -1},

				{256, 256, -1},
			},
			Out: []*image.Rectangle{
				rect(0, 0, 256, 256),
				rect(0, 256, 256, 512),
				rect(0, 512, 256, 768),
				rect(0, 768, 256, 1024),

				rect(256, 0, 512, 256),
				rect(512, 0, 768, 256),
				rect(768, 0, 1024, 256),
				rect(256, 256, 512, 512),

				rect(256, 512, 512, 768),
				rect(256, 768, 512, 1024),
				rect(512, 256, 768, 512),
				rect(768, 256, 1024, 512),

				rect(512, 512, 768, 768),
				rect(512, 768, 768, 1024),
				rect(768, 512, 1024, 768),
				rect(768, 768, 1024, 1024),

				nil,
			},
		},
		{
			Name: "fill not fitting squares",
			In: []Op{
				{300, 300, -1},
				{300, 300, -1},
				{300, 300, -1},
				{300, 300, -1},
				{300, 300, -1},
				{300, 300, -1},
				{300, 300, -1},
				{300, 300, -1},
				{300, 300, -1},
				{300, 300, -1},
			},
			Out: []*image.Rectangle{
				rect(0, 0, 300, 300),
				rect(0, 300, 300, 600),
				rect(0, 600, 300, 900),
				rect(300, 0, 600, 300),
				rect(600, 0, 900, 300),
				rect(300, 300, 600, 600),
				rect(300, 600, 600, 900),
				rect(600, 300, 900, 600),
				rect(600, 600, 900, 900),
				nil,
			},
		},
	}

	for _, c := range cases {
		p := packing.NewPage(1024, 1024, 1024)
		nodes := []*packing.Node{}
		nnodes := 0
		for i, in := range c.In {
			if in.FreeNodeID == -1 {
				n := p.Alloc(in.Width, in.Height)
				nodes = append(nodes, n)
				nnodes++
			} else {
				p.Free(nodes[in.FreeNodeID])
				nodes = append(nodes, nil)
				nnodes--
			}
			if nnodes < 0 {
				panic("not reached")
			}

			if p.IsEmpty() != (nnodes == 0) {
				t.Errorf("%s: nodes[%d]: page.IsEmpty(): got: %v, want: %v", c.Name, i, p.IsEmpty(), (nnodes == 0))
			}
		}
		for i, out := range c.Out {
			if nodes[i] == nil {
				if out != nil {
					t.Errorf("%s: nodes[%d]: should be nil but %v", c.Name, i, out)
				}
				continue
			}
			got := nodes[i].Region()
			if out == nil {
				t.Errorf("%s: nodes[%d]: got: %v, want: %v", c.Name, i, got, nil)
				continue
			}
			want := *out
			if got != want {
				t.Errorf("%s: nodes[%d]: got: %v, want: %v", c.Name, i, got, want)
			}
		}
	}
}

func TestAlloc(t *testing.T) {
	p := packing.NewPage(1024, 1024, 2048)
	w, h := p.Size()
	p.Alloc(w/2, h/2)

	n0 := p.Alloc(w*3/2, h*2)
	if n0 == nil {
		t.Errorf("p.Alloc failed: width: %d, height: %d", w*3/2, h*2)
	}
	n1 := p.Alloc(w/2, h*3/2)
	if n1 == nil {
		t.Errorf("p.Alloc failed: width: %d, height: %d", w/2, h*3/2)
	}
	if p.Alloc(1, 1) != nil {
		t.Errorf("p.Alloc(1, 1) must fail but not")
	}
	p.Free(n1)
	if p.Alloc(1, 1) == nil {
		t.Errorf("p.Alloc(1, 1) failed")
	}
	p.Free(n0)
}

func TestAlloc2(t *testing.T) {
	p := packing.NewPage(1024, 1024, 2048)
	w, h := p.Size()
	p.Alloc(w/2, h/2)
	n1 := p.Alloc(w/2, h/2)
	n2 := p.Alloc(w/2, h/2)
	p.Alloc(w/2, h/2)
	p.Free(n1)
	p.Free(n2)

	n3 := p.Alloc(w, h*2)
	if n3 == nil {
		t.Errorf("p.Alloc failed: width: %d, height: %d", w, h*2)
	}
	n4 := p.Alloc(w, h)
	if n4 == nil {
		t.Errorf("p.Alloc failed: width: %d, height: %d", w, h)
	}
	p.Free(n4)
	p.Free(n3)
}

func TestAllocJustSize(t *testing.T) {
	p := packing.NewPage(1024, 1024, 4096)
	if p.Alloc(4096, 4096) == nil {
		t.Errorf("got: nil, want: non-nil")
	}
}

// Issue #1454
func TestAllocTooMuch(t *testing.T) {
	p := packing.NewPage(1024, 1024, 4096)
	p.Alloc(1, 1)
	if p.Alloc(4096, 4096) != nil {
		t.Errorf("got: non-nil, want: nil")
	}
}

func TestNonSquareAlloc(t *testing.T) {
	p := packing.NewPage(1024, 1024, 16384)
	n0 := p.Alloc(16384, 1)
	if _, h := p.Size(); h != 1024 {
		t.Errorf("got: %d, want: 1024", h)
	}
	n1 := p.Alloc(16384, 1)
	if _, h := p.Size(); h != 1024 {
		t.Errorf("got: %d, want: 1024", h)
	}
	p.Free(n0)
	p.Free(n1)
}

func TestExtend(t *testing.T) {
	p := packing.NewPage(1024, 1024, 2048)
	n0 := p.Alloc(1024, 1024)
	if n0 == nil {
		t.Errorf("p.Alloc(1024, 1024) failed")
	}

	// In the current implementation, the page is extended in a horizontal direction first.
	n1 := p.Alloc(1024, 1024)
	if n1 == nil {
		t.Errorf("p.Alloc(1024, 1024) failed")
	}
	gotWidth, gotHeight := p.Size()
	if wantWidth, wantHeight := 2048, 1024; gotWidth != wantWidth || gotHeight != wantHeight {
		t.Errorf("got: (%d, %d), want: (%d, %d)", gotWidth, gotHeight, wantWidth, wantHeight)
	}

	// Then, allocating (1024, 2048) fails unfortunately.
	if p.Alloc(1024, 2048) != nil {
		t.Errorf("p.Alloc(1024, 2048) must fail but not")
	}

	n2 := p.Alloc(2048, 1024)
	if n2 == nil {
		t.Errorf("p.Alloc(1024, 2048) failed")
	}
	gotWidth, gotHeight = p.Size()
	if wantWidth, wantHeight := 2048, 2048; gotWidth != wantWidth || gotHeight != wantHeight {
		t.Errorf("got: (%d, %d), want: (%d, %d)", gotWidth, gotHeight, wantWidth, wantHeight)
	}

	p.Free(n0)
	p.Free(n1)
	p.Free(n2)
}

func TestExtend2(t *testing.T) {
	p := packing.NewPage(1024, 1024, 2048)
	n0 := p.Alloc(1024, 1024)
	if n0 == nil {
		t.Errorf("p.Alloc(1024, 1024) failed")
	}

	// Extend the page in the both directions. (1024, 2048) should be allocated on the right side.
	n1 := p.Alloc(1024, 2048)
	if n1 == nil {
		t.Errorf("p.Alloc(1024, 2048) failed")
	}
	gotWidth, gotHeight := p.Size()
	if wantWidth, wantHeight := 2048, 2048; gotWidth != wantWidth || gotHeight != wantHeight {
		t.Errorf("got: (%d, %d), want: (%d, %d)", gotWidth, gotHeight, wantWidth, wantHeight)
	}

	// There should be a space in the lower-left corner.
	n2 := p.Alloc(1024, 1024)
	if n2 == nil {
		t.Errorf("p.Alloc(1024, 1024) failed")
	}
	gotWidth, gotHeight = p.Size()
	if wantWidth, wantHeight := 2048, 2048; gotWidth != wantWidth || gotHeight != wantHeight {
		t.Errorf("got: (%d, %d), want: (%d, %d)", gotWidth, gotHeight, wantWidth, wantHeight)
	}

	p.Free(n0)
	p.Free(n1)
	p.Free(n2)
}

func TestExtend3(t *testing.T) {
	p := packing.NewPage(1024, 1024, 2048)

	// Allocate a small area that doesn't touch the left edge and the bottom edge.
	// Allocating (1, 1) would split the entire region into left and right in the current implementation,
	// so allocate (2, 1) here.
	n0 := p.Alloc(2, 1)
	if n0 == nil {
		t.Errorf("p.Alloc(1, 1) failed")
	}

	// Extend the page in the vertical direction.
	n1 := p.Alloc(1024, 2047)
	if n1 == nil {
		t.Errorf("p.Alloc(1024, 2047) failed")
	}
	gotWidth, gotHeight := p.Size()
	if wantWidth, wantHeight := 1024, 2048; gotWidth != wantWidth || gotHeight != wantHeight {
		t.Errorf("got: (%d, %d), want: (%d, %d)", gotWidth, gotHeight, wantWidth, wantHeight)
	}

	// There should be a space on the right side.
	n2 := p.Alloc(1024, 2048)
	if n2 == nil {
		t.Errorf("p.Alloc(1024, 2048) failed")
	}
	gotWidth, gotHeight = p.Size()
	if wantWidth, wantHeight := 2048, 2048; gotWidth != wantWidth || gotHeight != wantHeight {
		t.Errorf("got: (%d, %d), want: (%d, %d)", gotWidth, gotHeight, wantWidth, wantHeight)
	}

	p.Free(n0)
	p.Free(n1)
	p.Free(n2)
}

// Issue #2584
func TestRemoveAtRootsChild(t *testing.T) {
	p := packing.NewPage(32, 32, 1024)
	n0 := p.Alloc(18, 18)
	n1 := p.Alloc(28, 59)
	n2 := p.Alloc(18, 18)
	n3 := p.Alloc(18, 18)
	n4 := p.Alloc(8, 10)
	n5 := p.Alloc(322, 242)
	_ = n5
	p.Free(n0)
	p.Free(n2)
	p.Free(n1)
	p.Free(n3)
	p.Free(n4)

	n6 := p.Alloc(18, 18)
	p.Free(n6)
}
