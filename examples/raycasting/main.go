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

// +build example jsgo

package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/ebitenutil"

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

func directedRay(x, y, length, v float64) Line {

	return Line{
		X1: x,
		Y1: y,
		X2: x + length*math.Cos(v),
		Y2: y + length*math.Sin(v),
	}
}

// Algebra from wikipedia
// https://en.wikipedia.org/wiki/Line%E2%80%93line_intersection#Given_two_points_on_each_line
func intersection(l1, l2 Line) (float64, float64, error) {
	denom := (l1.X1-l1.X2)*(l2.Y1-l2.Y2) - (l1.Y1-l1.Y2)*(l2.X1-l2.X2)
	tNum := (l1.X1-l2.X1)*(l2.Y1-l2.Y2) - (l1.Y1-l2.Y1)*(l2.X1-l2.X2)
	uNum := -((l1.X1-l1.X2)*(l1.Y1-l2.Y1) - (l1.Y1-l1.Y2)*(l1.X1-l2.X1))

	if denom == 0 {
		return 0, 0, errors.New("lines parallel or coincident")
	}

	t := tNum / denom
	if t > 1. || t < 0 {
		return 0, 0, errors.New("lines intersect, segments do not")
	}

	u := uNum / denom
	if u > 1. || u < 0 {
		return 0, 0, errors.New("lines intersect, segments do not")
	}

	x := l1.X1 + t*(l1.X2-l1.X1)
	y := l1.Y1 + t*(l1.Y2-l1.Y1)
	return x, y, nil
}

func rayCasting(cx, cy float64, walls []Line, numRays float64) []Line {
	var rays []Line

	rayLength := 1000. // something large
	for i := 0.; i < numRays; i++ {
		v := (i / numRays) * 2 * math.Pi

		// Create a new line to serve as ray
		ray := directedRay(cx, cy, rayLength, v)

		// Check for intersection and save collision points
		points := [][2]float64{}
		for _, wall := range walls {
			if px, py, err := intersection(ray, wall); err == nil {
				points = append(points, [2]float64{px, py})
			}
		}

		// Find the point closest to start of ray
		min := math.Inf(1)
		var minI = -1
		for i, p := range points {
			d2 := (cx-p[0])*(cx-p[0]) + (cy-p[1])*(cy-p[1])
			if d2 < min {
				min = d2
				minI = i
			}
		}
		rays = append(rays, Line{cx, cy, points[minI][0], points[minI][1]})
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

func handleRays() {
	var raysFactor float64
	switch {
	case inpututil.IsKeyJustPressed(ebiten.Key0):
		numRays = 0
		return
	case inpututil.IsKeyJustPressed(ebiten.Key1):
		raysFactor = 2
	case inpututil.IsKeyJustPressed(ebiten.Key2):
		raysFactor = 3
	case inpututil.IsKeyJustPressed(ebiten.Key3):
		raysFactor = 4
	case inpututil.IsKeyJustPressed(ebiten.Key4):
		raysFactor = 5
	case inpututil.IsKeyJustPressed(ebiten.Key5):
		raysFactor = 6
	case inpututil.IsKeyJustPressed(ebiten.Key6):
		raysFactor = 7
	case inpututil.IsKeyJustPressed(ebiten.Key7):
		raysFactor = 8
	case inpututil.IsKeyJustPressed(ebiten.Key8):
		raysFactor = 9
	case inpututil.IsKeyJustPressed(ebiten.Key9):
		raysFactor = 10
	default:
		return
	}
	numRays = float64(math.Pow(2, raysFactor))
}

func handleMovement() {
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
		px += 2
	}

	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyDown) {
		py += 2
	}

	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
		px -= 2
	}

	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp) {
		py -= 2
	}

	// +1/-1 is to stop player before it reaches the border
	if px >= screenHeight-padding {
		px = screenHeight - padding - 1
	}

	if px <= padding {
		px = padding + 1
	}

	if py >= screenWidth-padding {
		py = screenWidth - padding - 1
	}

	if py <= padding {
		py = padding + 1
	}
}

func update(screen *ebiten.Image) error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return errors.New("game ended by player")
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		showRays = !showRays
	}

	handleRays()
	handleMovement()

	if ebiten.IsDrawingSkipped() {
		return nil
	}

	// Reset the shadowImage
	shadowImage.Fill(color.Black)
	rays := rayCasting(px, py, walls, numRays)

	// Subtract ray triangles from shadow
	opt := &ebiten.DrawTrianglesOptions{}
	opt.Address = ebiten.AddressRepeat
	opt.CompositeMode = ebiten.CompositeModeSourceOut

	for i, line := range rays {
		nextLine := rays[(i+1)%len(rays)]

		// Draw triangle of area between rays
		v := vertices(px, py, nextLine.X2, nextLine.Y2, line.X2, line.Y2)
		shadowImage.DrawTriangles(v, []uint16{0, 1, 2}, triangleImage, opt)
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
	ebitenutil.DrawRect(screen, px-2, py-2, 4, 4, color.Black)
	ebitenutil.DrawRect(screen, px-1, py-1, 2, 2, color.RGBA{255, 100, 100, 255})

	if showRays {
		ebitenutil.DebugPrintAt(screen, "R: hide rays", padding, 0)
	} else {
		ebitenutil.DebugPrintAt(screen, "R: show rays", padding, 0)
	}

	ebitenutil.DebugPrintAt(screen, "WASD: move", 160, 0)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("TPS: %0.2f", ebiten.CurrentTPS()), 51, 51)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("0-9: control # rays (%4.0f)", numRays), padding, screenHeight-20)
	return nil
}

var (
	showRays bool
	numRays  float64
	px, py   float64
	walls    []Line
)

const padding = 20

func main() {
	px = screenWidth / 2
	py = screenHeight / 2
	numRays = 128

	// Add outer walls
	walls = append(walls, rect(padding, padding, screenWidth-2*padding, screenHeight-2*padding)...)

	// Angled wall
	walls = append(walls, Line{50, 110, 100, 150})

	// Rectangles
	walls = append(walls, rect(45, 50, 70, 20)...)
	walls = append(walls, rect(150, 50, 30, 60)...)

	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Ray casting and shadows (Ebiten demo)"); err != nil {
		log.Fatal(err)
	}
}
