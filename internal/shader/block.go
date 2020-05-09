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
	"go/token"
	"strings"
)

type block struct {
	vars  []variable
	stmts []stmt
	pos   token.Pos
}

func (b *block) dump(indent int) []string {
	idt := strings.Repeat("\t", indent)

	var lines []string

	for _, v := range b.vars {
		lines = append(lines, fmt.Sprintf("%svar %s %s", idt, v.name, v.typ))
	}

	for _, s := range b.stmts {
		lines = append(lines, s.dump(indent)...)
	}

	return lines
}

type stmtType int

const (
	stmtNone stmtType = iota
	stmtReturn
)

type stmt struct {
	stmtType stmtType
	exprs    []expr
}

func (s *stmt) dump(indent int) []string {
	idt := strings.Repeat("\t", indent)

	var lines []string
	switch s.stmtType {
	case stmtNone:
		lines = append(lines, "%s(none)", idt)
	case stmtReturn:
		var expr string
		if len(s.exprs) > 0 {
			var strs []string
			for _, e := range s.exprs {
				strs = append(strs, e.dump())
			}
			expr = " " + strings.Join(strs, ", ")
		}
		lines = append(lines, fmt.Sprintf("%sreturn%s", idt, expr))
	default:
		lines = append(lines, fmt.Sprintf("%s(unknown stmt: %d)", idt, s.stmtType))
	}

	return lines
}

type exprType int

const (
	exprNone exprType = iota
	exprIdent
)

type expr struct {
	exprType exprType
	value    string
}

func (e *expr) dump() string {
	switch e.exprType {
	case exprNone:
		return "(none)"
	case exprIdent:
		return e.value
	default:
		return fmt.Sprintf("(unkown expr: %d)", e.exprType)
	}
}
