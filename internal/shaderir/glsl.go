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

func isValidSwizzling(s string) bool {
	if len(s) < 1 || 4 < len(s) {
		return false
	}

	const (
		xyzw = "xyzw"
		rgba = "rgba"
		strq = "strq"
	)

	switch {
	case strings.IndexByte(xyzw, s[0]) >= 0:
		for _, c := range s {
			if strings.IndexRune(xyzw, c) == -1 {
				return false
			}
		}
		return true
	case strings.IndexByte(rgba, s[0]) >= 0:
		for _, c := range s {
			if strings.IndexRune(rgba, c) == -1 {
				return false
			}
		}
		return true
	case strings.IndexByte(strq, s[0]) >= 0:
		for _, c := range s {
			if strings.IndexRune(strq, c) == -1 {
				return false
			}
		}
		return true
	}
	return false
}

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
	for i, t := range p.Uniforms {
		lines = append(lines, fmt.Sprintf("uniform %s;", p.glslVarDecl(&t, fmt.Sprintf("U%d", i))))
	}
	for i, t := range p.Attributes {
		lines = append(lines, fmt.Sprintf("attribute %s;", p.glslVarDecl(&t, fmt.Sprintf("A%d", i))))
	}
	for i, t := range p.Varyings {
		lines = append(lines, fmt.Sprintf("varying %s;", p.glslVarDecl(&t, fmt.Sprintf("V%d", i))))
	}
	for _, f := range p.Funcs {
		lines = append(lines, p.glslFunc(&f)...)
	}

	// Vertex func
	if len(p.VertexFunc.Block.Stmts) > 0 {
		lines = append(lines, "#if defined(COMPILING_VERTEX_SHADER)")
		lines = append(lines, "void main(void) {")
		lines = append(lines, p.glslBlock(&p.VertexFunc.Block, 0, 0)...)
		lines = append(lines, "}")
		lines = append(lines, "#endif")
	}

	// Fragment func
	if len(p.FragmentFunc.Block.Stmts) > 0 {
		lines = append(lines, "#if defined(COMPILING_FRAGMENT_SHADER)")
		lines = append(lines, "void main(void) {")
		lines = append(lines, p.glslBlock(&p.FragmentFunc.Block, 0, 0)...)
		lines = append(lines, "}")
		lines = append(lines, "#endif")
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

func (p *Program) glslType(t *Type) string {
	switch t.Main {
	case None:
		return "void"
	case Array:
		panic("not implemented")
	case Struct:
		return p.structName(t)
	default:
		return t.Main.Glsl()
	}
}

func (p *Program) glslVarDecl(t *Type, varname string) string {
	switch t.Main {
	case None:
		return "?(none)"
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

	var lines []string
	lines = append(lines, fmt.Sprintf("%s F%d(%s) {", p.glslType(&f.Return), f.Index, argsstr))
	lines = append(lines, p.glslBlock(&f.Block, 0, idx)...)
	lines = append(lines, "}")

	return lines
}

func (p *Program) glslBlock(b *Block, level int, localVarIndex int) []string {
	idt := strings.Repeat("\t", level+1)

	var lines []string
	for _, t := range b.LocalVars {
		lines = append(lines, fmt.Sprintf("%s%s;", idt, p.glslVarDecl(&t, fmt.Sprintf("l%d", localVarIndex))))
		localVarIndex++
	}

	var glslExpr func(e *Expr) string
	glslExpr = func(e *Expr) string {
		switch e.Type {
		case IntExpr:
			return fmt.Sprintf("%d", e.Int)
		case FloatExpr:
			return fmt.Sprintf("%.9e", e.Float)
		case UniformVariable:
			return fmt.Sprintf("U%d", e.Index)
		case LocalVariable:
			idx := e.Index
			switch b {
			case &p.VertexFunc.Block:
				na := len(p.Attributes)
				nv := len(p.Varyings)
				switch {
				case idx < na:
					return fmt.Sprintf("A%d", idx)
				case idx < na+nv:
					return fmt.Sprintf("V%d", idx-na)
				case idx == na+nv:
					return "gl_Position"
				default:
					return fmt.Sprintf("l%d", idx-(na+nv+1))
				}
			case &p.FragmentFunc.Block:
				nv := len(p.Varyings)
				switch {
				case idx < nv:
					return fmt.Sprintf("V%d", idx)
				case idx == nv:
					return "gl_FragCoord"
				default:
					return fmt.Sprintf("l%d", idx-(nv+1))
				}
			default:
				return fmt.Sprintf("l%d", idx)
			}
		case StructMember:
			return fmt.Sprintf("M%d", e.Index)
		case BuiltinFuncExpr:
			return string(e.BuiltinFunc)
		case SwizzlingExpr:
			if !isValidSwizzling(e.Swizzling) {
				return fmt.Sprintf("?(unexpected swizzling: %s)", e.Swizzling)
			}
			return e.Swizzling
		case FunctionExpr:
			return fmt.Sprintf("F%d", e.Index)
		case Unary:
			var op string
			switch e.Op {
			case Add, Sub, Neg:
				op = string(e.Op)
			default:
				op = fmt.Sprintf("?(unexpected op: %s)", string(e.Op))
			}
			return fmt.Sprintf("%s(%s)", op, glslExpr(&e.Exprs[0]))
		case Binary:
			return fmt.Sprintf("(%s) %s (%s)", glslExpr(&e.Exprs[0]), e.Op, glslExpr(&e.Exprs[1]))
		case Selection:
			return fmt.Sprintf("(%s) ? (%s) : (%s)", glslExpr(&e.Exprs[0]), glslExpr(&e.Exprs[1]), glslExpr(&e.Exprs[2]))
		case Call:
			var args []string
			for _, exp := range e.Exprs[1:] {
				args = append(args, glslExpr(&exp))
			}
			return fmt.Sprintf("(%s)(%s)", glslExpr(&e.Exprs[0]), strings.Join(args, ", "))
		case FieldSelector:
			return fmt.Sprintf("(%s).%s", glslExpr(&e.Exprs[0]), glslExpr(&e.Exprs[1]))
		case Index:
			return fmt.Sprintf("(%s)[%s]", glslExpr(&e.Exprs[0]), glslExpr(&e.Exprs[1]))
		default:
			return fmt.Sprintf("?(unexpected expr: %d)", e.Type)
		}
	}

	for _, s := range b.Stmts {
		switch s.Type {
		case ExprStmt:
			lines = append(lines, fmt.Sprintf("%s%s;", idt, glslExpr(&s.Exprs[0])))
		case BlockStmt:
			lines = append(lines, idt+"{")
			lines = append(lines, p.glslBlock(&s.Blocks[0], level+1, localVarIndex)...)
			lines = append(lines, idt+"}")
		case Assign:
			lines = append(lines, fmt.Sprintf("%s%s = %s;", idt, glslExpr(&s.Exprs[0]), glslExpr(&s.Exprs[1])))
		case If:
			lines = append(lines, fmt.Sprintf("%sif (%s) {", idt, glslExpr(&s.Exprs[0])))
			lines = append(lines, p.glslBlock(&s.Blocks[0], level+1, localVarIndex)...)
			if len(s.Blocks) > 1 {
				lines = append(lines, fmt.Sprintf("%s} else {", idt))
				lines = append(lines, p.glslBlock(&s.Blocks[1], level+1, localVarIndex)...)
			}
			lines = append(lines, fmt.Sprintf("%s}", idt))
		case For:
			v := localVarIndex
			localVarIndex++
			var delta string
			switch s.ForDelta {
			case 0:
				delta = fmt.Sprintf("?(unexpected delta: %d)", s.ForDelta)
			case 1:
				delta = fmt.Sprintf("l%d++", v)
			case -1:
				delta = fmt.Sprintf("l%d--", v)
			default:
				if s.ForDelta > 0 {
					delta = fmt.Sprintf("l%d += %d", v, s.ForDelta)
				} else {
					delta = fmt.Sprintf("l%d -= %d", v, -s.ForDelta)
				}
			}
			var op string
			switch s.ForOp {
			case LessThanOp, LessThanEqualOp, GreaterThanOp, GreaterThanEqualOp, EqualOp, NotEqualOp:
				op = string(s.ForOp)
			default:
				op = fmt.Sprintf("?(unexpected op: %s)", string(s.ForOp))
			}
			lines = append(lines, fmt.Sprintf("%sfor (int l%d = %d; l%d %s %d; %s) {", idt, v, s.ForInit, v, op, s.ForEnd, delta))
			lines = append(lines, p.glslBlock(&s.Blocks[0], level+1, localVarIndex)...)
			lines = append(lines, fmt.Sprintf("%s}", idt))
		case Continue:
			lines = append(lines, idt+"continue;")
		case Break:
			lines = append(lines, idt+"break;")
		case Return:
			if len(s.Exprs) == 0 {
				lines = append(lines, idt+"return;")
			} else {
				lines = append(lines, fmt.Sprintf("%sreturn %s;", idt, glslExpr(&s.Exprs[0])))
			}
		case Discard:
			lines = append(lines, idt+"discard;")
		default:
			lines = append(lines, fmt.Sprintf("%s?(unexpected stmt: %d)", idt, s.Type))
		}
	}

	return lines
}
