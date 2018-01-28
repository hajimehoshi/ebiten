// The MIT License (MIT)
//
// Copyright (c) 2015-2016 Martin Lindhe
// Copyright (c) 2016      Hajime Hoshi
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
// THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

// +build example

package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten"
)

// World represents the game state.
type World struct {
	area [][]bool
}

func newArea(width, height int) [][]bool {
	a := make([][]bool, height)
	for i := 0; i < height; i++ {
		a[i] = make([]bool, width)
	}
	return a
}

// NewWorld creates a new world.
func NewWorld(width, height int, maxInitLiveCells int) *World {
	w := &World{
		area: newArea(width, height),
	}
	w.init(maxInitLiveCells)
	return w
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// init inits world with a random state.
func (w *World) init(maxLiveCells int) {
	height := len(w.area)
	width := len(w.area[0])
	for i := 0; i < maxLiveCells; i++ {
		x := rand.Intn(width)
		y := rand.Intn(height)
		w.area[y][x] = true
	}
}

// Update game state by one tick.
func (w *World) Update() {
	height := len(w.area)
	width := len(w.area[0])
	next := newArea(width, height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pop := neighbourCount(w.area, x, y)
			switch {
			case pop < 2:
				// rule 1. Any live cell with fewer than two live neighbours
				// dies, as if caused by under-population.
				next[y][x] = false

			case (pop == 2 || pop == 3) && w.area[y][x]:
				// rule 2. Any live cell with two or three live neighbours
				// lives on to the next generation.
				next[y][x] = true

			case pop > 3:
				// rule 3. Any live cell with more than three live neighbours
				// dies, as if by over-population.
				next[y][x] = false

			case pop == 3:
				// rule 4. Any dead cell with exactly three live neighbours
				// becomes a live cell, as if by reproduction.
				next[y][x] = true
			}
		}
	}
	w.area = next
}

// Draw paints current game state.
func (w *World) Draw(pix []byte) {
	height := len(w.area)
	width := len(w.area[0])
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := 4*y*width + 4*x
			if w.area[y][x] {
				pix[idx] = 0xff
				pix[idx+1] = 0xff
				pix[idx+2] = 0xff
				pix[idx+3] = 0xff
			} else {
				pix[idx] = 0
				pix[idx+1] = 0
				pix[idx+2] = 0
				pix[idx+3] = 0
			}
		}
	}
}

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// neighbourCount calculates the Moore neighborhood of (x, y).
func neighbourCount(a [][]bool, x, y int) int {
	w := len(a[0])
	h := len(a)
	minI := max(x-1, 0)
	minJ := max(y-1, 0)
	maxI := min(x+1, w-1)
	maxJ := min(y+1, h-1)

	c := 0
	for j := minJ; j <= maxJ; j++ {
		for i := minI; i <= maxI; i++ {
			if i == x && j == y {
				continue
			}
			if a[j][i] {
				c++
			}
		}
	}
	return c
}

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	world  = NewWorld(screenWidth, screenHeight, int((screenWidth*screenHeight)/10))
	pixels = make([]byte, screenWidth*screenHeight*4)
)

func update(screen *ebiten.Image) error {
	world.Update()

	if ebiten.IsRunningSlowly() {
		return nil
	}

	world.Draw(pixels)
	screen.ReplacePixels(pixels)
	return nil
}

func main() {
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Game of Life (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
