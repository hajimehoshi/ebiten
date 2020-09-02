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
	"go/parser"
	"go/token"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	. "github.com/hajimehoshi/ebiten/internal/shader"
	"github.com/hajimehoshi/ebiten/internal/shaderir/glsl"
	"github.com/hajimehoshi/ebiten/internal/shaderir/metal"
)

func normalize(str string) string {
	if strings.HasPrefix(str, glsl.FragmentPrelude) {
		str = str[len(glsl.FragmentPrelude):]
	}
	return strings.TrimSpace(str)
}

func TestCompile(t *testing.T) {
	if runtime.GOOS == "js" {
		t.Skip("file open might not be implemented in this environment")
	}

	files, err := ioutil.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}

	type testcase struct {
		Name string
		Src  []byte
		VS   []byte
		FS   []byte
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

		src, err := ioutil.ReadFile(filepath.Join("testdata", n))
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
			vs, err := ioutil.ReadFile(filepath.Join("testdata", vsn))
			if err != nil {
				t.Fatal(err)
			}
			tc.VS = vs
		}

		fsn := name + ".expected.fs"
		if _, ok := fnames[fsn]; ok {
			fs, err := ioutil.ReadFile(filepath.Join("testdata", fsn))
			if err != nil {
				t.Fatal(err)
			}
			tc.FS = fs
		}

		if tc.VS == nil && tc.FS == nil {
			t.Fatalf("no expected file for %s", name)
		}

		tests = append(tests, tc)
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, "", tc.Src, parser.AllErrors)
			if err != nil {
				t.Fatal(err)
			}

			s, err := Compile(fset, f, "Vertex", "Fragment", 0)
			if err != nil {
				t.Error(err)
				return
			}
			vs, fs := glsl.Compile(s)
			if got, want := normalize(vs), normalize(string(tc.VS)); got != want {
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
				t.Errorf("got: %v, want: %v\n\n%s", got, want, msg)
			}
			if tc.FS != nil {
				if got, want := normalize(fs), normalize(string(tc.FS)); got != want {
					t.Errorf("got: %v, want: %v", got, want)
				}
			}

			// Just check that Compile doesn't cause panic.
			// TODO: Should the results be tested?
			metal.Compile(s, "Vertex", "Fragmentp")
		})
	}
}
