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

package opengl

import (
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/glsl"
)

func (c *context) glslVersion() glsl.GLSLVersion {
	switch c.webGLVersion {
	case webGLVersion1:
		return glsl.GLSLVersionES100
	case webGLVersion2:
		return glsl.GLSLVersionES300
	}
	panic("opengl: WebGL context is not initialized yet at glslVersion")
}
