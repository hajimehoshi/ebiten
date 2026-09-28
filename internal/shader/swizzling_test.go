// Copyright 2026 The Ebiten Authors
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

package shader_test

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"text/template"

	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/glsl"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/hlsl"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/msl"
)

func walkSwizzlingExprs(p *shaderir.Program, f func(e *shaderir.Expr)) {
	var walkExpr func(e *shaderir.Expr)
	walkExpr = func(e *shaderir.Expr) {
		if e.Type == shaderir.SwizzlingExpr {
			f(e)
		}
		for i := range e.Exprs {
			walkExpr(&e.Exprs[i])
		}
	}
	var walkBlock func(b *shaderir.Block)
	walkBlock = func(b *shaderir.Block) {
		if b == nil {
			return
		}
		for i := range b.Stmts {
			for j := range b.Stmts[i].Exprs {
				walkExpr(&b.Stmts[i].Exprs[j])
			}
			for _, child := range b.Stmts[i].Blocks {
				walkBlock(child)
			}
		}
	}
	walkBlock(p.VertexFunc.Block)
	walkBlock(p.FragmentFunc.Block)
	for _, fn := range p.Funcs {
		walkBlock(fn.Block)
	}
}

func TestSwizzlingCanonical(t *testing.T) {
	const source = `//kage:unit pixels
package main

func swizzle(v vec4) vec4 {
	v.{{.X}}{{.Y}} = v.{{.Y}}{{.X}}
	v.{{.Z}}{{.W}} += v.{{.X}}{{.Y}}
	v.{{.X}}, v.{{.Y}} = v.{{.Y}}, v.{{.X}}
	v.{{.W}}{{.Z}}{{.Y}}{{.X}}.xy = v.{{.Z}}{{.W}}
	return v.{{.W}}{{.Z}}{{.Y}}{{.X}}.rgba.stpq + v.{{.X}}{{.X}}{{.Y}}{{.Y}}
}

func Fragment(position vec4) vec4 {
	v := ivec4(1, 2, 3, 4)
	v.{{.X}}{{.Y}} = v.{{.Y}}{{.X}}
	v.{{.Z}}{{.W}} += v.{{.X}}{{.Y}}
	v.{{.X}}, v.{{.Y}} = v.{{.Y}}, v.{{.X}}
	v.{{.W}}{{.Z}}{{.Y}}{{.X}}.xy = v.{{.Z}}{{.W}}
	return swizzle(position) + vec4(v.{{.W}}{{.Z}}{{.Y}}{{.X}}.rgba.stpq + v.{{.X}}{{.X}}{{.Y}}{{.Y}})
}`
	tmpl, err := template.New("swizzling").Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	compile := func(t *testing.T, set string) *shaderir.Program {
		t.Helper()
		var src strings.Builder
		if err := tmpl.Execute(&src, struct {
			X, Y, Z, W string
		}{
			X: set[0:1],
			Y: set[1:2],
			Z: set[2:3],
			W: set[3:4],
		}); err != nil {
			t.Fatal(err)
		}
		p, err := compileToIR([]byte(src.String()))
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	outputs := func(p *shaderir.Program) []string {
		vs, fs := glsl.Compile(p, glsl.GLSLVersionDefault)
		esVS, esFS := glsl.Compile(p, glsl.GLSLVersionES300)
		hvs, hfs, _, _ := hlsl.Compile(p)
		return []string{vs, fs, esVS, esFS, hvs, hfs, msl.Compile(p)}
	}
	clearSwizzlingSet := func(e *shaderir.Expr) {
		e.SwizzlingSet = shaderir.SwizzlingSetXYZW
	}
	want := compile(t, "xyzw")
	walkSwizzlingExprs(want, func(e *shaderir.Expr) {
		if len(e.Swizzling) < 1 || len(e.Swizzling) > 4 || strings.Trim(e.Swizzling, "xyzw") != "" {
			t.Errorf("noncanonical IR swizzle: %q", e.Swizzling)
		}
	})
	wantOutputs := outputs(want)
	walkSwizzlingExprs(want, clearSwizzlingSet)
	for _, set := range []string{"xyzw", "rgba", "stpq"} {
		t.Run(set, func(t *testing.T) {
			got := compile(t, set)
			gotOutputs := outputs(got)
			walkSwizzlingExprs(got, clearSwizzlingSet)
			if !reflect.DeepEqual(got.FragmentFunc, want.FragmentFunc) || !reflect.DeepEqual(got.Funcs, want.Funcs) {
				t.Error("equivalent swizzles produced different IR")
			}
			for i, output := range gotOutputs {
				if output != wantOutputs[i] {
					t.Errorf("backend output %d differs for equivalent swizzles", i)
				}
			}
		})
	}
}

func TestSwizzlingSet(t *testing.T) {
	testCases := []struct {
		Selectors  string
		Swizzlings []string
		Sets       []shaderir.SwizzlingSet
	}{
		{
			Selectors:  "xyzw",
			Swizzlings: []string{"xyzw"},
			Sets:       []shaderir.SwizzlingSet{shaderir.SwizzlingSetXYZW},
		},
		{
			Selectors:  "rgba",
			Swizzlings: []string{"xyzw"},
			Sets:       []shaderir.SwizzlingSet{shaderir.SwizzlingSetRGBA},
		},
		{
			Selectors:  "stpq",
			Swizzlings: []string{"xyzw"},
			Sets:       []shaderir.SwizzlingSet{shaderir.SwizzlingSetSTPQ},
		},
		{
			Selectors:  "x",
			Swizzlings: []string{"x"},
			Sets:       []shaderir.SwizzlingSet{shaderir.SwizzlingSetXYZW},
		},
		{
			Selectors:  "rgr",
			Swizzlings: []string{"xyx"},
			Sets:       []shaderir.SwizzlingSet{shaderir.SwizzlingSetRGBA},
		},
		{
			Selectors:  "qpts",
			Swizzlings: []string{"wzyx"},
			Sets:       []shaderir.SwizzlingSet{shaderir.SwizzlingSetSTPQ},
		},
		{
			Selectors:  "xy.ts",
			Swizzlings: []string{"xy", "yx"},
			Sets:       []shaderir.SwizzlingSet{shaderir.SwizzlingSetXYZW, shaderir.SwizzlingSetSTPQ},
		},
		{
			Selectors:  "rg.yx",
			Swizzlings: []string{"xy", "yx"},
			Sets:       []shaderir.SwizzlingSet{shaderir.SwizzlingSetRGBA, shaderir.SwizzlingSetXYZW},
		},
		{
			Selectors:  "st.gr",
			Swizzlings: []string{"xy", "yx"},
			Sets:       []shaderir.SwizzlingSet{shaderir.SwizzlingSetSTPQ, shaderir.SwizzlingSetRGBA},
		},
		{
			Selectors:  "wzyx.rgba.stpq",
			Swizzlings: []string{"wzyx", "xyzw", "xyzw"},
			Sets:       []shaderir.SwizzlingSet{shaderir.SwizzlingSetXYZW, shaderir.SwizzlingSetRGBA, shaderir.SwizzlingSetSTPQ},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Selectors, func(t *testing.T) {
			p, err := compileToIR([]byte(fmt.Sprintf(`//kage:unit pixels
package main
func Fragment() vec4 {
	var v vec4
	w := v.%s
	_ = w
	return v
}`, tc.Selectors)))
			if err != nil {
				t.Fatal(err)
			}
			var swizzlings, sources []string
			var sets []shaderir.SwizzlingSet
			walkSwizzlingExprs(p, func(e *shaderir.Expr) {
				swizzlings = append(swizzlings, e.Swizzling)
				sets = append(sets, e.SwizzlingSet)
				sources = append(sources, e.SourceSwizzling())
			})
			if !slices.Equal(swizzlings, tc.Swizzlings) {
				t.Errorf("Swizzling: got: %q, want: %q", swizzlings, tc.Swizzlings)
			}
			if !slices.Equal(sets, tc.Sets) {
				t.Errorf("SwizzlingSet: got: %v, want: %v", sets, tc.Sets)
			}
			if want := strings.Split(tc.Selectors, "."); !slices.Equal(sources, want) {
				t.Errorf("SourceSwizzling: got: %q, want: %q", sources, want)
			}
		})
	}
}

func TestSwizzlingValidation(t *testing.T) {
	check := func(t *testing.T, typ, swizzle string, valid bool) {
		t.Helper()
		for _, statement := range []string{
			"_ = v." + swizzle,
			"v." + swizzle + " = v." + swizzle,
		} {
			source := fmt.Sprintf(`//kage:unit pixels
package main
func Fragment() vec4 {
	var v %s
	%s
	return vec4(0)
}`, typ, statement)
			_, err := compileToIR([]byte(source))
			if (err == nil) != valid {
				t.Errorf("%s: %s: success = %t, want %t (error: %v)", typ, statement, err == nil, valid, err)
			}
		}
	}
	for _, prefix := range []string{"vec", "ivec"} {
		for size := 2; size <= 4; size++ {
			typ := fmt.Sprintf("%s%d", prefix, size)
			t.Run(typ, func(t *testing.T) {
				for _, set := range []string{"xyzw", "rgba", "stpq"} {
					for i := range set {
						check(t, typ, set[i:i+1], i < size)
					}
				}
				for _, swizzle := range []string{"rx", "sr", "strq", "xs", "rt", "pqx", "xyzwx", "xy.p", "rg.b", "st.z"} {
					check(t, typ, swizzle, false)
				}
				for _, swizzle := range []string{"xy.ts", "rg.yx", "st.gr"} {
					check(t, typ, swizzle, true)
				}
			})
		}
	}
}
