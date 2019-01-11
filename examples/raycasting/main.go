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
	"errors"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/ebitenutil"

	geo "github.com/paulmach/go.geo"

	"github.com/hajimehoshi/ebiten/inpututil"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/examples/resources/images"
)

const (
	screenWidth  = 240
	screenHeight = 240
)

var (
	bgImage       *ebiten.Image
	shadowImage   *ebiten.Image
	triangleImage *ebiten.Image
)

func init() {
	// Decode image from a byte slice instead of a file so that
	// this example works in any working directory.
	// If you want to use a file, there are some options:
	// 1) Use os.Open and pass the file to the image decoder.
	//    This is a very regular way, but doesn't work on browsers.
	// 2) Use ebitenutil.OpenFile and pass the file to the image decoder.
	//    This works even on browsers.
	// 3) Use ebitenutil.NewImageFromFile to create an ebiten.Image directly from a file.
	//    This also works on browsers.
	img, _, err := image.Decode(bytes.NewReader(images.Tile_png))
	if err != nil {
		log.Fatal(err)
	}
	bgImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)
	shadowImage, _ = ebiten.NewImage(screenWidth, screenHeight, ebiten.FilterDefault)
	triangleImage, _ = ebiten.NewImage(screenWidth, screenHeight, ebiten.FilterDefault)
	triangleImage.Fill(color.White)
}

type Line struct {
	X1, Y1, X2, Y2 float64
}

func rayCasting(cx, cy float64, walls []Line) []Line {
	start := geo.NewPoint(cx, cy)

	var rays []Line

	rayLength := 1000. // something large
	sparser := 2.      // decrease number of rays
	for i := 0.; i < 360/sparser; i++ {
		v := sparser * math.Pi * i / 180
		ray := geo.NewLine(start, geo.NewPoint(rayLength*math.Cos(v), rayLength*math.Sin(v)))

		// Check for intersection
		// Save all collision points
		points := []*geo.Point{}
		for _, wall := range walls {
			p := geo.NewPath()
			p.InsertAt(0, geo.NewPoint(wall.X1, wall.Y1))
			p.InsertAt(1, geo.NewPoint(wall.X2, wall.Y2))
			pts, _ := p.IntersectionLine(ray)
			points = append(points, pts...)
		}

		// Find the point closest to start of ray
		min := math.Inf(1)
		minP := &geo.Point{}
		for i := range points {

			d := points[i].DistanceFrom(start)

			if d < min {
				min = d
				minP = points[i]
			}
		}

		rays = append(rays, Line{cx, cy, minP.X(), minP.Y()})
	}
	return rays
}

func vertices(x1, y1, x2, y2, x3, y3 float64) []ebiten.Vertex {
	return []ebiten.Vertex{
		ebiten.Vertex{float32(x1), float32(y1), 0, 0, 1, 1, 1, 1},
		ebiten.Vertex{float32(x2), float32(y2), 0, 0, 1, 1, 1, 1},
		ebiten.Vertex{float32(x3), float32(y3), 0, 0, 1, 1, 1, 1},
	}
}

func rect(x, y, w, h float64) []Line {
	var lines []Line
	lines = append(lines, Line{x, y, x, y + h})
	lines = append(lines, Line{x, y + h, x + w, y + h})
	lines = append(lines, Line{x + w, y + h, x + w, y})
	lines = append(lines, Line{x + w, y, x, y})
	return lines
}

func handleMovement() {
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
		x += 2
	}

	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyDown) {
		y += 2
	}

	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
		x -= 2
	}

	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp) {
		y -= 2
	}

	// +1/-1 is to stop player before it reaches the border
	if x >= screenHeight-padding {
		x = screenHeight - padding - 1
	}

	if x <= padding {
		x = padding + 1
	}

	if y >= screenWidth-padding {
		y = screenWidth - padding - 1
	}

	if y <= padding {
		y = padding + 1
	}

}

func update(screen *ebiten.Image) error {

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return errors.New("game ended by player")
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		showRays = !showRays
	}

	handleMovement()

	if ebiten.IsDrawingSkipped() {
		return nil
	}

	// x = 50 + 50*math.Cos(float64(time.Now().Nanosecond()/10000000)/100)

	// Reset the shadowImage
	shadowImage.Fill(color.Black)

	rays := rayCasting(x, y, walls)

	// Subtract ray triangles from shadow
	opt := &ebiten.DrawTrianglesOptions{}
	opt.Address = ebiten.AddressRepeat
	opt.CompositeMode = ebiten.CompositeModeSourceOut

	prevLine := rays[len(rays)-1]
	for _, line := range rays {

		// Draw triangle of area between rays
		v := vertices(x, y, prevLine.X2, prevLine.Y2, line.X2, line.Y2)
		shadowImage.DrawTriangles(v, []uint16{0, 1, 2}, triangleImage, opt)
		prevLine = line
	}

	// Draw background
	screen.DrawImage(bgImage, &ebiten.DrawImageOptions{})

	if showRays {
		// Draw rays
		for _, r := range rays {
			ebitenutil.DrawLine(screen, r.X1, r.Y1, r.X2, r.Y2, color.RGBA{255, 255, 0, 150})
		}
	}

	// Draw shadow
	op := &ebiten.DrawImageOptions{}
	op.ColorM.Scale(1, 1, 1, 0.7)
	screen.DrawImage(shadowImage, op)

	// Draw walls
	for _, w := range walls {
		ebitenutil.DrawLine(screen, w.X1, w.Y1, w.X2, w.Y2, color.RGBA{255, 0, 0, 255})
	}

	// Draw player as a rect
	ebitenutil.DrawRect(screen, x-2, y-2, 4, 4, color.Black)
	ebitenutil.DrawRect(screen, x-1, y-1, 2, 2, color.RGBA{255, 100, 100, 255})

	ebitenutil.DebugPrint(screen, "   R: toggle rays          WASD: move")
	return nil
}

var (
	showRays         = true
	x, y     float64 = screenWidth / 2, screenHeight / 2
	walls    []Line
)

const padding = 20

func main() {

	// Add outer walls
	walls = append(walls, rect(padding, padding, screenWidth-2*padding, screenHeight-2*padding)...)

	// Angled wall
	walls = append(walls, Line{50, 80, 100, 150})

	// Rectangle
	walls = append(walls, rect(150, 50, 30, 60)...)

	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Ray casting and shadows (Ebiten demo)"); err != nil {
		log.Fatal(err)
	}
}
