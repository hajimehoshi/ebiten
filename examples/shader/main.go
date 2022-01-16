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

//go:build example
// +build example

package main

import (
	"bytes"
	"image"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	resources "github.com/hajimehoshi/ebiten/v2/examples/resources/images/shader"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

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
	// Now the byte slice is generated with //go:generate for Go 1.15 or older.
	// If you use Go 1.16 or newer, it is strongly recommended to use //go:embed to embed the image file.
	// See https://pkg.go.dev/embed for more details.
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
	shaders map[int]*ebiten.Shader
	idx     int
	time    int
}

func (g *Game) Update() error {
	g.time++
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		g.idx++
		g.idx %= len(shaderSrcs)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		g.idx += len(shaderSrcs) - 1
		g.idx %= len(shaderSrcs)
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

	w, h := screen.Size()
	cx, cy := ebiten.CursorPosition()

	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]interface{}{
		"Time":       float32(g.time) / 60,
		"Cursor":     []float32{float32(cx), float32(cy)},
		"ScreenSize": []float32{float32(w), float32(h)},
	}
	op.Images[0] = gopherImage
	op.Images[1] = normalImage
	op.Images[2] = gopherBgImage
	op.Images[3] = noiseImage
	screen.DrawRectShader(w, h, s, op)

	msg := "Press Up/Down to switch the shader."
	ebitenutil.DebugPrint(screen, msg)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Shader (Ebiten Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
