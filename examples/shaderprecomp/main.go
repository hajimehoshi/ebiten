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

// This example demonstrates shader precompilation.
//
// Run 'go generate' to compile the shaders used by this example for the current
// platform and register them via the exp/shaderprecomp package. Ebitengine then
// uses the precompiled shaders instead of compiling them at runtime, which
// shortens the loading time. The example also works without 'go generate', in
// which case the shaders are compiled at runtime as usual.
package main

//go:generate go run gen.go

import (
	_ "embed"
	"fmt"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

//go:embed defaultshader.go
var defaultShaderSource []byte

// precompiledShadersRegistered reports whether precompiled shaders are registered.
var precompiledShadersRegistered bool

type Game struct {
	shader *ebiten.Shader
}

func (g *Game) Update() error {
	if g.shader == nil {
		s, err := ebiten.NewShader(defaultShaderSource)
		if err != nil {
			return err
		}
		g.shader = s
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	cx, cy := ebiten.CursorPosition()
	w, h := screen.Bounds().Dx(), screen.Bounds().Dy()
	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]any{
		"Time":   float32(ebiten.Tick()) / float32(ebiten.TPS()),
		"Cursor": []float32{float32(cx), float32(cy)},
	}
	screen.DrawRectShader(w, h, g.shader, op)

	state := "OFF (shaders are compiled at runtime; run 'go generate' to precompile them)"
	if precompiledShadersRegistered {
		state = "ON (precompiled shaders are registered for this platform)"
	}
	msg := fmt.Sprintf(`This is a test for shader precompilation.
Precompilation: %s
Note that this example works whether or not shaders are precompiled.`, state)
	ebitenutil.DebugPrint(screen, msg)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func main() {
	ebiten.SetWindowTitle("Ebitengine Example (Shader Precompilation)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}
