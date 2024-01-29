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

	"golang.org/x/sync/errgroup"

	"github.com/hajimehoshi/ebiten/v2/internal/builtinshader"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

type Shader struct {
	ir     *shaderir.Program
	shader *graphicscommand.Shader
}

func NewShader(ir *shaderir.Program) *Shader {
	// A shader is initialized lazily, and the lock is not needed.
	return &Shader{
		ir: ir,
	}
}

func (s *Shader) finalize() {
	// A function from finalizer must not be blocked, but disposing operation can be blocked.
	// Defer this operation until it becomes safe. (#913)
	appendDeferred(func() {
		s.deallocate()
	})
}

func (s *Shader) ensureShader() *graphicscommand.Shader {
	if s.shader != nil {
		return s.shader
	}
	s.shader = graphicscommand.NewShader(s.ir)
	runtime.SetFinalizer(s, (*Shader).finalize)
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
	runtime.SetFinalizer(s, nil)
	if s.shader == nil {
		return
	}
	s.shader.Dispose()
	s.shader = nil
}

var (
	NearestFilterShader *Shader
	LinearFilterShader  *Shader
	clearShader         *Shader
)

func init() {
	var wg errgroup.Group
	wg.Go(func() error {
		ir, err := graphics.CompileShader([]byte(builtinshader.Shader(builtinshader.FilterNearest, builtinshader.AddressUnsafe, false)))
		if err != nil {
			return fmt.Errorf("atlas: compiling the nearest shader failed: %w", err)
		}
		NearestFilterShader = NewShader(ir)
		return nil
	})
	wg.Go(func() error {
		ir, err := graphics.CompileShader([]byte(builtinshader.Shader(builtinshader.FilterLinear, builtinshader.AddressUnsafe, false)))
		if err != nil {
			return fmt.Errorf("atlas: compiling the linear shader failed: %w", err)
		}
		LinearFilterShader = NewShader(ir)
		return nil
	})
	wg.Go(func() error {
		ir, err := graphics.CompileShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(0)
}`))
		if err != nil {
			return fmt.Errorf("atlas: compiling the clear shader failed: %w", err)
		}
		clearShader = NewShader(ir)
		return nil
	})
	if err := wg.Wait(); err != nil {
		panic(err)
	}
}
