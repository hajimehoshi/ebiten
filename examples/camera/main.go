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

package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"log"
	"math"

	"golang.org/x/image/math/f64"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

const (
	tileSize   = 16
	tileXCount = 25
)

const (
	worldWidth  = 480
	worldHeight = 320
	worldSizeX  = worldWidth / tileSize
)

var (
	tilesImage *ebiten.Image
)

func init() {
	// Decode an image from the image file's byte slice.
	img, _, err := image.Decode(bytes.NewReader(images.Tiles_png))
	if err != nil {
		log.Fatal(err)
	}
	tilesImage = ebiten.NewImageFromImage(img)
}

type Camera struct {
	ViewPort   f64.Vec2
	Position   f64.Vec2
	ZoomFactor int
	Rotation   int
}

func (c *Camera) String() string {
	return fmt.Sprintf(
		"T: %.1f, R: %d, S: %d",
		c.Position, c.Rotation, c.ZoomFactor,
	)
}

func (c *Camera) viewportCenter() f64.Vec2 {
	return f64.Vec2{
		c.ViewPort[0] * 0.5,
		c.ViewPort[1] * 0.5,
	}
}

func (c *Camera) worldMatrix() ebiten.GeoM {
	m := ebiten.GeoM{}
	m.Translate(-c.Position[0], -c.Position[1])
	// We want to scale and rotate around center of image / screen
	m.Translate(-c.viewportCenter()[0], -c.viewportCenter()[1])
	m.Scale(
		math.Pow(1.01, float64(c.ZoomFactor)),
		math.Pow(1.01, float64(c.ZoomFactor)),
	)
	m.Rotate(float64(c.Rotation) * 2 * math.Pi / 360)
	m.Translate(c.viewportCenter()[0], c.viewportCenter()[1])
	return m
}

func (c *Camera) Render(world, screen *ebiten.Image) {
	screen.DrawImage(world, &ebiten.DrawImageOptions{
		GeoM: c.worldMatrix(),
	})
}

func (c *Camera) ScreenToWorld(posX, posY int) (float64, float64) {
	inverseMatrix := c.worldMatrix()
	if inverseMatrix.IsInvertible() {
		inverseMatrix.Invert()
		return inverseMatrix.Apply(float64(posX), float64(posY))
	} else {
		// When scaling it can happened that matrix is not invertable
		return math.NaN(), math.NaN()
	}
}

func (c *Camera) Reset() {
	c.Position[0] = 0
	c.Position[1] = 0
	c.Rotation = 0
	c.ZoomFactor = 0
}

type Game struct {
	layers [][]int
	world  *ebiten.Image
	camera Camera
}

func (g *Game) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		g.camera.Position[0] -= 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		g.camera.Position[0] += 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyArrowUp) {
		g.camera.Position[1] -= 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyArrowDown) {
		g.camera.Position[1] += 1
	}

	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		if g.camera.ZoomFactor > -2400 {
			g.camera.ZoomFactor -= 1
		}
	}
	if ebiten.IsKeyPressed(ebiten.KeyE) {
		if g.camera.ZoomFactor < 2400 {
			g.camera.ZoomFactor += 1
		}
	}

	if ebiten.IsKeyPressed(ebiten.KeyR) {
		g.camera.Rotation += 1
	}

	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		g.camera.Reset()
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Draw each tile with each DrawImage call.
	// As the source images of all DrawImage calls are always same,
	// this rendering is done very effectively.
	// For more detail, see https://pkg.go.dev/github.com/hajimehoshi/ebiten/v2#Image.DrawImage
	for _, l := range g.layers {
		for i, t := range l {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64((i%worldSizeX)*tileSize), float64((i/worldSizeX)*tileSize))

			sx := (t % tileXCount) * tileSize
			sy := (t / tileXCount) * tileSize
			g.world.DrawImage(tilesImage.SubImage(image.Rect(sx, sy, sx+tileSize, sy+tileSize)).(*ebiten.Image), op)
		}
	}
	g.camera.Render(g.world, screen)

	worldX, worldY := g.camera.ScreenToWorld(ebiten.CursorPosition())
	ebitenutil.DebugPrint(
		screen,
		fmt.Sprintf("TPS: %0.2f\nMove (WASD/Arrows)\nZoom (QE)\nRotate (R)\nReset (Space)", ebiten.ActualTPS()),
	)

	ebitenutil.DebugPrintAt(
		screen,
		fmt.Sprintf("%s\nCursor World Pos: %.2f,%.2f",
			g.camera.String(),
			worldX, worldY),
		0, screenHeight-32,
	)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	g := &Game{
		layers: [][]int{
			{
				243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243,
				243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243,
				243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 218, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 218, 243, 243, 243, 243, 243,
				243, 218, 243, 243, 243, 243, 243, 243, 243, 243, 243, 218, 243, 243, 243, 243, 243, 243, 244, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243,
				243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243,

				243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243,
				243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243,
				243, 243, 244, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 218, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 218, 243, 243,
				243, 243, 243, 243, 243, 243, 243, 243, 243, 219, 243, 243, 243, 219, 243, 243, 243, 243, 243, 243, 243, 218, 243, 243, 243, 243, 243, 243, 243, 243,
				243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243,

				243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243,
				243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243,
				243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243,
				243, 218, 243, 243, 243, 243, 243, 243, 243, 243, 243, 244, 243, 243, 243, 243, 243, 243, 243, 218, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243,
				243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 218, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243,

				243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243,
				243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243,
				243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 218, 243, 243, 243, 243, 243, 243, 243, 243, 243, 218, 243, 243, 243,
				243, 218, 243, 243, 243, 243, 243, 243, 243, 243, 243, 244, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243,
				243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243, 243,
			},
			{
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 26, 27, 28, 29, 30, 31, 0, 0, 0, 0, 0, 0, 0, 0, 0, 63, 64, 65, 66, 67, 68, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 51, 52, 53, 54, 55, 56, 0, 0, 0, 0, 0, 0, 0, 0, 0, 88, 89, 90, 91, 92, 93, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 76, 77, 78, 79, 80, 81, 0, 0, 0, 0, 0, 0, 0, 0, 0, 113, 114, 115, 116, 117, 118, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 101, 102, 103, 104, 105, 106, 0, 0, 0, 0, 0, 0, 0, 0, 0, 138, 139, 140, 141, 142, 143, 0, 0, 0, 0,

				0, 0, 0, 0, 0, 126, 127, 128, 129, 130, 131, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 245, 242, 288, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 303, 303, 245, 242, 303, 303, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 245, 242, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 245, 242, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 245, 242, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 245, 242, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 245, 242, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 245, 242, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 245, 242, 0, 0, 0, 0, 0,

				0, 0, 0, 0, 0, 0, 0, 245, 242, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 245, 242, 0, 45, 46, 47, 48,
				0, 0, 0, 0, 0, 0, 0, 245, 242, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 245, 242, 0, 70, 71, 72, 73,
				0, 0, 0, 0, 0, 0, 0, 245, 242, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 245, 242, 0, 95, 96, 97, 98,
				0, 0, 0, 0, 0, 0, 0, 245, 242, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 245, 242, 0, 120, 121, 122, 123,
				0, 0, 0, 0, 0, 0, 0, 245, 242, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 245, 242, 0, 145, 146, 147, 148,

				0, 0, 0, 0, 0, 0, 0, 245, 242, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 245, 242, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 245, 242, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 245, 242, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 245, 267, 268, 268, 268, 268, 268, 268, 268, 268, 268, 268, 268, 268, 268, 268, 270, 242, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 245, 192, 193, 193, 193, 193, 193, 193, 193, 193, 193, 193, 193, 193, 193, 193, 193, 222, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 245, 242, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			},
		},
		camera: Camera{ViewPort: f64.Vec2{screenWidth, screenHeight}},
	}
	g.world = ebiten.NewImage(worldWidth, worldHeight)

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Camera (Ebitengine Demo)")
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
