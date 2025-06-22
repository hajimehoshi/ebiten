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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/shader"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
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
	prelude := msl.Prelude(shaderir.Texels)
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
