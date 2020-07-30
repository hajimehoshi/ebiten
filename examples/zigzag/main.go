// Copyright 2020 The Ebiten Authors
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

package main

import (
	"fmt"
	"image/color"
	"log"
	"math/rand"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/inpututil"
)

const (
	screenWidth  = 640
	screenHeight = 480
	Bit32Max     = 0xFFFFFFFF
)

var (
	emptyImage *ebiten.Image
)

// Scale ...
type Scale struct {
	X int64
	Y int64
}

// Game ...
type Game struct {
	moveDirection int8
	AppleX        int64
	AppleY        int64
	scalesLoc     map[int64]*Scale
	timer         uint64
	speed         int8
	Score         int64
	bestScore     int64
	Level         int8
	nearapple     bool
}

func init() {
	emptyImage, _ = ebiten.NewImage(1, 1, ebiten.FilterDefault)
	_ = emptyImage.Fill(color.White)
}

func (g *Game) collides() bool {
	return g.scalesLoc[0].X < g.AppleX+10 &&
		g.scalesLoc[0].X+10 > g.AppleX &&
		g.scalesLoc[0].Y < g.AppleY+10 &&
		g.scalesLoc[0].Y+10 > g.AppleY
}

func (g *Game) nearApple() bool {
	return g.scalesLoc[0].X < (g.AppleX-30)+60 &&
		g.scalesLoc[0].X+10 > g.AppleX-30 &&
		g.scalesLoc[0].Y < (g.AppleY-30)+50 &&
		g.scalesLoc[0].Y+10 > g.AppleY-30
}

func (g *Game) collidesWithSelf() bool {
	for i, v := range g.scalesLoc {
		if i > 0 {
			if g.scalesLoc[0].X < v.X+10 &&
				g.scalesLoc[0].X+10 > v.X &&
				g.scalesLoc[0].Y < v.Y+10 &&
				g.scalesLoc[0].Y+10 > v.Y {
				return true
			}
		}
	}
	return false
}

func (g *Game) wallCollision() bool {
	return g.scalesLoc[0].X > screenWidth/2 ||
		g.scalesLoc[0].X < -(screenWidth/2) ||
		g.scalesLoc[0].Y > screenHeight/2 ||
		g.scalesLoc[0].Y < -(screenHeight/2)
}

func (g *Game) updateTimer() {
	if g.timer > Bit32Max {
		g.timer = 0
	}
	g.timer++
}

func (g *Game) controlSpeed() bool {
	if g.timer%uint64(g.speed) == 0 {
		return true
	}
	return false
}

func (g *Game) reset() {
	g.AppleX = 30
	g.AppleY = 30
	g.speed = 5 // means 83.33 ms
	l := len(g.scalesLoc)
	for i := int64(1); i < int64(l); i++ {
		delete(g.scalesLoc, i)
	}
	g.scalesLoc[0].X = 0
	g.scalesLoc[0].Y = 0
	g.Score = 0
	g.Level = 1
	g.moveDirection = 0
}

// Update ...
func (g *Game) Update(screen *ebiten.Image) error {

	if inpututil.IsKeyJustPressed(ebiten.KeyLeft) || inpututil.IsKeyJustPressed(ebiten.KeyA) {
		if g.moveDirection != 2 {
			g.moveDirection = 1
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyRight) || inpututil.IsKeyJustPressed(ebiten.KeyD) {
		if g.moveDirection != 1 {
			g.moveDirection = 2
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		if g.moveDirection != 4 {
			g.moveDirection = 3
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		if g.moveDirection != 3 {
			g.moveDirection = 4
		}
	} else if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.reset()
	}

	if g.controlSpeed() {
		if g.collidesWithSelf() || g.wallCollision() {
			g.reset()
		}
		g.nearapple = g.nearApple()

		if g.collides() {
			g.AppleX = rand.Int63n(screenWidth-20) + -(screenWidth / 2)
			g.AppleY = rand.Int63n(screenHeight-20) + -(screenHeight / 2)
			g.scalesLoc[int64(len(g.scalesLoc))] = &Scale{
				X: g.scalesLoc[int64(len(g.scalesLoc)-1)].X,
				Y: g.scalesLoc[int64(len(g.scalesLoc)-1)].Y,
			}
			if len(g.scalesLoc) > 7 && len(g.scalesLoc) < 15 {
				g.Level = 2
				g.speed = 4
			} else if len(g.scalesLoc) > 14 && len(g.scalesLoc) < 21 {
				g.Level = 3
				g.speed = 3
			} else if len(g.scalesLoc) > 20 && len(g.scalesLoc) < 30 {
				g.Level = 4
				g.speed = 2
			} else if len(g.scalesLoc) > 29 {
				g.Level = 5
				g.speed = 1
			} else {
				g.Level = 1
			}
			g.Score++
			if g.bestScore < g.Score {
				g.bestScore = g.Score
			}
		}

		for i := int64(len(g.scalesLoc)) - 1; i > 0; i-- {
			g.scalesLoc[i].X = g.scalesLoc[i-1].X
			g.scalesLoc[i].Y = g.scalesLoc[i-1].Y
		}
		switch g.moveDirection {
		case 1:
			g.scalesLoc[0].X = g.scalesLoc[0].X - 10
		case 2:
			g.scalesLoc[0].X += 10
		case 3:
			g.scalesLoc[0].Y += 10
		case 4:
			g.scalesLoc[0].Y -= 10
		}
	}

	g.updateTimer()

	return nil
}

// Draw ...
func (g *Game) Draw(screen *ebiten.Image) {

	if g.moveDirection == 0 {
		ebitenutil.DebugPrint(screen, fmt.Sprintf("Press up/down/left/right to start, M to Enable/Disable Sound"))
	} else {
		if g.nearapple {
			ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %0.2f Level: %d Score: %d Best Score: %d Near", ebiten.CurrentFPS(), g.Level, g.Score, g.bestScore))
		} else {
			ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %0.2f Level: %d Score: %d Best Score: %d", ebiten.CurrentFPS(), g.Level, g.Score, g.bestScore))
		}
	}

	for _, v := range g.scalesLoc {
		ebitenutil.DrawRect(screen, (screenWidth/2)+float64(v.X), (screenHeight/2)+float64(v.Y), 10, 10, color.RGBA{0x80, 0xa0, 0xc0, 0xff})
	}
	ebitenutil.DrawRect(screen, (screenWidth/2)+float64(g.AppleX), (screenHeight/2)+float64(g.AppleY), 10, 10, color.RGBA{0xFF, 0x00, 0x00, 0xff})
}

// Layout ...
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func newGame() *Game {
	g := &Game{AppleX: 30, AppleY: 30}
	g.speed = 5 // means 100 ms
	g.scalesLoc = make(map[int64]*Scale)
	g.scalesLoc[0] = &Scale{X: 0, Y: 0}
	return g
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("ZigZag")
	if err := ebiten.RunGame(newGame()); err != nil {
		log.Fatal(err)
	}
}
