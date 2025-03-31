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

// This demo is inspired by the xscreensaver 'squirals'.

package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math/rand/v2"

	"github.com/ebitengine/debugui"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	width         = 800
	height        = 600
	scale         = 1
	numOfSquirals = width / 32
)

type palette struct {
	name   string
	colors []color.Color
}

var (
	background = color.Black

	palettes = []palette{
		{
			name: "sand dunes",
			colors: []color.Color{
				color.RGBA{0xF2, 0x74, 0x05, 0xFF}, // #F27405
				color.RGBA{0xD9, 0x52, 0x04, 0xFF}, // #D95204
				color.RGBA{0x40, 0x18, 0x01, 0xFF}, // #401801
				color.RGBA{0xA6, 0x2F, 0x03, 0xFF}, // #A62F03
				color.RGBA{0x73, 0x2A, 0x19, 0xFF}, // #732A19
			},
		},
		{
			name: "mono desert sand",
			colors: []color.Color{
				color.RGBA{0x7F, 0x6C, 0x52, 0xFF}, // #7F6C52
				color.RGBA{0xFF, 0xBA, 0x58, 0xFF}, // #FFBA58
				color.RGBA{0xFF, 0xD9, 0xA5, 0xFF}, // #FFD9A5
				color.RGBA{0x7F, 0x50, 0x0F, 0xFF}, // #7F500F
				color.RGBA{0xCC, 0xAE, 0x84, 0xFF}, // #CCAE84
			},
		},
		{
			name: "land sea gradient",
			colors: []color.Color{
				color.RGBA{0x00, 0xA2, 0xE8, 0xFF}, // #00A2E8
				color.RGBA{0x67, 0xA3, 0xF5, 0xFF}, // #67A3F5
				color.RGBA{0xFF, 0xFF, 0xD5, 0xFF}, // #FFFFD5
				color.RGBA{0xDD, 0xE8, 0x0C, 0xFF}, // #DDE80C
				color.RGBA{0x74, 0x9A, 0x0D, 0xFF}, // #749A0D
			},
		},
	}

	// blocker is an arbitrary color used to prevent the
	// squirals from leaving the canvas.
	blocker = color.RGBA{0, 0, 0, 254}

	// dirCycles defines by offset which direction a squiral
	// should try next for the two cases:
	// clockwise:
	//	1. try to turn right (index+1)
	//  2. try to go straight (index+0)
	//  3. try to turn left (index+3)
	// counter-clockwise:
	//  1. try to turn left (index+3)
	//  2. try to go straight (index+0)
	//  3. try to turn right (index+1)
	dirCycles = [2][3]int{
		{1, 0, 3}, // cw
		{3, 0, 1}, // ccw
	}

	// dirs contains vectors for the directions: east, south, west, north
	// in the specified order.
	dirs = [4]vec2{{1, 0}, {0, 1}, {-1, 0}, {0, -1}}

	// neighbors defines neighboring cells depending on the moving
	// direction of the squiral:
	// index of 0 -> squiral moves vertically,
	// index of 1 -> squiral moves horizontally.
	// These neighbors are tested for "collisions" during simulation.
	neighbors = [2][2]vec2{
		{{0, 1}, {0, -1}}, // east, west
		{{1, 0}, {-1, 0}}, // south, north
	}
)

type vec2 struct {
	x int
	y int
}

type squiral struct {
	speed int
	pos   vec2
	dir   int
	rot   int
	col   color.Color
	dead  bool
}

func (s *squiral) spawn(game *Game) {
	s.dead = false

	rx := rand.IntN(width-4) + 2
	ry := rand.IntN(height-4) + 2

	for dx := -2; dx <= 2; dx++ {
		for dy := -2; dy <= 2; dy++ {
			tx, ty := rx+dx, ry+dy
			if game.auto.colorMap[tx][ty] != background {
				s.dead = true
				return
			}
		}
	}

	s.speed = rand.IntN(5) + 1
	s.pos.x = rx
	s.pos.y = ry
	s.dir = rand.IntN(4)

	game.colorCycle = (game.colorCycle + 1) % len(palettes[game.selectedPalette].colors)
	s.col = palettes[game.selectedPalette].colors[game.colorCycle]

	s.rot = rand.IntN(2)
}

func (s *squiral) step(game *Game) {
	if s.dead {
		return
	}
	x, y := s.pos.x, s.pos.y // shorthands

	change := rand.IntN(1000)
	if change < 2 {
		// On 0.2% of iterations, switch rotation direction.
		s.rot = (s.rot + 1) % 2
	}

	// 1. try to advance the spiral in its rotation
	// direction (clockwise or counter-clockwise).
	for _, next := range dirCycles[s.rot] {
		dir := (s.dir + next) % 4
		off := dirs[dir]
		// Peek all targets by priority.
		target := vec2{
			x: x + off.x,
			y: y + off.y,
		}
		if game.auto.colorMap[target.x][target.y] == background {
			// If the target is free we need to also check the
			// surrounding cells.

			// a. Test if next cell in direction dir does not have
			// the same color as this squiral.
			ntarg := vec2{
				x: target.x + off.x,
				y: target.y + off.y,
			}
			if game.auto.colorMap[ntarg.x][ntarg.y] == s.col {
				// If this has the same color, we cannot go into this direction,
				// to avoid ugly blocks of equal color.
				continue // try next direction
			}

			// b. Test all outer fields for the color of the
			// squiral itself.
			horivert := dir % 2
			xtarg := vec2{}
			set := true
			for _, out := range neighbors[horivert] {
				xtarg.x = target.x + out.x
				xtarg.y = target.y + out.y

				// If one of the outer targets equals the squiral's
				// color, again continue with next direction.
				if game.auto.colorMap[xtarg.x][xtarg.y] == s.col {
					// If this is not free we cannot go into this direction.
					set = false
					break // try next direction
				}

				xtarg.x = ntarg.x + out.x
				xtarg.y = ntarg.y + out.y

				// If one of the outer targets equals the squiral's
				// color, again continue with next direction.
				if game.auto.colorMap[xtarg.x][xtarg.y] == s.col {
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
				game.setpix(s.pos, s.col)
				return
			}
		}
	}

	s.dead = true
}

type automaton struct {
	squirals [numOfSquirals]squiral
	colorMap [width][height]color.Color
}

func (au *automaton) init(game *Game) {
	// Init the test grid with color (0,0,0,0) and the borders of
	// it with color(0,0,0,254) as a blocker color, so the squirals
	// cannot escape the scene.
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			if x == 0 || x == width-1 || y == 0 || y == height-1 {
				au.colorMap[x][y] = blocker
			} else {
				au.colorMap[x][y] = background
			}
		}
	}

	for i := 0; i < numOfSquirals; i++ {
		au.squirals[i].spawn(game)
	}
}

func (a *automaton) step(game *Game) {
	for i := 0; i < numOfSquirals; i++ {
		for s := 0; s < a.squirals[i].speed; s++ {
			a.squirals[i].step(game)
			if a.squirals[i].dead {
				a.squirals[i].spawn(game)
			}
		}
	}
}

type Game struct {
	debugui debugui.DebugUI

	selectedPalette int
	colorCycle      int
	canvas          *ebiten.Image
	auto            automaton
	resetDraw       bool
}

func NewGame() *Game {
	g := &Game{
		canvas: ebiten.NewImage(width, height),
	}
	g.canvas.Fill(background)
	g.auto.init(g)
	return g
}

func (g *Game) setpix(xy vec2, col color.Color) {
	g.canvas.Set(xy.x, xy.y, col)
	g.auto.colorMap[xy.x][xy.y] = col
}

func (g *Game) Update() error {
	reset := false

	if _, err := g.debugui.Update(func(ctx *debugui.Context) error {
		ctx.Window("Squirals", image.Rect(10, 10, 210, 160), func(layout debugui.ContainerLayout) {
			ctx.Text(fmt.Sprintf("TPS: %0.2f", ebiten.ActualTPS()))
			ctx.Text(fmt.Sprintf("FPS: %0.2f", ebiten.ActualFPS()))
			ctx.Button("Respawn").On(func() {
				reset = true
			})
			ctx.Button("Toggle Background").On(func() {
				if background == color.White {
					background = color.Black
				} else {
					background = color.White
				}
				reset = true
			})
			ctx.Button("Cycle Theme").On(func() {
				g.selectedPalette = (g.selectedPalette + 1) % len(palettes)
				reset = true
			})
		})
		return nil
	}); err != nil {
		return err
	}

	if reset {
		g.resetDraw = true
		g.auto.init(g)
	}

	g.auto.step(g)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.resetDraw {
		g.canvas.Fill(background)
		g.resetDraw = false
	}
	screen.DrawImage(g.canvas, nil)
	g.debugui.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return width, height
}

func main() {
	ebiten.SetTPS(250)
	ebiten.SetWindowSize(width*scale, height*scale)
	ebiten.SetWindowTitle("Squirals (Ebitengine Demo)")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
