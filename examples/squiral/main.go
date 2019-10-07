// Copyright 2019 The Ebiten Authors
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

// +build example jsgo

// This demo is inspired by the xscreensaver 'squirals'.

package main

import (
	"fmt"
	"image/color"
	"math/rand"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"golang.org/x/image/colornames"
)

type direction int

const (
	width  = 800
	height = 600
	scale  = 2
	amount = width / 32
)

const (
	east direction = iota
	south
	west
	north

	horizontal direction = iota
	vertical
)

type palette struct {
	name   string
	colors []color.Color
}

var (
	quit = fmt.Errorf("quit")

	background = colornames.Black

	palettes = []palette{
		palette{
			name: "sand dunes",
			colors: []color.Color{
				color.RGBA{0xF2, 0x74, 0x05, 0xFF}, //#F27405
				color.RGBA{0xD9, 0x52, 0x04, 0xFF}, //#D95204
				color.RGBA{0x40, 0x18, 0x01, 0xFF}, //#401801
				color.RGBA{0xA6, 0x2F, 0x03, 0xFF}, //#A62F03
				color.RGBA{0x73, 0x2A, 0x19, 0xFF}, //#732A19
			},
		},
		palette{
			name: "mono desert sand",
			colors: []color.Color{
				color.RGBA{0x7F, 0x6C, 0x52, 0xFF}, //#7F6C52
				color.RGBA{0xFF, 0xBA, 0x58, 0xFF}, //#FFBA58
				color.RGBA{0xFF, 0xD9, 0xA5, 0xFF}, //#FFD9A5
				color.RGBA{0x7F, 0x50, 0x0F, 0xFF}, //#7F500F
				color.RGBA{0xCC, 0xAE, 0x84, 0xFF}, //#CCAE84
			},
		},
		palette{
			name: "land sea gradient",
			colors: []color.Color{
				color.RGBA{0x00, 0xA2, 0xE8, 0xFF}, //#00A2E8
				color.RGBA{0x67, 0xA3, 0xF5, 0xFF}, //#67A3F5
				color.RGBA{0xFF, 0xFF, 0xD5, 0xFF}, //#FFFFD5
				color.RGBA{0xDD, 0xE8, 0x0C, 0xFF}, //#DDE80C
				color.RGBA{0x74, 0x9A, 0x0D, 0xFF}, //#749A0D
			},
		},
	}
	selectedPalette = 0
	colorCycle      = 0
	canvas          *ebiten.Image
	auto            *automaton
	once            sync.Once
	blocker         = color.RGBA{0, 0, 0, 254}
	free            = color.RGBA{0, 0, 0, 0}

	rotors = [2][3]direction{
		[3]direction{1, 0, 3},
		[3]direction{3, 0, 1},
	}

	// east, south, west, north
	dirs   = [4]tup{tup{1, 0}, tup{0, 1}, tup{-1, 0}, tup{0, -1}}
	outers = [2][2]tup{
		[2]tup{tup{0, 1}, tup{0, -1}}, // east, west
		[2]tup{tup{1, 0}, tup{-1, 0}}, // south, north
	}
)

type tup struct {
	x int
	y int
}

type squiral struct {
	speed int
	pos   tup
	dir   direction
	rot   int
	col   color.Color
	dead  bool
}

func (s *squiral) spawn() {
	s.dead = false

	rx := rand.Intn(width-4) + 2
	ry := rand.Intn(height-4) + 2

	for dx := -2; dx <= 2; dx++ {
		for dy := -2; dy <= 2; dy++ {
			tx, ty := rx+dx, ry+dy
			if auto.colorMap[tx][ty] != free {
				s.dead = true
				return
			}
		}
	}

	s.speed = rand.Intn(5) + 1
	s.pos.x = rx
	s.pos.y = ry
	s.dir = direction(rand.Intn(5))

	colorCycle = (colorCycle + 1) % len(palettes[selectedPalette].colors)
	s.col = palettes[selectedPalette].colors[colorCycle]

	s.rot = rand.Intn(2)
}

func (s *squiral) step(debug int) {
	if s.dead {
		return
	}
	x, y := s.pos.x, s.pos.y // shorthands

	change := rand.Intn(1000)
	if change < 2 {
		// On 0.2% of iterations, switch rotation direction.
		s.rot = (s.rot + 1) % 2
	}

	// 1. try to advance the spiral in its rotation
	// direction (clockwise or counter-clockwise).
	for _, next := range rotors[s.rot] {
		dir := (s.dir + next) % 4
		off := dirs[dir]
		// Peek all targets by priority.
		target := tup{
			x: x + off.x,
			y: y + off.y,
		}
		if auto.colorMap[target.x][target.y] == free {
			// If the target is free we need to also check the
			// surrounding cells.

			// a. Test if next cell in direction dir does not have
			// the same color as this squiral.
			ntarg := tup{
				x: target.x + off.x,
				y: target.y + off.y,
			}
			if auto.colorMap[ntarg.x][ntarg.y] == s.col {
				// If this has the same color, we cannot go into this direction,
				// to avoid ugly blocks of equal color.
				continue // try next direction
			}

			// b. Test all outer fields for the color of the
			// squiral itself.
			horivert := dir % 2
			xtarg := tup{}
			set := true
			for _, out := range outers[horivert] {
				xtarg.x = target.x + out.x
				xtarg.y = target.y + out.y

				// If one of the outer targets equals the squiral's
				// color, again continue with next direction.
				if auto.colorMap[xtarg.x][xtarg.y] == s.col {
					// If this is not free we cannot go into this direction.
					set = false
					break // try next direction
				}

				xtarg.x = ntarg.x + out.x
				xtarg.y = ntarg.y + out.y

				// If one of the outer targets equals the squiral's
				// color, again continue with next direction.
				if auto.colorMap[xtarg.x][xtarg.y] == s.col {
					// If this is not free we cannot go into this direction.
					set = false
					break // try next direction
				}
			}

			if set {
				s.pos = target
				s.dir = dir
				// 2. set the color of this squiral to its
				// current position.
				setpix(s.pos, s.col)
				return
			}
		}
	}

	s.dead = true
}

type automaton struct {
	squirals [amount]squiral
	colorMap [width][height]color.Color
}

func (au *automaton) init() {
	// Init the test grid with color (0,0,0,0) and the borders of
	// it with color(0,0,0,254) as a blocker color, so the squirals
	// cannot escape the scene.
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			if x == 0 || x == width-1 || y == 0 || y == height-1 {
				auto.colorMap[x][y] = blocker
			} else {
				auto.colorMap[x][y] = free
			}
		}
	}

	for i := 0; i < amount; i++ {
		auto.squirals[i].spawn()
	}
}

func (au *automaton) step() {
	for i := 0; i < amount; i++ {
		for s := 0; s < au.squirals[i].speed; s++ {
			au.squirals[i].step(i)
			if au.squirals[i].dead {
				au.squirals[i].spawn()
			}
		}
	}
}

func setpix(xy tup, col color.Color) {
	canvas.Set(xy.x, xy.y, col)
	auto.colorMap[xy.x][xy.y] = col
}

func init() {
	rand.Seed(time.Now().UnixNano())
	c, err := ebiten.NewImage(width, height, ebiten.FilterNearest)
	if err != nil {
		panic(err)
	}
	canvas = c
	canvas.Fill(colornames.Black)

	auto = &automaton{}
	auto.init()
}

func update(screen *ebiten.Image) error {
	reset := false

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return quit
	}

	if ebiten.IsKeyPressed(ebiten.KeyB) {
		if background == colornames.White {
			background = colornames.Black
		} else {
			background = colornames.White
		}
		reset = true
	} else if ebiten.IsKeyPressed(ebiten.KeyT) {
		selectedPalette = (selectedPalette + 1) % len(palettes)
		reset = true
	}

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		reset = true
	}

	if reset {
		screen.Fill(background)
		canvas.Fill(background)
		auto.init()
	}

	auto.step()

	if ebiten.IsDrawingSkipped() {
		return nil
	}

	draw(screen)

	return nil
}

func draw(screen *ebiten.Image) {
	screen.DrawImage(canvas, nil)
	ebitenutil.DebugPrintAt(
		screen,
		fmt.Sprintf("TPS: %0.2f, FPS: %0.2f", ebiten.CurrentTPS(), ebiten.CurrentFPS()),
		1, 1,
	)
	ebitenutil.DebugPrintAt(
		screen,
		"left mouse click: respawn",
		1, 16,
	)
	ebitenutil.DebugPrintAt(
		screen,
		"b: toggle background (white/black)",
		1, 32,
	)
	ebitenutil.DebugPrintAt(
		screen,
		fmt.Sprintf("t: cycle theme (current: %s)", palettes[selectedPalette].name),
		1, 48,
	)
	ebitenutil.DebugPrintAt(
		screen,
		"esc: quit",
		1, 64,
	)
}

func main() {
	ebiten.SetMaxTPS(250)
	if err := ebiten.Run(update, width, height, scale, "Squirals"); err != nil {
		if err == quit {
			return
		}
		panic(err)
	}
}
