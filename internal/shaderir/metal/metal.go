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

package metal

import (
	"fmt"
	"go/constant"
	"go/token"
	"regexp"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

const (
	vertexOut   = "varyings"
	fragmentOut = "out"
)

type compileContext struct {
	structNames map[string]string
	structTypes []shaderir.Type
}

func (c *compileContext) structName(p *shaderir.Program, t *shaderir.Type) string {
	if t.Main != shaderir.Struct {
		panic("metal: the given type at structName must be a struct")
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

const Prelude = `#include <metal_stdlib>

using namespace metal;

constexpr sampler texture_sampler{filter::nearest};`

func Compile(p *shaderir.Program, vertex, fragment string) (shader string) {
	c := &compileContext{
		structNames: map[string]string{},
	}

	var lines []string
	lines = append(lines, strings.Split(Prelude, "\n")...)
	lines = append(lines, "", "{{.Structs}}")

	if len(p.Attributes) > 0 {
		lines = append(lines, "")
		lines = append(lines, "struct Attributes {")
		for i, a := range p.Attributes {
			lines = append(lines, fmt.Sprintf("\t%s;", c.metalVarDecl(p, &a, fmt.Sprintf("M%d", i), true, false)))
		}
		lines = append(lines, "};")
	}

	if len(p.Varyings) > 0 {
		lines = append(lines, "")
		lines = append(lines, "struct Varyings {")
		lines = append(lines, "\tfloat4 Position [[position]];")
		for i, v := range p.Varyings {
			lines = append(lines, fmt.Sprintf("\t%s;", c.metalVarDecl(p, &v, fmt.Sprintf("M%d", i), false, false)))
		}
		lines = append(lines, "};")
	}

	if len(p.Funcs) > 0 {
		lines = append(lines, "")
		for _, f := range p.Funcs {
			lines = append(lines, c.metalFunc(p, &f, true)...)
		}
		for _, f := range p.Funcs {
			if len(lines) > 0 && lines[len(lines)-1] != "" {
				lines = append(lines, "")
			}
			lines = append(lines, c.metalFunc(p, &f, false)...)
		}
	}

	if p.VertexFunc.Block != nil && len(p.VertexFunc.Block.Stmts) > 0 {
		lines = append(lines, "")
		lines = append(lines,
			fmt.Sprintf("vertex Varyings %s(", vertex),
			"\tuint vid [[vertex_id]],",
			"\tconst device Attributes* attributes [[buffer(0)]]")
		for i, u := range p.Uniforms {
			lines[len(lines)-1] += ","
			lines = append(lines, fmt.Sprintf("\tconstant %s [[buffer(%d)]]", c.metalVarDecl(p, &u, fmt.Sprintf("U%d", i), false, true), i+1))
		}
		for i := 0; i < p.TextureNum; i++ {
			lines[len(lines)-1] += ","
			lines = append(lines, fmt.Sprintf("\ttexture2d<float> T%[1]d [[texture(%[1]d)]]", i))
		}
		lines[len(lines)-1] += ") {"
		lines = append(lines, fmt.Sprintf("\tVaryings %s = {};", vertexOut))
		lines = append(lines, c.metalBlock(p, p.VertexFunc.Block, p.VertexFunc.Block, 0)...)
		if last := fmt.Sprintf("\treturn %s;", vertexOut); lines[len(lines)-1] != last {
			lines = append(lines, last)
		}
		lines = append(lines, "}")
	}

	if p.FragmentFunc.Block != nil && len(p.FragmentFunc.Block.Stmts) > 0 {
		lines = append(lines, "")
		lines = append(lines,
			fmt.Sprintf("fragment float4 %s(", fragment),
			"\tVaryings varyings [[stage_in]]")
		for i, u := range p.Uniforms {
			lines[len(lines)-1] += ","
			lines = append(lines, fmt.Sprintf("\tconstant %s [[buffer(%d)]]", c.metalVarDecl(p, &u, fmt.Sprintf("U%d", i), false, true), i+1))
		}
		for i := 0; i < p.TextureNum; i++ {
			lines[len(lines)-1] += ","
			lines = append(lines, fmt.Sprintf("\ttexture2d<float> T%[1]d [[texture(%[1]d)]]", i))
		}
		lines[len(lines)-1] += ") {"
		lines = append(lines, fmt.Sprintf("\tfloat4 %s = float4(0);", fragmentOut))
		lines = append(lines, c.metalBlock(p, p.FragmentFunc.Block, p.FragmentFunc.Block, 0)...)
		if last := fmt.Sprintf("\treturn %s;", fragmentOut); lines[len(lines)-1] != last {
			lines = append(lines, last)
		}
		lines = append(lines, "}")
	}

	ls := strings.Join(lines, "\n")

	// Struct types are determined after converting the program.
	if len(c.structTypes) > 0 {
		var stlines []string
		for i, t := range c.structTypes {
			stlines = append(stlines, fmt.Sprintf("struct S%d {", i))
			for j, st := range t.Sub {
				stlines = append(stlines, fmt.Sprintf("\t%s;", c.metalVarDecl(p, &st, fmt.Sprintf("M%d", j), false, false)))
			}
			stlines = append(stlines, "};")
		}
		ls = strings.ReplaceAll(ls, "{{.Structs}}", strings.Join(stlines, "\n"))
	} else {
		ls = strings.ReplaceAll(ls, "{{.Structs}}", "")
	}

	nls := regexp.MustCompile(`\n\n+`)
	ls = nls.ReplaceAllString(ls, "\n\n")
	ls = strings.TrimSpace(ls) + "\n"

	return ls
}

func (c *compileContext) metalType(p *shaderir.Program, t *shaderir.Type, packed bool, ref bool) string {
	switch t.Main {
	case shaderir.None:
		return "void"
	case shaderir.Struct:
		return c.structName(p, t)
	default:
		return typeString(t, packed, ref)
	}
}

func (c *compileContext) metalVarDecl(p *shaderir.Program, t *shaderir.Type, varname string, packed bool, ref bool) string {
	switch t.Main {
	case shaderir.None:
		return "?(none)"
	case shaderir.Struct:
		s := c.structName(p, t)
		if ref {
			s += "&"
		}
		return fmt.Sprintf("%s %s", s, varname)
	default:
		t := typeString(t, packed, ref)
		return fmt.Sprintf("%s %s", t, varname)
	}
}

func (c *compileContext) metalVarInit(p *shaderir.Program, t *shaderir.Type) string {
	switch t.Main {
	case shaderir.None:
		return "?(none)"
	case shaderir.Array:
		return "{}"
	case shaderir.Struct:
		return "{}"
	case shaderir.Bool:
		return "false"
	case shaderir.Int:
		return "0"
	case shaderir.Float, shaderir.Vec2, shaderir.Vec3, shaderir.Vec4, shaderir.Mat2, shaderir.Mat3, shaderir.Mat4:
		return fmt.Sprintf("%s(0)", basicTypeString(t.Main, false))
	default:
		t := c.metalType(p, t, false, false)
		panic(fmt.Sprintf("?(unexpected type: %s)", t))
	}
}

func (c *compileContext) metalFunc(p *shaderir.Program, f *shaderir.Func, prototype bool) []string {
	var args []string

	// Uniform variables and texture variables. In Metal, non-const global variables are not available.
	for i, u := range p.Uniforms {
		args = append(args, "constant "+c.metalVarDecl(p, &u, fmt.Sprintf("U%d", i), false, true))
	}
	for i := 0; i < p.TextureNum; i++ {
		args = append(args, fmt.Sprintf("texture2d<float> T%d", i))
	}

	var idx int
	for _, t := range f.InParams {
		args = append(args, c.metalVarDecl(p, &t, fmt.Sprintf("l%d", idx), false, false))
		idx++
	}
	for _, t := range f.OutParams {
		args = append(args, "thread "+c.metalVarDecl(p, &t, fmt.Sprintf("l%d", idx), false, true))
		idx++
	}
	argsstr := "void"
	if len(args) > 0 {
		argsstr = strings.Join(args, ", ")
	}

	t := c.metalType(p, &f.Return, false, false)
	sig := fmt.Sprintf("%s F%d(%s)", t, f.Index, argsstr)

	var lines []string
	if prototype {
		lines = append(lines, fmt.Sprintf("%s;", sig))
		return lines
	}
	lines = append(lines, fmt.Sprintf("%s {", sig))
	lines = append(lines, c.metalBlock(p, f.Block, f.Block, 0)...)
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

func localVariableName(p *shaderir.Program, topBlock *shaderir.Block, idx int) string {
	switch topBlock {
	case p.VertexFunc.Block:
		na := len(p.Attributes)
		nv := len(p.Varyings)
		switch {
		case idx < na:
			return fmt.Sprintf("attributes[vid].M%d", idx)
		case idx == na:
			return fmt.Sprintf("%s.Position", vertexOut)
		case idx < na+nv+1:
			return fmt.Sprintf("%s.M%d", vertexOut, idx-na-1)
		default:
			return fmt.Sprintf("l%d", idx-(na+nv+1))
		}
	case p.FragmentFunc.Block:
		nv := len(p.Varyings)
		switch {
		case idx == 0:
			return fmt.Sprintf("varyings.Position")
		case idx < nv+1:
			return fmt.Sprintf("varyings.M%d", idx-1)
		case idx == nv+1:
			return fragmentOut
		default:
			return fmt.Sprintf("l%d", idx-(nv+2))
		}
	default:
		return fmt.Sprintf("l%d", idx)
	}
}

func (c *compileContext) initVariable(p *shaderir.Program, topBlock, block *shaderir.Block, index int, decl bool, level int) []string {
	idt := strings.Repeat("\t", level+1)
	name := localVariableName(p, topBlock, index)
	t := p.LocalVariableType(topBlock, block, index)

	var lines []string
	if decl {
		lines = append(lines, fmt.Sprintf("%s%s = %s;", idt, c.metalVarDecl(p, &t, name, false, false), c.metalVarInit(p, &t)))
	} else {
		lines = append(lines, fmt.Sprintf("%s%s = %s;", idt, name, c.metalVarInit(p, &t)))
	}
	return lines
}

func (c *compileContext) metalBlock(p *shaderir.Program, topBlock, block *shaderir.Block, level int) []string {
	if block == nil {
		return nil
	}

	idt := strings.Repeat("\t", level+1)

	var lines []string
	for i, t := range block.LocalVars {
		// The type is None e.g., when the variable is a for-loop counter.
		if t.Main != shaderir.None {
			lines = append(lines, c.initVariable(p, topBlock, block, block.LocalVarIndexOffset+i, true, level)...)
		}
	}

	var metalExpr func(e *shaderir.Expr) string
	metalExpr = func(e *shaderir.Expr) string {
		switch e.Type {
		case shaderir.NumberExpr:
			return constantToNumberLiteral(e.ConstType, e.Const)
		case shaderir.UniformVariable:
			return fmt.Sprintf("U%d", e.Index)
		case shaderir.TextureVariable:
			return fmt.Sprintf("T%d", e.Index)
		case shaderir.LocalVariable:
			return localVariableName(p, topBlock, e.Index)
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
			return fmt.Sprintf("%s(%s)", op, metalExpr(&e.Exprs[0]))
		case shaderir.Binary:
			return fmt.Sprintf("(%s) %s (%s)", metalExpr(&e.Exprs[0]), e.Op, metalExpr(&e.Exprs[1]))
		case shaderir.Selection:
			return fmt.Sprintf("(%s) ? (%s) : (%s)", metalExpr(&e.Exprs[0]), metalExpr(&e.Exprs[1]), metalExpr(&e.Exprs[2]))
		case shaderir.Call:
			callee := e.Exprs[0]
			var args []string
			if callee.Type != shaderir.BuiltinFuncExpr {
				for i := range p.Uniforms {
					args = append(args, fmt.Sprintf("U%d", i))
				}
				for i := 0; i < p.TextureNum; i++ {
					args = append(args, fmt.Sprintf("T%d", i))
				}
			}
			for _, exp := range e.Exprs[1:] {
				args = append(args, metalExpr(&exp))
			}
			if callee.Type == shaderir.BuiltinFuncExpr && callee.BuiltinFunc == shaderir.Texture2DF {
				return fmt.Sprintf("%s.sample(texture_sampler, %s)", args[0], strings.Join(args[1:], ", "))
			}
			return fmt.Sprintf("%s(%s)", metalExpr(&callee), strings.Join(args, ", "))
		case shaderir.FieldSelector:
			return fmt.Sprintf("(%s).%s", metalExpr(&e.Exprs[0]), metalExpr(&e.Exprs[1]))
		case shaderir.Index:
			return fmt.Sprintf("(%s)[%s]", metalExpr(&e.Exprs[0]), metalExpr(&e.Exprs[1]))
		default:
			return fmt.Sprintf("?(unexpected expr: %d)", e.Type)
		}
	}

	for _, s := range block.Stmts {
		switch s.Type {
		case shaderir.ExprStmt:
			lines = append(lines, fmt.Sprintf("%s%s;", idt, metalExpr(&s.Exprs[0])))
		case shaderir.BlockStmt:
			lines = append(lines, idt+"{")
			lines = append(lines, c.metalBlock(p, topBlock, s.Blocks[0], level+1)...)
			lines = append(lines, idt+"}")
		case shaderir.Assign:
			lines = append(lines, fmt.Sprintf("%s%s = %s;", idt, metalExpr(&s.Exprs[0]), metalExpr(&s.Exprs[1])))
		case shaderir.Init:
			init := true
			if topBlock == p.VertexFunc.Block {
				// In the vertex function, varying values are the output parameters.
				// These values are represented as a struct and not needed to be initialized.
				na := len(p.Attributes)
				nv := len(p.Varyings)
				if s.InitIndex < na+nv+1 {
					init = false
				}
			}
			if init {
				lines = append(lines, c.initVariable(p, topBlock, block, s.InitIndex, false, level)...)
			}
		case shaderir.If:
			lines = append(lines, fmt.Sprintf("%sif (%s) {", idt, metalExpr(&s.Exprs[0])))
			lines = append(lines, c.metalBlock(p, topBlock, s.Blocks[0], level+1)...)
			if len(s.Blocks) > 1 {
				lines = append(lines, fmt.Sprintf("%s} else {", idt))
				lines = append(lines, c.metalBlock(p, topBlock, s.Blocks[1], level+1)...)
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

			v := localVariableName(p, topBlock, s.ForVarIndex)
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
			ts := typeString(&t, false, false)
			lines = append(lines, fmt.Sprintf("%sfor (%s %s = %s; %s %s %s; %s) {", idt, ts, v, init, v, op, end, delta))
			lines = append(lines, c.metalBlock(p, topBlock, s.Blocks[0], level+1)...)
			lines = append(lines, fmt.Sprintf("%s}", idt))
		case shaderir.Continue:
			lines = append(lines, idt+"continue;")
		case shaderir.Break:
			lines = append(lines, idt+"break;")
		case shaderir.Return:
			switch {
			case topBlock == p.VertexFunc.Block:
				lines = append(lines, fmt.Sprintf("%sreturn %s;", idt, vertexOut))
			case topBlock == p.FragmentFunc.Block:
				lines = append(lines, fmt.Sprintf("%sreturn %s;", idt, fragmentOut))
			case len(s.Exprs) == 0:
				lines = append(lines, idt+"return;")
			default:
				lines = append(lines, fmt.Sprintf("%sreturn %s;", idt, metalExpr(&s.Exprs[0])))
			}
		case shaderir.Discard:
			lines = append(lines, idt+"discard;")
		default:
			lines = append(lines, fmt.Sprintf("%s?(unexpected stmt: %d)", idt, s.Type))
		}
	}

	return lines
}
