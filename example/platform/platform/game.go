// Copyright 2015 Hajime Hoshi
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

package platform

import (
	"github.com/hajimehoshi/ebiten"
)

// NOTE: It's painful to prepare a struct to draw an image part.
type playerRect struct {
	game *Game
}

func (p *playerRect) Len() int {
	return 1
}

func (p *playerRect) Src(i int) (x0, y0, x1, y1 int) {
	return 0, 128, TileSize, 128 + TileSize
}

func (p *playerRect) Dst(i int) (x0, y0, x1, y1 int) {
	x, y := (p.game.player.x)/unit-8, (p.game.player.y)/unit-16
	return x, y, x + TileSize, y + TileSize
}

type player struct {
	x  int
	y  int
	vy int
	ay int
}

type Game struct {
	level        *Level
	player       *player
	jumpKeyState int
}

func NewGame() *Game {
	x, y := (TileSize*3+8)*unit, (TileSize*13)*unit
	return &Game{
		level: &Level{},
		player: &player{
			x:  x,
			y:  y,
			vy: 0,
			ay: 0,
		},
	}
}

// TODO: Rename
const unit = 16

func (g *Game) inAir() bool {
	return g.player.y < 13*unit*TileSize
}

func (g *Game) Update() {
	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		g.jumpKeyState++
	} else {
		g.jumpKeyState = 0
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.player.x -= 1.5 * unit
	} else if ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.player.x += 1.5 * unit
	}
	if g.inAir() {
		g.player.ay = unit
		if g.player.vy < 0 && 0 < g.jumpKeyState {
			g.player.ay -= int(0.5 * unit)
		}
	} else if g.jumpKeyState == 1 {
		g.player.vy = 0
		g.player.ay = -10 * unit
	} else {
		g.player.vy = 0
		g.player.ay = 0
	}
	if 5*unit < g.player.vy {
		g.player.vy = 5 * unit
	}
	g.player.vy += g.player.ay
	g.player.y += g.player.vy
	if 13*unit*TileSize <= g.player.y {
		g.player.y = 13 * unit * TileSize
	}
}

func (g *Game) Draw(screen *ebiten.Image) error {
	if err := g.level.Draw(screen); err != nil {
		return err
	}
	if err := screen.DrawImage(tileSet, &ebiten.DrawImageOptions{
		ImageParts: &playerRect{g},
	}); err != nil {
		return err
	}
	return nil
}
