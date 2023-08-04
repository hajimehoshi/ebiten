// Copyright 2023 The Ebitengine Authors
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

//go:build ignore

package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl/gl"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	f, err := os.Create("debug.go")
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.WriteString(`// Copyright 2023 The Ebitengine Authors
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

package gl

import (
	"fmt"
	"os"
)

type DebugContext struct {
	Context Context
}

var _ Context = (*DebugContext)(nil)
`); err != nil {
		return err
	}

	t := reflect.TypeOf((*gl.Context)(nil)).Elem()
	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		name := m.Name

		var argNames []string
		var argNamesAndTypes []string
		for j := 0; j < m.Type.NumIn(); j++ {
			n := fmt.Sprintf("arg%d", j)
			argNames = append(argNames, n)
			argNamesAndTypes = append(argNamesAndTypes, n+" "+typeName(m.Type.In(j)))
		}

		var outTypes []string
		var outNames []string
		for j := 0; j < m.Type.NumOut(); j++ {
			outTypes = append(outTypes, typeName(m.Type.Out(j)))
			outNames = append(outNames, fmt.Sprintf("out%d", j))
		}

		if _, err := fmt.Fprintf(f, "\nfunc (d *DebugContext) %s(%s) (%s) {\n", name, strings.Join(argNamesAndTypes, ", "), strings.Join(outTypes, ",")); err != nil {
			return err
		}
		if len(outTypes) > 0 {
			if _, err := fmt.Fprintf(f, "\t%s := ", strings.Join(outNames, ", ")); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(f, "\t"); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(f, "d.Context.%s(%s)\n", name, strings.Join(argNames, ", ")); err != nil {
			return err
		}

		// Print logs.
		if name != "LoadFunctions" && name != "IsES" {
			if _, err := fmt.Fprintf(f, "\tfmt.Fprintln(os.Stderr, %q)\n", name); err != nil {
				return err
			}
		}

		// Check errors.
		if name != "LoadFunctions" && name != "IsES" && name != "GetError" {
			if _, err := fmt.Fprintf(f, `	if e := d.Context.GetError(); e != NO_ERROR {
		panic(fmt.Sprintf("gl: GetError() returned %%d at %s", e))
	}
`, name); err != nil {
				return err
			}
		}

		if len(outTypes) > 0 {
			if _, err := fmt.Fprintf(f, "\treturn %s\n", strings.Join(outNames, ", ")); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(f, "}\n"); err != nil {
			return err
		}
	}

	return nil
}

func typeName(t reflect.Type) string {
	if t.Kind() == reflect.Slice {
		return "[]" + typeName(t.Elem())
	}
	return t.Name()
}
