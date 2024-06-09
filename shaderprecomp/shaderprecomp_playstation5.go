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

//go:build playstation5

package shaderprecomp

import (
	"io"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/playstation5"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/pssl"
)

// CompileToPSSL compiles the shader source to PlayStaton Shader Language to writers.
//
// CompileToPSSL is concurrent-safe.
func CompileToPSSL(vertexWriter, pixelWriter io.Writer, source *ShaderSource) error {
	ir, err := graphics.CompileShader(source.source)
	if err != nil {
		return err
	}
	vs, ps := pssl.Compile(ir)
	if _, err = vertexWriter.Write([]byte(vs)); err != nil {
		return err
	}
	if _, err = pixelWriter.Write([]byte(ps)); err != nil {
		return err
	}
	return nil
}

// RegisterPlayStationShaders registers a precompiled PlayStation Shader for a shader source.
//
// RegisterPlayStationShaders is concurrent-safe.
func RegisterPlayStationShaders(source *ShaderSource, vertexShader, pixelShader []byte) {
	playstation5.RegisterPrecompiledShaders(source.source, vertexShader, pixelShader)
}
