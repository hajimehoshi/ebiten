// Copyright 2020 The Ebiten Authors
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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/shader"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/glsl"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/hlsl"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/msl"
)

func glslVertexNormalize(str string) string {
	p := glsl.VertexPrelude(glsl.GLSLVersionDefault)
	str = strings.TrimPrefix(str, p)
	return strings.TrimSpace(str)
}

func glslFragmentNormalize(str string) string {
	p := glsl.FragmentPrelude(glsl.GLSLVersionDefault)
	str = strings.TrimPrefix(str, p)
	return strings.TrimSpace(str)
}

func hlslNormalize(str string, prelude string) string {
	str = strings.TrimPrefix(str, prelude)
	return strings.TrimSpace(str)
}

func metalNormalize(str string) string {
	prelude := msl.Prelude()
	str = strings.TrimPrefix(str, prelude)
	return strings.TrimSpace(str)
}

func compare(t *testing.T, title, got, want string) {
	var msg string
	gotlines := strings.Split(got, "\n")
	wantlines := strings.Split(want, "\n")
	for i := range gotlines {
		if len(wantlines) <= i {
			msg = fmt.Sprintf(`lines %d:
got:  %s
want: (out of range)`, i+1, gotlines[i])
			break
		}
		if gotlines[i] != wantlines[i] {
			msg = fmt.Sprintf(`lines %d:
got:  %s
want: %s`, i+1, gotlines[i], wantlines[i])
			break
		}
	}
	t.Errorf("%s: got: %v, want: %v\n\n%s", title, got, want, msg)
}

func TestCompile(t *testing.T) {
	if runtime.GOOS == "js" {
		t.Skip("file open might not be implemented in this environment")
	}

	files, err := os.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}

	type testcase struct {
		Name  string
		Src   []byte
		VS    []byte
		FS    []byte
		HLSL  []byte
		Metal []byte
	}

	fnames := map[string]struct{}{}
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		fnames[f.Name()] = struct{}{}
	}

	tests := []testcase{}
	for n := range fnames {
		if !strings.HasSuffix(n, ".go") {
			continue
		}

		src, err := os.ReadFile(filepath.Join("testdata", n))
		if err != nil {
			t.Fatal(err)
		}

		name := n[:len(n)-len(".go")]
		tc := testcase{
			Name: name,
			Src:  src,
		}

		vsn := name + ".expected.vs"
		if _, ok := fnames[vsn]; ok {
			vs, err := os.ReadFile(filepath.Join("testdata", vsn))
			if err != nil {
				t.Fatal(err)
			}
			tc.VS = vs
		}

		fsn := name + ".expected.fs"
		if _, ok := fnames[fsn]; ok {
			fs, err := os.ReadFile(filepath.Join("testdata", fsn))
			if err != nil {
				t.Fatal(err)
			}
			tc.FS = fs
		}

		if tc.VS == nil && tc.FS == nil {
			t.Fatalf("no expected file for %s", name)
		}

		hlsln := name + ".expected.hlsl"
		if _, ok := fnames[hlsln]; ok {
			hlsl, err := os.ReadFile(filepath.Join("testdata", hlsln))
			if err != nil {
				t.Fatal(err)
			}
			tc.HLSL = hlsl
		}

		metaln := name + ".expected.metal"
		if _, ok := fnames[metaln]; ok {
			metal, err := os.ReadFile(filepath.Join("testdata", metaln))
			if err != nil {
				t.Fatal(err)
			}
			tc.Metal = metal
		}

		tests = append(tests, tc)
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			s, err := shader.Compile(tc.Src, "Vertex", "Fragment", 0)
			if err != nil {
				t.Error(err)
				return
			}

			// GLSL
			vs, fs := glsl.Compile(s, glsl.GLSLVersionDefault)
			if got, want := glslVertexNormalize(vs), glslVertexNormalize(string(tc.VS)); got != want {
				compare(t, "GLSL Vertex", got, want)
			}
			if tc.FS != nil {
				if got, want := glslFragmentNormalize(fs), glslFragmentNormalize(string(tc.FS)); got != want {
					compare(t, "GLSL Fragment", got, want)
				}
			}

			if tc.HLSL != nil {
				vs, _, vertexPrelude, _ := hlsl.Compile(s)
				if got, want := hlslNormalize(vs, vertexPrelude), hlslNormalize(string(tc.HLSL), vertexPrelude); got != want {
					compare(t, "HLSL", got, want)
				}
			}

			if tc.Metal != nil {
				m := msl.Compile(s)
				if got, want := metalNormalize(m), metalNormalize(string(tc.Metal)); got != want {
					compare(t, "Metal", got, want)
				}
			}

			// Just check that Compile doesn't cause panic.
			// TODO: Should the results be tested?
			msl.Compile(s)
		})
	}
}

func TestCompileAssignFromNoReturnValue(t *testing.T) {
	srcs := []string{
		`package main

func bar() {
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	x := bar()
	return vec4(1)
}`,
		`package main

func bar() {
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var x = bar()
	return vec4(1)
}`,
	}
	for _, src := range srcs {
		_, err := shader.Compile([]byte(src), "Vertex", "Fragment", 0)
		if err == nil {
			t.Errorf("Compile must return an error for a function call with no return values, but got nil")
		}
	}
}

func TestCompileHLSLIntModulo(t *testing.T) {
	src := []byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, src0Pos vec2, color vec4) vec4 {
	a := int(src0Pos.x)
	b := a % 3
	return vec4(float(b))
}`)
	s, err := graphics.CompileShader(src)
	if err != nil {
		t.Fatal(err)
	}
	_, ps, _, _ := hlsl.Compile(s)
	if !strings.Contains(ps, "modInt(") {
		t.Errorf("HLSL pixel shader should use modInt for integer modulo, but got:\n%s", ps)
	}
	// The PSMain body must not use the '%' operator directly. modInt's own definition
	// uses '%' on uints (which fxc does not warn about), so check only the body.
	i := strings.Index(ps, "PSMain")
	if i < 0 {
		t.Fatalf("HLSL pixel shader should have a PSMain function, but got:\n%s", ps)
	}
	if body := ps[i:]; strings.Contains(body, " % ") {
		t.Errorf("HLSL PSMain should not use the '%%' operator for integer modulo, but got:\n%s", body)
	}
}

func TestCompileVaryingTypeMismatchPosition(t *testing.T) {
	src := `package main

func Vertex(position vec4, texCoord vec2, color vec4) (vec4, vec2) {
	return position, texCoord
}

func Fragment(position vec4, texCoord vec3, color vec4) vec4 {
	return vec4(1)
}`
	_, err := shader.Compile([]byte(src), "Vertex", "Fragment", 0)
	if err == nil {
		t.Fatalf("Compile must return an error for a mismatched fragment argument, but got nil")
	}
	var perr *shader.ParseError
	if !errors.As(err, &perr) {
		t.Fatalf("Compile must return a *shader.ParseError, but got %v", err)
	}
	positions := perr.Positions()
	if len(positions) == 0 {
		t.Fatalf("Compile must report at least one error position, but got none")
	}
	// The errors must point at the fragment entry point, not at the beginning of the source.
	for _, p := range positions {
		if got, want := p.Line, 7; got != want {
			t.Errorf("the error line: got: %d, want: %d (%v)", got, want, err)
		}
	}
}

func TestCompileBlankAsValue(t *testing.T) {
	values := []string{
		"min(_, 1)",
		"1 + _",
		"min(-_, 1)",
		"max(+_, 1)",
		"clamp(-_, 0, 1)",
		"1 + (-_)",
	}
	for _, v := range values {
		src := []byte(fmt.Sprintf(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	const x = %s
	return vec4(x)
}`, v))
		t.Run(v, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Compile must not panic when _ is used as a value, but panicked: %v", r)
				}
			}()
			if _, err := shader.Compile(src, "Vertex", "Fragment", 0); err == nil {
				t.Fatalf("Compile must return an error when _ is used as a value, but got nil")
			}
		})
	}
}
func TestCompileHugeShift(t *testing.T) {
	src := `package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	x := 1 << (1 << 40)
	return vec4(x)
}`
	_, err := shader.Compile([]byte(src), "Vertex", "Fragment", 0)
	if err == nil {
		t.Errorf("Compile must return an error for a huge constant shift, but got nil")
	}
}

func TestCompileVarCountMismatch(t *testing.T) {
	srcs := []string{
		`package main

func f() (int, int) {
	return 1, 2
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var a, b, c = f()
	return vec4(a + b + c)
}`,
		`package main

func f() (int, int) {
	return 1, 2
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var a, b, c int = f()
	return vec4(a + b + c)
}`,
		`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var a, b = 1
	return vec4(a + b)
}`,
		`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var a, b int = 1
	return vec4(a + b)
}`,
	}
	for _, src := range srcs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Compile must not panic for a var count mismatch, but panicked: %v", r)
				}
			}()
			if _, err := shader.Compile([]byte(src), "Vertex", "Fragment", 0); err == nil {
				t.Errorf("Compile must return an error for a var count mismatch, but got nil")
			}
		}()
	}
}

func TestCompileBoolArithmetic(t *testing.T) {
	srcs := []string{
		`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(float(true + true))
}`,
		`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(float(true - true))
}`,
		`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(float(true * true))
}`,
		`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(float(true / true))
}`,
		`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a := true
	b := true
	c := a + b
	if c {
		return vec4(1)
	}
	return vec4(0)
}`,
	}
	for _, src := range srcs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Compile must not panic for bool arithmetic, but panicked: %v", r)
				}
			}()
			if _, err := shader.Compile([]byte(src), "Vertex", "Fragment", 0); err == nil {
				t.Errorf("Compile must return an error for bool arithmetic, but got nil")
			}
		}()
	}
}
