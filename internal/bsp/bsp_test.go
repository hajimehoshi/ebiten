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

package bsp_test

import (
	"testing"

	. "github.com/hajimehoshi/ebiten/internal/bsp"
)

func TestBSP(t *testing.T) {
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
		In  []Op
		Out []*Rect
	}{
		{
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
				{MaxSize, MaxSize, -1},
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
				{0, 0, MaxSize, MaxSize},
			},
		},
		{
			In: []Op{
				{200, 400, -1},
				{MaxSize, MaxSize, -1},
				{200, 400, -1},
				{100, 100, -1},
				{400, 400, -1},
				{MaxSize, MaxSize, -1},
				{1000, 1000, -1},
				{1200, 1200, -1},
				{200, 200, -1},
				{0, 0, 2},
				{200, 400, -1},
			},
			Out: []*Rect{
				{0, 0, 200, 400},
				nil,
				{0, 400, 200, 400},
				{0, 800, 100, 100},
				{200, 0, 400, 400},
				nil,
				{200, 400, 1000, 1000},
				nil,
				{0, 900, 200, 200},
				nil,
				{0, 400, 200, 400},
			},
		},
		{
			In: []Op{
				{512, 512, -1},
				{512, 512, -1},
				{512, 512, -1},
				{512, 512, -1},

				{512, 512, -1},
				{512, 512, -1},
				{512, 512, -1},
				{512, 512, -1},

				{512, 512, -1},
				{512, 512, -1},
				{512, 512, -1},
				{512, 512, -1},

				{512, 512, -1},
				{512, 512, -1},
				{512, 512, -1},
				{512, 512, -1},

				{512, 512, -1},
			},
			Out: []*Rect{
				{0, 0, 512, 512},
				{0, 512, 512, 512},
				{0, 1024, 512, 512},
				{0, 1536, 512, 512},

				{512, 0, 512, 512},
				{1024, 0, 512, 512},
				{1536, 0, 512, 512},
				{512, 512, 512, 512},

				{512, 1024, 512, 512},
				{512, 1536, 512, 512},
				{1024, 512, 512, 512},
				{1536, 512, 512, 512},

				{1024, 1024, 512, 512},
				{1024, 1536, 512, 512},
				{1536, 1024, 512, 512},
				{1536, 1536, 512, 512},

				nil,
			},
		},
		{
			In: []Op{
				{600, 600, -1},
				{600, 600, -1},
				{600, 600, -1},
				{600, 600, -1},
				{600, 600, -1},
				{600, 600, -1},
				{600, 600, -1},
				{600, 600, -1},
				{600, 600, -1},
				{600, 600, -1},
			},
			Out: []*Rect{
				{0, 0, 600, 600},
				{0, 600, 600, 600},
				{0, 1200, 600, 600},
				{600, 0, 600, 600},
				{1200, 0, 600, 600},
				{600, 600, 600, 600},
				{600, 1200, 600, 600},
				{1200, 600, 600, 600},
				{1200, 1200, 600, 600},
				nil,
			},
		},
	}

	for caseIndex, c := range cases {
		p := &Page{}
		nodes := []*Node{}
		for _, in := range c.In {
			if in.FreeNodeID == -1 {
				n := p.Alloc(in.Width, in.Height)
				nodes = append(nodes, n)
			} else {
				p.Free(nodes[in.FreeNodeID])
				nodes = append(nodes, nil)
			}
		}
		for i, out := range c.Out {
			if nodes[i] == nil {
				if out != nil {
					t.Errorf("(%d) nodes[%d]: should be nil but %v", caseIndex, i, out)
				}
				continue
			}
			x, y, width, height := nodes[i].Region()
			got := Rect{x, y, width, height}
			if out == nil {
				t.Errorf("(%d) nodes[%d]: got: %v, want: %v", caseIndex, i, got, nil)
				continue
			}
			want := *out
			if got != want {
				t.Errorf("(%d) nodes[%d]: got: %v, want: %v", caseIndex, i, got, want)
			}
		}
	}
}
