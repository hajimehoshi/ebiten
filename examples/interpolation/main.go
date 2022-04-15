// Copyright 2022 The Ebiten Authors
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
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"log"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth  = 320
	screenHeight = 240

	maxAngle      = 256
	movementSpeed = 10
)

var (
	ebitenImage *ebiten.Image

	tps = []int{10, 30, 60}
)

type Game struct {
	lastUpdate time.Time

	tpsIndex      int
	interpolation bool

	x, y   float64
	vx, vy float64
	angle  float64

	prevGeom ebiten.GeoM
	currGeom ebiten.GeoM
}

func LerpGeoM(a, b ebiten.GeoM, t float64) ebiten.GeoM {
	for y := 0; y < 3; y++ {
		for x := 0; x < 2; x++ {
			ea := a.Element(x, y)
			eb := b.Element(x, y)
			a.SetElement(x, y, ea+t*(eb-ea))
		}
	}

	return a
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.interpolation = !g.interpolation
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.tpsIndex = (g.tpsIndex + 1) % len(tps)
		ebiten.SetMaxTPS(tps[g.tpsIndex])
	}

	g.x += g.vx
	g.y += g.vy
	if g.x < 0 {
		g.x = -g.x
		g.vx = -g.vx
	} else if mx := float64(screenWidth - ebitenImage.Bounds().Dx()); mx <= g.x {
		g.x = 2*mx - g.x
		g.vx = -g.vx
	}
	if g.y < 0 {
		g.y = -g.y
		g.vy = -g.vy
	} else if my := float64(screenHeight - ebitenImage.Bounds().Dy()); my <= g.y {
		g.y = 2*my - g.y
		g.vy = -g.vy
	}
	g.angle++
	if g.angle == maxAngle {
		g.angle = 0
	}

	w, h := ebitenImage.Size()
	g.prevGeom = g.currGeom
	g.currGeom.Reset()
	g.currGeom.Translate(-float64(w)/2, -float64(h)/2)
	g.currGeom.Rotate(2 * math.Pi * float64(g.angle) / maxAngle)
	g.currGeom.Translate(float64(w)/2, float64(h)/2)
	g.currGeom.Translate(float64(g.x), float64(g.y))

	g.lastUpdate = time.Now()

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	op := ebiten.DrawImageOptions{
		GeoM: g.currGeom,
	}
	if g.interpolation {
		tickDuration := float64(time.Second) / float64(tps[g.tpsIndex])
		t := float64(time.Since(g.lastUpdate)) / tickDuration
		op.GeoM = LerpGeoM(g.prevGeom, g.currGeom, t)
	}

	screen.DrawImage(ebitenImage, &op)

	str := "Press <space> to toggle interpolation\n"
	str += fmt.Sprintf("TPS: %0.2f\nFPS: %0.2f\nInterpolation: %v", ebiten.CurrentTPS(), ebiten.CurrentFPS(), g.interpolation)
	ebitenutil.DebugPrint(screen, str)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	// Decode an image from the image file's byte slice.
	// Now the byte slice is generated with //go:generate for Go 1.15 or older.
	// If you use Go 1.16 or newer, it is strongly recommended to use //go:embed to embed the image file.
	// See https://pkg.go.dev/embed for more details.
	img, _, err := image.Decode(bytes.NewReader(images.Ebiten_png))
	if err != nil {
		log.Fatal(err)
	}
	ebitenImage = ebiten.NewImageFromImage(img)

	ebiten.SetMaxTPS(tps[0])
	ebiten.SetFPSMode(ebiten.FPSModeVsyncOn)
	ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
	ebiten.SetWindowTitle("Interpolation (Ebiten Demo)")

	g := &Game{
		lastUpdate: time.Now(),

		x:     screenWidth / 2,
		y:     screenHeight / 2,
		vx:    movementSpeed,
		vy:    movementSpeed,
		angle: 0,
	}
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
