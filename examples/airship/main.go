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
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
)

const (
	screenWidth  = 640
	screenHeight = 480
	maxAngle     = 256
	maxLean      = 16
)

var (
	skyColor = color.RGBA{0x66, 0xcc, 0xff, 0xff}

	gophersImage         *ebiten.Image
	repeatedGophersImage *ebiten.Image
)

func init() {
	// Decode an image from the image file's byte slice.
	img, _, err := image.Decode(bytes.NewReader(images.Gophers_jpg))
	if err != nil {
		log.Fatal(err)
	}
	gophersImage = ebiten.NewImageFromImage(img)

	const (
		xrepeat = 7
		yrepeat = 8
	)
	w, h := gophersImage.Bounds().Dx(), gophersImage.Bounds().Dy()
	repeatedGophersImage = ebiten.NewImage(w*xrepeat, h*yrepeat)
	for j := range yrepeat {
		for i := range xrepeat {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(w*i), float64(h*j))
			repeatedGophersImage.DrawImage(gophersImage, op)
		}
	}
}

// player represents the current airship's position.
type player struct {
	// x16, y16 represents the position in XY plane in fixed float format.
	// The fractional part has 16 bits of precision.
	x16 int
	y16 int

	// angle represents the player's angle in XY plane.
	// angle takes an integer value in [0, maxAngle).
	angle int

	// lean represents the player's leaning.
	// lean takes an integer value in [-maxLean, maxLean].
	lean int
}

func round(x float64) float64 {
	return math.Floor(x + 0.5)
}

// MoveForward moves the player p forward.
func (p *player) MoveForward() {
	w, h := gophersImage.Bounds().Dx(), gophersImage.Bounds().Dy()
	mx := w * 16
	my := h * 16
	s, c := math.Sincos(float64(p.angle) * 2 * math.Pi / maxAngle)
	p.x16 += int(round(16*c) * 2)
	p.y16 += int(round(16*s) * 2)
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

// RotateRight rotates the player p in the right direction.
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

// RotateLeft rotates the player p in the left direction.
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

// Stabilize tries to move the player in the stable position (lean).
func (p *player) Stabilize() {
	if 0 < p.lean {
		p.lean--
	}
	if p.lean < 0 {
		p.lean++
	}
}

// Position returns the player p's position.
func (p *player) Position() (int, int) {
	return p.x16, p.y16
}

// Angle returns the player p's angle.
func (p *player) Angle() int {
	return p.angle
}

// updateGroundImage updates the ground image according to the current player's position.
func (g *Game) updateGroundImage(ground *ebiten.Image) {
	ground.Clear()

	x16, y16 := g.player.Position()
	a := g.player.Angle()
	rw, rh := repeatedGophersImage.Bounds().Dx(), repeatedGophersImage.Bounds().Dy()
	gw, gh := ground.Bounds().Dx(), ground.Bounds().Dy()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(-x16)/16, float64(-y16)/16)
	op.GeoM.Translate(float64(-rw)/2, float64(-rh)/2)
	op.GeoM.Rotate(float64(-a)*2*math.Pi/maxAngle + math.Pi*3.0/2.0)
	op.GeoM.Translate(float64(gw)/2, float64(gh)/2)
	ground.DrawImage(repeatedGophersImage, op)
}

// drawGroundImage draws the ground image to the given screen image.
func (g *Game) drawGroundImage(screen *ebiten.Image, ground *ebiten.Image) {
	g.perspectiveGroundImage.Clear()
	gw := ground.Bounds().Dx()
	pw, ph := g.perspectiveGroundImage.Bounds().Dx(), g.perspectiveGroundImage.Bounds().Dy()
	for j := 0; j < ph; j++ {
		// z is in [2, -1]
		rate := float64(j) / float64(ph)
		z := (1-rate)*2 + rate*-1
		// Avoid too small z, or the scale (1/z) can be too big.
		if z <= 0.1 {
			break
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-float64(pw)/2, 0)
		op.GeoM.Scale(1/z, 8) // 8 is an arbitrary number not to make empty lines.
		op.GeoM.Translate(float64(pw)/2, float64(j)/z)

		g.perspectiveGroundImage.DrawImage(ground.SubImage(image.Rect(0, j, gw, j+1)).(*ebiten.Image), op)
	}

	g.perspectiveGroundImage.DrawImage(g.fogImage, nil)

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(pw)/2, 0)
	op.GeoM.Rotate(-1 * float64(g.player.lean) / maxLean * math.Pi / 8)
	op.GeoM.Translate(float64(screenWidth)/2, screenHeight/3)
	screen.DrawImage(g.perspectiveGroundImage, op)
}

type Game struct {
	player *player

	groundImage            *ebiten.Image
	perspectiveGroundImage *ebiten.Image
	fogImage               *ebiten.Image
}

func NewGame() *Game {
	g := &Game{
		player: &player{
			x16:   16 * 100,
			y16:   16 * 200,
			angle: maxAngle * 3 / 4,
		},
		groundImage:            ebiten.NewImage(screenWidth*3, screenHeight*2/3+200),
		perspectiveGroundImage: ebiten.NewImage(screenWidth*3, screenHeight),
	}

	const fogHeight = 16
	w := g.perspectiveGroundImage.Bounds().Dx()
	fogRGBA := image.NewRGBA(image.Rect(0, 0, w, fogHeight))
	for j := 0; j < fogHeight; j++ {
		a := uint32(float64(fogHeight-1-j) * 0xff / (fogHeight - 1))
		clr := skyColor
		r, g, b, oa := uint32(clr.R), uint32(clr.G), uint32(clr.B), uint32(clr.A)
		clr.R = uint8(r * a / oa)
		clr.G = uint8(g * a / oa)
		clr.B = uint8(b * a / oa)
		clr.A = uint8(a)
		for i := 0; i < w; i++ {
			fogRGBA.SetRGBA(i, j, clr)
		}
	}
	g.fogImage = ebiten.NewImageFromImage(fogRGBA)

	return g
}

func (g *Game) Update() error {
	// Manipulate the player by the input.
	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		g.player.MoveForward()
	}
	rotated := false
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		g.player.RotateRight()
		rotated = true
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		g.player.RotateLeft()
		rotated = true
	}
	if !rotated {
		g.player.Stabilize()
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Draw the ground image.
	screen.Fill(skyColor)
	g.updateGroundImage(g.groundImage)
	g.drawGroundImage(screen, g.groundImage)

	// Draw the message.
	tutorial := "Space: Move forward\nLeft/Right: Rotate"
	msg := fmt.Sprintf("TPS: %0.2f\nFPS: %0.2f\n%s", ebiten.ActualTPS(), ebiten.ActualFPS(), tutorial)
	ebitenutil.DebugPrint(screen, msg)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Air Ship (Ebitengine Demo)")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}
