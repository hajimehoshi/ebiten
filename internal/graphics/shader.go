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
attribute vec4 tex_coord;
attribute vec4 geo_matrix_body;
attribute vec2 geo_matrix_translation;
varying vec2 varying_tex_coord;
varying vec2 varying_tex_coord_min;
varying vec2 varying_tex_coord_max;

void main(void) {
  varying_tex_coord = vec2(tex_coord[0], tex_coord[1]);
  varying_tex_coord_min =
    vec2(min(tex_coord[0], tex_coord[2]), min(tex_coord[1], tex_coord[3]));
  varying_tex_coord_max =
    vec2(max(tex_coord[0], tex_coord[2]), max(tex_coord[1], tex_coord[3]));
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

uniform sampler2D texture;
uniform mat4 color_matrix;
uniform vec4 color_matrix_translation;
varying vec2 varying_tex_coord;
varying vec2 varying_tex_coord_min;
varying vec2 varying_tex_coord_max;

void main(void) {
  vec4 color = vec4(0, 0, 0, 0);
  if (varying_tex_coord_min.x <= varying_tex_coord.x &&
      varying_tex_coord_min.y <= varying_tex_coord.y &&
      varying_tex_coord.x < varying_tex_coord_max.x &&
      varying_tex_coord.y < varying_tex_coord_max.y) {
    color = texture2D(texture, varying_tex_coord);
  }

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
