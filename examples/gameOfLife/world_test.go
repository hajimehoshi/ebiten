package main

import (
	"testing"
)

func TestNewWorld(t *testing.T) {
	wl := NewWorld(20, 10)
	if wl.width != 20 {
		t.Error(t, "Width did not set correctly")
	}
	if wl.height != 10 {
		t.Error(t, "Height did not set correctly")
	}

	if len(wl.cells) != 200 {
		t.Error(t, "Cell did not set correctly")
	}
}

func TestWorld_isInside(t *testing.T) {
	world := NewWorld(10, 10)
	isInBounds := world.isInside(5, 9)
	if !isInBounds {
		t.Error(t, "Given vector should be inside world boundaries")
	}
}

func TestWorld_isInsideNot(t *testing.T) {
	world := NewWorld(10, 10)
	isInBounds := world.isInside(10, 10)
	if isInBounds {
		t.Error(t, "Given vector should not be inside world boundaries")
	}
}

func TestWorld_LookNotFound(t *testing.T) {
	world := NewWorld(10, 10)

	if found := world.Look(0, 0, Directions["n"]); found {
		t.Error(t, "No Cell should be found, since direction North out of range")
	}

	if found := world.Look(1, 1, Directions["n"]); found {
		t.Error(t, "No Cell should be found, since direction North out of range")
	}

	if found := world.Look(1, 1, Directions["ne"]); found {
		t.Error(t, "No Cell should be found, since direction North East is empty.")
	}

	if found := world.Look(1, 1, Directions["e"]); found {
		t.Error(t, "No Cell should be found, since direction East is empty.")
	}

	if found := world.Look(1, 1, Directions["se"]); found {
		t.Error(t, "Cell should be found, at x 1 y 1, we set before")
	}
	if found := world.Look(1, 1, Directions["s"]); found {
		t.Error(t, "Cell should be found")
	}
	if found := world.Look(1, 1, Directions["w"]); found {
		t.Error(t, "No Cell should be found, since direction West is empty.")
	}
	if found := world.Look(1, 1, Directions["sw"]); found {
		t.Error(t, "No Cell should be found, since direction South West is empty.")
	}
}

func TestWorld_Plus(t *testing.T) {
	var wl *World = NewWorld(5, 5)
	wl.setCell(1, 1, true)
	wl.setCell(2, 1, true) //eastCell

	if alive := wl.Plus(1, 1, Directions["e"]); !alive {
		t.Error(t, "Cell on East should be found Alive")
	}

	if alive := wl.Plus(1, 1, Directions["s"]); alive {
		t.Error(t, "Cell on South direction should be found Dead")
	}
}

func TestWorld_setCell(t *testing.T) {
	var wl *World = NewWorld(20, 10)
	wl.setCell(1, 1, true)

	if cell := wl.getCell(1, 1); !cell {
		t.Error("Cell should be Alive")
	}
}

func TestWorld_findNeighbours(t *testing.T) {
	var wl *World = NewWorld(5, 5)
	/*
		X	_ 0 1 2 3
		Y	0       D
			1   A C
			2   B   E
	*/
	wl.setCell(1, 1, true) //A
	wl.setCell(1, 2, true) //B
	wl.setCell(2, 1, true) //C
	wl.setCell(3, 0, true) //D
	wl.setCell(3, 2, true) //E

	if ANeighbours := wl.findNeighbours(1, 1); ANeighbours != 2 {
		t.Error("Neighbours count should be 2 for given vector")
	}

	if CNeighbours := wl.findNeighbours(2, 1); CNeighbours != 4 {
		t.Error("Neighbours count should be 4 for given vector")
	}
}

func TestWorld_GenerateGrid(t *testing.T) {
	var wl *World = NewWorld(5, 5)
	wl.GenerateGrid(40)

	if total := len(wl.cells); total != 25 {
		t.Error("Total cells count should be 25")
	}

	aliveCount := 0
	for _, cell := range wl.cells {
		if cell.Alive {
			aliveCount++
		}
	}
	if aliveCount != 10 {
		t.Error("Alive cells count should be 40 percent")
	}
}

func TestWorld_Next(t *testing.T) {
	var wl *World = NewWorld(5, 5)
	wl.GenerateGrid(40)

	if total := len(wl.cells); total != 25 {
		t.Error("Total cells count should be 25")
	}
}
