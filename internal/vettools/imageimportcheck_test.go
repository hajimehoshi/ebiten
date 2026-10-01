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

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"golang.org/x/tools/go/analysis"
	"testing"
)

func TestImageImportCheckPosition(t *testing.T) {
	fset := token.NewFileSet()
	// A preceding file makes the token position differ from a file offset.
	if _, err := parser.ParseFile(fset, "first.go", "package p\nvar padding = 1\n", 0); err != nil {
		t.Fatal(err)
	}
	f, err := parser.ParseFile(fset, "second.go", "package p\n\nimport _ \"image/png\"\n", 0)
	if err != nil {
		t.Fatal(err)
	}
	var diagnostics []analysis.Diagnostic
	pass := &analysis.Pass{
		Fset:   fset,
		Files:  []*ast.File{f},
		Pkg:    types.NewPackage("example.com/p", "p"),
		Report: func(d analysis.Diagnostic) { diagnostics = append(diagnostics, d) },
	}
	if _, err := runImageImportCheck(pass); err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("got %d diagnostics, want 1", len(diagnostics))
	}
	got := fset.Position(diagnostics[0].Pos)
	if got.Filename != "second.go" || got.Line != 3 || got.Column != 8 {
		t.Errorf("got %v, want second.go:3:8", got)
	}
}
