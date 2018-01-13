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
)

type shaderID int

const (
	shaderVertexModelview shaderID = iota
	shaderFragmentNearest
	shaderFragmentLinear
)

func shader(id shaderID) string {
	if id == shaderVertexModelview {
		return shaderStrVertex
	}
	defs := []string{}
	switch id {
	case shaderFragmentNearest:
		defs = append(defs, "#define FILTER_NEAREST")
	case shaderFragmentLinear:
		defs = append(defs, "#define FILTER_LINEAR")
	default:
		panic("not reached")
	}
	return strings.Replace(shaderStrFragment, "{{Definitions}}", strings.Join(defs, "\n"), -1)
}

const (
	shaderStrVertex = `
uniform mat4 projection_matrix;
attribute vec2 vertex;
attribute vec4 tex_coord;
varying vec2 varying_tex_coord;
varying vec2 varying_tex_coord_min;
varying vec2 varying_tex_coord_max;

void main(void) {
  varying_tex_coord = vec2(tex_coord[0], tex_coord[1]);
  varying_tex_coord_min =
    vec2(min(tex_coord[0], tex_coord[2]), min(tex_coord[1], tex_coord[3]));
  varying_tex_coord_max =
    vec2(max(tex_coord[0], tex_coord[2]), max(tex_coord[1], tex_coord[3]));
  gl_Position = projection_matrix * vec4(vertex, 0, 1);
}
`
	shaderStrFragment = `
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

#if defined(FILTER_LINEAR)
uniform highp vec2 source_size;
#endif

varying highp vec2 varying_tex_coord;
varying highp vec2 varying_tex_coord_min;
varying highp vec2 varying_tex_coord_max;

highp vec2 roundTexel(highp vec2 p) {
  // highp (relative) precision is 2^(-16) in the spec.
  // The minimum value for a denominator is half of 65536.
  highp float factor = 1.0 / 32768.0;
  p.x -= mod(p.x + factor * 0.5, factor) - factor * 0.5;
  p.y -= mod(p.y + factor * 0.5, factor) - factor * 0.5;
  return p;
}

void main(void) {
  highp vec2 pos = varying_tex_coord;

#if defined(FILTER_NEAREST)
  vec4 color = texture2D(texture, pos);
  if (pos.x < varying_tex_coord_min.x ||
    pos.y < varying_tex_coord_min.y ||
    varying_tex_coord_max.x <= pos.x ||
    varying_tex_coord_max.y <= pos.y) {
    color = vec4(0, 0, 0, 0);
  }
#endif

#if defined(FILTER_LINEAR)
  pos = roundTexel(pos);
  highp vec2 texel_size = 1.0 / source_size;
  pos -= texel_size * 0.5;

  highp vec2 p0 = pos;
  highp vec2 p1 = pos + texel_size;
  vec4 c0 = texture2D(texture, p0);
  vec4 c1 = texture2D(texture, vec2(p1.x, p0.y));
  vec4 c2 = texture2D(texture, vec2(p0.x, p1.y));
  vec4 c3 = texture2D(texture, p1);
  if (p0.x < varying_tex_coord_min.x) {
    c0 = vec4(0, 0, 0, 0);
    c2 = vec4(0, 0, 0, 0);
  }
  if (p0.y < varying_tex_coord_min.y) {
    c0 = vec4(0, 0, 0, 0);
    c1 = vec4(0, 0, 0, 0);
  }
  if (varying_tex_coord_max.x <= p1.x) {
    c1 = vec4(0, 0, 0, 0);
    c3 = vec4(0, 0, 0, 0);
  }
  if (varying_tex_coord_max.y <= p1.y) {
    c2 = vec4(0, 0, 0, 0);
    c3 = vec4(0, 0, 0, 0);
  }

  vec2 rate = fract(pos * source_size);
  vec4 color = mix(mix(c0, c1, rate.x), mix(c2, c3, rate.x), rate.y);
#endif

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
`
)
