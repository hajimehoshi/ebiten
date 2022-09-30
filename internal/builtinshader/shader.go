// Copyright 2022 The Ebitengine Authors
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

package builtinshader

import (
	"bytes"
	"fmt"
	"sync"
	"text/template"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

const (
	UniformColorMBody        = "ColorMBody"
	UniformColorMTranslation = "ColorMTranslation"
)

type key struct {
	Filter    graphicsdriver.Filter
	Address   graphicsdriver.Address
	UseColorM bool
}

var (
	shaders  = map[key][]byte{}
	shadersM sync.Mutex
)

var tmpl = template.Must(template.New("tmpl").Parse(`package main

{{if .UseColorM}}
var ColorMBody mat4
var ColorMTranslation vec4
{{end}}

{{if eq .Address .AddressRepeat}}
func adjustTexelForAddressRepeat(p vec2) vec2 {
	origin, size := imageSrcRegionOnTexture()
	return mod(p - origin, size) + origin
}
{{end}}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
{{if eq .Filter .FilterNearest}}
{{if eq .Address .AddressUnsafe}}
	clr := imageSrc0UnsafeAt(texCoord)
{{else if eq .Address .AddressClampToZero}}
	clr := imageSrc0At(texCoord)
{{else if eq .Address .AddressRepeat}}
	clr := imageSrc0At(adjustTexelForAddressRepeat(texCoord))
{{end}}
{{else if eq .Filter .FilterLinear}}
	sourceSize := imageSrcTextureSize()
	texelSize := 1 / sourceSize

	// Shift 1/512 [texel] to avoid the tie-breaking issue (#1212).
	// As all the vertex positions are aligned to 1/16 [pixel], this shiting should work in most cases.
	p0 := texCoord - texelSize/2 + texelSize/512
	p1 := texCoord + texelSize/2 + texelSize/512

{{if eq .Address .AddressRpeat}}
	p0 = adjustTexelForAddressRepeat(p0)
	p1 = adjustTexelForAddressRepeat(p1)
{{end}}

{{if eq .Address .AddressUnsafe}}
	c0 := imageSrc0UnsafeAt(p0)
	c1 := imageSrc0UnsafeAt(vec2(p1.x, p0.y))
	c2 := imageSrc0UnsafeAt(vec2(p0.x, p1.y))
	c3 := imageSrc0UnsafeAt(p1)
{{else}}
	c0 := imageSrc0At(p0)
	c1 := imageSrc0At(vec2(p1.x, p0.y))
	c2 := imageSrc0At(vec2(p0.x, p1.y))
	c3 := imageSrc0At(p1)
{{end}}

	rate := fract(p0 * sourceSize)
	clr := mix(mix(c0, c1, rate.x), mix(c2, c3, rate.x), rate.y)
{{end}}

{{if .UseColorM}}
	// Un-premultiply alpha.
	// When the alpha is 0, 1-sign(alpha) is 1.0, which means division does nothing.
	clr.rgb /= clr.a + (1-sign(clr.a))
	// Apply the clr matrix or scale.
	clr = (ColorMBody * clr) + ColorMTranslation
	// Premultiply alpha
	clr.rgb *= clr.a
	// Do not apply the color scale assuming the scale is (1, 1, 1, 1).
	// Clamp the output.
	clr.rgb = min(clr.rgb, clr.a)
{{else}}
	// Apply the color scale.
	clr *= color
{{end}}

	return clr
}

`))

// Shader returns the built-in shader based on the given parameters.
//
// The returned shader always uses a color matrix so far.
func Shader(filter graphicsdriver.Filter, address graphicsdriver.Address) []byte {
	shadersM.Lock()
	defer shadersM.Unlock()

	// Now UseColorM is always true. When UseColorM is true, the color scale in vertices is not used.
	// In the built-in shaders, the color scale is modified at the vertex shader (#1996),
	// and this modification cannot be represented in a Kage program.
	// A custom vertex shader in Kage (#2209), or changing the interpretation of scales (#2365) would solve the issue.
	const useColorM = true

	k := key{
		Filter:    filter,
		Address:   address,
		UseColorM: useColorM,
	}
	if s, ok := shaders[k]; ok {
		return s
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]interface{}{
		"Filter":             filter,
		"FilterNearest":      graphicsdriver.FilterNearest,
		"FilterLinear":       graphicsdriver.FilterLinear,
		"Address":            address,
		"AddressUnsafe":      graphicsdriver.AddressUnsafe,
		"AddressClampToZero": graphicsdriver.AddressClampToZero,
		"AddressRepeat":      graphicsdriver.AddressRepeat,
		"UseColorM":          useColorM,
	}); err != nil {
		panic(fmt.Sprintf("builtinshader: tmpl.Execute failed: %v", err))
	}

	b := buf.Bytes()
	shaders[k] = b
	return b
}
