// Copyright 2024 The Ebitengine Authors
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
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/packages"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/glsl"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/hlsl"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/msl"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/pssl"
)

var flagTarget = flag.String("target", "", "shader compilation targets separated by comma (e.g. 'glsl,glsles,hlsl,msl')")

func main() {
	if err := xmain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type Shader struct {
	Package    string
	File       string
	Source     string
	SourceHash string
	GLSL       *GLSL   `json:",omitempty"`
	GLSLES     *GLSLES `json:",omitempty"`
	HLSL       *HLSL   `json:",omitempty"`
	MSL        *MSL    `json:",omitempty"`
	PSSL       *PSSL   `json:",omitempty"`
}

type GLSL struct {
	Vertex   string
	Fragment string
}

type GLSLES struct {
	Vertex   string
	Fragment string
}

type HLSL struct {
	Vertex string
	Pixel  string
}

type MSL struct {
	Shader string
}

type PSSL struct {
	Vertex string
	Pixel  string
}

func xmain() error {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "shaderlister [-target=TARGET] [package]")
		os.Exit(2)
	}
	flag.Parse()
	if len(flag.Args()) < 1 {
		flag.Usage()
	}

	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
	}, flag.Args()...)
	if err != nil {
		return err
	}

	var targets []string
	if ft := strings.TrimSpace(*flagTarget); ft != "" {
		for _, t := range strings.Split(ft, ",") {
			targets = append(targets, strings.TrimSpace(t))
		}
	}

	// Collect shader information.
	// Even if no shader is found, the output should be a JSON array. Start with an empty slice, not nil.
	shaders := []Shader{}

	var visitErr error
	packages.Visit(pkgs, func(pkg *packages.Package) bool {
		path := pkg.PkgPath
		// A standard library should not have a directive for shaders. Skip them.
		if isStandardImportPath(path) {
			return true
		}
		// A semi-standard library should not have a directive for shaders. Skip them.
		if strings.HasPrefix(path, "golang.org/x/") {
			return true
		}

		origN := len(shaders)
		shaders = appendShaderSources(shaders, pkg)

		// Add source hashes.
		for i := range shaders[origN:] {
			shader := &shaders[i]
			hash, err := graphics.CalcSourceHash([]byte(shader.Source))
			if err != nil {
				visitErr = err
				return false
			}
			shader.SourceHash = hash.String()
		}

		// Compile shaders.
		if len(targets) == 0 {
			return true
		}
		for i := range shaders[origN:] {
			if err := compile(&shaders[i], targets); err != nil {
				visitErr = err
				return false
			}
		}

		return true
	}, nil)
	if visitErr != nil {
		return visitErr
	}

	w := bufio.NewWriter(os.Stdout)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")
	if err := enc.Encode(shaders); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}

	return nil
}

// isStandardImportPath reports whether $GOROOT/src/path should be considered part of the standard distribution.
//
// This is based on the implementation in the standard library (cmd/go/internal/search/search.go).
func isStandardImportPath(path string) bool {
	head, _, _ := strings.Cut(path, "/")
	return !strings.Contains(head, ".")
}

const directive = "ebitengine:shader"

var reDirective = regexp.MustCompile(`(?m)^\s*//` + regexp.QuoteMeta(directive))

func hasShaderDirectiveInComment(commentGroup *ast.CommentGroup) bool {
	for _, line := range commentGroup.List {
		if reDirective.MatchString(line.Text) {
			return true
		}
	}
	return false
}

func appendShaderSources(shaders []Shader, pkg *packages.Package) []Shader {
	topLevelDecls := map[ast.Decl]struct{}{}
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			topLevelDecls[decl] = struct{}{}
		}
	}
	isTopLevelDecl := func(decl ast.Decl) bool {
		_, ok := topLevelDecls[decl]
		return ok
	}

	var genDeclStack []*ast.GenDecl

	in := inspector.New(pkg.Syntax)
	in.Nodes([]ast.Node{
		(*ast.GenDecl)(nil),
		(*ast.ValueSpec)(nil),
	}, func(n ast.Node, push bool) bool {
		switch n := n.(type) {
		case *ast.GenDecl:
			genDecl := n

			// It is possible to check whether decl.Tok is token.CONST or not,
			// but move on without checking it for better warning messages.

			if push {
				// If the GenDecl is with parentheses (e.g. `const ( ... )`), check the GenDecl's comment.
				// The directive doesn't work, so if the directive is found, warn it.
				if genDecl.Lparen != token.NoPos {
					if genDecl.Doc != nil && hasShaderDirectiveInComment(genDecl.Doc) {
						pos := pkg.Fset.Position(genDecl.Doc.Pos())
						slog.Warn(fmt.Sprintf("misplaced %s directive", directive),
							"package", pkg.PkgPath,
							"file", pos.Filename,
							"line", pos.Line,
							"column", pos.Column)
					}
				} else {
					if genDecl.Doc == nil {
						return false
					}
					if !hasShaderDirectiveInComment(genDecl.Doc) {
						return false
					}
				}

				// It is possible to check whether genCecl is top-level or not,
				// but move on without checking it for better warning messages.

				genDeclStack = append(genDeclStack, genDecl)
			} else {
				genDeclStack = genDeclStack[:len(genDeclStack)-1]
			}
			return true

		case *ast.ValueSpec:
			spec := n

			genDecl := genDeclStack[len(genDeclStack)-1]

			// If the ValueSpec is in parentheses (e.g. `const ( ... )`), check the ValueSpec's comment.
			if genDecl.Lparen != token.NoPos {
				if spec.Doc == nil {
					return false
				}
				if !hasShaderDirectiveInComment(spec.Doc) {
					return false
				}
			}

			var docPos token.Pos
			if spec.Doc != nil {
				docPos = spec.Doc.Pos()
			} else {
				docPos = genDecl.Doc.Pos()
			}

			if !isTopLevelDecl(genDecl) {
				pos := pkg.Fset.Position(docPos)
				slog.Warn(fmt.Sprintf("misplaced %s directive", directive),
					"package", pkg.PkgPath,
					"file", pos.Filename,
					"line", pos.Line,
					"column", pos.Column)
				return false
			}

			// Avoid multiple names like `const a, b = "foo", "bar"` to avoid confusions.
			if len(spec.Names) != 1 {
				pos := pkg.Fset.Position(docPos)
				slog.Warn(fmt.Sprintf("%s cannot apply to multiple declarations", directive),
					"package", pkg.PkgPath,
					"file", pos.Filename,
					"line", pos.Line,
					"column", pos.Column)
				return false
			}

			// Check if the ValueSpec is a const declaration.
			name := spec.Names[0]
			def := pkg.TypesInfo.Defs[name]
			c, ok := def.(*types.Const)
			if !ok {
				pos := pkg.Fset.Position(docPos)
				slog.Warn(fmt.Sprintf("%s cannot apply to %s", directive, objectTypeString(def)),
					"package", pkg.PkgPath,
					"file", pos.Filename,
					"line", pos.Line,
					"column", pos.Column)
				return false
			}

			// Check the constant type.
			val := c.Val()
			if val.Kind() != constant.String {
				pos := pkg.Fset.Position(docPos)
				slog.Warn(fmt.Sprintf("%s cannot apply to const type of %s", directive, val.Kind()),
					"package", pkg.PkgPath,
					"file", pos.Filename,
					"line", pos.Line,
					"column", pos.Column)
				return false
			}

			shaders = append(shaders, Shader{
				Package: pkg.PkgPath,
				File:    pkg.Fset.Position(spec.Pos()).Filename,
				Source:  constant.StringVal(val),
			})
			return false

		default:
			return false
		}
	})

	return shaders
}

func objectTypeString(obj types.Object) string {
	switch obj := obj.(type) {
	case *types.PkgName:
		return "package"
	case *types.Const:
		return "const"
	case *types.TypeName:
		return "type"
	case *types.Var:
		if obj.IsField() {
			return "field"
		}
		return "var"
	case *types.Func:
		return "func"
	case *types.Label:
		return "label"
	case *types.Builtin:
		return "builtin"
	case *types.Nil:
		return "nil"
	default:
		return fmt.Sprintf("objectTypeString(%T)", obj)
	}
}

func compile(shader *Shader, targets []string) error {
	ir, err := graphics.CompileShader([]byte(shader.Source))
	if err != nil {
		return fmt.Errorf("compiling shader failed: %w", err)
	}

	for _, target := range targets {
		switch target {
		case "glsl":
			vs, fs := glsl.Compile(ir, glsl.GLSLVersionDefault)
			shader.GLSL = &GLSL{
				Vertex:   vs,
				Fragment: fs,
			}
		case "glsles":
			vs, fs := glsl.Compile(ir, glsl.GLSLVersionES300)
			shader.GLSLES = &GLSLES{
				Vertex:   vs,
				Fragment: fs,
			}
		case "hlsl":
			vs, ps, _ := hlsl.Compile(ir)
			shader.HLSL = &HLSL{
				Vertex: vs,
				Pixel:  ps,
			}
		case "msl":
			s := msl.Compile(ir)
			shader.MSL = &MSL{
				Shader: s,
			}
		case "pssl":
			vs, ps := pssl.Compile(ir)
			shader.PSSL = &PSSL{
				Vertex: vs,
				Pixel:  ps,
			}
		default:
			return fmt.Errorf("unsupported target: %s", target)
		}
	}

	return nil
}
