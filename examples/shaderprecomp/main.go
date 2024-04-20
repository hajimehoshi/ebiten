// Copyright 2024 The Ebitengine Authors
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
	_ "embed"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

//go:embed defaultshader.go
var defaultShaderSourceBytes []byte

type Game struct {
	defaultShader *ebiten.Shader
	counter       int
}

func (g *Game) Update() error {
	g.counter++

	if g.defaultShader == nil {
		s, err := ebiten.NewShader(defaultShaderSourceBytes)
		if err != nil {
			return err
		}
		g.defaultShader = s
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	cx, cy := ebiten.CursorPosition()
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]interface{}{
		"Time":   float32(g.counter) / float32(ebiten.TPS()),
		"Cursor": []float32{float32(cx), float32(cy)},
	}
	screen.DrawRectShader(w, h, g.defaultShader, op)

	msg := `This is a test for shader precompilation.
Precompilation works only on macOS so far.
Note that this example still works even without shader precompilation.`
	ebitenutil.DebugPrint(screen, msg)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func main() {
	if err := registerPrecompiledShaders(); err != nil {
		log.Fatal(err)
	}
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
