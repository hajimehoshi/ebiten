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

type Address int

const (
	AddressUnsafe Address = iota
	AddressClampToZero
	AddressRepeat
)

const (
	UniformColorMBody        = "ColorMBody"
	UniformColorMTranslation = "ColorMTranslation"
)

type key struct {
	Filter    Filter
	Address   Address
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
	p0 := texCoord - texelSize/2 + texelSize/512
	p1 := texCoord + texelSize/2 + texelSize/512

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

	rate := fract(p0 * sourceSize)
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

// Shader returns the built-in shader based on the given parameters.
//
// The returned shader always uses a color matrix so far.
func Shader(filter Filter, address Address, useColorM bool) []byte {
	shadersM.Lock()
	defer shadersM.Unlock()

	k := key{
		Filter:    filter,
		Address:   address,
		UseColorM: useColorM,
	}
	if s, ok := shaders[k]; ok {
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
	shaders[k] = b
	return b
}
