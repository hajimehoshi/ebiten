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
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
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
	GoFile     string `json:",omitempty"`
	KageFile   string `json:",omitempty"`
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
		shaders, err = appendShaderSources(shaders, pkg)
		if err != nil {
			visitErr = err
			return false
		}
		newShaders := shaders[origN:]

		// Add source hashes.
		for i := range newShaders {
			shader := &newShaders[i]
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
		for i := range newShaders {
			if err := compile(&newShaders[i], targets); err != nil {
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

const (
	shaderSourceDirective = "ebitengine:shadersource"
	shaderFileDirective   = "ebitengine:shaderfile"
)

var (
	reShaderSourceDirective = regexp.MustCompile(`^\s*//` + regexp.QuoteMeta(shaderSourceDirective) + `$`)
	reShaderFileDirective   = regexp.MustCompile(`^\s*//` + regexp.QuoteMeta(shaderFileDirective) + ` `)
)

func hasShaderSourceDirectiveInComment(commentGroup *ast.CommentGroup) bool {
	for _, c := range commentGroup.List {
		for _, line := range strings.Split(c.Text, "\n") {
			if reShaderSourceDirective.MatchString(line) {
				return true
			}
		}
	}
	return false
}

func isAsciiSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\v' || r == '\n' || r == '\r'
}

func includesGlobMetaChar(str string) bool {
	// '-' and '^' are meta characters only when these are in brackets.
	// So, these don't need to be checked.
	return strings.ContainsAny(str, "*?[]")
}

func appendShaderSources(shaders []Shader, pkg *packages.Package) ([]Shader, error) {
	// Resolve ebitengine:shaderfile directives.
	visitedPatterns := map[string]struct{}{}
	visitedPaths := map[string]struct{}{}
	for _, f := range pkg.Syntax {
		var funcs []*ast.FuncDecl
		for _, decl := range f.Decls {
			if f, ok := decl.(*ast.FuncDecl); ok {
				funcs = append(funcs, f)
			}
		}
		for _, cg := range f.Comments {
			for _, c := range cg.List {
				// Ignore the line if it is in a function declaration.
				if slices.ContainsFunc(funcs, func(f *ast.FuncDecl) bool {
					return f.Pos() <= c.Pos() && c.Pos() < f.End()
				}) {
					continue
				}

				for _, line := range strings.Split(c.Text, "\n") {
					m := reShaderFileDirective.FindString(line)
					if len(m) == 0 {
						continue
					}
					patterns := strings.TrimPrefix(line, m)
					for _, pattern := range strings.FieldsFunc(patterns, isAsciiSpace) {
						pattern := filepath.Join(pkg.Dir, filepath.FromSlash(pattern))
						if _, ok := visitedPatterns[pattern]; ok {
							continue
						}
						visitedPatterns[pattern] = struct{}{}
						if !includesGlobMetaChar(pattern) {
							stat, err := os.Stat(pattern)
							if err == nil && stat.IsDir() {
								// If the pattern is a directory, read all files in the directory recursively.
								if err := filepath.WalkDir(pattern, func(path string, d os.DirEntry, err error) error {
									if err != nil {
										return err
									}
									if d.IsDir() {
										return nil
									}
									if _, ok := visitedPaths[path]; ok {
										return nil
									}
									visitedPaths[path] = struct{}{}
									goFile := pkg.Fset.Position(cg.Pos()).Filename
									shaders, err = appendShaderFromFile(shaders, pkg.PkgPath, goFile, path)
									if err != nil {
										return err
									}
									return nil
								}); err != nil {
									return nil, err
								}
								continue
							}
							if err != nil && !errors.Is(err, os.ErrNotExist) {
								return nil, err
							}
						}
						paths, err := filepath.Glob(pattern)
						if err != nil {
							return nil, err
						}
						for _, path := range paths {
							if _, ok := visitedPaths[path]; ok {
								continue
							}
							visitedPaths[path] = struct{}{}
							goFile := pkg.Fset.Position(cg.Pos()).Filename
							shaders, err = appendShaderFromFile(shaders, pkg.PkgPath, goFile, path)
							if err != nil {
								return nil, err
							}
						}
					}
				}
			}
		}
	}

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

	// Resolve ebitengine:shadersource directives.
	var genDeclStack []*ast.GenDecl
	// inspector.Inspector doesn't iterate comments that are not attached to any other nodes.
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
					if genDecl.Doc != nil && hasShaderSourceDirectiveInComment(genDecl.Doc) {
						pos := pkg.Fset.Position(genDecl.Doc.Pos())
						slog.Warn(fmt.Sprintf("misplaced %s directive", shaderSourceDirective),
							"package", pkg.PkgPath,
							"file", pos.Filename,
							"line", pos.Line,
							"column", pos.Column)
					}
				} else {
					if genDecl.Doc == nil {
						return false
					}
					if !hasShaderSourceDirectiveInComment(genDecl.Doc) {
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
				if !hasShaderSourceDirectiveInComment(spec.Doc) {
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
				slog.Warn(fmt.Sprintf("misplaced %s directive", shaderSourceDirective),
					"package", pkg.PkgPath,
					"file", pos.Filename,
					"line", pos.Line,
					"column", pos.Column)
				return false
			}

			// Avoid multiple names like `const a, b = "foo", "bar"` to avoid confusions.
			if len(spec.Names) != 1 {
				pos := pkg.Fset.Position(docPos)
				slog.Warn(fmt.Sprintf("%s cannot apply to multiple declarations", shaderSourceDirective),
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
				slog.Warn(fmt.Sprintf("%s cannot apply to %s", shaderSourceDirective, objectTypeString(def)),
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
				slog.Warn(fmt.Sprintf("%s cannot apply to const type of %s", shaderSourceDirective, val.Kind()),
					"package", pkg.PkgPath,
					"file", pos.Filename,
					"line", pos.Line,
					"column", pos.Column)
				return false
			}

			shaders = append(shaders, Shader{
				Package: pkg.PkgPath,
				GoFile:  pkg.Fset.Position(spec.Pos()).Filename,
				Source:  constant.StringVal(val),
			})
			return false

		default:
			return false
		}
	})

	return shaders, nil
}

func appendShaderFromFile(shaders []Shader, pkgPath string, goFile string, kageFile string) ([]Shader, error) {
	content, err := os.ReadFile(kageFile)
	if err != nil {
		return nil, err
	}
	shaders = append(shaders, Shader{
		Package:  pkgPath,
		GoFile:   goFile,
		KageFile: kageFile,
		Source:   string(content),
	})
	return shaders, nil
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
