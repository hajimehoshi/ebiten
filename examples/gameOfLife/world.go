package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"image/color"
	"math/rand"
	"os"
	"time"
)

type World struct {
	cells  []Cell
	width  int
	height int
}

/*
 x → →

 y
 ↓
 ↓
*/

func NewWorld(width, height int) *World {
	return &World{
		cells:  make([]Cell, width*height),
		width:  width,
		height: height,
	}
}

func (wl *World) GenerateGrid(percent int) {
	percentageAlive := percent * len(wl.cells) / 100

	// Fill Alive Cells as per percentage given
	for i := percentageAlive; i > 0; i-- {
		wl.cells[i] = NewCell(true)
	}

	// Randomize Alive Cells
	cellsClone := wl.cells
	seed := rand.New(rand.NewSource(time.Now().Unix()))
	for i := len(wl.cells); i > 0; i-- {
		randomIndex := seed.Intn(i)
		wl.cells[i-1], cellsClone[randomIndex] = cellsClone[randomIndex], wl.cells[i-1]
	}
	wl.cells = cellsClone
}

func (wl *World) Next() {
	oldWorld := NewWorld(wl.width, wl.height)
	copy(oldWorld.cells, wl.cells)

	for y := 0; y < oldWorld.height; y++ {
		for x := 0; x < oldWorld.width; x++ {
			cell := NewCell(oldWorld.getCell(x, y))
			count := oldWorld.findNeighbours(x, y)
			cell.NextState(count)
			wl.setCell(x, y, cell.Alive)
		}
	}
}

//Print to screen
func (wl World) Print(background *ebiten.Image) {
	for y := 0; y < wl.height; y++ {
		//columns
		for x := 0; x < wl.width; x++ {
			renderCharacter(x, y, wl.getCell(x, y), background)
		}
	}
}

func (wl World) isInside(x, y int) bool {
	return x >= 0 && x < wl.width && y >= 0 && y < wl.height
}

// Look checks a Cell at given direction
// and returns true if a Cell is found
func (wl World) Look(x, y int, dir Vector) bool {
	if !wl.isInside(x, y) {
		return false
	}
	return wl.Plus(x, y, dir)
}

func (wl World) getCell(x, y int) bool {
	return wl.cells[x+(y*wl.width)].Alive
}

func (wl *World) setCell(x, y int, alive bool) {
	if !wl.isInside(x, y) {
		fmt.Printf("Cordinates %v and %v are not in range! \n", x, y)
		os.Exit(1)
	}

	wl.cells[x+(y*wl.width)] = NewCell(alive)
}

// Plus accepts x, y of Cell PLUS direction vector
// returns if Cell exists in the given direction
func (wl World) Plus(x, y int, vec Vector) bool {
	return wl.isInside(x+vec.x, y+vec.y) && wl.getCell(x+vec.x, y+vec.y)
}

// findNeighbours of given co-ordinates
func (wl World) findNeighbours(x, y int) (count int) {
	for _, direction := range DirectionNames {
		if wl.Look(x, y, Directions[direction]) {
			count++
		}
	}
	return
}

func renderCharacter(x, y int, isAlive bool, background *ebiten.Image) {
	if isAlive {
		ebitenutil.DrawRect(
			background,
			float64(x), float64(y),
			32, 32,
			color.RGBA{255, 137, 0, 250},
		)
	}
}
