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
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

// The implementation is based on the following article:
// https://medium.com/@evanwallace/easy-scalable-text-rendering-on-the-gpu-c3f4d782c5ac

// These values are protected by stencilBufferM.

var (
	stencilBufferFillShader      *ebiten.Shader
	stencilBufferNonZeroShader   *ebiten.Shader
	stencilBufferNonZeroAAShader *ebiten.Shader
	stencilBufferEvenOddShader   *ebiten.Shader
	stencilBufferEvenOddAAShader *ebiten.Shader

	stencilBufferM sync.Mutex
)

func ensureStencilBufferShader() (*ebiten.Shader, error) {
	stencilBufferM.Lock()
	defer stencilBufferM.Unlock()

	if stencilBufferFillShader != nil {
		return stencilBufferFillShader, nil
	}
	s, err := ebiten.NewShader([]byte(stencilBufferFillShaderSrc))
	if err != nil {
		return nil, err
	}
	stencilBufferFillShader = s
	return stencilBufferFillShader, err
}

func ensureStencilBufferNonZeroShader(antialias bool) (*ebiten.Shader, error) {
	stencilBufferM.Lock()
	defer stencilBufferM.Unlock()

	if antialias {
		if stencilBufferNonZeroAAShader != nil {
			return stencilBufferNonZeroAAShader, nil
		}
		s, err := ebiten.NewShader([]byte(stencilBufferNonZeroAAShaderSrc))
		if err != nil {
			return nil, err
		}
		stencilBufferNonZeroAAShader = s
		return stencilBufferNonZeroAAShader, nil
	}

	if stencilBufferNonZeroShader != nil {
		return stencilBufferNonZeroShader, nil
	}
	s, err := ebiten.NewShader([]byte(stencilBufferNonZeroShaderSrc))
	if err != nil {
		return nil, err
	}
	stencilBufferNonZeroShader = s
	return stencilBufferNonZeroShader, nil
}

func ensureStencilBufferEvenOddShader(antialias bool) (*ebiten.Shader, error) {
	stencilBufferM.Lock()
	defer stencilBufferM.Unlock()

	if antialias {
		if stencilBufferEvenOddAAShader != nil {
			return stencilBufferEvenOddAAShader, nil
		}
		s, err := ebiten.NewShader([]byte(stencilBufferEvenOddAAShaderSrc))
		if err != nil {
			return nil, err
		}
		stencilBufferEvenOddAAShader = s
		return stencilBufferEvenOddAAShader, nil
	}

	if stencilBufferEvenOddShader != nil {
		return stencilBufferEvenOddShader, nil
	}
	s, err := ebiten.NewShader([]byte(stencilBufferEvenOddShaderSrc))
	if err != nil {
		return nil, err
	}
	stencilBufferEvenOddShader = s
	return stencilBufferEvenOddShader, nil
}

//ebitengine:shadersource
const stencilBufferFillShaderSrc = `//kage:unit pixels

package main

func Fragment(dstPos vec4, src0Pos vec2, color vec4, custom vec4) vec4 {
	// Loop-Blinn algorithm.
	// https://developer.nvidia.com/gpugems/gpugems3/part-iv-image-effects/chapter-25-rendering-vector-art-gpu
	// A fan triangle has (u, v) = (0, 1) at every vertex, which is always inside the curve.
	uv := custom.xy
	v := clamp(-sign(uv.x * uv.x - uv.y), 0, 1) * 1.0/255.0
	return v * color
}
`

//ebitengine:shadersource
const stencilBufferNonZeroShaderSrc = `//kage:unit pixels

package main

func Fragment(dstPos vec4, src0Pos vec2, color vec4) vec4 {
	c := imageSrc0UnsafeAt(src0Pos)
	// The winding number is biased by 128.
	if floor(c.r*255 + 0.5) == 128 {
		discard()
	}
	return color
}
`

//ebitengine:shadersource
const stencilBufferNonZeroAAShaderSrc = `//kage:unit pixels

package main

func Fragment(dstPos vec4, src0Pos vec2, color vec4, custom vec4) vec4 {
	c0 := imageSrc0UnsafeAt(src0Pos)
	// imageSrc1UnsafeAt uses the offset info, which would prevent batching.
	// Use a custom offset instead.
	c1 := imageSrc0UnsafeAt(src0Pos + custom.xy)
	// The winding numbers are biased by 128.
	v0 := min(abs(floor(c0*255 + 0.5) - 128), 1)
	v1 := min(abs(floor(c1*255 + 0.5) - 128), 1)
	v := dot(v0, vec4(1.0/8.0)) + dot(v1, vec4(1.0/8.0))
	if v == 0 {
		discard()
	}
	return v * color
}
`

//ebitengine:shadersource
const stencilBufferEvenOddShaderSrc = `//kage:unit pixels

package main

func Fragment(dstPos vec4, src0Pos vec2, color vec4) vec4 {
	c := imageSrc0UnsafeAt(src0Pos)
	// The winding number is biased by 128, which is even.
	if mod(floor(c.r*255 + 0.5), 2) == 0 {
		discard()
	}
	return color
}
`

//ebitengine:shadersource
const stencilBufferEvenOddAAShaderSrc = `//kage:unit pixels

package main

func Fragment(dstPos vec4, src0Pos vec2, color vec4, custom vec4) vec4 {
	c0 := imageSrc0UnsafeAt(src0Pos)
	// imageSrc1UnsafeAt uses the offset info, which would prevent batching.
	// Use a custom offset instead.
	c1 := imageSrc0UnsafeAt(src0Pos + custom.xy)
	// The winding numbers are biased by 128, which is even.
	v0 := mod(floor(c0*255 + 0.5), 2)
	v1 := mod(floor(c1*255 + 0.5), 2)
	v := dot(v0, vec4(1.0/8.0)) + dot(v1, vec4(1.0/8.0))
	if v == 0 {
		discard()
	}
	return v * color
}
`
