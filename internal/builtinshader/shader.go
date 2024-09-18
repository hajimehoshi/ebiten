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
)

type Filter int

const (
	FilterNearest Filter = iota
	FilterLinear
)

const FilterCount = 2

type Address int

const (
	AddressUnsafe Address = iota
	AddressClampToZero
	AddressRepeat
)

const AddressCount = 3

const (
	UniformColorMBody        = "ColorMBody"
	UniformColorMTranslation = "ColorMTranslation"
)

var (
	shaders  [FilterCount][AddressCount][2][]byte
	shadersM sync.Mutex
)

var tmpl = template.Must(template.New("tmpl").Parse(`//kage:unit pixels

package main

{{if .UseColorM}}
var ColorMBody mat4
var ColorMTranslation vec4
{{end}}

{{if eq .Address .AddressRepeat}}
func adjustTexelForAddressRepeat(p vec2) vec2 {
	origin := imageSrc0Origin()
	size := imageSrc0Size()
	return mod(p - origin, size) + origin
}
{{end}}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
{{if eq .Filter .FilterNearest}}
{{if eq .Address .AddressUnsafe}}
	clr := imageSrc0UnsafeAt(srcPos)
{{else if eq .Address .AddressClampToZero}}
	clr := imageSrc0At(srcPos)
{{else if eq .Address .AddressRepeat}}
	clr := imageSrc0At(adjustTexelForAddressRepeat(srcPos))
{{end}}
{{else if eq .Filter .FilterLinear}}
	p0 := srcPos - 1/2.0
	p1 := srcPos + 1/2.0

{{if eq .Address .AddressRepeat}}
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

	rate := fract(p1)
	clr := mix(mix(c0, c1, rate.x), mix(c2, c3, rate.x), rate.y)
{{end}}

{{if .UseColorM}}
	// Un-premultiply alpha.
	// When the alpha is 0, 1-sign(alpha) is 1.0, which means division does nothing.
	clr.rgb /= clr.a + (1-sign(clr.a))
	// Apply the clr matrix.
	clr = (ColorMBody * clr) + ColorMTranslation
	// Premultiply alpha
	clr.rgb *= clr.a
	// Apply the color scale.
	clr *= color
	// Clamp the output.
	clr.rgb = min(clr.rgb, clr.a)
{{else}}
	// Apply the color scale.
	clr *= color
{{end}}

	return clr
}

`))

// ShaderSource returns the built-in shader source based on the given parameters.
//
// The returned shader always uses a color matrix so far.
func ShaderSource(filter Filter, address Address, useColorM bool) []byte {
	shadersM.Lock()
	defer shadersM.Unlock()

	var c int
	if useColorM {
		c = 1
	}
	if s := shaders[filter][address][c]; s != nil {
		return s
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct {
		Filter             Filter
		FilterNearest      Filter
		FilterLinear       Filter
		Address            Address
		AddressUnsafe      Address
		AddressClampToZero Address
		AddressRepeat      Address
		UseColorM          bool
	}{
		Filter:             filter,
		FilterNearest:      FilterNearest,
		FilterLinear:       FilterLinear,
		Address:            address,
		AddressUnsafe:      AddressUnsafe,
		AddressClampToZero: AddressClampToZero,
		AddressRepeat:      AddressRepeat,
		UseColorM:          useColorM,
	}); err != nil {
		panic(fmt.Sprintf("builtinshader: tmpl.Execute failed: %v", err))
	}

	b := buf.Bytes()
	shaders[filter][address][c] = b
	return b
}

var ScreenShaderSource = []byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2) vec4 {
	// Blend source colors in a square region, which size is 1/scale.
	scale := imageDstSize()/imageSrc0Size()
	pos := srcPos
	p0 := pos - 1/2.0/scale
	p1 := pos + 1/2.0/scale

	// Texels must be in the source rect, so it is not necessary to check.
	c0 := imageSrc0UnsafeAt(p0)
	c1 := imageSrc0UnsafeAt(vec2(p1.x, p0.y))
	c2 := imageSrc0UnsafeAt(vec2(p0.x, p1.y))
	c3 := imageSrc0UnsafeAt(p1)

	// p is the p1 value in one pixel assuming that the pixel's upper-left is (0, 0) and the lower-right is (1, 1).
	rate := clamp(fract(p1)*scale, 0, 1)
	return mix(mix(c0, c1, rate.x), mix(c2, c3, rate.x), rate.y)
}
`)

var ClearShaderSource = []byte(`//kage:unit pixels

package main

func Fragment() vec4 {
	return vec4(0)
}
`)

func AppendShaderSources(sources [][]byte) [][]byte {
	for filter := Filter(0); filter < FilterCount; filter++ {
		for address := Address(0); address < AddressCount; address++ {
			sources = append(sources, ShaderSource(filter, address, false), ShaderSource(filter, address, true))
		}
	}
	sources = append(sources, ScreenShaderSource, ClearShaderSource)
	return sources
}
