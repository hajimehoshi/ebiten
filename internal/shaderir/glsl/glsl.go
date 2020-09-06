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

package glsl

import (
	"fmt"
	"go/constant"
	"go/token"
	"regexp"
	"strings"

	"github.com/hajimehoshi/ebiten/internal/shaderir"
)

const FragmentPrelude = `#if defined(GL_ES)
precision highp float;
#else
#define lowp
#define mediump
#define highp
#endif`

type compileContext struct {
	structNames map[string]string
	structTypes []shaderir.Type
}

func (c *compileContext) structName(p *shaderir.Program, t *shaderir.Type) string {
	if t.Main != shaderir.Struct {
		panic("glsl: the given type at structName must be a struct")
	}
	s := t.String()
	if n, ok := c.structNames[s]; ok {
		return n
	}
	n := fmt.Sprintf("S%d", len(c.structNames))
	c.structNames[s] = n
	c.structTypes = append(c.structTypes, *t)
	return n
}

func Compile(p *shaderir.Program) (vertexShader, fragmentShader string) {
	c := &compileContext{
		structNames: map[string]string{},
	}

	// Vertex func
	var vslines []string
	{
		vslines = append(vslines, "{{.Structs}}")
		if len(p.Uniforms) > 0 || p.TextureNum > 0 || len(p.Attributes) > 0 || len(p.Varyings) > 0 {
			vslines = append(vslines, "")
			for i, t := range p.Uniforms {
				vslines = append(vslines, fmt.Sprintf("uniform %s;", c.glslVarDecl(p, &t, fmt.Sprintf("U%d", i))))
			}
			for i := 0; i < p.TextureNum; i++ {
				vslines = append(vslines, fmt.Sprintf("uniform sampler2D T%d;", i))
			}
			for i, t := range p.Attributes {
				vslines = append(vslines, fmt.Sprintf("attribute %s;", c.glslVarDecl(p, &t, fmt.Sprintf("A%d", i))))
			}
			for i, t := range p.Varyings {
				vslines = append(vslines, fmt.Sprintf("varying %s;", c.glslVarDecl(p, &t, fmt.Sprintf("V%d", i))))
			}
		}
		if len(p.Funcs) > 0 {
			vslines = append(vslines, "")
			for _, f := range p.Funcs {
				vslines = append(vslines, c.glslFunc(p, &f, true)...)
			}
			for _, f := range p.Funcs {
				if len(vslines) > 0 && vslines[len(vslines)-1] != "" {
					vslines = append(vslines, "")
				}
				vslines = append(vslines, c.glslFunc(p, &f, false)...)
			}
		}

		if p.VertexFunc.Block != nil && len(p.VertexFunc.Block.Stmts) > 0 {
			vslines = append(vslines, "")
			vslines = append(vslines, "void main(void) {")
			vslines = append(vslines, c.glslBlock(p, p.VertexFunc.Block, p.VertexFunc.Block, 0)...)
			vslines = append(vslines, "}")
		}
	}

	// Fragment func
	var fslines []string
	{
		fslines = append(fslines, strings.Split(FragmentPrelude, "\n")...)
		fslines = append(fslines, "", "{{.Structs}}")
		if len(p.Uniforms) > 0 || p.TextureNum > 0 || len(p.Varyings) > 0 {
			fslines = append(fslines, "")
			for i, t := range p.Uniforms {
				fslines = append(fslines, fmt.Sprintf("uniform %s;", c.glslVarDecl(p, &t, fmt.Sprintf("U%d", i))))
			}
			for i := 0; i < p.TextureNum; i++ {
				fslines = append(fslines, fmt.Sprintf("uniform sampler2D T%d;", i))
			}
			for i, t := range p.Varyings {
				fslines = append(fslines, fmt.Sprintf("varying %s;", c.glslVarDecl(p, &t, fmt.Sprintf("V%d", i))))
			}
		}
		if len(p.Funcs) > 0 {
			fslines = append(fslines, "")
			for _, f := range p.Funcs {
				fslines = append(fslines, c.glslFunc(p, &f, true)...)
			}
			for _, f := range p.Funcs {
				if len(fslines) > 0 && fslines[len(fslines)-1] != "" {
					fslines = append(fslines, "")
				}
				fslines = append(fslines, c.glslFunc(p, &f, false)...)
			}
		}

		if p.FragmentFunc.Block != nil && len(p.FragmentFunc.Block.Stmts) > 0 {
			fslines = append(fslines, "")
			fslines = append(fslines, "void main(void) {")
			fslines = append(fslines, c.glslBlock(p, p.FragmentFunc.Block, p.FragmentFunc.Block, 0)...)
			fslines = append(fslines, "}")
		}
	}

	vs := strings.Join(vslines, "\n")
	fs := strings.Join(fslines, "\n")

	// Struct types are determined after converting the program.
	if len(c.structTypes) > 0 {
		var stlines []string
		for i, t := range c.structTypes {
			stlines = append(stlines, fmt.Sprintf("struct S%d {", i))
			for j, st := range t.Sub {
				stlines = append(stlines, fmt.Sprintf("\t%s;", c.glslVarDecl(p, &st, fmt.Sprintf("M%d", j))))
			}
			stlines = append(stlines, "};")
		}
		st := strings.Join(stlines, "\n")
		vs = strings.ReplaceAll(vs, "{{.Structs}}", st)
		fs = strings.ReplaceAll(fs, "{{.Structs}}", st)
	} else {
		vs = strings.ReplaceAll(vs, "{{.Structs}}", "")
		fs = strings.ReplaceAll(fs, "{{.Structs}}", "")
	}

	nls := regexp.MustCompile(`\n\n+`)
	vs = nls.ReplaceAllString(vs, "\n\n")
	fs = nls.ReplaceAllString(fs, "\n\n")

	vs = strings.TrimSpace(vs) + "\n"
	fs = strings.TrimSpace(fs) + "\n"

	return vs, fs
}

func (c *compileContext) glslType(p *shaderir.Program, t *shaderir.Type) (string, string) {
	switch t.Main {
	case shaderir.None:
		return "void", ""
	case shaderir.Struct:
		return c.structName(p, t), ""
	default:
		return typeString(t)
	}
}

func (c *compileContext) glslVarDecl(p *shaderir.Program, t *shaderir.Type, varname string) string {
	switch t.Main {
	case shaderir.None:
		return "?(none)"
	case shaderir.Struct:
		return fmt.Sprintf("%s %s", c.structName(p, t), varname)
	default:
		t0, t1 := typeString(t)
		return fmt.Sprintf("%s %s%s", t0, varname, t1)
	}
}

func (c *compileContext) glslVarInit(p *shaderir.Program, t *shaderir.Type) string {
	switch t.Main {
	case shaderir.None:
		return "?(none)"
	case shaderir.Array:
		init := c.glslVarInit(p, &t.Sub[0])
		es := make([]string, 0, t.Length)
		for i := 0; i < t.Length; i++ {
			es = append(es, init)
		}
		t0, t1 := typeString(t)
		return fmt.Sprintf("%s%s(%s)", t0, t1, strings.Join(es, ", "))
	case shaderir.Struct:
		panic("not implemented")
	case shaderir.Bool:
		return "false"
	case shaderir.Int:
		return "0"
	case shaderir.Float, shaderir.Vec2, shaderir.Vec3, shaderir.Vec4, shaderir.Mat2, shaderir.Mat3, shaderir.Mat4:
		return fmt.Sprintf("%s(0)", basicTypeString(t.Main))
	default:
		t0, t1 := c.glslType(p, t)
		panic(fmt.Sprintf("?(unexpected type: %s%s)", t0, t1))
	}
}

func (c *compileContext) glslFunc(p *shaderir.Program, f *shaderir.Func, prototype bool) []string {
	var args []string
	var idx int
	for _, t := range f.InParams {
		args = append(args, "in "+c.glslVarDecl(p, &t, fmt.Sprintf("l%d", idx)))
		idx++
	}
	for _, t := range f.OutParams {
		args = append(args, "out "+c.glslVarDecl(p, &t, fmt.Sprintf("l%d", idx)))
		idx++
	}
	argsstr := "void"
	if len(args) > 0 {
		argsstr = strings.Join(args, ", ")
	}

	t0, t1 := c.glslType(p, &f.Return)
	sig := fmt.Sprintf("%s%s F%d(%s)", t0, t1, f.Index, argsstr)

	var lines []string
	if prototype {
		lines = append(lines, fmt.Sprintf("%s;", sig))
		return lines
	}
	lines = append(lines, fmt.Sprintf("%s {", sig))
	lines = append(lines, c.glslBlock(p, f.Block, f.Block, 0)...)
	lines = append(lines, "}")

	return lines
}

func constantToNumberLiteral(t shaderir.ConstType, v constant.Value) string {
	switch t {
	case shaderir.ConstTypeNone:
		if v.Kind() == constant.Bool {
			if constant.BoolVal(v) {
				return "true"
			}
			return "false"
		}
		fallthrough
	case shaderir.ConstTypeFloat:
		if i := constant.ToInt(v); i.Kind() == constant.Int {
			x, _ := constant.Int64Val(i)
			return fmt.Sprintf("%d.0", x)
		}
		if i := constant.ToFloat(v); i.Kind() == constant.Float {
			x, _ := constant.Float64Val(i)
			return fmt.Sprintf("%.10e", x)
		}
	case shaderir.ConstTypeInt:
		if i := constant.ToInt(v); i.Kind() == constant.Int {
			x, _ := constant.Int64Val(i)
			return fmt.Sprintf("%d", x)
		}
	}
	return fmt.Sprintf("?(unexpected literal: %s)", v)
}

func localVariableName(p *shaderir.Program, topBlock, block *shaderir.Block, idx int) string {
	switch topBlock {
	case p.VertexFunc.Block:
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
	case p.FragmentFunc.Block:
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

func (c *compileContext) initVariable(p *shaderir.Program, topBlock, block *shaderir.Block, index int, decl bool, level int) []string {
	idt := strings.Repeat("\t", level+1)
	name := localVariableName(p, topBlock, block, index)
	t := p.LocalVariableType(topBlock, block, index)

	var lines []string
	switch t.Main {
	case shaderir.Array:
		if decl {
			lines = append(lines, fmt.Sprintf("%s%s;", idt, c.glslVarDecl(p, &t, name)))
		}
		init := c.glslVarInit(p, &t.Sub[0])
		for i := 0; i < t.Length; i++ {
			lines = append(lines, fmt.Sprintf("%s%s[%d] = %s;", idt, name, i, init))
		}
	case shaderir.None:
		// The type is None e.g., when the variable is a for-loop counter.
	default:
		if decl {
			lines = append(lines, fmt.Sprintf("%s%s = %s;", idt, c.glslVarDecl(p, &t, name), c.glslVarInit(p, &t)))
		} else {
			lines = append(lines, fmt.Sprintf("%s%s = %s;", idt, name, c.glslVarInit(p, &t)))
		}
	}
	return lines
}

func (c *compileContext) glslBlock(p *shaderir.Program, topBlock, block *shaderir.Block, level int) []string {
	if block == nil {
		return nil
	}

	var lines []string
	for i := range block.LocalVars {
		lines = append(lines, c.initVariable(p, topBlock, block, block.LocalVarIndexOffset+i, true, level)...)
	}

	var glslExpr func(e *shaderir.Expr) string
	glslExpr = func(e *shaderir.Expr) string {
		switch e.Type {
		case shaderir.NumberExpr:
			return constantToNumberLiteral(e.ConstType, e.Const)
		case shaderir.UniformVariable:
			return fmt.Sprintf("U%d", e.Index)
		case shaderir.TextureVariable:
			return fmt.Sprintf("T%d", e.Index)
		case shaderir.LocalVariable:
			return localVariableName(p, topBlock, block, e.Index)
		case shaderir.StructMember:
			return fmt.Sprintf("M%d", e.Index)
		case shaderir.BuiltinFuncExpr:
			return builtinFuncString(e.BuiltinFunc)
		case shaderir.SwizzlingExpr:
			if !shaderir.IsValidSwizzling(e.Swizzling) {
				return fmt.Sprintf("?(unexpected swizzling: %s)", e.Swizzling)
			}
			return e.Swizzling
		case shaderir.FunctionExpr:
			return fmt.Sprintf("F%d", e.Index)
		case shaderir.Unary:
			var op string
			switch e.Op {
			case shaderir.Add, shaderir.Sub, shaderir.NotOp:
				op = string(e.Op)
			default:
				op = fmt.Sprintf("?(unexpected op: %s)", string(e.Op))
			}
			return fmt.Sprintf("%s(%s)", op, glslExpr(&e.Exprs[0]))
		case shaderir.Binary:
			return fmt.Sprintf("(%s) %s (%s)", glslExpr(&e.Exprs[0]), e.Op, glslExpr(&e.Exprs[1]))
		case shaderir.Selection:
			return fmt.Sprintf("(%s) ? (%s) : (%s)", glslExpr(&e.Exprs[0]), glslExpr(&e.Exprs[1]), glslExpr(&e.Exprs[2]))
		case shaderir.Call:
			var args []string
			for _, exp := range e.Exprs[1:] {
				args = append(args, glslExpr(&exp))
			}
			// Using parentheses at the callee is illegal.
			return fmt.Sprintf("%s(%s)", glslExpr(&e.Exprs[0]), strings.Join(args, ", "))
		case shaderir.FieldSelector:
			return fmt.Sprintf("(%s).%s", glslExpr(&e.Exprs[0]), glslExpr(&e.Exprs[1]))
		case shaderir.Index:
			return fmt.Sprintf("(%s)[%s]", glslExpr(&e.Exprs[0]), glslExpr(&e.Exprs[1]))
		default:
			return fmt.Sprintf("?(unexpected expr: %d)", e.Type)
		}
	}

	idt := strings.Repeat("\t", level+1)
	for _, s := range block.Stmts {
		switch s.Type {
		case shaderir.ExprStmt:
			lines = append(lines, fmt.Sprintf("%s%s;", idt, glslExpr(&s.Exprs[0])))
		case shaderir.BlockStmt:
			lines = append(lines, idt+"{")
			lines = append(lines, c.glslBlock(p, topBlock, s.Blocks[0], level+1)...)
			lines = append(lines, idt+"}")
		case shaderir.Assign:
			lhs := s.Exprs[0]
			rhs := s.Exprs[1]
			if lhs.Type == shaderir.LocalVariable {
				if t := p.LocalVariableType(topBlock, block, lhs.Index); t.Main == shaderir.Array {
					for i := 0; i < t.Length; i++ {
						lines = append(lines, fmt.Sprintf("%[1]s%[2]s[%[3]d] = %[4]s[%[3]d];", idt, glslExpr(&lhs), i, glslExpr(&rhs)))
					}
					continue
				}
			}
			lines = append(lines, fmt.Sprintf("%s%s = %s;", idt, glslExpr(&lhs), glslExpr(&rhs)))
		case shaderir.Init:
			lines = append(lines, c.initVariable(p, topBlock, block, s.InitIndex, false, level)...)
		case shaderir.If:
			lines = append(lines, fmt.Sprintf("%sif (%s) {", idt, glslExpr(&s.Exprs[0])))
			lines = append(lines, c.glslBlock(p, topBlock, s.Blocks[0], level+1)...)
			if len(s.Blocks) > 1 {
				lines = append(lines, fmt.Sprintf("%s} else {", idt))
				lines = append(lines, c.glslBlock(p, topBlock, s.Blocks[1], level+1)...)
			}
			lines = append(lines, fmt.Sprintf("%s}", idt))
		case shaderir.For:
			var ct shaderir.ConstType
			switch s.ForVarType.Main {
			case shaderir.Int:
				ct = shaderir.ConstTypeInt
			case shaderir.Float:
				ct = shaderir.ConstTypeFloat
			}

			v := localVariableName(p, topBlock, block, s.ForVarIndex)
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
			case shaderir.LessThanOp, shaderir.LessThanEqualOp, shaderir.GreaterThanOp, shaderir.GreaterThanEqualOp, shaderir.EqualOp, shaderir.NotEqualOp:
				op = string(s.ForOp)
			default:
				op = fmt.Sprintf("?(unexpected op: %s)", string(s.ForOp))
			}

			t := s.ForVarType
			init := constantToNumberLiteral(ct, s.ForInit)
			end := constantToNumberLiteral(ct, s.ForEnd)
			t0, t1 := typeString(&t)
			lines = append(lines, fmt.Sprintf("%sfor (%s %s%s = %s; %s %s %s; %s) {", idt, t0, v, t1, init, v, op, end, delta))
			lines = append(lines, c.glslBlock(p, topBlock, s.Blocks[0], level+1)...)
			lines = append(lines, fmt.Sprintf("%s}", idt))
		case shaderir.Continue:
			lines = append(lines, idt+"continue;")
		case shaderir.Break:
			lines = append(lines, idt+"break;")
		case shaderir.Return:
			if len(s.Exprs) == 0 {
				lines = append(lines, idt+"return;")
			} else {
				lines = append(lines, fmt.Sprintf("%sreturn %s;", idt, glslExpr(&s.Exprs[0])))
			}
		case shaderir.Discard:
			lines = append(lines, idt+"discard;")
		default:
			lines = append(lines, fmt.Sprintf("%s?(unexpected stmt: %d)", idt, s.Type))
		}
	}

	return lines
}
