// Copyright 2014 Hajime Hoshi
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

package graphics

type shaderId int

const (
	shaderVertexModelview shaderId = iota
	shaderFragmentTexture
)

func shader(id shaderId) string {
	return shaders[id]
}

var shaders = map[shaderId]string{
	shaderVertexModelview: `
uniform mat4 projection_matrix;
attribute vec2 vertex;
attribute vec2 tex_coord;
attribute vec4 geo_matrix_body;
attribute vec2 geo_matrix_translation;
varying vec2 vertex_out_tex_coord;

void main(void) {
  vertex_out_tex_coord = tex_coord;
  mat4 geo_matrix = mat4(
    vec4(geo_matrix_body[0], geo_matrix_body[2], 0, 0),
    vec4(geo_matrix_body[1], geo_matrix_body[3], 0, 0),
    vec4(0, 0, 1, 0),
    vec4(geo_matrix_translation, 0, 1)
  );
  gl_Position = projection_matrix * geo_matrix * vec4(vertex, 0, 1);
}
`,
	shaderFragmentTexture: `
#if defined(GL_ES)
precision mediump float;
#else
#define lowp
#define mediump
#define highp
#endif

uniform lowp sampler2D texture;
uniform lowp mat4 color_matrix;
uniform lowp vec4 color_matrix_translation;
#if defined(GL_ES) && defined(GL_FRAGMENT_PRECISION_HIGH)
varying highp vec2 vertex_out_tex_coord;
#else
varying vec2 vertex_out_tex_coord;
#endif

void main(void) {
  lowp vec4 color = texture2D(texture, vertex_out_tex_coord);

  // Un-premultiply alpha
  if (0.0 < color.a) {
    color.rgb /= color.a;
  }
  // Apply the color matrix
  color = (color_matrix * color) + color_matrix_translation;
  color = clamp(color, 0.0, 1.0);
  // Premultiply alpha
  color.rgb *= color.a;

  gl_FragColor = color;
}
`,
}
