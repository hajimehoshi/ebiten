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
	"strings"
	"testing"
	"text/template"

	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/glsl"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/hlsl"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/msl"
)

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
	var checkExpr func(shaderir.Expr)
	checkExpr = func(e shaderir.Expr) {
		if e.Type == shaderir.SwizzlingExpr {
			if len(e.Swizzling) < 1 || len(e.Swizzling) > 4 || strings.Trim(e.Swizzling, "xyzw") != "" {
				t.Errorf("noncanonical IR swizzle: %q", e.Swizzling)
			}
		}
		for _, child := range e.Exprs {
			checkExpr(child)
		}
	}
	var checkBlock func(*shaderir.Block)
	checkBlock = func(b *shaderir.Block) {
		for _, stmt := range b.Stmts {
			for _, e := range stmt.Exprs {
				checkExpr(e)
			}
			for _, child := range stmt.Blocks {
				checkBlock(child)
			}
		}
	}
	want := compile(t, "xyzw")
	checkBlock(want.FragmentFunc.Block)
	for _, f := range want.Funcs {
		checkBlock(f.Block)
	}
	wantOutputs := outputs(want)
	for _, set := range []string{"xyzw", "rgba", "stpq"} {
		t.Run(set, func(t *testing.T) {
			got := compile(t, set)
			if !reflect.DeepEqual(got.FragmentFunc, want.FragmentFunc) || !reflect.DeepEqual(got.Funcs, want.Funcs) {
				t.Error("equivalent swizzles produced different IR")
			}
			for i, output := range outputs(got) {
				if output != wantOutputs[i] {
					t.Errorf("backend output %d differs for equivalent swizzles", i)
				}
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
