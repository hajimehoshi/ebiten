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

package restorable

import (
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/hajimehoshi/ebiten/v2/internal/builtinshader"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

type Shader struct {
	shader *graphicscommand.Shader
	ir     *shaderir.Program
	name   string
}

func NewShader(ir *shaderir.Program, name string) *Shader {
	s := &Shader{
		shader: graphicscommand.NewShader(ir, name),
		ir:     ir,
		name:   name,
	}
	theImages.addShader(s)
	return s
}

func (s *Shader) Dispose() {
	theImages.removeShader(s)
	s.shader.Dispose()
	s.shader = nil
	s.ir = nil
}

func (s *Shader) restore() {
	s.shader = graphicscommand.NewShader(s.ir, s.name)
}

func (s *Shader) Unit() shaderir.Unit {
	return s.ir.Unit
}

var (
	NearestFilterShader *Shader
	LinearFilterShader  *Shader
	clearShader         *Shader
)

func init() {
	var wg errgroup.Group
	var nearestIR, linearIR, clearIR *shaderir.Program
	wg.Go(func() error {
		ir, err := graphics.CompileShader([]byte(builtinshader.ShaderSource(builtinshader.FilterNearest, builtinshader.AddressUnsafe, false)))
		if err != nil {
			return fmt.Errorf("restorable: compiling the nearest shader failed: %w", err)
		}
		nearestIR = ir
		return nil
	})
	wg.Go(func() error {
		ir, err := graphics.CompileShader([]byte(builtinshader.ShaderSource(builtinshader.FilterLinear, builtinshader.AddressUnsafe, false)))
		if err != nil {
			return fmt.Errorf("restorable: compiling the linear shader failed: %w", err)
		}
		linearIR = ir
		return nil
	})
	wg.Go(func() error {
		ir, err := graphics.CompileShader([]byte(builtinshader.ClearShaderSource))
		if err != nil {
			return fmt.Errorf("restorable: compiling the clear shader failed: %w", err)
		}
		clearIR = ir
		return nil
	})
	if err := wg.Wait(); err != nil {
		panic(err)
	}
	NearestFilterShader = NewShader(nearestIR, "nearest")
	LinearFilterShader = NewShader(linearIR, "linear")
	clearShader = NewShader(clearIR, "clear")
}
