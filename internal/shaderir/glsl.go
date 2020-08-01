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
	"go/constant"
	"go/token"
	"strings"
)

const GlslFragmentPrelude = `#if defined(GL_ES)
precision highp float;
#else
#define lowp
#define mediump
#define highp
#endif
`

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

func (p *Program) Glsl() (vertexShader, fragmentShader string) {
	p.structNames = map[string]string{}
	p.structTypes = nil

	// Vertex func
	var vslines []string
	{
		for i, t := range p.Uniforms {
			vslines = append(vslines, fmt.Sprintf("uniform %s;", p.glslVarDecl(&t, fmt.Sprintf("U%d", i))))
		}
		for i := 0; i < p.TextureNum; i++ {
			vslines = append(vslines, fmt.Sprintf("uniform sampler2D T%d;", i))
		}
		for i, t := range p.Attributes {
			vslines = append(vslines, fmt.Sprintf("attribute %s;", p.glslVarDecl(&t, fmt.Sprintf("A%d", i))))
		}
		for i, t := range p.Varyings {
			vslines = append(vslines, fmt.Sprintf("varying %s;", p.glslVarDecl(&t, fmt.Sprintf("V%d", i))))
		}
		if len(p.Funcs) > 0 {
			if len(vslines) > 0 && vslines[len(vslines)-1] != "" {
				vslines = append(vslines, "")
			}
			for _, f := range p.Funcs {
				vslines = append(vslines, p.glslFunc(&f, true)...)
			}
			for _, f := range p.Funcs {
				if len(vslines) > 0 && vslines[len(vslines)-1] != "" {
					vslines = append(vslines, "")
				}
				vslines = append(vslines, p.glslFunc(&f, false)...)
			}
		}

		if len(p.VertexFunc.Block.Stmts) > 0 {
			if len(vslines) > 0 && vslines[len(vslines)-1] != "" {
				vslines = append(vslines, "")
			}
			vslines = append(vslines, "void main(void) {")
			vslines = append(vslines, p.glslBlock(&p.VertexFunc.Block, &p.VertexFunc.Block, 0, 0)...)
			vslines = append(vslines, "}")
		}
	}

	// Fragment func
	var fslines []string
	{
		for i, t := range p.Uniforms {
			fslines = append(fslines, fmt.Sprintf("uniform %s;", p.glslVarDecl(&t, fmt.Sprintf("U%d", i))))
		}
		for i := 0; i < p.TextureNum; i++ {
			fslines = append(fslines, fmt.Sprintf("uniform sampler2D T%d;", i))
		}
		for i, t := range p.Varyings {
			fslines = append(fslines, fmt.Sprintf("varying %s;", p.glslVarDecl(&t, fmt.Sprintf("V%d", i))))
		}
		if len(p.Funcs) > 0 {
			if len(fslines) > 0 && fslines[len(fslines)-1] != "" {
				fslines = append(fslines, "")
			}
			for _, f := range p.Funcs {
				fslines = append(fslines, p.glslFunc(&f, true)...)
			}
			for _, f := range p.Funcs {
				if len(fslines) > 0 && fslines[len(fslines)-1] != "" {
					fslines = append(fslines, "")
				}
				fslines = append(fslines, p.glslFunc(&f, false)...)
			}
		}

		if len(p.FragmentFunc.Block.Stmts) > 0 {
			if len(fslines) > 0 && fslines[len(fslines)-1] != "" {
				fslines = append(fslines, "")
			}
			fslines = append(fslines, "void main(void) {")
			fslines = append(fslines, p.glslBlock(&p.FragmentFunc.Block, &p.FragmentFunc.Block, 0, 0)...)
			fslines = append(fslines, "}")
		}
	}

	var stlines []string
	for i, t := range p.structTypes {
		stlines = append(stlines, fmt.Sprintf("struct S%d {", i))
		for j, st := range t.Sub {
			stlines = append(stlines, fmt.Sprintf("\t%s;", p.glslVarDecl(&st, fmt.Sprintf("M%d", j))))
		}
		stlines = append(stlines, "};")
	}

	vslines = append(stlines, vslines...)

	tmp := make([]string, len(stlines))
	copy(tmp, stlines)
	fslines = append(tmp, fslines...)

	fslines = append(strings.Split(GlslFragmentPrelude, "\n"), fslines...)

	return strings.Join(vslines, "\n") + "\n", strings.Join(fslines, "\n") + "\n"
}

func (p *Program) glslType(t *Type) (string, string) {
	switch t.Main {
	case None:
		return "void", ""
	case Struct:
		return p.structName(t), ""
	default:
		return t.Glsl()
	}
}

func (p *Program) glslVarDecl(t *Type, varname string) string {
	switch t.Main {
	case None:
		return "?(none)"
	case Struct:
		return fmt.Sprintf("%s %s", p.structName(t), varname)
	default:
		t0, t1 := t.Glsl()
		return fmt.Sprintf("%s %s%s", t0, varname, t1)
	}
}

func (p *Program) glslVarInit(t *Type) string {
	switch t.Main {
	case None:
		return "?(none)"
	case Array:
		init := p.glslVarInit(&t.Sub[0])
		es := make([]string, 0, t.Length)
		for i := 0; i < t.Length; i++ {
			es = append(es, init)
		}
		t0, t1 := t.Glsl()
		return fmt.Sprintf("%s%s(%s)", t0, t1, strings.Join(es, ", "))
	case Struct:
		panic("not implemented")
	case Bool:
		return "false"
	case Int:
		return "0"
	case Float:
		return "float(0)"
	case Vec2:
		return "vec2(0)"
	case Vec3:
		return "vec3(0)"
	case Vec4:
		return "vec4(0)"
	case Mat2:
		return "mat2(0)"
	case Mat3:
		return "mat3(0)"
	case Mat4:
		return "mat4(0)"
	default:
		t0, t1 := p.glslType(t)
		panic(fmt.Sprintf("?(unexpected type: %s%s)", t0, t1))
	}
}

func (p *Program) glslFunc(f *Func, prototype bool) []string {
	var args []string
	var idx int
	for _, t := range f.InParams {
		args = append(args, "in "+p.glslVarDecl(&t, fmt.Sprintf("l%d", idx)))
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

	t0, t1 := p.glslType(&f.Return)
	sig := fmt.Sprintf("%s%s F%d(%s)", t0, t1, f.Index, argsstr)

	var lines []string
	if prototype {
		lines = append(lines, fmt.Sprintf("%s;", sig))
		return lines
	}
	lines = append(lines, fmt.Sprintf("%s {", sig))
	lines = append(lines, p.glslBlock(&f.Block, &f.Block, 0, idx)...)
	lines = append(lines, "}")

	return lines
}

func constantToNumberLiteral(t ConstType, v constant.Value) string {
	switch t {
	case ConstTypeNone:
		if v.Kind() == constant.Bool {
			if constant.BoolVal(v) {
				return "true"
			}
			return "false"
		}
		fallthrough
	case ConstTypeFloat:
		if i := constant.ToInt(v); i.Kind() == constant.Int {
			x, _ := constant.Int64Val(i)
			return fmt.Sprintf("%d.0", x)
		}
		if i := constant.ToFloat(v); i.Kind() == constant.Float {
			x, _ := constant.Float64Val(i)
			return fmt.Sprintf("%.9e", x)
		}
	case ConstTypeInt:
		if i := constant.ToInt(v); i.Kind() == constant.Int {
			x, _ := constant.Int64Val(i)
			return fmt.Sprintf("%d", x)
		}
	}
	return fmt.Sprintf("?(unexpected literal: %s)", v)
}

func (p *Program) localVariableName(topBlock *Block, idx int) string {
	switch topBlock {
	case &p.VertexFunc.Block:
		na := len(p.Attributes)
		nv := len(p.Varyings)
		switch {
		case idx < na:
			return fmt.Sprintf("A%d", idx)
		case idx == na:
			return "gl_Position"
		case idx < na+nv+1:
			return fmt.Sprintf("V%d", idx-na-1)
		default:
			return fmt.Sprintf("l%d", idx-(na+nv+1))
		}
	case &p.FragmentFunc.Block:
		nv := len(p.Varyings)
		switch {
		case idx == 0:
			return "gl_FragCoord"
		case idx < nv+1:
			return fmt.Sprintf("V%d", idx-1)
		case idx == nv+1:
			return "gl_FragColor"
		default:
			return fmt.Sprintf("l%d", idx-(nv+2))
		}
	default:
		return fmt.Sprintf("l%d", idx)
	}
}

func (p *Program) glslBlock(topBlock, block *Block, level int, localVarIndex int) []string {
	idt := strings.Repeat("\t", level+1)

	var lines []string
	for _, t := range block.LocalVars {
		// The type is None e.g., when the variable is a for-loop counter.
		if t.Main != None {
			lines = append(lines, fmt.Sprintf("%s%s = %s;", idt, p.glslVarDecl(&t, fmt.Sprintf("l%d", localVarIndex)), p.glslVarInit(&t)))
		}
		localVarIndex++
	}

	var glslExpr func(e *Expr) string
	glslExpr = func(e *Expr) string {
		switch e.Type {
		case NumberExpr:
			return constantToNumberLiteral(e.ConstType, e.Const)
		case UniformVariable:
			return fmt.Sprintf("U%d", e.Index)
		case TextureVariable:
			return fmt.Sprintf("T%d", e.Index)
		case LocalVariable:
			return p.localVariableName(topBlock, e.Index)
		case StructMember:
			return fmt.Sprintf("M%d", e.Index)
		case BuiltinFuncExpr:
			return e.BuiltinFunc.Glsl()
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
			case Add, Sub, NotOp:
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
			// Using parentheses at the callee is illegal.
			return fmt.Sprintf("%s(%s)", glslExpr(&e.Exprs[0]), strings.Join(args, ", "))
		case FieldSelector:
			return fmt.Sprintf("(%s).%s", glslExpr(&e.Exprs[0]), glslExpr(&e.Exprs[1]))
		case Index:
			return fmt.Sprintf("(%s)[%s]", glslExpr(&e.Exprs[0]), glslExpr(&e.Exprs[1]))
		default:
			return fmt.Sprintf("?(unexpected expr: %d)", e.Type)
		}
	}

	for _, s := range block.Stmts {
		switch s.Type {
		case ExprStmt:
			lines = append(lines, fmt.Sprintf("%s%s;", idt, glslExpr(&s.Exprs[0])))
		case BlockStmt:
			lines = append(lines, idt+"{")
			lines = append(lines, p.glslBlock(topBlock, &s.Blocks[0], level+1, localVarIndex)...)
			lines = append(lines, idt+"}")
		case Assign:
			// TODO: Give an appropriate context
			lines = append(lines, fmt.Sprintf("%s%s = %s;", idt, glslExpr(&s.Exprs[0]), glslExpr(&s.Exprs[1])))
		case If:
			lines = append(lines, fmt.Sprintf("%sif (%s) {", idt, glslExpr(&s.Exprs[0])))
			lines = append(lines, p.glslBlock(topBlock, &s.Blocks[0], level+1, localVarIndex)...)
			if len(s.Blocks) > 1 {
				lines = append(lines, fmt.Sprintf("%s} else {", idt))
				lines = append(lines, p.glslBlock(topBlock, &s.Blocks[1], level+1, localVarIndex)...)
			}
			lines = append(lines, fmt.Sprintf("%s}", idt))
		case For:
			var ct ConstType
			switch s.ForVarType.Main {
			case Int:
				ct = ConstTypeInt
			case Float:
				ct = ConstTypeFloat
			}

			v := p.localVariableName(topBlock, s.ForVarIndex)
			var delta string
			switch val, _ := constant.Float64Val(s.ForDelta); val {
			case 0:
				delta = fmt.Sprintf("?(unexpected delta: %v)", s.ForDelta)
			case 1:
				delta = fmt.Sprintf("%s++", v)
			case -1:
				delta = fmt.Sprintf("%s--", v)
			default:
				d := s.ForDelta
				if val > 0 {
					delta = fmt.Sprintf("%s += %s", v, constantToNumberLiteral(ct, d))
				} else {
					d = constant.UnaryOp(token.SUB, d, 0)
					delta = fmt.Sprintf("%s -= %s", v, constantToNumberLiteral(ct, d))
				}
			}
			var op string
			switch s.ForOp {
			case LessThanOp, LessThanEqualOp, GreaterThanOp, GreaterThanEqualOp, EqualOp, NotEqualOp:
				op = string(s.ForOp)
			default:
				op = fmt.Sprintf("?(unexpected op: %s)", string(s.ForOp))
			}

			t := s.ForVarType
			init := constantToNumberLiteral(ct, s.ForInit)
			end := constantToNumberLiteral(ct, s.ForEnd)
			t0, t1 := t.Glsl()
			lines = append(lines, fmt.Sprintf("%sfor (%s %s%s = %s; %s %s %s; %s) {", idt, t0, v, t1, init, v, op, end, delta))
			lines = append(lines, p.glslBlock(topBlock, &s.Blocks[0], level+1, localVarIndex)...)
			lines = append(lines, fmt.Sprintf("%s}", idt))
		case Continue:
			lines = append(lines, idt+"continue;")
		case Break:
			lines = append(lines, idt+"break;")
		case Return:
			if len(s.Exprs) == 0 {
				lines = append(lines, idt+"return;")
			} else {
				// TODO: Give an appropriate context.
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
