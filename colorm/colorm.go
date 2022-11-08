// Copyright 2022 The Ebitengine Authors
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

package colorm

import (
	"fmt"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/internal/builtinshader"
)

// Dim is a dimension of a ColorM.
const Dim = ebiten.ColorMDim

// ColorM represents a matrix to transform coloring when rendering an image.
//
// ColorM is applied to the straight alpha color
// while an Image's pixels' format is alpha premultiplied.
// Before applying a matrix, a color is un-multiplied, and after applying the matrix,
// the color is multiplied again.
//
// The initial value is identity.
type ColorM = ebiten.ColorM

func uniforms(c ColorM) map[string]interface{} {
	var body [16]float32
	var translation [4]float32
	c.ReadElements(body[:], translation[:])

	uniforms := map[string]interface{}{}
	uniforms[builtinshader.UniformColorMBody] = body[:]
	uniforms[builtinshader.UniformColorMTranslation] = translation[:]
	return uniforms
}

type builtinShaderKey struct {
	filter  builtinshader.Filter
	address builtinshader.Address
}

var (
	builtinShaders  = map[builtinShaderKey]*ebiten.Shader{}
	builtinShadersM sync.Mutex
)

func builtinShader(filter builtinshader.Filter, address builtinshader.Address) *ebiten.Shader {
	builtinShadersM.Lock()
	defer builtinShadersM.Unlock()

	key := builtinShaderKey{
		filter:  filter,
		address: address,
	}
	if s, ok := builtinShaders[key]; ok {
		return s
	}

	src := builtinshader.Shader(filter, address, true)
	s, err := ebiten.NewShader(src)
	if err != nil {
		panic(fmt.Sprintf("colorm: NewShader for a built-in shader failed: %v", err))
	}
	shader := s

	builtinShaders[key] = shader
	return shader
}
