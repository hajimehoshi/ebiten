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

// Package packing offers a packing algorithm in 2D space.
package packing

const (
	minSize = 1
)

type Page struct {
	root    *Node
	maxSize int
}

func NewPage(maxSize int) *Page {
	return &Page{
		maxSize: maxSize,
	}
}

func (p *Page) IsEmpty() bool {
	if p.root == nil {
		return true
	}
	return !p.root.used && p.root.child0 == nil && p.root.child1 == nil
}

type Node struct {
	x      int
	y      int
	width  int
	height int

	used   bool
	parent *Node
	child0 *Node
	child1 *Node
}

func (n *Node) canFree() bool {
	if n.used {
		return false
	}
	if n.child0 == nil && n.child1 == nil {
		return true
	}
	return n.child0.canFree() && n.child1.canFree()
}

func (n *Node) Region() (x, y, width, height int) {
	return n.x, n.y, n.width, n.height
}

// square returns a float value indicating how much the given rectangle is close to a square.
// If the given rectangle is square, this return 1 (maximum value).
// Otherwise, this returns a value in [0, 1).
func square(width, height int) float64 {
	if width == 0 && height == 0 {
		return 0
	}
	if width <= height {
		return float64(width) / float64(height)
	}
	return float64(height) / float64(width)
}

func (p *Page) alloc(n *Node, width, height int) *Node {
	if n.width < width || n.height < height {
		return nil
	}
	if n.used {
		return nil
	}
	if n.child0 == nil && n.child1 == nil {
		if n.width == width && n.height == height {
			n.used = true
			return n
		}
		if square(n.width-width, n.height) >= square(n.width, n.height-height) {
			// Split vertically
			n.child0 = &Node{
				x:      n.x,
				y:      n.y,
				width:  width,
				height: n.height,
				parent: n,
			}
			n.child1 = &Node{
				x:      n.x + width,
				y:      n.y,
				width:  n.width - width,
				height: n.height,
				parent: n,
			}
		} else {
			// Split holizontally
			n.child0 = &Node{
				x:      n.x,
				y:      n.y,
				width:  n.width,
				height: height,
				parent: n,
			}
			n.child1 = &Node{
				x:      n.x,
				y:      n.y + height,
				width:  n.width,
				height: n.height - height,
				parent: n,
			}
		}
		return p.alloc(n.child0, width, height)
	}
	if n.child0 == nil || n.child1 == nil {
		panic("not reached")
	}
	if node := p.alloc(n.child0, width, height); node != nil {
		return node
	}
	if node := p.alloc(n.child1, width, height); node != nil {
		return node
	}
	return nil
}

func (p *Page) Alloc(width, height int) *Node {
	if width <= 0 || height <= 0 {
		panic("bsp: width and height must > 0")
	}
	if p.root == nil {
		p.root = &Node{
			width:  p.maxSize,
			height: p.maxSize,
		}
	}
	if width < minSize {
		width = minSize
	}
	if height < minSize {
		height = minSize
	}
	n := p.alloc(p.root, width, height)
	return n
}

func (p *Page) Free(node *Node) {
	if node.child0 != nil || node.child1 != nil {
		panic("bsp: can't free the node including children")
	}
	node.used = false
	if node.parent == nil {
		return
	}
	if node.parent.child0 == nil || node.parent.child1 == nil {
		panic("not reached")
	}
	if node.parent.child0.canFree() && node.parent.child1.canFree() {
		node.parent.child0 = nil
		node.parent.child1 = nil
		p.Free(node.parent)
	}
}
