// Copyright 2026 The Ebitengine Authors
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

// Package pinch is a game that moves, scales, and turns a square with gestures.
// It is shared by the desktop and mobile builds of the example.
package pinch

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	ScreenWidth  = 640
	ScreenHeight = 480

	minScale = 0.1
	maxScale = 10

	// gridPitch is the distance between the grid lines in the world.
	gridPitch = 80
	// gridLines is the number of the grid lines on each side of the origin.
	gridLines = 20
)

var (
	gridColor   = color.RGBA{0x40, 0x40, 0x40, 0xff}
	axisColor   = color.RGBA{0x80, 0x80, 0x80, 0xff}
	squareColor = color.RGBA{0xff, 0, 0, 0xff}
)

// Game is a game that moves, scales, and turns a square with gestures.
type Game struct {
	// view maps the world to the screen.
	view ebiten.GeoM

	gestures GestureTracker
	gesture  ViewGesture

	path vector.Path

	buttons  []*button
	points   []image.Point
	touchIDs []ebiten.TouchID
}

// NewGame returns a new game.
func NewGame() *Game {
	g := &Game{}
	g.gestures.Separate = GestureScale | GestureRotate
	g.reset()

	// The buttons for a gesture are lit when the gesture is reported, and the buttons to separate gestures are lit
	// when the gestures are separated.
	toggle := func(set *Gesture, gesture Gesture, lit bool) (func() bool, func()) {
		return func() bool { return (*set&gesture != 0) == lit }, func() { *set ^= gesture }
	}
	type spec struct {
		label   string
		set     *Gesture
		gesture Gesture
		lit     bool
	}
	specs := []spec{
		{"PanX", &g.gestures.Ignore, GesturePanX, false},
		{"PanY", &g.gestures.Ignore, GesturePanY, false},
		{"Scale", &g.gestures.Ignore, GestureScale, false},
		{"Rotate", &g.gestures.Ignore, GestureRotate, false},
		{"X or Y", &g.gestures.Separate, GesturePanX | GesturePanY, true},
		{"S or R", &g.gestures.Separate, GestureScale | GestureRotate, true},
	}
	const (
		w   = 84
		h   = 32
		gap = 4
		n   = 7
		x0  = (ScreenWidth - n*w - (n-1)*gap) / 2
		y0  = ScreenHeight - h - 16
	)
	rect := func(i int) image.Rectangle {
		x := x0 + i*(w+gap)
		return image.Rect(x, y0, x+w, y0+h)
	}
	for i, s := range specs {
		on, press := toggle(s.set, s.gesture, s.lit)
		g.buttons = append(g.buttons, &button{label: s.label, rect: rect(i), on: on, press: press})
	}
	g.buttons = append(g.buttons, &button{label: "Reset", rect: rect(len(specs)), press: g.reset})
	return g
}

func (g *Game) reset() {
	g.view.Reset()
	g.view.Translate(ScreenWidth/2, ScreenHeight/2)
}

// scale returns the scale of the view.
func (g *Game) scale() float64 {
	a, b := g.view.Element(0, 0), g.view.Element(0, 1)
	c, d := g.view.Element(1, 0), g.view.Element(1, 1)
	return math.Sqrt(math.Abs(a*d - b*c))
}

// angle returns the angle of the view, in radians.
func (g *Game) angle() float64 {
	return math.Atan2(g.view.Element(1, 0), g.view.Element(0, 0))
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.reset()
	}
	g.points, g.touchIDs = pressedPoints(g.points[:0], g.touchIDs)
	for _, b := range g.buttons {
		b.update(g.points)
	}

	g.gesture = g.gestures.Update()
	// Keep the scale in the range.
	s := g.scale()
	g.gesture.Scale = min(max(s*g.gesture.Scale, minScale), maxScale) / s
	g.gesture.Apply(&g.view)
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// The grid shows how the world is moved, scaled, and turned.
	const r = gridPitch * gridLines
	for i := -gridLines; i <= gridLines; i++ {
		clr := gridColor
		if i == 0 {
			clr = axisColor
		}
		p := float64(i * gridPitch)
		g.strokeWorldLine(screen, p, -r, p, r, clr)
		g.strokeWorldLine(screen, -r, p, r, p, clr)
	}

	// The square is centered at the origin of the world.
	g.path.Reset()
	const h = gridPitch / 2
	for i, p := range [][2]float64{{-h, -h}, {h, -h}, {h, h}, {-h, h}} {
		x, y := g.view.Apply(p[0], p[1])
		if i == 0 {
			g.path.MoveTo(float32(x), float32(y))
		} else {
			g.path.LineTo(float32(x), float32(y))
		}
	}
	g.path.Close()
	op := &vector.DrawPathOptions{
		AntiAlias: true,
	}
	op.ColorScale.ScaleWithColor(squareColor)
	vector.FillPath(screen, &g.path, nil, op)

	// The center of the gesture.
	if g.gesture.Scale != 1 || g.gesture.Rotation != 0 {
		vector.StrokeCircle(screen, float32(g.gesture.CenterX), float32(g.gesture.CenterY), 6, 1, color.White, true)
	}

	ebitenutil.DebugPrint(screen, fmt.Sprintf(`Pinch to scale, rotate with two fingers to turn, and scroll or drag to move
on a touchpad or a touch screen. The buttons choose the gestures to report and to separate.
Ignore: %s, Separate: %s
Scale: %0.3f, Angle: %0.1f deg
Scale change: %+0.4f, Turn: %+0.4f rad`,
		g.gestures.Ignore, g.gestures.Separate, g.scale(), g.angle()*180/math.Pi, g.gesture.Scale-1, g.gesture.Rotation))

	for _, b := range g.buttons {
		b.draw(screen)
	}
}

func (g *Game) strokeWorldLine(screen *ebiten.Image, x0, y0, x1, y1 float64, clr color.Color) {
	sx0, sy0 := g.view.Apply(x0, y0)
	sx1, sy1 := g.view.Apply(x1, y1)
	vector.StrokeLine(screen, float32(sx0), float32(sy0), float32(sx1), float32(sy1), 1, clr, true)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}
