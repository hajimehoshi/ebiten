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
	_ "embed"
	"fmt"
	"image"
	_ "image/png"
	"log"

	"github.com/ebitengine/debugui"

	"github.com/hajimehoshi/ebiten/v2"
	resources "github.com/hajimehoshi/ebiten/v2/examples/resources/images/shader"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

var (
	//go:embed default.go
	default_go []byte

	//go:embed texel.go
	texel_go []byte

	//go:embed lighting.go
	lighting_go []byte

	//go:embed radialblur.go
	radialblur_go []byte

	//go:embed chromaticaberration.go
	chromaticaberration_go []byte

	//go:embed dissolve.go
	dissolve_go []byte

	//go:embed water.go
	water_go []byte
)

// These directives are used for an shader analyzer in the future.
// See also #3157.

//ebitengine:shaderfile default.go
//ebitengine:shaderfile texel.go
//ebitengine:shaderfile lighting.go
//ebitengine:shaderfile radialblur.go
//ebitengine:shaderfile chromaticaberration.go
//ebitengine:shaderfile dissolve.go
//ebitengine:shaderfile water.go

const (
	screenWidth  = 640
	screenHeight = 480
)

var (
	gopherImage   *ebiten.Image
	gopherBgImage *ebiten.Image
	normalImage   *ebiten.Image
	noiseImage    *ebiten.Image
)

func init() {
	// Decode an image from the image file's byte slice.
	img, _, err := image.Decode(bytes.NewReader(resources.Gopher_png))
	if err != nil {
		log.Fatal(err)
	}
	gopherImage = ebiten.NewImageFromImage(img)
}

func init() {
	img, _, err := image.Decode(bytes.NewReader(resources.GopherBg_png))
	if err != nil {
		log.Fatal(err)
	}
	gopherBgImage = ebiten.NewImageFromImage(img)
}

func init() {
	img, _, err := image.Decode(bytes.NewReader(resources.Normal_png))
	if err != nil {
		log.Fatal(err)
	}
	normalImage = ebiten.NewImageFromImage(img)
}

func init() {
	img, _, err := image.Decode(bytes.NewReader(resources.Noise_png))
	if err != nil {
		log.Fatal(err)
	}
	noiseImage = ebiten.NewImageFromImage(img)
}

var shaderSrcs = [][]byte{
	default_go,
	texel_go,
	lighting_go,
	radialblur_go,
	chromaticaberration_go,
	dissolve_go,
	water_go,
}

type Game struct {
	debugui debugui.DebugUI

	shaders   map[int]*ebiten.Shader
	idx       int
	gamepadID ebiten.GamepadID
}

func (g *Game) Update() error {
	if g.gamepadID < 0 {
		if ids := inpututil.AppendJustConnectedGamepadIDs(nil); len(ids) > 0 {
			g.gamepadID = ids[0]
		}
	} else {
		if inpututil.IsGamepadJustDisconnected(g.gamepadID) {
			if ids := ebiten.AppendGamepadIDs(nil); len(ids) > 0 {
				g.gamepadID = ids[0]
			} else {
				g.gamepadID = -1
			}
		}
	}

	if _, err := g.debugui.Update(func(ctx *debugui.Context) error {
		ctx.Window("Shader", image.Rect(10, 10, 210, 110), func(layout debugui.ContainerLayout) {
			ctx.Text(fmt.Sprintf("%d / %d", g.idx+1, len(shaderSrcs)))
			ctx.SetGridLayout([]int{-1, -1}, nil)
			dec := func() {
				g.idx += len(shaderSrcs) - 1
				g.idx %= len(shaderSrcs)
			}
			ctx.Button("Prev").On(dec)
			if inpututil.IsStandardGamepadButtonJustPressed(g.gamepadID, ebiten.StandardGamepadButtonLeftTop) {
				dec()
			}
			inc := func() {
				g.idx++
				g.idx %= len(shaderSrcs)
			}
			ctx.Button("Next").On(inc)
			if inpututil.IsStandardGamepadButtonJustPressed(g.gamepadID, ebiten.StandardGamepadButtonLeftBottom) {
				inc()
			}
		})
		return nil
	}); err != nil {
		return err
	}

	if g.shaders == nil {
		g.shaders = map[int]*ebiten.Shader{}
	}
	if _, ok := g.shaders[g.idx]; !ok {
		s, err := ebiten.NewShader([]byte(shaderSrcs[g.idx]))
		if err != nil {
			return err
		}
		g.shaders[g.idx] = s
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	s, ok := g.shaders[g.idx]
	if !ok {
		return
	}

	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	cx, cy := ebiten.CursorPosition()

	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]any{
		"Time":   float32(ebiten.Tick()) / float32(ebiten.TPS()),
		"Cursor": []float32{float32(cx), float32(cy)},
	}
	op.Images[0] = gopherImage
	op.Images[1] = normalImage
	op.Images[2] = gopherBgImage
	op.Images[3] = noiseImage
	screen.DrawRectShader(w, h, s, op)

	g.debugui.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Shader (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{
		gamepadID: -1,
	}); err != nil {
		log.Fatal(err)
	}
}
