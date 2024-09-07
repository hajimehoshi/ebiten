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
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

var nextShaderID = 1

func genNextShaderID() int {
	id := nextShaderID
	nextShaderID++
	return id
}

type Shader struct {
	shader graphicsdriver.Shader
	ir     *shaderir.Program
	id     int

	// name is used only for logging.
	name string
}

func NewShader(ir *shaderir.Program, name string) *Shader {
	s := &Shader{
		ir:   ir,
		id:   genNextShaderID(),
		name: name,
	}
	c := &newShaderCommand{
		result: s,
		ir:     ir,
	}
	theCommandQueueManager.enqueueCommand(c)
	return s
}

func (s *Shader) Dispose() {
	c := &disposeShaderCommand{
		target: s,
	}
	theCommandQueueManager.enqueueCommand(c)
}

func (s *Shader) unit() shaderir.Unit {
	return s.ir.Unit
}
