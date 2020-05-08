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

package shader

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"sort"
	"strings"
)

const (
	varyingStructName = "VertexOut"
)

var (
	kageTagRe = regexp.MustCompile("^`" + `kage:\"(.+)\"` + "`$")
)

type nameAndType struct {
	name string
	typ  typ
}

type Shader struct {
	// position is the field name of VertexOut that represents a vertex position (gl_Position in GLSL).
	position nameAndType

	// varyings is a set of varying variables.
	varyings []nameAndType

	errs []string
}

type ParseError struct {
	errs []string
}

func (p *ParseError) Error() string {
	return strings.Join(p.errs, "\n")
}

func NewShader(src []byte) (*Shader, error) {
	f, err := parser.ParseFile(token.NewFileSet(), "", src, parser.AllErrors)
	if err != nil {
		return nil, err
	}

	s := &Shader{}
	s.parse(f)

	if len(s.errs) > 0 {
		return nil, &ParseError{s.errs}
	}

	// TODO: Make a call graph and reorder the elements.
	return s, nil
}

func (s *Shader) parse(f *ast.File) {
	// TODO: Accumulate errors
	for name, obj := range f.Scope.Objects {
		switch name {
		case varyingStructName:
			s.parseVaryingStruct(obj)
		}
	}
}

func (sh *Shader) parseVaryingStruct(obj *ast.Object) {
	name := obj.Name
	if obj.Kind != ast.Typ {
		sh.errs = append(sh.errs, fmt.Sprintf("%s must be a type but %s", name, obj.Kind))
		return
	}
	t := obj.Decl.(*ast.TypeSpec).Type
	s, ok := t.(*ast.StructType)
	if !ok {
		sh.errs = append(sh.errs, fmt.Sprintf("%s must be a struct but not", name))
		return
	}

	for _, f := range s.Fields.List {
		if f.Tag != nil {
			tag := f.Tag.Value
			m := kageTagRe.FindStringSubmatch(tag)
			if m == nil {
				sh.errs = append(sh.errs, fmt.Sprintf("invalid struct tag: %s", tag))
				continue
			}
			if m[1] != "position" {
				sh.errs = append(sh.errs, fmt.Sprintf("struct tag value must be position in %s but %s", varyingStructName, m[1]))
				continue
			}
			if len(f.Names) != 1 {
				sh.errs = append(sh.errs, fmt.Sprintf("position members must be one"))
				continue
			}
			t, err := parseType(f.Type)
			if err != nil {
				sh.errs = append(sh.errs, err.Error())
				continue
			}
			if t != typVec4 {
				sh.errs = append(sh.errs, fmt.Sprintf("position must be vec4 but %s", t))
				continue
			}
			sh.position = nameAndType{
				name: f.Names[0].Name,
				typ:  t,
			}
			continue
		}
		t, err := parseType(f.Type)
		if err != nil {
			sh.errs = append(sh.errs, err.Error())
			continue
		}
		if !t.numeric() {
			sh.errs = append(sh.errs, fmt.Sprintf("members in %s must be numeric but %s", varyingStructName, t))
			continue
		}
		for _, n := range f.Names {
			sh.varyings = append(sh.varyings, nameAndType{
				name: n.Name,
				typ:  t,
			})
		}
	}
	sort.Slice(sh.varyings, func(a, b int) bool {
		return sh.varyings[a].name < sh.varyings[b].name
	})
}

// Dump dumps the shader state in an intermediate language.
func (s *Shader) Dump() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("var %s varying %s // position", s.position.name, s.position.typ))
	for _, v := range s.varyings {
		lines = append(lines, fmt.Sprintf("var %s varying %s", v.name, v.typ))
	}
	return strings.Join(lines, "\n") + "\n"
}

func (s *Shader) GlslVertex() string {
	var lines []string

	for _, v := range s.varyings {
		// TODO: variable names must be escaped not to conflict with keywords.
		lines = append(lines, fmt.Sprintf("varying %s %s;", v.typ.glslString(), v.name))
	}
	return strings.Join(lines, "\n") + "\n"
}
