// Copyright 2018 The Ebiten Authors
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
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	exec "golang.org/x/sys/execabs"
	"golang.org/x/tools/go/ast/astutil"
)

func pngDir() (string, error) {
	dir, err := exec.Command("go", "list", "-f", "{{.Dir}}", "image/png").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(dir)), nil
}

func pngFiles() ([]string, error) {
	files, err := exec.Command("go", "list", "-f", `{{join .GoFiles ","}}`, "image/png").Output()
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSpace(string(files)), ","), nil
}

func run() error {
	reVer := regexp.MustCompile(`^go1\.(\d+)(\.\d+)?$`)
	verStr := runtime.Version()
	m := reVer.FindStringSubmatch(verStr)
	if m == nil {
		return fmt.Errorf("png: unexpected Go version: %s", verStr)
	}
	ver, err := strconv.Atoi(m[1])
	if err != nil {
		return err
	}
	if ver < 22 {
		return errors.New("png: use Go 1.22 or newer")
	}

	dir, err := pngDir()
	if err != nil {
		return err
	}

	files, err := pngFiles()
	if err != nil {
		return err
	}

	const prefix = "stdlib"

	matches, err := filepath.Glob(prefix + "*.go")
	if err != nil {
		return err
	}
	for _, f := range matches {
		if err := os.Remove(f); err != nil {
			return err
		}
	}

	for _, f := range files {
		out, err := os.Create(prefix + f)
		if err != nil {
			return err
		}
		defer out.Close()

		// TODO: Remove call of RegisterDecoder

		data, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			return err
		}
		fset := token.NewFileSet()
		tree, err := parser.ParseFile(fset, "", string(data), parser.ParseComments)
		if err != nil {
			return err
		}

		astutil.Apply(tree, func(c *astutil.Cursor) bool {
			stmt, ok := c.Node().(*ast.ExprStmt)
			if !ok {
				return true
			}
			call, ok := stmt.X.(*ast.CallExpr)
			if !ok {
				return true
			}
			s, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			receiver, ok := s.X.(*ast.Ident)
			if !ok {
				return true
			}
			// Delete registering PNG format.
			if receiver.Name == "image" && s.Sel.Name == "RegisterFormat" {
				c.Delete()
			}
			return true
		}, nil)

		fmt.Fprintln(out, "// Code generated by gen.go. DO NOT EDIT.")
		fmt.Fprintln(out)
		format.Node(out, fset, tree)

		if f == "reader.go" {
			// The min function was removed as of Go 1.22, but this is needed for old Go.
			// TODO: Remove this when Go 1.21 is the minimum supported version.
			fmt.Fprintln(out, `
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}`)
		}
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}
