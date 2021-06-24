// Copyright 2021 The Ebiten Authors
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
//go:build example
// +build example

package main

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"image/color"
	"log"
	"math/rand"
	"time"
)

const (
	screenWidth  = 640
	screenHeight = 480
	scale        = 64
	stars        = 1024
)

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

type Star struct {
	fromx, fromy, tox, toy, brightness float64
}

func (s *Star) Init() {
	s.tox = rand.Float64() * screenWidth * scale
	s.fromx = s.tox
	s.toy = rand.Float64() * screenHeight * scale
	s.fromy = s.toy
	s.brightness = rand.Float64() * 0xff
}

func (s *Star) Out() bool {
	return s.fromx < 0 || screenWidth*scale < s.fromx || s.fromy < 0 || screenHeight*scale < s.fromy
}

func (s *Star) Update(x, y float64) {
	s.fromx = s.tox
	s.fromy = s.toy
	s.tox += (s.tox - x) / 32
	s.toy += (s.toy - y) / 32
	s.brightness += 1
	if 0xff < s.brightness {
		s.brightness = 0xff
	}
	if s.Out() {
		s.Init()
	}
}

func (s *Star) Pos() (float64, float64, float64, float64) {
	return s.fromx / scale, s.fromy / scale, s.tox / scale, s.toy / scale
}

func (s *Star) Colors() (uint8, uint8, uint8) {
	return uint8(0xbb * s.brightness / 0xff), // Red
		uint8(0xdd * s.brightness / 0xff), // Green
		uint8(0xff * s.brightness / 0xff) // Blue
}

type Game struct {
	stars [stars]Star
}

func NewGame() *Game {
	g := &Game{}
	for i := 0; i < stars; i++ {
		g.stars[i].Init()
	}
	return g
}

func (g *Game) Update() error {
	x, y := ebiten.CursorPosition()
	for i := 0; i < stars; i++ {
		g.stars[i].Update(float64(x*scale), float64(y*scale))
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	for i := 0; i < stars; i++ {
		s := &g.stars[i]
		fx, fy, tx, ty := s.Pos()
		r, g, b := s.Colors()
		ebitenutil.DrawLine(screen, fx, fy, tx, ty, color.RGBA{r, g, b, 0xff})
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	rand.Seed(time.Now().UnixNano())
	ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
	ebiten.SetWindowTitle("Stars (Ebiten Demo)")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
