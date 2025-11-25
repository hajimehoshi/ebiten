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

package atlas

import (
	"fmt"
	"runtime"
	"sync"
	"weak"

	"golang.org/x/sync/errgroup"

	"github.com/hajimehoshi/ebiten/v2/internal/builtinshader"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

// shadersWithInternalShader keeps track of shaders that have internal graphicscommand.Shader.
type shadersWithInternalShader struct {
	shaders map[weak.Pointer[Shader]]struct{}
	m       sync.Mutex
}

func (s *shadersWithInternalShader) add(shader *Shader) {
	s.m.Lock()
	defer s.m.Unlock()
	if s.shaders == nil {
		s.shaders = map[weak.Pointer[Shader]]struct{}{}
	}
	s.shaders[weak.Make(shader)] = struct{}{}
}

func (s *shadersWithInternalShader) remove(shader *Shader) {
	s.m.Lock()
	defer s.m.Unlock()
	delete(s.shaders, weak.Make(shader))
}

func (s *shadersWithInternalShader) deallocateInternalShaders() {
	s.m.Lock()
	shaders := make([]weak.Pointer[Shader], 0, len(s.shaders))
	for shader := range s.shaders {
		shaders = append(shaders, shader)
	}
	s.m.Unlock()

	for _, shader := range shaders {
		// deallocate invokes the internal shader's Dispose method.
		if shader := shader.Value(); shader != nil {
			shader.deallocate()
		}
	}

	s.m.Lock()
	clear(s.shaders)
	s.m.Unlock()
}

var theShadersWithInternalShader shadersWithInternalShader

type Shader struct {
	ir      *shaderir.Program
	shader  *graphicscommand.Shader
	unit    shaderir.Unit
	name    string
	cleanup runtime.Cleanup
}

func NewShader(ir *shaderir.Program, name string) *Shader {
	// A shader is initialized lazily, and the lock is not needed.
	return &Shader{
		ir:   ir,
		name: name,
		unit: ir.Unit,
	}
}

func (s *Shader) ensureShader() *graphicscommand.Shader {
	if s.shader != nil {
		return s.shader
	}
	s.shader = graphicscommand.NewShader(s.ir, s.name)
	s.cleanup = runtime.AddCleanup(s, func(shader *graphicscommand.Shader) {
		// A function from cleanup must not be blocked, but disposing operation can be blocked.
		// Defer this operation until it becomes safe. (#913)
		appendDeferred(func() {
			shader.Dispose()
		})
	}, s.shader)
	theShadersWithInternalShader.add(s)
	return s.shader
}

// Deallocate deallocates the internal state.
func (s *Shader) Deallocate() {
	backendsM.Lock()
	defer backendsM.Unlock()

	if !inFrame {
		appendDeferred(func() {
			s.deallocate()
		})
		return
	}

	s.deallocate()
}

func (s *Shader) deallocate() {
	s.cleanup.Stop()
	if s.shader == nil {
		return
	}
	s.shader.Dispose()
	s.shader = nil
	theShadersWithInternalShader.remove(s)
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
			return fmt.Errorf("atlas: compiling the nearest shader failed: %w", err)
		}
		nearestIR = ir
		return nil
	})
	wg.Go(func() error {
		ir, err := graphics.CompileShader([]byte(builtinshader.ShaderSource(builtinshader.FilterLinear, builtinshader.AddressUnsafe, false)))
		if err != nil {
			return fmt.Errorf("atlas: compiling the linear shader failed: %w", err)
		}
		linearIR = ir
		return nil
	})
	wg.Go(func() error {
		ir, err := graphics.CompileShader([]byte(builtinshader.ClearShaderSource))
		if err != nil {
			return fmt.Errorf("atlas: compiling the clear shader failed: %w", err)
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
