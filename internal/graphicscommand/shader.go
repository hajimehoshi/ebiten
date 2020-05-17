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

package graphicscommand

import (
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/shaderir"
)

type Shader struct {
	shader driver.Shader
}

func NewShader(ir *shaderir.Program) *Shader {
	s := &Shader{}
	c := &newShaderCommand{
		result: s,
		ir:     ir,
	}
	theCommandQueue.Enqueue(c)
	return s
}

func (s *Shader) Dispose() {
	c := &disposeShaderCommand{
		target: s,
	}
	theCommandQueue.Enqueue(c)
}
