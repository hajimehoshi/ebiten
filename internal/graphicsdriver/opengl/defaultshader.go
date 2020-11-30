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

package opengl

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/driver"
)

// glslReservedKeywords is a set of reserved keywords that cannot be used as an indentifier on some environments.
// See https://www.khronos.org/registry/OpenGL/specs/gl/GLSLangSpec.4.60.pdf.
var glslReservedKeywords = map[string]struct{}{
	"common": {}, "partition": {}, "active": {},
	"asm":   {},
	"class": {}, "union": {}, "enum": {}, "typedef": {}, "template": {}, "this": {},
	"resource": {},
	"goto":     {},
	"inline":   {}, "noinline": {}, "public": {}, "static": {}, "extern": {}, "external": {}, "interface": {},
	"long": {}, "short": {}, "half": {}, "fixed": {}, "unsigned": {}, "superp": {},
	"input": {}, "output": {},
	"hvec2": {}, "hvec3": {}, "hvec4": {}, "fvec2": {}, "fvec3": {}, "fvec4": {},
	"filter": {},
	"sizeof": {}, "cast": {},
	"namespace": {}, "using": {},
	"sampler3DRect": {},
}

var glslIdentifier = regexp.MustCompile(`[_a-zA-Z][_a-zA-Z0-9]*`)

func checkGLSL(src string) {
	for _, l := range strings.Split(src, "\n") {
		if strings.Contains(l, "//") {
			l = l[:strings.Index(l, "//")]
		}
		for _, token := range glslIdentifier.FindAllString(l, -1) {
			if _, ok := glslReservedKeywords[token]; ok {
				panic(fmt.Sprintf("opengl: %q is a reserved keyword", token))
			}
		}
	}
}

func vertexShaderStr() string {
	src := shaderStrVertex
	checkGLSL(src)
	return src
}

func fragmentShaderStr(useColorM bool, filter driver.Filter, address driver.Address) string {
	replaces := map[string]string{
		"{{.AddressClampToZero}}": fmt.Sprintf("%d", driver.AddressClampToZero),
		"{{.AddressRepeat}}":      fmt.Sprintf("%d", driver.AddressRepeat),
		"{{.AddressUnsafe}}":      fmt.Sprintf("%d", driver.AddressUnsafe),
	}
	src := shaderStrFragment
	for k, v := range replaces {
		src = strings.Replace(src, k, v, -1)
	}

	var defs []string

	if useColorM {
		defs = append(defs, "#define USE_COLOR_MATRIX")
	}

	switch filter {
	case driver.FilterNearest:
		defs = append(defs, "#define FILTER_NEAREST")
	case driver.FilterLinear:
		defs = append(defs, "#define FILTER_LINEAR")
	case driver.FilterScreen:
		defs = append(defs, "#define FILTER_SCREEN")
	default:
		panic(fmt.Sprintf("opengl: invalid filter: %d", filter))
	}

	switch address {
	case driver.AddressClampToZero:
		defs = append(defs, "#define ADDRESS_CLAMP_TO_ZERO")
	case driver.AddressRepeat:
		defs = append(defs, "#define ADDRESS_REPEAT")
	case driver.AddressUnsafe:
		defs = append(defs, "#define ADDRESS_UNSAFE")
	default:
		panic(fmt.Sprintf("opengl: invalid address: %d", address))
	}

	src = strings.Replace(src, "{{.Definitions}}", strings.Join(defs, "\n"), -1)

	checkGLSL(src)
	return src
}

const (
	shaderStrVertex = `
uniform vec2 viewport_size;
attribute vec2 A0;
attribute vec2 A1;
attribute vec4 A2;
varying vec2 varying_tex;
varying vec4 varying_color_scale;

void main(void) {
  varying_tex = A1;
  varying_color_scale = A2;

  mat4 projection_matrix = mat4(
    vec4(2.0 / viewport_size.x, 0, 0, 0),
    vec4(0, 2.0 / viewport_size.y, 0, 0),
    vec4(0, 0, 1, 0),
    vec4(-1, -1, 0, 1)
  );
  gl_Position = projection_matrix * vec4(A0, 0, 1);
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

{{.Definitions}}

uniform sampler2D T0;
uniform vec4 source_region;

#if defined(USE_COLOR_MATRIX)
uniform mat4 color_matrix_body;
uniform vec4 color_matrix_translation;
#endif

uniform highp vec2 source_size;

#if defined(FILTER_SCREEN)
uniform highp float scale;
#endif

varying highp vec2 varying_tex;
varying highp vec4 varying_color_scale;

highp float floorMod(highp float x, highp float y) {
  if (x < 0.0) {
    return y - (-x - y * floor(-x/y));
  }
  return x - y * floor(x/y);
}

highp vec2 adjustTexelByAddress(highp vec2 p, highp vec4 source_region) {
#if defined(ADDRESS_CLAMP_TO_ZERO)
  return p;
#endif

#if defined(ADDRESS_REPEAT)
  highp vec2 o = vec2(source_region[0], source_region[1]);
  highp vec2 size = vec2(source_region[2] - source_region[0], source_region[3] - source_region[1]);
  return vec2(floorMod((p.x - o.x), size.x) + o.x, floorMod((p.y - o.y), size.y) + o.y);
#endif

#if defined(ADDRESS_UNSAFE)
  return p;
#endif
}

void main(void) {
  highp vec2 pos = varying_tex;

#if defined(FILTER_NEAREST)
  vec4 color;
# if defined(ADDRESS_UNSAFE)
  color = texture2D(T0, pos);
# else
  pos = adjustTexelByAddress(pos, source_region);
  if (source_region[0] <= pos.x &&
      source_region[1] <= pos.y &&
      pos.x < source_region[2] &&
      pos.y < source_region[3]) {
    color = texture2D(T0, pos);
  } else {
    color = vec4(0, 0, 0, 0);
  }
# endif  // defined(ADDRESS_UNSAFE)
#endif  // defined(FILTER_NEAREST)

#if defined(FILTER_LINEAR)
  vec4 color;
  highp vec2 texel_size = 1.0 / source_size;

  // Shift 1/512 [texel] to avoid the tie-breaking issue.
  // As all the vertex positions are aligned to 1/16 [pixel], this shiting should work in most cases.
  highp vec2 p0 = pos - (texel_size) / 2.0 + (texel_size / 512.0);
  highp vec2 p1 = pos + (texel_size) / 2.0 + (texel_size / 512.0);

# if !defined(ADDRESS_UNSAFE)
  p0 = adjustTexelByAddress(p0, source_region);
  p1 = adjustTexelByAddress(p1, source_region);
# endif  // defined(ADDRESS_UNSAFE)

  vec4 c0 = texture2D(T0, p0);
  vec4 c1 = texture2D(T0, vec2(p1.x, p0.y));
  vec4 c2 = texture2D(T0, vec2(p0.x, p1.y));
  vec4 c3 = texture2D(T0, p1);
# if !defined(ADDRESS_UNSAFE)
  if (p0.x < source_region[0]) {
    c0 = vec4(0, 0, 0, 0);
    c2 = vec4(0, 0, 0, 0);
  }
  if (p0.y < source_region[1]) {
    c0 = vec4(0, 0, 0, 0);
    c1 = vec4(0, 0, 0, 0);
  }
  if (source_region[2] <= p1.x) {
    c1 = vec4(0, 0, 0, 0);
    c3 = vec4(0, 0, 0, 0);
  }
  if (source_region[3] <= p1.y) {
    c2 = vec4(0, 0, 0, 0);
    c3 = vec4(0, 0, 0, 0);
  }
# endif  // defined(ADDRESS_UNSAFE)

  vec2 rate = fract(p0 * source_size);
  color = mix(mix(c0, c1, rate.x), mix(c2, c3, rate.x), rate.y);
#endif  // defined(FILTER_LINEAR)

#if defined(FILTER_SCREEN)
  highp vec2 texel_size = 1.0 / source_size;
  highp vec2 half_scaled_texel_size = texel_size / 2.0 / scale;

  highp vec2 p0 = pos - half_scaled_texel_size + (texel_size / 512.0);
  highp vec2 p1 = pos + half_scaled_texel_size + (texel_size / 512.0);

  vec4 c0 = texture2D(T0, p0);
  vec4 c1 = texture2D(T0, vec2(p1.x, p0.y));
  vec4 c2 = texture2D(T0, vec2(p0.x, p1.y));
  vec4 c3 = texture2D(T0, p1);
  // Texels must be in the source rect, so it is not necessary to check that like linear filter.

  vec2 rate_center = vec2(1.0, 1.0) - half_scaled_texel_size;
  vec2 rate = clamp(((fract(p0 * source_size) - rate_center) * scale) + rate_center, 0.0, 1.0);
  gl_FragColor = mix(mix(c0, c1, rate.x), mix(c2, c3, rate.x), rate.y);

  // Assume that a color matrix and color vector values are not used with FILTER_SCREEN.

#else

# if defined(USE_COLOR_MATRIX)
  // Un-premultiply alpha.
  // When the alpha is 0, 1.0 - sign(alpha) is 1.0, which means division does nothing.
  color.rgb /= color.a + (1.0 - sign(color.a));
  // Apply the color matrix or scale.
  color = (color_matrix_body * color) + color_matrix_translation;
  color *= varying_color_scale;
  // Premultiply alpha
  color.rgb *= color.a;
# else
  vec4 s = varying_color_scale;
  color *= vec4(s.r, s.g, s.b, 1.0) * s.a;
# endif  // defined(USE_COLOR_MATRIX)

  color = min(color, color.a);

  gl_FragColor = color;

#endif  // defined(FILTER_SCREEN)

}
`
)
