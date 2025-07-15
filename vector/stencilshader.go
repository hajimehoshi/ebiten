// Copyright 2025 The Ebitengine Authors
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

package vector

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// The implementation is based on the following article:
// https://medium.com/@evanwallace/easy-scalable-text-rendering-on-the-gpu-c3f4d782c5ac

// These values are protected by cacheM.

var (
	stencilBufferFillShader      *ebiten.Shader
	stencilBufferBezierShader    *ebiten.Shader
	stencilBufferNonZeroShader   *ebiten.Shader
	stencilBufferNonZeroAAShader *ebiten.Shader
	stencilBufferEvenOddShader   *ebiten.Shader
	stencilBufferEvenOddAAShader *ebiten.Shader
)

//ebitengine:shadersource
const stencilBufferFillShaderSrc = `//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4, custom vec4) vec4 {
	v := 1.0 / 255.0
	if frontfacing() {
		v *= 16
	}
	return v * color
}
`

//ebitengine:shadersource
const stencilBufferBezierShaderSrc = `//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4, custom vec4) vec4 {
	// Loop-Blinn algorithm.
	// https://developer.nvidia.com/gpugems/gpugems3/part-iv-image-effects/chapter-25-rendering-vector-art-gpu
	uv := custom.xy
	v := clamp(-sign(uv.x * uv.x - uv.y), 0, 1) * 1.0/255.0
	// This is opposite to the fill shader, especially for the non-zero fill rule.
	if !frontfacing() {
		v *= 16
	}
	return v * color
}
`

//ebitengine:shadersource
const stencilBufferNonZeroShaderSrc = `//kage:unit pixels

package main

func round(x float) float {
	return floor(x + 0.5)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	c := imageSrc0UnsafeAt(srcPos)
	r := int(round(c.r*255))
	w := abs((r >> 4) - (r & 0x0F))
	v := min(float(w), 1)
	return v * color
}
`

//ebitengine:shadersource
const stencilBufferNonZeroAAShaderSrc = `//kage:unit pixels

package main

func round(x vec4) vec4 {
	return floor(x + 0.5)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4, custom vec4) vec4 {
	c0 := imageSrc0UnsafeAt(srcPos)
	// imageSrc1UnsafeAt uses the offset info, which would prevent batching.
	// Use a custom offset instead.
	c1 := imageSrc0UnsafeAt(srcPos + custom.xy)
	ci0 := ivec4(round(c0*255))
	ci1 := ivec4(round(c1*255))
	w0 := abs((ci0 >> 4) - (ci0 & 0x0F))
	w1 := abs((ci1 >> 4) - (ci1 & 0x0F))
	v0 := min(vec4(w0), 1)
	v1 := min(vec4(w1), 1)
	return (dot(v0, vec4(1.0/8.0)) + dot(v1, vec4(1.0/8.0))) * color
}
`

//ebitengine:shadersource
const stencilBufferEvenOddShaderSrc = `//kage:unit pixels

package main

func round(x float) float {
	return floor(x + 0.5)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	c := imageSrc0UnsafeAt(srcPos)
	r := int(round(c.r*255))
	v := abs((r >> 4) - (r & 0x0F))
	return float(v % 2) * color
}
`

//ebitengine:shadersource
const stencilBufferEvenOddAAShaderSrc = `//kage:unit pixels

package main

func round(x vec4) vec4 {
	return floor(x + 0.5)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4, custom vec4) vec4 {
	c0 := imageSrc0UnsafeAt(srcPos)
	// imageSrc1UnsafeAt uses the offset info, which would prevent batching.
	// Use a custom offset instead.
	c1 := imageSrc0UnsafeAt(srcPos + custom.xy)
	ci0 := ivec4(round(c0*255))
	ci1 := ivec4(round(c1*255))
	w0 := abs((ci0 >> 4) - (ci0 & 0x0F))
	w1 := abs((ci1 >> 4) - (ci1 & 0x0F))
	v0 := vec4(w0 % 2)
	v1 := vec4(w1 % 2)
	return (dot(v0, vec4(1.0/8.0)) + dot(v1, vec4(1.0/8.0))) * color
}
`
