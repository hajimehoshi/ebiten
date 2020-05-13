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

package shaderir

import (
	"fmt"
	"strings"
)

func (p *Program) structName(t *Type) string {
	if t.Main != Struct {
		panic("shaderir: the given type at structName must be a struct")
	}
	s := t.serialize()
	if n, ok := p.structNames[s]; ok {
		return n
	}
	n := fmt.Sprintf("S%d", len(p.structNames))
	p.structNames[s] = n
	p.structTypes = append(p.structTypes, *t)
	return n
}

func (p *Program) Glsl() string {
	p.structNames = map[string]string{}
	p.structTypes = nil

	var lines []string
	for _, u := range p.Uniforms {
		lines = append(lines, fmt.Sprintf("uniform %s;", p.glslVarDecl(&u.Type, u.Name)))
	}
	for _, a := range p.Attributes {
		lines = append(lines, fmt.Sprintf("attribute %s;", p.glslVarDecl(&a.Type, a.Name)))
	}
	for _, v := range p.Varyings {
		lines = append(lines, fmt.Sprintf("varying %s;", p.glslVarDecl(&v.Type, v.Name)))
	}
	for _, f := range p.Funcs {
		lines = append(lines, p.glslFunc(&f)...)
	}

	var stLines []string
	for i, t := range p.structTypes {
		stLines = append(stLines, fmt.Sprintf("struct S%d {", i))
		for j, st := range t.Sub {
			stLines = append(stLines, fmt.Sprintf("\t%s;", p.glslVarDecl(&st, fmt.Sprintf("M%d", j))))
		}
		stLines = append(stLines, "};")
	}
	lines = append(stLines, lines...)

	return strings.Join(lines, "\n") + "\n"
}

func (p *Program) glslVarDecl(t *Type, varname string) string {
	switch t.Main {
	case None:
		return "?(none)"
	case Image2D:
		panic("not implemented")
	case Array:
		panic("not implemented")
	case Struct:
		return fmt.Sprintf("%s %s", p.structName(t), varname)
	default:
		return fmt.Sprintf("%s %s", t.Main.Glsl(), varname)
	}
}

func (p *Program) glslFunc(f *Func) []string {
	var args []string
	var idx int
	for _, t := range f.InParams {
		args = append(args, "in "+p.glslVarDecl(&t, fmt.Sprintf("l%d", idx)))
		idx++
	}
	for _, t := range f.InOutParams {
		args = append(args, "inout "+p.glslVarDecl(&t, fmt.Sprintf("l%d", idx)))
		idx++
	}
	for _, t := range f.OutParams {
		args = append(args, "out "+p.glslVarDecl(&t, fmt.Sprintf("l%d", idx)))
		idx++
	}
	argsstr := "void"
	if len(args) > 0 {
		argsstr = strings.Join(args, ", ")
	}

	return []string{
		fmt.Sprintf(`void %s(%s) {`, f.Name, argsstr),
		`}`,
	}
}
