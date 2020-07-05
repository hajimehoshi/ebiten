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

// +build example jsgo

package main

import (
	"bytes"
	"image"
	_ "image/jpeg"
	"log"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/examples/resources/images"
	"github.com/hajimehoshi/ebiten/inpututil"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

var gophersImage *ebiten.Image

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
	img, _, err := image.Decode(bytes.NewReader(images.Gophers_jpg))
	if err != nil {
		log.Fatal(err)
	}
	gophersImage, _ = ebiten.NewImageFromImage(img, ebiten.FilterDefault)
}

var shaderSrcs = [][]byte{
	default_go,
	image_go,
}

type Game struct {
	shaders map[int]*ebiten.Shader
	idx     int
	time    int
}

func (g *Game) Update(screen *ebiten.Image) error {
	g.time++
	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		g.idx++
		g.idx %= len(shaderSrcs)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		g.idx += len(shaderSrcs) - 1
		g.idx %= len(shaderSrcs)
	}

	if g.shaders == nil {
		g.shaders = map[int]*ebiten.Shader{}
	}
	if _, ok := g.shaders[g.idx]; !ok {
		s, err := ebiten.NewShader([]byte(shaderSrcs[g.idx]), 1)
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
	vs := []ebiten.Vertex{
		{
			DstX: 0,
			DstY: 0,
			SrcX: 0,
			SrcY: 0,
		},
		{
			DstX: float32(w),
			DstY: 0,
			SrcX: float32(w),
			SrcY: 0,
		},
		{
			DstX: 0,
			DstY: float32(h),
			SrcX: 0,
			SrcY: float32(h),
		},
		{
			DstX: float32(w),
			DstY: float32(h),
			SrcX: float32(w),
			SrcY: float32(h),
		},
	}
	is := []uint16{0, 1, 2, 1, 2, 3}

	cx, cy := ebiten.CursorPosition()

	op := &ebiten.DrawTrianglesWithShaderOptions{}
	op.Uniforms = []interface{}{
		float32(g.time) / 60,                // Time
		[]float32{float32(cx), float32(cy)}, // Cursor
	}
	if g.idx != 0 {
		op.Uniforms = append(op.Uniforms,
			gophersImage, // Image
		)
	}
	screen.DrawTrianglesWithShader(vs, is, s, op)

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
