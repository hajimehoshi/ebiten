// Copyright 2018 The Ebiten Authors
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

package main

import (
	"fmt"
	"image/color"
	"log"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
)

var (
	mplusNormalFont font.Face
)

func init() {
	tt, err := opentype.Parse(fonts.MPlus1pRegular_ttf)
	if err != nil {
		log.Fatal(err)
	}

	const dpi = 72
	mplusNormalFont, err = opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    24,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}
}

const (
	screenWidth  = 640
	screenHeight = 480
)

type Game struct {
	monitors []string
}

func (g *Game) Update() error {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0) {
		cx, cy := ebiten.CursorPosition()
		l := text.BoundString(mplusNormalFont, "|").Dy()
		y := l
		for _, m := range g.monitors {
			b := text.BoundString(mplusNormalFont, m)
			if cx >= b.Min.X && cx <= b.Max.X && cy >= b.Min.Y+y && cy <= b.Max.Y+y {
				ebiten.SetWindowTitle(m)
				ebiten.SetWindowMonitor(m)
				break
			}
			y += l
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	const x = 0
	l := text.BoundString(mplusNormalFont, "|").Dy()
	y := l
	for _, m := range g.monitors {
		text.Draw(screen, m, mplusNormalFont, x, y, color.White)
		y += l
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	g := &Game{}

	fmt.Println("Monitor", ebiten.Monitor())

	targetMonitor := ""
	for _, m := range ebiten.Monitors() {
		g.monitors = append(g.monitors, m.Name())
		x, y := m.Position()
		w, h := m.Size()
		targetMonitor = m.Name()
		fmt.Println("Monitor", m.Index(), m.Name(), x, y, w, h, m.RefreshRate())
	}

	ebiten.SetWindowMonitor(targetMonitor)
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle(targetMonitor)
	if err := ebiten.RunGameWithOptions(g, &ebiten.RunGameOptions{
		//Monitor: targetMonitor,
	}); err != nil {
		log.Fatal(err)
	}
}
