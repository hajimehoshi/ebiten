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

	"github.com/hajimehoshi/ebiten/internal/web"
)

type shaderId int

const (
	shaderVertexModelview shaderId = iota
	shaderFragmentTexture
)

func shader(id shaderId) string {
	s := shaders[id]
	defs := []string{}
	if web.IsMobileBrowser() {
		defs = append(defs, "#define IS_MOBILE_BROWSER")
	}
	s = strings.Replace(s, "{{Definitions}}", strings.Join(defs, "\n"), -1)
	return s
}

var shaders = map[shaderId]string{
	shaderVertexModelview: `
{{Definitions}}

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

{{Definitions}}

uniform sampler2D texture;
uniform mat4 color_matrix;
uniform vec4 color_matrix_translation;
uniform vec2 source_size;
uniform int filter_type;

varying vec2 varying_tex_coord;
varying vec2 varying_tex_coord_min;
varying vec2 varying_tex_coord_max;

vec2 roundTexel(vec2 p) {
#if defined(IS_MOBILE_BROWSER)
  // This is a temporary hack: on mobile browsers like Android Chrome,
  // the code to round a float might cause unexpected behavior.
  // This is confirmed with rendering a relatively large (e.g. 4096x512) image
  // with specifying a source rectangle near its edges.
  // TODO: examples/moire doesn't work anyway.
  // TODO: Does this function work on an Android native app?
  return p;
#else
  vec2 factor = 1.0 / (source_size * 256.0);
  if (factor.x > 0.0) {
    p.x -= mod(p.x + factor.x * 0.5, factor.x) - factor.x * 0.5;
  }
  if (factor.y > 0.0) {
    p.y -= mod(p.y + factor.y * 0.5, factor.y) - factor.y * 0.5;
  }
  return p;
#endif
}

vec4 getColorAt(vec2 pos) {
  if (pos.x < varying_tex_coord_min.x ||
      pos.y < varying_tex_coord_min.y ||
      varying_tex_coord_max.x <= pos.x ||
      varying_tex_coord_max.y <= pos.y) {
    return vec4(0, 0, 0, 0);
  }
  return texture2D(texture, pos);
}

void main(void) {
  vec4 color = vec4(0, 0, 0, 0);

  vec2 pos = roundTexel(varying_tex_coord);
  if (filter_type == 1) {
    // Nearest neighbor
    color = getColorAt(pos);
  } else if (filter_type == 2) {
    // Bi-linear
    vec2 texel_size = 1.0 / source_size;
    pos -= texel_size * 0.5;
    vec4 c0 = getColorAt(pos);
    vec4 c1 = getColorAt(pos + vec2(texel_size.x, 0));
    vec4 c2 = getColorAt(pos + vec2(0, texel_size.y));
    vec4 c3 = getColorAt(pos + texel_size);
    float rateX = fract(pos.x * source_size.x);
    float rateY = fract(pos.y * source_size.y);
    color = mix(mix(c0, c1, rateX), mix(c2, c3, rateX), rateY);
  } else {
    color = vec4(1, 0, 0, 1);
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
