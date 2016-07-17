// Copyright 2016 The Ebiten Authors
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
	"math"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/examples/common"
)

const (
	screenWidth  = 320
	screenHeight = 240
	maxAngle     = 256
	maxLean      = 16
)

var (
	skyColor  = color.RGBA{0x66, 0xcc, 0xff, 0xff}
	thePlayer = &player{
		x16:   16 * 100,
		y16:   16 * 200,
		angle: maxAngle * 3 / 4,
	}
	gophersImage *ebiten.Image
	groundImage  *ebiten.Image
)

type player struct {
	x16   int
	y16   int
	angle int
	lean  int
}

func round(x float64) float64 {
	return math.Floor(x + 0.5)
}

func (p *player) MoveForward() {
	w, h := gophersImage.Size()
	mx := w * 16
	my := h * 16
	s, c := math.Sincos(float64(p.angle) * 2 * math.Pi / maxAngle)
	p.x16 += int(round(16 * c))
	p.y16 += int(round(16 * s))
	for mx <= p.x16 {
		p.x16 -= mx
	}
	for my <= p.y16 {
		p.y16 -= my
	}
	for p.x16 < 0 {
		p.x16 += mx
	}
	for p.y16 < 0 {
		p.y16 += my
	}
}

func (p *player) RotateRight() {
	p.angle++
	if maxAngle <= p.angle {
		p.angle -= maxAngle
	}
	p.lean++
	if maxLean < p.lean {
		p.lean = maxLean
	}
}

func (p *player) RotateLeft() {
	p.angle--
	if p.angle < 0 {
		p.angle += maxAngle
	}
	p.lean--
	if p.lean < -maxLean {
		p.lean = -maxLean
	}
}

func (p *player) Stabilize() {
	if 0 < p.lean {
		p.lean--
	}
	if p.lean < 0 {
		p.lean++
	}
}

func (p *player) Position() (int, int) {
	return p.x16, p.y16
}

func (p *player) Angle() int {
	return p.angle
}

func updateGroundImage(ground *ebiten.Image) error {
	x16, y16 := thePlayer.Position()
	a := thePlayer.Angle()
	gw, gh := ground.Size()
	w, h := gophersImage.Size()
	for j := -1; j <= 1; j++ {
		for i := -1; i <= 1; i++ {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(-x16)/16, float64(-y16)/16)
			op.GeoM.Translate(float64(w*i), float64(h*j))
			op.GeoM.Rotate(float64(-a)*2*math.Pi/maxAngle + math.Pi*3.0/2.0)
			op.GeoM.Translate(float64(gw)/2, float64(gh))
			if err := ground.DrawImage(gophersImage, op); err != nil {
				return err
			}
		}
	}
	return nil
}

type groundParts struct {
	image *ebiten.Image
}

func (g *groundParts) Len() int {
	_, h := g.image.Size()
	return h
}

func (g *groundParts) Src(i int) (int, int, int, int) {
	w, _ := g.image.Size()
	return 0, i, w, i + 1
}

func (g *groundParts) Dst(i int) (int, int, int, int) {
	w, _ := g.image.Size()
	r := i * 2
	j := 1.0
	for idx := 0; idx < i; idx++ {
		j += float64(idx) / 40
	}
	return -r, int(j), w + r, int(float64(j)+float64(i)/40) + 1
}

func drawGroundImage(screen *ebiten.Image, ground *ebiten.Image) error {
	w, _ := ground.Size()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(w)/2, 0)
	op.GeoM.Rotate(-1 * float64(thePlayer.lean) / maxLean * math.Pi / 8)
	op.GeoM.Translate(float64(w)/2, 0)
	op.GeoM.Translate(float64(screenWidth-w)/2, screenHeight/3)
	op.ImageParts = &groundParts{ground}
	if err := screen.DrawImage(ground, op); err != nil {
		return err
	}
	return nil
}

func update(screen *ebiten.Image) error {
	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		thePlayer.MoveForward()
	}
	rotated := false
	if ebiten.IsKeyPressed(ebiten.KeyRight) {
		thePlayer.RotateRight()
		rotated = true
	}
	if ebiten.IsKeyPressed(ebiten.KeyLeft) {
		thePlayer.RotateLeft()
		rotated = true
	}
	if !rotated {
		thePlayer.Stabilize()
	}

	if err := screen.Fill(skyColor); err != nil {
		return err
	}
	if err := updateGroundImage(groundImage); err != nil {
		return err
	}
	if err := drawGroundImage(screen, groundImage); err != nil {
		return err
	}

	tutrial := "Space: Move foward\nLeft/Right: Rotate"
	msg := fmt.Sprintf("FPS: %0.2f\n%s", ebiten.CurrentFPS(), tutrial)
	if err := ebitenutil.DebugPrint(screen, msg); err != nil {
		return err
	}
	return nil
}

func main() {
	var err error
	gophersImage, _, err = common.AssetImage("gophers.jpg", ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	groundImage, err = ebiten.NewImage(screenWidth+70, screenHeight*2/3+50, ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Air Ship (Ebiten Demo)"); err != nil {
		log.Fatal(err)
	}
}
