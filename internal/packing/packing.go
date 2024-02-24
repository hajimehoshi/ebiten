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
	"fmt"
	"image"
)

type Page struct {
	root    *Node
	width   int
	height  int
	maxSize int
}

func isPositivePowerOf2(x int) bool {
	if x <= 0 {
		return false
	}
	return x&(x-1) == 0
}

func NewPage(initWidth, initHeight int, maxSize int) *Page {
	if !isPositivePowerOf2(initWidth) {
		panic(fmt.Sprintf("packing: initWidth must be a positive power of 2 but %d", initWidth))
	}
	if !isPositivePowerOf2(initHeight) {
		panic(fmt.Sprintf("packing: initHeight must be a positive power of 2 but %d", initHeight))
	}
	if !isPositivePowerOf2(maxSize) {
		panic(fmt.Sprintf("packing: maxSize must be a positive power of 2 but %d", maxSize))
	}
	return &Page{
		width:   initWidth,
		height:  initHeight,
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
	region image.Rectangle
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

func (n *Node) Region() image.Rectangle {
	return n.region
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

func alloc(n *Node, width, height int) *Node {
	if n.region.Dx() < width || n.region.Dy() < height {
		return nil
	}
	if n.used {
		return nil
	}
	if n.child0 == nil && n.child1 == nil {
		if n.region.Dx() == width && n.region.Dy() == height {
			n.used = true
			return n
		}
		if square(n.region.Dx()-width, n.region.Dy()) >= square(n.region.Dx(), n.region.Dy()-height) {
			// Split vertically
			n.child0 = &Node{
				region: image.Rect(n.region.Min.X, n.region.Min.Y, n.region.Min.X+width, n.region.Max.Y),
				parent: n,
			}
			n.child1 = &Node{
				region: image.Rect(n.region.Min.X+width, n.region.Min.Y, n.region.Max.X, n.region.Max.Y),
				parent: n,
			}
		} else {
			// Split horizontally
			n.child0 = &Node{
				region: image.Rect(n.region.Min.X, n.region.Min.Y, n.region.Max.X, n.region.Min.Y+height),
				parent: n,
			}
			n.child1 = &Node{
				region: image.Rect(n.region.Min.X, n.region.Min.Y+height, n.region.Max.X, n.region.Max.Y),
				parent: n,
			}
		}
		// Note: it now MUST fit, due to above preconditions (repeated here).
		if n.child0.region.Dx() < width || n.child0.region.Dy() < height {
			panic(fmt.Sprintf("packing: the newly created child node (%d, %d) unexpectedly does not contain the requested size (%d, %d)", n.child0.region.Dx(), n.child0.region.Dy(), width, height))
		}
		// Thus, alloc can't return nil, but it may do another split along the other dimension
		// to get a node with the exact size (width, height).
		node := alloc(n.child0, width, height)
		if node == nil {
			panic(fmt.Sprintf("packing: could not allocate the requested size (%d, %d) in the newly created child node (%d, %d)", width, height, n.child0.region.Dx(), n.child0.region.Dy()))
		}
		return node
	}
	if n.child0 == nil || n.child1 == nil {
		panic("packing: both two children must not be nil at alloc")
	}
	if node := alloc(n.child0, width, height); node != nil {
		return node
	}
	if node := alloc(n.child1, width, height); node != nil {
		return node
	}
	return nil
}

func (p *Page) Size() (int, int) {
	return p.width, p.height
}

func (p *Page) Alloc(width, height int) *Node {
	if width <= 0 || height <= 0 {
		panic("packing: width and height must > 0")
	}

	if p.root == nil {
		p.root = &Node{
			region: image.Rect(0, 0, p.width, p.height),
		}
	}
	return p.extendAndAlloc(width, height)
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

func (p *Page) extendAndAlloc(width, height int) *Node {
	if n := alloc(p.root, width, height); n != nil {
		return n
	}

	if p.width >= p.maxSize && p.height >= p.maxSize {
		return nil
	}

	// (1, 0), (0, 1), (2, 0), (1, 1), (0, 2), (3, 0), (2, 1), (1, 2), (0, 3), ...
	for i := 1; ; i++ {
		for j := 0; j <= i; j++ {
			newWidth := p.width
			for k := 0; k < i-j; k++ {
				newWidth *= 2
			}
			newHeight := p.height
			for k := 0; k < j; k++ {
				newHeight *= 2
			}

			if newWidth > p.maxSize || newHeight > p.maxSize {
				if newWidth > p.maxSize && newHeight > p.maxSize {
					panic(fmt.Sprintf("packing: too big extension: allocating size: (%d, %d), current size: (%d, %d), new size: (%d, %d), (i, j): (%d, %d), max size: %d", width, height, p.width, p.height, newWidth, newHeight, i, j, p.maxSize))
				}
				continue
			}

			rollback := p.extend(newWidth, newHeight)
			if n := alloc(p.root, width, height); n != nil {
				return n
			}
			rollback()

			// If the allocation failed even with a maximized page, give up the allocation.
			if newWidth >= p.maxSize && newHeight >= p.maxSize {
				return nil
			}
		}
	}
}

func (p *Page) extend(newWidth int, newHeight int) func() {
	edgeNodes := []*Node{}
	abort := errors.New("abort")
	aborted := false
	if p.root != nil {
		_ = walk(p.root, func(n *Node) error {
			if n.region.Max.X < p.width && n.region.Max.Y < p.height {
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

	var rollback func()

	if aborted {
		origRoot := p.root
		origRootCloned := *p.root

		// Extend the page in the vertical direction.
		if newHeight-p.height > 0 {
			upper := p.root
			lower := &Node{
				region: image.Rect(0, p.height, p.width, newHeight),
			}
			p.root = &Node{
				region: image.Rect(0, 0, p.width, newHeight),
				child0: upper,
				child1: lower,
			}
			upper.parent = p.root
			lower.parent = p.root
		}

		// Extend the page in the horizontal direction.
		if newWidth-p.width > 0 {
			left := p.root
			right := &Node{
				region: image.Rect(p.width, 0, newWidth, newHeight),
			}
			p.root = &Node{
				region: image.Rect(0, 0, newWidth, newHeight),
				child0: left,
				child1: right,
			}
			left.parent = p.root
			right.parent = p.root
		}

		origWidth, origHeight := p.width, p.height
		rollback = func() {
			p.width = origWidth
			p.height = origHeight
			// The node address must not be changed, so use the original root node's pointer (#2584).
			// As the root node might be modified, restore the content by the cloned content.
			p.root = origRoot
			*p.root = origRootCloned
		}
	} else {
		origWidth, origHeight := p.width, p.height
		origMaxXs := map[*Node]int{}
		origMaxYs := map[*Node]int{}

		for _, n := range edgeNodes {
			if n.region.Max.X == p.width {
				origMaxXs[n] = n.region.Max.X
				n.region.Max.X = newWidth
			}
			if n.region.Max.Y == p.height {
				origMaxYs[n] = n.region.Max.Y
				n.region.Max.Y = newHeight
			}
		}

		rollback = func() {
			p.width = origWidth
			p.height = origHeight
			for n, x := range origMaxXs {
				n.region.Max.X = x
			}
			for n, y := range origMaxYs {
				n.region.Max.Y = y
			}
		}
	}

	p.width = newWidth
	p.height = newHeight

	return rollback
}
