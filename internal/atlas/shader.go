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
	"runtime"

	"github.com/hajimehoshi/ebiten/v2/internal/restorable"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

type Shader struct {
	ir      *shaderir.Program
	shader  *restorable.Shader
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

func (s *Shader) ensureShader() *restorable.Shader {
	if s.shader != nil {
		return s.shader
	}
	s.shader = restorable.NewShader(s.ir, s.name)
	s.cleanup = runtime.AddCleanup(s, func(shader *restorable.Shader) {
		// A function from cleanup must not be blocked, but disposing operation can be blocked.
		// Defer this operation until it becomes safe. (#913)
		appendDeferred(func() {
			shader.Dispose()
		})
	}, s.shader)
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
}

var (
	NearestFilterShader = &Shader{
		shader: restorable.NearestFilterShader,
		unit:   restorable.NearestFilterShader.Unit(),
	}
	LinearFilterShader = &Shader{
		shader: restorable.LinearFilterShader,
		unit:   restorable.LinearFilterShader.Unit(),
	}
)
