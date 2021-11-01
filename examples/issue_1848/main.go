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

//go:build example
// +build example

package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/examples/issue_1848/shaders"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

var (
	shaderSinLayer0            *ebiten.Shader
	shaderEightCirclesLayer1   *ebiten.Shader
	shaderDancingX             *ebiten.Shader
	shaderRandomPostprocessing *ebiten.Shader
	shaderPrograms             []*ebiten.Shader

	uniformsByShaderIndex = make([]map[string]interface{}, 5)
)

func init() {
	rand.Seed(time.Now().UnixNano())

	shaderSinLayer0, _ = ebiten.NewShader(shaders.SinLayer0Shader)
	shaderEightCirclesLayer1, _ = ebiten.NewShader(shaders.EightCircleBackgroundShader)
	shaderDancingX, _ = ebiten.NewShader(shaders.DancingXShader)
	shaderRandomPostprocessing, _ := ebiten.NewShader(shaders.RandomMaskShader)

	shaderPrograms = []*ebiten.Shader{
		shaderSinLayer0,
		shaderEightCirclesLayer1,
		shaderDancingX,
		shaderRandomPostprocessing,
	}

	randomizeUniforms()
}

func randomizeUniforms() {
	// Shader 0
	uniformsByShaderIndex[0] = map[string]interface{}{
		"ScreenSize": []float32{float32(screenWidth), float32(screenHeight)},
		"Palette": []float32{
			rand.Float32(), rand.Float32(), rand.Float32(), 0.0,
			rand.Float32(), rand.Float32(), rand.Float32(), 1. / 3.,
			rand.Float32(), rand.Float32(), rand.Float32(), 2. / 3.,
			rand.Float32(), rand.Float32(), rand.Float32(), 1.0,
		},
		"Frequency": rand.Float32() * 100,
		"Scale":     0.2 + rand.Float32()*0.3,
	}
	// Shader 1
	positions := make([]float32, 8*2)
	for i := range positions {
		positions[i] = rand.Float32() - 0.5
	}
	radiuses := make([]float32, 8)
	for i := range radiuses {
		radiuses[i] = rand.Float32() * 0.2
	}
	palettes := make([][]float32, 8)
	for i := range palettes {
		palettes[i] = []float32{
			rand.Float32(), rand.Float32(), rand.Float32(), 0.0,
			rand.Float32(), rand.Float32(), rand.Float32(), 1. / 3.,
			rand.Float32(), rand.Float32(), rand.Float32(), 2. / 3.,
			rand.Float32(), rand.Float32(), rand.Float32(), 1.0,
		}
	}
	uniformsByShaderIndex[1] = map[string]interface{}{
		"ScreenSize": []float32{float32(screenWidth), float32(screenHeight)},
		"Positions":  positions,
		"Aliasing":   rand.Float32() * 0.01,
		"Radiuses":   radiuses,
		"Palette0":   palettes[0],
		"Palette1":   palettes[1],
		"Palette2":   palettes[2],
		"Palette3":   palettes[3],
		"Palette4":   palettes[4],
		"Palette5":   palettes[5],
		"Palette6":   palettes[6],
		"Palette7":   palettes[7],
	}
	// Shader 2
	uniformsByShaderIndex[2] = map[string]interface{}{
		"ScreenSize": []float32{float32(screenWidth), float32(screenHeight)},
		"Width":      0.1 + rand.Float32()*0.1,
		"Radius":     rand.Float32() * 0.1,
		"Aliasing":   rand.Float32() * 0.01,
		"Frequency":  rand.Float32() * 25,
		"Palette": []float32{
			rand.Float32() * 0.1, rand.Float32() * 0.1, rand.Float32() * 0.1, 0.0,
			rand.Float32() * 0.1, rand.Float32() * 0.1, rand.Float32() * 0.1, 1. / 3.,
			rand.Float32() * 0.1, rand.Float32() * 0.1, rand.Float32() * 0.1, 2. / 3.,
			rand.Float32() * 0.1, rand.Float32() * 0.1, rand.Float32() * 0.1, 1.0,
		},
	}
	// Shader 3
	uniformsByShaderIndex[3] = map[string]interface{}{}
	for i := 0; i < 25; i++ {
		uniformsByShaderIndex[3][fmt.Sprintf("Maskvar%d", i)] = rand.Float32() * 0.1
	}
}

type Game struct {
	ticks  int64
	idx    int
	buffer *ebiten.Image
}

func (g *Game) Update() error {
	var switched bool

	g.ticks++
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		g.idx++
		g.idx %= len(shaderPrograms)
		switched = true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		g.idx += len(shaderPrograms) - 1
		g.idx %= len(shaderPrograms)
		switched = true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		randomizeUniforms()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return errors.New("soft kill")
	}
	if switched {
		fmt.Println("shader switch")
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.buffer.Clear()
	// Sin layer 0 background
	g.buffer.DrawRectShader(
		screenWidth,
		screenHeight,
		shaderPrograms[0],
		&ebiten.DrawRectShaderOptions{
			Uniforms: uniformsByShaderIndex[0],
		},
	)
	// 8 circles background
	g.buffer.DrawRectShader(
		screenWidth,
		screenHeight,
		shaderPrograms[1],
		&ebiten.DrawRectShaderOptions{
			Uniforms: uniformsByShaderIndex[1],
		},
	)
	// Dancing X
	uniformsByShaderIndex[2]["Time"] = float32(g.ticks%240)/240 - 0.5
	g.buffer.DrawRectShader(
		screenWidth,
		screenHeight,
		shaderPrograms[2],
		&ebiten.DrawRectShaderOptions{
			Uniforms: uniformsByShaderIndex[2],
		},
	)
	// Random post processing 1 out of desperation :)
	screen.DrawRectShader(
		screenWidth,
		screenHeight,
		shaderPrograms[3],
		&ebiten.DrawRectShaderOptions{
			Uniforms: uniformsByShaderIndex[3],
			Images: [4]*ebiten.Image{
				g.buffer,
			},
		},
	)

	ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f - FPS: %0.2f\n", ebiten.CurrentTPS(), ebiten.CurrentFPS()))
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetFPSMode(ebiten.FPSModeVsyncOffMaximum)
	ebiten.SetMaxTPS(60)
	ebiten.SetFullscreen(true)
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Uniforms uploads test")
	if err := ebiten.RunGame(&Game{
		buffer: ebiten.NewImage(screenWidth, screenHeight),
	}); err != nil {
		log.Fatal(err)
	}
}
