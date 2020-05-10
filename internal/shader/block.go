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
	"go/token"
	"strings"
)

type block struct {
	vars   []variable
	consts []constant
	funcs  []function
	stmts  []stmt
	pos    token.Pos
	outer  *block
}

func (b *block) dump(indent int) []string {
	idt := strings.Repeat("\t", indent)

	var lines []string

	for _, v := range b.vars {
		init := ""
		if v.init != nil {
			init = " = " + dumpExpr(v.init)
		}
		lines = append(lines, fmt.Sprintf("%svar %s %s%s", idt, v.name, v.typ, init))
	}
	for _, c := range b.consts {
		lines = append(lines, fmt.Sprintf("%sconst %s %s = %s", idt, c.name, c.typ, dumpExpr(c.init)))
	}
	for _, f := range b.funcs {
		var args []string
		for _, a := range f.args {
			args = append(args, fmt.Sprintf("%s %s", a.name, a.typ))
		}
		var rets []string
		for _, r := range f.rets {
			name := r.name
			if name == "" {
				name = "_"
			}
			rets = append(rets, fmt.Sprintf("%s %s", name, r.typ))
		}
		l := fmt.Sprintf("func %s(%s)", f.name, strings.Join(args, ", "))
		if len(rets) > 0 {
			l += " (" + strings.Join(rets, ", ") + ")"
		}
		l += " {"
		lines = append(lines, l)
		lines = append(lines, f.body.dump(indent+1)...)
		lines = append(lines, "}")
	}

	for _, s := range b.stmts {
		lines = append(lines, s.dump(indent)...)
	}

	return lines
}

type stmtType int

const (
	stmtNone stmtType = iota
	stmtAssign
	stmtReturn
)

type stmt struct {
	stmtType stmtType
	exprs    []ast.Expr
}

func (s *stmt) dump(indent int) []string {
	idt := strings.Repeat("\t", indent)

	var lines []string
	switch s.stmtType {
	case stmtNone:
		lines = append(lines, "%s(none)", idt)
	case stmtAssign:
		lines = append(lines, fmt.Sprintf("%s%s = %s", idt, dumpExpr(s.exprs[0]), dumpExpr(s.exprs[1])))
	case stmtReturn:
		var expr string
		if len(s.exprs) > 0 {
			var strs []string
			for _, e := range s.exprs {
				strs = append(strs, dumpExpr(e))
			}
			expr = " " + strings.Join(strs, ", ")
		}
		lines = append(lines, fmt.Sprintf("%sreturn%s", idt, expr))
	default:
		lines = append(lines, fmt.Sprintf("%s(unknown stmt: %d)", idt, s.stmtType))
	}

	return lines
}

func dumpExpr(e ast.Expr) string {
	switch e := e.(type) {
	case *ast.BasicLit:
		return e.Value
	case *ast.CompositeLit:
		t := parseType(e.Type)
		var vals []string
		for _, e := range e.Elts {
			vals = append(vals, dumpExpr(e))
		}
		return fmt.Sprintf("%s{%s}", t, strings.Join(vals, ", "))
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return fmt.Sprintf("%s.%s", dumpExpr(e.X), dumpExpr(e.Sel))
	default:
		return fmt.Sprintf("(unkown expr: %#v)", e)
	}
}
