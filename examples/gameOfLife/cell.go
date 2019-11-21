package main

import "strings"

var DirectionNames = strings.Split("n ne e se s sw w nw", " ")
var Directions = map[string]Vector{
	"n":  {0, -1},
	"ne": {1, -1},
	"e":  {1, 0},
	"se": {1, 1},
	"s":  {0, 1},
	"sw": {-1, 1},
	"w":  {-1, 0},
	"nw": {-1, -1},
}

type Vector struct {
	x int
	y int
}

type Cell struct {
	Alive bool
}

func NewCell(alive bool) Cell {
	return Cell{alive}
}

func (c *Cell) NextState(neighbours int) {
	if c.Alive && (neighbours < 2 || neighbours > 3) {
		c.Alive = false
	}

	if c.Alive && (neighbours == 2 || neighbours == 3) {
		c.Alive = true
	}

	if !c.Alive && neighbours == 3 {
		c.Alive = true
	}
}
