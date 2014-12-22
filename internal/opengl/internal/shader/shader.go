/*
Copyright 2014 Hajime Hoshi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package shader

import (
	"github.com/go-gl/gl"
	"log"
)

type shader struct {
	native     gl.Shader
	shaderType gl.GLenum
	source     string
}

type shaderId int

const (
	shaderVertex shaderId = iota
	shaderVertexFinal
	shaderColorMatrix
	shaderColorFinal
)

var shaders = map[shaderId]*shader{
	shaderVertex: {
		shaderType: gl.VERTEX_SHADER,
		source: `
uniform mat4 projection_matrix;
uniform mat4 modelview_matrix;
attribute vec2 vertex;
attribute vec2 tex_coord;
varying vec2 vertex_out_tex_coord;

void main(void) {
  vertex_out_tex_coord = tex_coord;
  gl_Position = projection_matrix * modelview_matrix * vec4(vertex, 0, 1);
}
`,
	},
	shaderColorMatrix: {
		shaderType: gl.FRAGMENT_SHADER,
		source: `
uniform sampler2D texture;
uniform mat4 color_matrix;
uniform vec4 color_matrix_translation;
varying vec2 vertex_out_tex_coord;

void main(void) {
  vec4 color = texture2D(texture, vertex_out_tex_coord);
  // Un-premultiply alpha
  color.rgb /= color.a;
  // Apply the color matrix
  color = (color_matrix * color) + color_matrix_translation;
  // Premultiply alpha
  color = clamp(color, 0.0, 1.0);
  color.rgb *= color.a;

  gl_FragColor = color;
}
`,
	},
	shaderColorFinal: {
		shaderType: gl.FRAGMENT_SHADER,
		source: `
uniform sampler2D texture;
varying vec2 vertex_out_tex_coord;

void main(void) {
  gl_FragColor = texture2D(texture, vertex_out_tex_coord);
}
`,
	},
}

func (s *shader) compile() {
	s.native = gl.CreateShader(s.shaderType)
	if s.native == 0 {
		panic("glCreateShader failed")
	}

	s.native.Source(s.source)
	s.native.Compile()

	if s.native.Get(gl.COMPILE_STATUS) == gl.FALSE {
		s.showShaderLog()
		panic("shader compile failed")
	}
}

func (s *shader) showShaderLog() {
	if s.native.Get(gl.INFO_LOG_LENGTH) == 0 {
		return
	}
	log.Fatalf("shader error: %s\n", s.native.GetInfoLog())
}

func (s *shader) delete() {
	s.native.Delete()
}
