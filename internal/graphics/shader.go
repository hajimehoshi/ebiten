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

import (
	"strings"

	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
)

type shaderId int

const (
	shaderVertexModelview shaderId = iota
	shaderFragmentTexture
)

func shader(c *opengl.Context, id shaderId) string {
	str := shaders[id]
	if !c.GlslHighpSupported() {
		str = strings.Replace(str, "highp ", "", -1)
		str = strings.Replace(str, "lowp ", "", -1)
	}
	return str
}

var shaders = map[shaderId]string{
	shaderVertexModelview: `
uniform highp mat4 projection_matrix;
uniform highp mat4 modelview_matrix; // TODO: Remove this
attribute highp vec2 vertex;
attribute highp vec2 tex_coord;
attribute highp vec4 geo_matrix;
attribute highp vec2 geo_matrix_translation;
varying highp vec2 vertex_out_tex_coord;

void main(void) {
  vertex_out_tex_coord = tex_coord;
  mat4 x = modelview_matrix; // temporary hack not to omit modelview_matrix
  mat4 geom = mat4(
    vec4(geo_matrix[0], geo_matrix[2], 0, 0),
    vec4(geo_matrix[1], geo_matrix[3], 0, 0),
    vec4(0, 0, 1, 0),
    vec4(geo_matrix_translation, 0, 1)
  );
  gl_Position = projection_matrix * geom * vec4(vertex, 0, 1);
}
`,
	shaderFragmentTexture: `
uniform lowp sampler2D texture;
uniform lowp mat4 color_matrix;
uniform lowp vec4 color_matrix_translation;
varying highp vec2 vertex_out_tex_coord;

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
