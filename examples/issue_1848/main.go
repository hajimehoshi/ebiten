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
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/issue_1848/shaders"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	screenWidth  = 640
	screenHeight = 480
)

var (
	shaderPaletteHorizontal *ebiten.Shader
	shaderPaletteVertical   *ebiten.Shader
	shaderShape             *ebiten.Shader
	shaderPrograms          []*ebiten.Shader

	uniformsByShaderIndex = make([]map[string]interface{}, 3)
)

func init() {
	rand.Seed(time.Now().UnixNano())
	shaderPaletteHorizontal, _ = ebiten.NewShader(shaders.ShaderPaletteHorizontalSrc)
	shaderPaletteVertical, _ = ebiten.NewShader(shaders.ShaderPaletteVerticalSrc)

	shaderPrograms = []*ebiten.Shader{
		shaderPaletteHorizontal,
		shaderPaletteVertical,
	}

	randomizeUniforms()
}

func randomizeUniforms() {
	uniformsByShaderIndex[0] = map[string]interface{}{
		"Palette0": []float32{
			rand.Float32(), rand.Float32(), rand.Float32(), 0.0,
			rand.Float32(), rand.Float32(), rand.Float32(), 1. / 3.,
			rand.Float32(), rand.Float32(), rand.Float32(), 2. / 3.,
			rand.Float32(), rand.Float32(), rand.Float32(), 1.0,
		},
	}
	uniformsByShaderIndex[1] = map[string]interface{}{
		"Palette1": []float32{
			rand.Float32(), rand.Float32(), rand.Float32(), 0.0,
			rand.Float32(), rand.Float32(), rand.Float32(), 1. / 3.,
			rand.Float32(), rand.Float32(), rand.Float32(), 2. / 3.,
			rand.Float32(), rand.Float32(), rand.Float32(), 1.0,
		},
	}
}

type Game struct {
	ticks int64
	idx   int
}

func (g *Game) Update() error {
	g.ticks++
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		g.idx++
		g.idx %= len(shaderPrograms)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		g.idx += len(shaderPrograms) - 1
		g.idx %= len(shaderPrograms)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		randomizeUniforms()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return errors.New("soft kill")
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.DrawRectShader(
		screenWidth,
		screenHeight,
		shaderPrograms[g.idx],
		&ebiten.DrawRectShaderOptions{
			Uniforms: uniformsByShaderIndex[g.idx],
		},
	)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Uniforms uploads test")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
