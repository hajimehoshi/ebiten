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

package shaderprecomp

import (
	"github.com/hajimehoshi/ebiten/v2/internal/builtinshader"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

// AppendBuildinShaderSources appends all the built-in shader sources to the given slice.
//
// Do not modify the content of the shader source.
//
// AppendBuildinShaderSources is concurrent-safe.
func AppendBuildinShaderSources(sources []*ShaderSource) []*ShaderSource {
	for _, s := range builtinshader.AppendShaderSources(nil) {
		src, err := NewShaderSource(s)
		if err != nil {
			panic(err)
		}
		sources = append(sources, src)
	}
	return sources
}

// ShaderSource is an object encapsulating a shader source code.
type ShaderSource struct {
	source []byte
	id     ShaderSourceID
}

// NewShaderSource creates a new ShaderSource object from the given source code.
func NewShaderSource(source []byte) (*ShaderSource, error) {
	hash, err := graphics.CalcSourceHash(source)
	if err != nil {
		return nil, err
	}
	return &ShaderSource{
		source: source,
		id:     ShaderSourceID(hash),
	}, nil
}

// ID returns a unique identifier for the shader source.
// The ShaderSourceID value must be the same for the same shader source and the same Ebitengine version.
// There is no guarantee that the ShaderSourceID value is the same between different Ebitengine versions.
func (s *ShaderSource) ID() ShaderSourceID {
	return s.id
}

// ShaderSourceID is a uniuqe identifier for a shader source.
type ShaderSourceID [16]byte

// ParseSourceID parses a string representation of the shader source ID.
func ParseSourceID(s string) (ShaderSourceID, error) {
	h, err := shaderir.ParseSourceHash(s)
	if err != nil {
		return ShaderSourceID{}, err
	}
	return ShaderSourceID(h), nil
}

// String returns a string representation of the shader source ID.
func (s ShaderSourceID) String() string {
	return shaderir.SourceHash(s).String()
}
