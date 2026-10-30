// Copyright 2026 The Ebitengine Authors
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

package vectorpaintshader

import (
	"bytes"
	"fmt"
	"sync"
	"text/template"
)

type Coverage int

const (
	CoverageNonZero Coverage = iota
	CoverageEvenOdd
	CoverageClipped
	CoverageCount
)

var (
	sources  [CoverageCount][2][2][]byte
	sourcesM sync.Mutex
)

var tmpl = template.Must(template.New("imagepaint").Parse(`//kage:unit pixels

package main

{{if ne .Coverage .CoverageClipped}}
func coverage(c {{if .AntiAlias}}vec4{{else}}float{{end}}) {{if .AntiAlias}}vec4{{else}}float{{end}} {
	// A winding number is biased by 128, which is even.
	v := floor(c*255 + 0.5)
{{if eq .Coverage .CoverageEvenOdd}}
	return mod(v, 2)
{{else}}
	return min(abs(v - 128), 1)
{{end}}
}
{{end}}

func Fragment(dstPos vec4, src0Pos vec2, color vec4, custom vec4) vec4 {
	p := imageSrc0Origin() + custom.xy
	c := imageSrc1UnsafeAtFromSrc0Pos(p){{if not .AntiAlias}}.r{{end}}
{{if ne .Coverage .CoverageClipped}}
	c = coverage(c)
{{end}}
{{if .AntiAlias}}
	c1 := imageSrc1UnsafeAtFromSrc0Pos(p + custom.zw)
{{if ne .Coverage .CoverageClipped}}
	c1 = coverage(c1)
{{end}}
	v := dot(c + c1, vec4(1.0/8.0))
{{else}}
	v := c
{{end}}
	if v == 0 {
		discard()
	}
{{if .Linear}}
	p0 := src0Pos - 0.5
	p1 := src0Pos + 0.5
	rate := fract(p1)
	a := mix(imageSrc0At(p0), imageSrc0At(vec2(p1.x, p0.y)), rate.x)
	b := mix(imageSrc0At(vec2(p0.x, p1.y)), imageSrc0At(p1), rate.x)
	clr := mix(a, b, rate.y)
{{else}}
	clr := imageSrc0At(src0Pos)
{{end}}
	return v * color * clr
}
`))

// ShaderSource returns the image-paint shader source for the specified coverage and sampling.
func ShaderSource(coverage Coverage, antialias, linear bool) []byte {
	sourcesM.Lock()
	defer sourcesM.Unlock()
	var antialiasIndex, filterIndex int
	if antialias {
		antialiasIndex = 1
	}
	if linear {
		filterIndex = 1
	}
	if source := sources[coverage][antialiasIndex][filterIndex]; source != nil {
		return source
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct {
		Coverage        Coverage
		CoverageEvenOdd Coverage
		CoverageClipped Coverage
		AntiAlias       bool
		Linear          bool
	}{
		Coverage:        coverage,
		CoverageEvenOdd: CoverageEvenOdd,
		CoverageClipped: CoverageClipped,
		AntiAlias:       antialias,
		Linear:          linear,
	}); err != nil {
		panic(fmt.Sprintf("vectorpaintshader: tmpl.Execute failed: %v", err))
	}
	source := buf.Bytes()
	sources[coverage][antialiasIndex][filterIndex] = source
	return source
}
