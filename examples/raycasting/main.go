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
	"sort"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/examples/resources/images"
	"github.com/hajimehoshi/ebiten/inpututil"
)

const (
	screenWidth  = 240
	screenHeight = 240
	padding      = 20
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

type line struct {
	X1, Y1, X2, Y2 float64
}

func (l *line) angle() float64 {
	return math.Atan2(l.Y2-l.Y1, l.X2-l.X1)
}

type object struct {
	walls []line
}

func (o object) points() [][2]float64 {
	// Get one of the endpoints for all segments,
	// + the startpoint of the first one, for non-closed paths
	var points [][2]float64
	for _, wall := range o.walls {
		points = append(points, [2]float64{wall.X2, wall.Y2})
	}
	points = append(points, [2]float64{o.walls[0].X1, o.walls[0].Y1})
	return points
}

func newRay(x, y, length, angle float64) line {
	return line{
		X1: x,
		Y1: y,
		X2: x + length*math.Cos(angle),
		Y2: y + length*math.Sin(angle),
	}
}

// intersection calculates the intersection of given two lines.
func intersection(l1, l2 line) (float64, float64, bool) {
	// https://en.wikipedia.org/wiki/Line%E2%80%93line_intersection#Given_two_points_on_each_line
	denom := (l1.X1-l1.X2)*(l2.Y1-l2.Y2) - (l1.Y1-l1.Y2)*(l2.X1-l2.X2)
	tNum := (l1.X1-l2.X1)*(l2.Y1-l2.Y2) - (l1.Y1-l2.Y1)*(l2.X1-l2.X2)
	uNum := -((l1.X1-l1.X2)*(l1.Y1-l2.Y1) - (l1.Y1-l1.Y2)*(l1.X1-l2.X1))

	if denom == 0 {
		return 0, 0, false
	}

	t := tNum / denom
	if t > 1 || t < 0 {
		return 0, 0, false
	}

	u := uNum / denom
	if u > 1 || u < 0 {
		return 0, 0, false
	}

	x := l1.X1 + t*(l1.X2-l1.X1)
	y := l1.Y1 + t*(l1.Y2-l1.Y1)
	return x, y, true
}

// rayCasting returns a slice of line originating from point cx, cy and intersecting with objects
func rayCasting(cx, cy float64, objects []object) []line {
	const rayLength = 1000 // something large enough to reach all objects

	var rays []line
	for _, obj := range objects {
		// Cast two rays per point
		for _, p := range obj.points() {
			l := line{cx, cy, p[0], p[1]}
			angle := l.angle()

			for _, offset := range []float64{-0.005, 0.005} {
				points := [][2]float64{}
				ray := newRay(cx, cy, rayLength, angle+offset)

				// Unpack all objects
				for _, o := range objects {
					for _, wall := range o.walls {
						if px, py, ok := intersection(ray, wall); ok {
							points = append(points, [2]float64{px, py})
						}
					}
				}

				// Find the point closest to start of ray
				min := math.Inf(1)
				minI := -1
				for i, p := range points {
					d2 := (cx-p[0])*(cx-p[0]) + (cy-p[1])*(cy-p[1])
					if d2 < min {
						min = d2
						minI = i
					}
				}
				rays = append(rays, line{cx, cy, points[minI][0], points[minI][1]})
			}
		}
	}

	// Sort rays based on angle, otherwise light triangles will not come out right
	sort.Slice(rays, func(i int, j int) bool {
		return rays[i].angle() < rays[j].angle()
	})
	return rays
}

func handleMovement() {
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
		px += 4
	}

	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyDown) {
		py += 4
	}

	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
		px -= 4
	}

	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp) {
		py -= 4
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

func rayVertices(x1, y1, x2, y2, x3, y3 float64) []ebiten.Vertex {
	return []ebiten.Vertex{
		{DstX: float32(x1), DstY: float32(y1), SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: float32(x2), DstY: float32(y2), SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
		{DstX: float32(x3), DstY: float32(y3), SrcX: 0, SrcY: 0, ColorR: 1, ColorG: 1, ColorB: 1, ColorA: 1},
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

	// Reset the shadowImage
	shadowImage.Fill(color.Black)
	rays := rayCasting(float64(px), float64(py), objects)

	// Subtract ray triangles from shadow
	opt := &ebiten.DrawTrianglesOptions{}
	opt.Address = ebiten.AddressRepeat
	opt.CompositeMode = ebiten.CompositeModeSourceOut
	for i, line := range rays {
		nextLine := rays[(i+1)%len(rays)]

		// Draw triangle of area between rays
		v := rayVertices(float64(px), float64(py), nextLine.X2, nextLine.Y2, line.X2, line.Y2)
		shadowImage.DrawTriangles(v, []uint16{0, 1, 2}, triangleImage, opt)
	}

	// Draw background
	screen.DrawImage(bgImage, nil)

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
	for _, obj := range objects {
		for _, w := range obj.walls {
			ebitenutil.DrawLine(screen, w.X1, w.Y1, w.X2, w.Y2, color.RGBA{255, 0, 0, 255})
		}
	}

	// Draw player as a rect
	ebitenutil.DrawRect(screen, float64(px)-2, float64(py)-2, 4, 4, color.Black)
	ebitenutil.DrawRect(screen, float64(px)-1, float64(py)-1, 2, 2, color.RGBA{255, 100, 100, 255})

	if showRays {
		ebitenutil.DebugPrintAt(screen, "R: hide rays", padding, 0)
	} else {
		ebitenutil.DebugPrintAt(screen, "R: show rays", padding, 0)
	}

	ebitenutil.DebugPrintAt(screen, "WASD: move", 160, 0)
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("TPS: %0.2f", ebiten.CurrentTPS()), 51, 51)
	return nil
}

var (
	showRays bool
	px, py   int
	objects  []object
)

func rect(x, y, w, h float64) []line {
	return []line{
		{x, y, x, y + h},
		{x, y + h, x + w, y + h},
		{x + w, y + h, x + w, y},
		{x + w, y, x, y},
	}
}

func main() {
	px = screenWidth / 2
	py = screenHeight / 2

	// Add outer walls
	objects = append(objects, object{rect(padding, padding, screenWidth-2*padding, screenHeight-2*padding)})

	// Angled wall
	objects = append(objects, object{[]line{{50, 110, 100, 150}}})

	// Rectangles
	objects = append(objects, object{rect(45, 50, 70, 20)})
	objects = append(objects, object{rect(150, 50, 30, 60)})

	if err := ebiten.Run(update, screenWidth, screenHeight, 2, "Ray casting and shadows (Ebiten demo)"); err != nil {
		log.Fatal(err)
	}
}
