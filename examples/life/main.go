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
	"image"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten"
)

var (
	randSource = rand.NewSource(time.Now().UnixNano())
	rnd        = rand.New(randSource)
)

// World represents the game state
type World struct {
	area [][]bool
}

// NewWorld creates a new world
func NewWorld(width, height int) *World {
	world := World{}
	world.area = makeArea(width, height)
	return &world
}

// RandomSeed inits world with a random state
func (w *World) RandomSeed(limit int) {
	height := len(w.area)
	width := len(w.area[0])

	for i := 0; i < limit; i++ {
		x := rnd.Intn(width)
		y := rnd.Intn(height)
		w.area[y][x] = true
	}
}

// Progress game state by one tick
func (w *World) Progress() {
	height := len(w.area)
	width := len(w.area[0])

	next := makeArea(width, height)

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

// DrawImage paints current game state
func (w *World) DrawImage(img *image.RGBA) {
	height := len(w.area)
	width := len(w.area[0])

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pos := 4*y*width + 4*x
			if w.area[y][x] {
				img.Pix[pos] = 0xff
				img.Pix[pos+1] = 0xff
				img.Pix[pos+2] = 0xff
				img.Pix[pos+3] = 0xff
			} else {
				img.Pix[pos] = 0
				img.Pix[pos+1] = 0
				img.Pix[pos+2] = 0
				img.Pix[pos+3] = 0
			}
		}
	}
}

// neighbourCount calculates the Moore neighborhood of x, y
func neighbourCount(a [][]bool, x, y int) int {
	height := len(a)
	width := len(a[0])

	lowX := 0
	if x > 0 {
		lowX = x - 1
	}

	lowY := 0
	if y > 0 {
		lowY = y - 1
	}

	highX := width - 1
	if x < width-1 {
		highX = x + 1
	}

	highY := height - 1
	if y < height-1 {
		highY = y + 1
	}

	near := 0
	for pY := lowY; pY <= highY; pY++ {
		for pX := lowX; pX <= highX; pX++ {
			if !(pX == x && pY == y) && a[pY][pX] {
				near++
			}
		}
	}

	return near
}

func makeArea(width, height int) [][]bool {
	area := make([][]bool, height)
	for i := 0; i < height; i++ {
		area[i] = make([]bool, width)
	}
	return area
}

const (
	screenWidth  = 320
	screenHeight = 240
)

var (
	world      *World
	noiseImage *image.RGBA
)

func update(screen *ebiten.Image) error {
	world.Progress()
	world.DrawImage(noiseImage)
	if err := screen.ReplacePixels(noiseImage.Pix); err != nil {
		return err
	}
	return nil
}

func main() {
	population := int((screenWidth * screenHeight) / 10)
	scale := 2.0

	world = NewWorld(screenWidth, screenHeight)
	world.RandomSeed(population)

	noiseImage = image.NewRGBA(image.Rect(0, 0, screenWidth, screenHeight))
	if err := ebiten.Run(update, screenWidth, screenHeight, scale, "Game of Life (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
