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

//go:generate go run gen.go
//go:generate gofmt -s -w .

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
	FilterPixelated
)

const FilterCount = 3

type Address int

const (
	AddressUnsafe Address = iota
	AddressClampToZero
	AddressRepeat
)

const AddressCount = 3

var (
	shaders  [FilterCount][AddressCount][]byte
	shadersM sync.Mutex
)

var tmpl = template.Must(template.New("tmpl").Parse(`//kage:unit pixels

package main

{{if eq .Address .AddressRepeat}}
func adjustSrcPosForAddressRepeat(p vec2) vec2 {
	origin := imageSrc0Origin()
	size := imageSrc0Size()
	return mod(p - origin, size) + origin
}
{{end}}

func Fragment(dstPos vec4, src0Pos vec2, color vec4) vec4 {
{{if eq .Filter .FilterNearest}}
{{if eq .Address .AddressUnsafe}}
	clr := imageSrc0UnsafeAt(src0Pos)
{{else if eq .Address .AddressClampToZero}}
	clr := imageSrc0At(src0Pos)
{{else if eq .Address .AddressRepeat}}
	clr := imageSrc0At(adjustSrcPosForAddressRepeat(src0Pos))
{{end}}
{{else}}
{{if eq .Filter .FilterLinear}}
	p0 := src0Pos - 1/2.0
	p1 := src0Pos + 1/2.0
{{else if eq .Filter .FilterPixelated}}
	// inversedScale is the size of the region on the source image.
	// The size is the inverse of the geometry-matrix scale.
	inversedScale := vec2(abs(dfdx(src0Pos.x)), abs(dfdy(src0Pos.y)))
	// Cap the inversedScale to 1 as dfdx/dfdy is not accurate on some machines (#3182).
	inversedScale = min(inversedScale, vec2(1))
	p0 := src0Pos - inversedScale/2.0
	p1 := src0Pos + inversedScale/2.0
{{end}}

{{if eq .Address .AddressRepeat}}
	p0 = adjustSrcPosForAddressRepeat(p0)
	p1 = adjustSrcPosForAddressRepeat(p1)
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

{{if eq .Filter .FilterLinear}}
	rate := fract(p1)
{{else if eq .Filter .FilterPixelated}}
	rate := clamp(fract(p1)/inversedScale, 0, 1)
{{end}}
	clr := mix(mix(c0, c1, rate.x), mix(c2, c3, rate.x), rate.y)
{{end}}

	// Apply the color scale.
	clr *= color

	return clr
}

`))

// ShaderSource returns the built-in shader source based on the given parameters.
func ShaderSource(filter Filter, address Address) []byte {
	shadersM.Lock()
	defer shadersM.Unlock()

	if s := shaders[filter][address]; s != nil {
		return s
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct {
		Filter             Filter
		FilterNearest      Filter
		FilterLinear       Filter
		FilterPixelated    Filter
		Address            Address
		AddressUnsafe      Address
		AddressClampToZero Address
		AddressRepeat      Address
	}{
		Filter:             filter,
		FilterNearest:      FilterNearest,
		FilterLinear:       FilterLinear,
		FilterPixelated:    FilterPixelated,
		Address:            address,
		AddressUnsafe:      AddressUnsafe,
		AddressClampToZero: AddressClampToZero,
		AddressRepeat:      AddressRepeat,
	}); err != nil {
		panic(fmt.Sprintf("builtinshader: tmpl.Execute failed: %v", err))
	}

	b := buf.Bytes()
	shaders[filter][address] = b
	return b
}

//ebitengine:shadersource
const ClearShaderSource = `//kage:unit pixels

package main

func Fragment() vec4 {
	return vec4(0)
}
`
