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
	"testing"

	. "github.com/hajimehoshi/ebiten/internal/packing"
)

func TestPage(t *testing.T) {
	type Rect struct {
		X      int
		Y      int
		Width  int
		Height int
	}

	type Op struct {
		Width      int
		Height     int
		FreeNodeID int
	}

	cases := []struct {
		Name string
		In   []Op
		Out  []*Rect
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
			Out: []*Rect{
				{0, 0, 100, 100},
				{0, 100, 100, 100},
				{0, 200, 100, 100},
				{0, 300, 100, 100},
				{0, 400, 100, 100},
				{0, 500, 100, 100},
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				{0, 0, 1024, 1024},
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
			Out: []*Rect{
				{0, 0, 31, 41},
				{31, 0, 59, 26},
				{31, 26, 53, 58},
				{31, 84, 97, 93},
				{0, 41, 28, 84},
				{31, 177, 62, 64},
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
			Out: []*Rect{
				{0, 0, 100, 200},
				nil,
				{0, 200, 100, 200},
				{0, 400, 50, 50},
				{100, 0, 200, 200},
				nil,
				{100, 200, 500, 500},
				nil,
				{0, 450, 100, 100},
				nil,
				{0, 200, 100, 200},
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
			Out: []*Rect{
				{0, 0, 256, 256},
				{0, 256, 256, 256},
				{0, 512, 256, 256},
				{0, 768, 256, 256},

				{256, 0, 256, 256},
				{512, 0, 256, 256},
				{768, 0, 256, 256},
				{256, 256, 256, 256},

				{256, 512, 256, 256},
				{256, 768, 256, 256},
				{512, 256, 256, 256},
				{768, 256, 256, 256},

				{512, 512, 256, 256},
				{512, 768, 256, 256},
				{768, 512, 256, 256},
				{768, 768, 256, 256},

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
			Out: []*Rect{
				{0, 0, 300, 300},
				{0, 300, 300, 300},
				{0, 600, 300, 300},
				{300, 0, 300, 300},
				{600, 0, 300, 300},
				{300, 300, 300, 300},
				{300, 600, 300, 300},
				{600, 300, 300, 300},
				{600, 600, 300, 300},
				nil,
			},
		},
	}

	for _, c := range cases {
		p := NewPage(1024, 1024)
		nodes := []*Node{}
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
			x, y, width, height := nodes[i].Region()
			got := Rect{x, y, width, height}
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

func TestExtend(t *testing.T) {
	p := NewPage(1024, 4096)
	s := p.Size()
	p.Alloc(s/2, s/2)
	p.Extend()
	if p.Size() != s*2 {
		t.Errorf("p.Size(): got: %d, want: %d", p.Size(), s*2)
	}
	if p.Alloc(s*3/2, s*2) == nil {
		t.Errorf("p.Alloc failed: width: %d, height: %d", s*3/2, s*2)
	}
	if p.Alloc(s/2, s*3/2) == nil {
		t.Errorf("p.Alloc failed: width: %d, height: %d", s/2, s*3/2)
	}
	if p.Alloc(1, 1) != nil {
		t.Errorf("p.Alloc must fail: width: %d, height: %d", 1, 1)
	}
}

func TestExtend2(t *testing.T) {
	p := NewPage(1024, 4096)
	s := p.Size()
	p.Alloc(s/2, s/2)
	n1 := p.Alloc(s/2, s/2)
	n2 := p.Alloc(s/2, s/2)
	p.Alloc(s/2, s/2)
	p.Free(n1)
	p.Free(n2)
	p.Extend()
	if p.Size() != s*2 {
		t.Errorf("p.Size(): got: %d, want: %d", p.Size(), s*2)
	}
	if p.Alloc(s, s*2) == nil {
		t.Errorf("p.Alloc failed: width: %d, height: %d", s, s*2)
	}
	if p.Alloc(s, s) == nil {
		t.Errorf("p.Alloc failed: width: %d, height: %d", s, s)
	}
	if p.Alloc(s, s) != nil {
		t.Errorf("p.Alloc must fail: width: %d, height: %d", s, s)
	}
}
