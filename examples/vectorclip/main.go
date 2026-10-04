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

package main

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"

	"github.com/ebitengine/debugui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	screenWidth  = 960
	screenHeight = 768
	pathSize     = 96
	panelSize    = 288
)

var scenes = []struct {
	name        string
	description string
}{
	{
		name:        "Identical clip",
		description: "Clipping a shape by itself leaves its coverage unchanged.",
	},
	{
		name:        "Duplicate restrictions",
		description: "Applying the same clip twice has the same effect as applying it once.",
	},
	{
		name:        "Partial curved clip",
		description: "A displaced circle clips away part of the shape.",
	},
	{
		name:        "Union",
		description: "Two independently filled circles form one clip: points inside either circle survive.",
	},
	{
		name:        "Intersection",
		description: "An intersection restricts the shape to the overlap of two circles.",
	},
	{
		name:        "Hole and fill rule",
		description: "The two concentric contours form a hole only with the even-odd rule.",
	},
	{
		name:        "Nested composition",
		description: "Union of a left circle clipped to the top and a right circle clipped to the bottom.",
	},
	{
		name:        "No restrictions",
		description: "A nil Clip leaves the drawing unchanged.",
	},
	{
		name:        "Empty clip",
		description: "An empty union contains no points, so the clipped result is empty.",
	},
}

var qualities = []struct {
	name      string
	scale     int
	antialias bool
}{
	{
		name:  "No antialiasing",
		scale: 1,
	},
	{
		name:      "Native antialiasing",
		scale:     1,
		antialias: true,
	},
	{
		name:  "2x point sampling",
		scale: 2,
	},
	{
		name:  "4x point sampling",
		scale: 4,
	},
}

type Game struct {
	debugui      debugui.DebugUI
	scene        int
	quality      int
	stroke       bool
	slightOffset bool
	evenOdd      bool
	showClips    bool
	layers       [3]*ebiten.Image
	resolved     [3]*ebiten.Image
	checker      *ebiten.Image
}

func (g *Game) changeScene(delta int) {
	g.scene = (g.scene + delta + len(scenes)) % len(scenes)
}

func (g *Game) changeQuality() {
	g.quality = (g.quality + 1) % len(qualities)
}

func (g *Game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		g.changeScene(-1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		g.changeScene(1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		g.changeQuality()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		g.stroke = !g.stroke
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyO) {
		g.slightOffset = !g.slightOffset
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyH) {
		g.evenOdd = !g.evenOdd
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		g.showClips = !g.showClips
	}
	_, err := g.debugui.Update(func(ctx *debugui.Context) error {
		ctx.Window("Clip case [Left/Right]", image.Rect(16, 16, 464, 208), func(layout debugui.ContainerLayout) {
			ctx.SetGridLayout([]int{-1, -1}, nil)
			ctx.Loop(len(scenes), func(i int) {
				label := scenes[i].name
				if i == g.scene {
					label = "* " + label
				}
				ctx.Button(label).On(func() {
					g.scene = i
				})
			})
		})
		ctx.Window("Quality [Q]", image.Rect(480, 16, 736, 208), func(layout debugui.ContainerLayout) {
			ctx.Loop(len(qualities), func(i int) {
				label := qualities[i].name
				if i == g.quality {
					label = "* " + label
				}
				ctx.Button(label).On(func() {
					g.quality = i
				})
			})
			ctx.Text("Same image at 1x and 3x")
		})
		ctx.Window("Display", image.Rect(752, 16, 944, 208), func(layout debugui.ContainerLayout) {
			ctx.Checkbox(&g.stroke, "Stroke [S]")
			ctx.Checkbox(&g.slightOffset, "Slight offset [O]")
			ctx.Checkbox(&g.evenOdd, "Even-odd holes [H]")
			ctx.Checkbox(&g.showClips, "Clip regions [C]")
		})
		return nil
	})
	return err
}

func circle(x, y, radius float32) *vector.Path {
	var p vector.Path
	p.Arc(x, y, radius, 0, 2*math.Pi, vector.Clockwise)
	p.Close()
	return &p
}

func rectangle(x0, y0, x1, y1 float32) *vector.Path {
	var p vector.Path
	p.MoveTo(x0, y0)
	p.LineTo(x1, y0)
	p.LineTo(x1, y1)
	p.LineTo(x0, y1)
	p.Close()
	return &p
}

func clip(paths ...*vector.Path) vector.Clip {
	if len(paths) == 1 {
		return &vector.PathClip{
			Path: paths[0],
		}
	}
	c := &vector.ClipSet{
		Operation: vector.ClipOperationUnion,
	}
	for _, p := range paths {
		c.Children = append(c.Children, &vector.PathClip{
			Path: p,
		})
	}
	return c
}

func (g *Game) paths() (*vector.Path, vector.Clip) {
	scale := float32(qualities[g.quality].scale)
	var offset float32
	if g.slightOffset {
		offset = 0.375
	}
	// Both drawing and clip paths use coordinates of the destination layer.
	transform := &vector.AddPathOptions{}
	transform.GeoM.Translate(float64(offset), float64(offset))
	transform.GeoM.Scale(float64(scale), float64(scale))
	transformed := func(p *vector.Path) *vector.Path {
		var dst vector.Path
		dst.AddPath(p, transform)
		return &dst
	}
	shape := transformed(circle(48, 48, 33))
	identity := shape
	if g.stroke {
		var outline vector.Path
		outline.AddStroke(shape, &vector.AddStrokeOptions{
			StrokeOptions: vector.StrokeOptions{
				Width: 5 * scale,
			},
		})
		identity = &outline
	}
	left := transformed(circle(33, 43, 27))
	right := transformed(circle(63, 53, 27))
	switch g.scene {
	case 0:
		return shape, clip(identity)
	case 1:
		c := clip(identity)
		return shape, &vector.ClipSet{
			Operation: vector.ClipOperationIntersection,
			Children:  []vector.Clip{c, c},
		}
	case 2:
		return shape, clip(left)
	case 3:
		return shape, clip(left, right)
	case 4:
		return shape, &vector.ClipSet{
			Operation: vector.ClipOperationIntersection,
			Children:  []vector.Clip{clip(left), clip(right)},
		}
	case 5:
		ring := circle(48, 48, 38)
		ring.AddPath(circle(48, 48, 22), nil)
		rule := vector.FillRuleNonZero
		if g.evenOdd {
			rule = vector.FillRuleEvenOdd
		}
		return shape, &vector.PathClip{
			Path:     transformed(ring),
			FillRule: rule,
		}
	case 6:
		return shape, &vector.ClipSet{
			Operation: vector.ClipOperationUnion,
			Children: []vector.Clip{
				&vector.ClipSet{
					Operation: vector.ClipOperationIntersection,
					Children: []vector.Clip{
						clip(left),
						clip(transformed(rectangle(0, 0, 96, 48))),
					},
				},
				&vector.ClipSet{
					Operation: vector.ClipOperationIntersection,
					Children: []vector.Clip{
						clip(right),
						clip(transformed(rectangle(0, 48, 96, 96))),
					},
				},
			},
		}
	case 7:
		return shape, nil
	default:
		return shape, &vector.ClipSet{}
	}
}

func (g *Game) prepareImages() {
	if g.checker == nil {
		img := image.NewRGBA(image.Rect(0, 0, panelSize, panelSize))
		for y := range panelSize {
			for x := range panelSize {
				shade := uint8(39 + 9*((x/12+y/12)%2))
				img.SetRGBA(x, y, color.RGBA{R: shade, G: shade, B: shade, A: 255})
			}
		}
		g.checker = ebiten.NewImageFromImage(img)
	}
	size := pathSize * qualities[g.quality].scale
	if g.layers[0] != nil && g.layers[0].Bounds().Dx() != size {
		for _, layer := range g.layers {
			layer.Deallocate()
		}
		g.layers = [3]*ebiten.Image{}
	}
	for i := range g.layers {
		if g.layers[i] == nil {
			g.layers[i] = ebiten.NewImage(size, size)
		}
		g.layers[i].Clear()
		if g.resolved[i] == nil {
			g.resolved[i] = ebiten.NewImage(pathSize, pathSize)
		}
		g.resolved[i].Clear()
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.prepareImages()
	shape, clip := g.paths()
	quality := qualities[g.quality]
	for i := range 2 {
		op := &vector.DrawPathOptions{
			AntiAlias: quality.antialias,
		}
		op.ColorScale.ScaleWithColor(color.RGBA{R: 70, G: 205, B: 235, A: 255})
		if i == 1 {
			op.Clip = clip
		}
		if g.stroke {
			vector.StrokePath(g.layers[i], shape, &vector.StrokeOptions{
				Width: float32(5 * quality.scale),
			}, op)
		} else {
			vector.FillPath(g.layers[i], shape, nil, op)
		}
	}
	// The left panel shows each restriction as a translucent region, including
	// the restriction's nested elements. Overlapping restrictions are intersected.
	if g.showClips {
		size := float32(pathSize * quality.scale)
		area := rectangle(0, 0, size, size)
		var clips []vector.Clip
		if clip != nil {
			clips = []vector.Clip{clip}
			if set, ok := clip.(*vector.ClipSet); ok && set.Operation == vector.ClipOperationIntersection {
				clips = set.Children
			}
		}
		for i, c := range clips {
			op := &vector.DrawPathOptions{
				AntiAlias: quality.antialias,
				Clip:      c,
			}
			if i%2 == 0 {
				op.ColorScale.Scale(1, 0.65, 0.1, 1)
			} else {
				op.ColorScale.Scale(1, 0.2, 0.65, 1)
			}
			op.ColorScale.ScaleAlpha(0.35)
			vector.FillPath(g.layers[2], area, nil, op)
		}
	}
	for i := range g.layers {
		// Supersampled layers retain point coverage until clipping is complete.
		reduce := &ebiten.DrawImageOptions{}
		reduce.GeoM.Scale(1/float64(quality.scale), 1/float64(quality.scale))
		reduce.Filter = ebiten.FilterLinear
		g.resolved[i].DrawImage(g.layers[i], reduce)
	}

	screen.Fill(color.RGBA{R: 24, G: 27, B: 33, A: 255})
	for i, label := range []string{"Input shape", "Clipped result"} {
		img := g.resolved[i]
		if i == 0 && g.showClips {
			label += " + clip regions"
		}

		x := 144 + i*384
		ebitenutil.DebugPrintAt(screen, label, x, 232)
		for _, zoom := range []int{1, 3} {
			px, py := x+(panelSize-pathSize)/2, 268
			if zoom == 3 {
				px, py = x, 408
			}
			ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%dx (%d x %d)", zoom, pathSize*zoom, pathSize*zoom), px, py-20)
			background := &ebiten.DrawImageOptions{}
			background.GeoM.Translate(float64(px), float64(py))
			region := g.checker.SubImage(image.Rect(0, 0, pathSize*zoom, pathSize*zoom)).(*ebiten.Image)
			screen.DrawImage(region, background)
			present := &ebiten.DrawImageOptions{}
			present.GeoM.Scale(float64(zoom), float64(zoom))
			present.GeoM.Translate(float64(px), float64(py))
			screen.DrawImage(img, present)
			if i == 0 && g.showClips {
				screen.DrawImage(g.resolved[2], present)
			}
		}
	}
	ebitenutil.DebugPrintAt(screen, scenes[g.scene].description, 24, 712)
	ebitenutil.DebugPrintAt(screen, "Amber / pink regions show separate clips. The result keeps points inside every clip.", 24, 736)

	g.debugui.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Vector clipping (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{
		scene:        2,
		quality:      1,
		showClips:    true,
		slightOffset: true,
		evenOdd:      true,
	}); err != nil {
		log.Fatal(err)
	}
}
