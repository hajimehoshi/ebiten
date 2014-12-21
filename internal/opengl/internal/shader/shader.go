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
uniform mat4 modelview_matrix_n;
attribute vec2 vertex;
attribute vec2 tex_coord0;
attribute vec2 tex_coord1;
varying vec2 vertex_out_tex_coord0;
varying vec2 vertex_out_tex_coord1;

void main(void) {
  vertex_out_tex_coord0 = tex_coord0;
  vertex_out_tex_coord1 = (modelview_matrix_n * vec4(tex_coord1, 0, 1)).xy;
  gl_Position = projection_matrix * modelview_matrix * vec4(vertex, 0, 1);
}
`,
	},
	shaderVertexFinal: {
		shaderType: gl.VERTEX_SHADER,
		source: `
uniform mat4 projection_matrix;
uniform mat4 modelview_matrix;
attribute vec2 vertex;
attribute vec2 tex_coord0;
varying vec2 vertex_out_tex_coord0;

void main(void) {
  vertex_out_tex_coord0 = tex_coord0;
  gl_Position = projection_matrix * modelview_matrix * vec4(vertex, 0, 1);
}
`,
	},
	shaderColorMatrix: {
		shaderType: gl.FRAGMENT_SHADER,
		source: `
uniform sampler2D texture0;
uniform sampler2D texture1;
uniform mat4 color_matrix;
uniform vec4 color_matrix_translation;
varying vec2 vertex_out_tex_coord0;
varying vec2 vertex_out_tex_coord1;

void main(void) {
  vec4 color0 = texture2D(texture0, vertex_out_tex_coord0);
  vec4 color1 = texture2D(texture1, vertex_out_tex_coord1);
  color0 = (color_matrix * color0) + color_matrix_translation;

  // Photoshop-like RGBA blending.
  //
  // NOTE: If the textures are alpha premultiplied, this calc would be much simpler,
  // but the color matrix must be applied to the straight alpha colors.
  // Thus, straight alpha colors are used in Ebiten.
  gl_FragColor.a = color0.a + (1.0 - color0.a) * color1.a;
  gl_FragColor.rgb = (color0.a * color0.rgb + (1.0 - color0.a) * color1.a * color1.rgb) / gl_FragColor.a;
}
`,
	},
	shaderColorFinal: {
		shaderType: gl.FRAGMENT_SHADER,
		source: `
uniform sampler2D texture0;
varying vec2 vertex_out_tex_coord0;

void main(void) {
  vec4 color0 = texture2D(texture0, vertex_out_tex_coord0);
  gl_FragColor.rgb = color0.a * color0.rgb;
  gl_FragColor.a = 1.0;
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
