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

import (
	"errors"
)

const (
	minSize = 1
)

type Page struct {
	root    *Node
	size    int
	maxSize int

	rollbackExtension func()
}

func NewPage(initSize int, maxSize int) *Page {
	return &Page{
		size:    initSize,
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
		panic("packing: both two children must not be nil at alloc")
	}
	if node := p.alloc(n.child0, width, height); node != nil {
		return node
	}
	if node := p.alloc(n.child1, width, height); node != nil {
		return node
	}
	return nil
}

func (p *Page) Size() int {
	return p.size
}

func (p *Page) SetMaxSize(size int) {
	if p.maxSize > size {
		panic("packing: maxSize cannot be decreased")
	}
	p.maxSize = size
}

func (p *Page) Alloc(width, height int) *Node {
	if width <= 0 || height <= 0 {
		panic("packing: width and height must > 0")
	}
	if p.root == nil {
		p.root = &Node{
			width:  p.size,
			height: p.size,
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
		panic("packing: can't free the node including children")
	}
	node.used = false
	if node.parent == nil {
		return
	}
	if node.parent.child0 == nil || node.parent.child1 == nil {
		panic("packing: both two children must not be nil at Free: double free happened?")
	}
	if node.parent.child0.canFree() && node.parent.child1.canFree() {
		node.parent.child0 = nil
		node.parent.child1 = nil
		p.Free(node.parent)
	}
}

func walk(n *Node, f func(n *Node) error) error {
	if err := f(n); err != nil {
		return err
	}
	if n.child0 != nil {
		if err := walk(n.child0, f); err != nil {
			return err
		}
	}
	if n.child1 != nil {
		if err := walk(n.child1, f); err != nil {
			return err
		}
	}
	return nil
}

func (p *Page) Extend(count int) bool {
	if p.rollbackExtension != nil {
		panic("packing: Extend cannot be called without rolling back or committing")
	}

	if p.size >= p.maxSize {
		return false
	}

	newSize := p.size
	for i := 0; i < count; i++ {
		newSize *= 2
	}

	if newSize > p.maxSize {
		return false
	}

	edgeNodes := []*Node{}
	abort := errors.New("abort")
	aborted := false
	if p.root != nil {
		_ = walk(p.root, func(n *Node) error {
			if n.x+n.width < p.size && n.y+n.height < p.size {
				return nil
			}
			if n.used {
				aborted = true
				return abort
			}
			edgeNodes = append(edgeNodes, n)
			return nil
		})
	}

	if aborted {
		origRoot := *p.root

		leftUpper := p.root
		leftLower := &Node{
			x:      0,
			y:      p.size,
			width:  p.size,
			height: newSize - p.size,
		}
		left := &Node{
			x:      0,
			y:      0,
			width:  p.size,
			height: p.size,
			child0: leftUpper,
			child1: leftLower,
		}
		leftUpper.parent = left
		leftLower.parent = left

		right := &Node{
			x:      p.size,
			y:      0,
			width:  newSize - p.size,
			height: newSize,
		}
		p.root = &Node{
			x:      0,
			y:      0,
			width:  newSize,
			height: newSize,
			child0: left,
			child1: right,
		}
		left.parent = p.root
		right.parent = p.root

		origSize := p.size
		p.rollbackExtension = func() {
			p.size = origSize
			p.root = &origRoot
		}
	} else {
		origSize := p.size
		origWidths := map[*Node]int{}
		origHeights := map[*Node]int{}

		for _, n := range edgeNodes {
			if n.x+n.width == p.size {
				origWidths[n] = n.width
				n.width += newSize - p.size
			}
			if n.y+n.height == p.size {
				origHeights[n] = n.height
				n.height += newSize - p.size
			}
		}

		p.rollbackExtension = func() {
			p.size = origSize
			for n, w := range origWidths {
				n.width = w
			}
			for n, h := range origHeights {
				n.height = h
			}
		}
	}

	p.size = newSize

	return true
}

// RollbackExtension rollbacks Extend call once.
func (p *Page) RollbackExtension() {
	if p.rollbackExtension == nil {
		panic("packing: RollbackExtension cannot be called without Extend")
	}
	p.rollbackExtension()
	p.rollbackExtension = nil
}

// CommitExtension commits Extend call.
func (p *Page) CommitExtension() {
	if p.rollbackExtension == nil {
		panic("packing: RollbackExtension cannot be called without Extend")
	}
	p.rollbackExtension = nil
}
