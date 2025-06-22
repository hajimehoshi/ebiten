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

package msl

import (
	"fmt"
	"go/constant"
	"go/token"
	"math"
	"regexp"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

const (
	vertexOut = "varyings"
)

type compileContext struct {
	structNames map[string]string
	structTypes []shaderir.Type
}

func (c *compileContext) structName(p *shaderir.Program, t *shaderir.Type) string {
	if t.Main != shaderir.Struct {
		panic("msl: the given type at structName must be a struct")
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

func Prelude(unit shaderir.Unit) string {
	str := `#include <metal_stdlib>

using namespace metal;

template<typename T, typename U>
T mod(T x, U y) {
	return x - y * floor(x/y);
}`
	if unit == shaderir.Texels {
		str += `

constexpr sampler texture_sampler{filter::nearest};`
	}
	return str
}

const (
	VertexName   = "Vertex"
	FragmentName = "Fragment"
)

func Compile(p *shaderir.Program) (shader string) {
	c := &compileContext{
		structNames: map[string]string{},
	}

	var lines []string
	lines = append(lines, strings.Split(Prelude(p.Unit), "\n")...)
	lines = append(lines, "", "{{.Structs}}")

	if len(p.Uniforms) > 0 {
		lines = append(lines, "")
		lines = append(lines, "struct Uniforms {")
		for i, u := range p.Uniforms {
			lines = append(lines, fmt.Sprintf("\t%s;", c.varDecl(p, &u, fmt.Sprintf("U%d", i), false)))
		}
		lines = append(lines, "};")
	}

	if len(p.Attributes) > 0 {
		lines = append(lines, "")
		lines = append(lines, "struct Attributes {")
		for i, a := range p.Attributes {
			lines = append(lines, fmt.Sprintf("\t%s;", c.varDecl(p, &a, fmt.Sprintf("M%d", i), false)))
		}
		lines = append(lines, "};")
	}

	if len(p.Varyings) > 0 {
		lines = append(lines, "")
		lines = append(lines, "struct Varyings {")
		lines = append(lines, "\tfloat4 Position [[position]];")
		for i, v := range p.Varyings {
			lines = append(lines, fmt.Sprintf("\t%s;", c.varDecl(p, &v, fmt.Sprintf("M%d", i), false)))
		}
		lines = append(lines, "};")
	}

	if len(p.Funcs) > 0 {
		lines = append(lines, "")
		for _, f := range p.Funcs {
			lines = append(lines, c.function(p, &f, true)...)
		}
		for _, f := range p.Funcs {
			if len(lines) > 0 && lines[len(lines)-1] != "" {
				lines = append(lines, "")
			}
			lines = append(lines, c.function(p, &f, false)...)
		}
	}

	if p.VertexFunc.Block != nil && len(p.VertexFunc.Block.Stmts) > 0 {
		lines = append(lines, "")
		lines = append(lines,
			fmt.Sprintf("vertex Varyings %s(", VertexName),
			"\tuint vid [[vertex_id]],",
			"\tconst device Attributes* attributes [[buffer(0)]]")
		if len(p.Uniforms) > 0 {
			lines[len(lines)-1] += ","
			lines = append(lines, "\tconstant Uniforms& uniforms [[buffer(1)]]")
		}
		for i := 0; i < p.TextureCount; i++ {
			lines[len(lines)-1] += ","
			lines = append(lines, fmt.Sprintf("\ttexture2d<float> T%[1]d [[texture(%[1]d)]]", i))
		}
		lines[len(lines)-1] += ") {"
		lines = append(lines, fmt.Sprintf("\tVaryings %s = {};", vertexOut))
		lines = append(lines, c.block(p, p.VertexFunc.Block, p.VertexFunc.Block, 0)...)
		if last := fmt.Sprintf("\treturn %s;", vertexOut); lines[len(lines)-1] != last {
			lines = append(lines, last)
		}
		lines = append(lines, "}")
	}

	if p.FragmentFunc.Block != nil && len(p.FragmentFunc.Block.Stmts) > 0 {
		lines = append(lines, "")
		lines = append(lines,
			fmt.Sprintf("fragment float4 %s(", FragmentName),
			"\tVaryings varyings [[stage_in]]")
		if len(p.Uniforms) > 0 {
			lines[len(lines)-1] += ","
			lines = append(lines, "\tconstant Uniforms& uniforms [[buffer(0)]]")
		}
		for i := 0; i < p.TextureCount; i++ {
			lines[len(lines)-1] += ","
			lines = append(lines, fmt.Sprintf("\ttexture2d<float> T%[1]d [[texture(%[1]d)]]", i))
		}
		lines[len(lines)-1] += ","
		lines = append(lines, "\tbool front_facing [[front_facing]]")
		lines[len(lines)-1] += ") {"
		lines = append(lines, c.block(p, p.FragmentFunc.Block, p.FragmentFunc.Block, 0)...)
		lines = append(lines, "}")
	}

	ls := strings.Join(lines, "\n")

	// Struct types are determined after converting the program.
	if len(c.structTypes) > 0 {
		var stlines []string
		for i, t := range c.structTypes {
			stlines = append(stlines, fmt.Sprintf("struct S%d {", i))
			for j, st := range t.Sub {
				stlines = append(stlines, fmt.Sprintf("\t%s;", c.varDecl(p, &st, fmt.Sprintf("M%d", j), false)))
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

func (c *compileContext) typ(p *shaderir.Program, t *shaderir.Type) string {
	switch t.Main {
	case shaderir.None:
		return "void"
	case shaderir.Struct:
		return c.structName(p, t)
	default:
		return typeString(t, false)
	}
}

func (c *compileContext) varDecl(p *shaderir.Program, t *shaderir.Type, varname string, ref bool) string {
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
		t := typeString(t, ref)
		return fmt.Sprintf("%s %s", t, varname)
	}
}

func (c *compileContext) varInit(p *shaderir.Program, t *shaderir.Type) string {
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
	case shaderir.Float, shaderir.Vec2, shaderir.Vec3, shaderir.Vec4,
		shaderir.IVec2, shaderir.IVec3, shaderir.IVec4,
		shaderir.Mat2, shaderir.Mat3, shaderir.Mat4:
		return fmt.Sprintf("%s(0)", basicTypeString(t.Main))
	default:
		t := c.typ(p, t)
		panic(fmt.Sprintf("?(unexpected type: %s)", t))
	}
}

func (c *compileContext) function(p *shaderir.Program, f *shaderir.Func, prototype bool) []string {
	var args []string

	// Uniform variables and texture variables. In Metal, non-const global variables are not available.
	if len(p.Uniforms) > 0 {
		args = append(args, "constant Uniforms& uniforms")
	}
	for i := 0; i < p.TextureCount; i++ {
		args = append(args, fmt.Sprintf("texture2d<float> T%d", i))
	}
	args = append(args, "bool front_facing")

	var idx int
	for _, t := range f.InParams {
		args = append(args, c.varDecl(p, &t, fmt.Sprintf("l%d", idx), false))
		idx++
	}
	for _, t := range f.OutParams {
		args = append(args, "thread "+c.varDecl(p, &t, fmt.Sprintf("l%d", idx), true))
		idx++
	}
	argsstr := "void"
	if len(args) > 0 {
		argsstr = strings.Join(args, ", ")
	}

	t := c.typ(p, &f.Return)
	sig := fmt.Sprintf("%s F%d(%s)", t, f.Index, argsstr)

	var lines []string
	if prototype {
		lines = append(lines, fmt.Sprintf("%s;", sig))
		return lines
	}
	lines = append(lines, fmt.Sprintf("%s {", sig))
	lines = append(lines, c.block(p, f.Block, f.Block, 0)...)
	lines = append(lines, "}")

	return lines
}

func constantToNumberLiteral(v constant.Value) string {
	switch v.Kind() {
	case constant.Bool:
		if constant.BoolVal(v) {
			return "true"
		}
		return "false"
	case constant.Int:
		x, _ := constant.Int64Val(v)
		return fmt.Sprintf("%d", x)
	case constant.Float:
		x, _ := constant.Float64Val(v)
		if i := math.Floor(x); i == x {
			return fmt.Sprintf("%d.0", int64(i))
		}
		return fmt.Sprintf("%.10e", x)
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
			return fmt.Sprintf("%s.Position", vertexOut)
		case idx < nv+1:
			return fmt.Sprintf("%s.M%d", vertexOut, idx-1)
		default:
			return fmt.Sprintf("l%d", idx-(nv+1))
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
		lines = append(lines, fmt.Sprintf("%s%s = %s;", idt, c.varDecl(p, &t, name, false), c.varInit(p, &t)))
	} else {
		lines = append(lines, fmt.Sprintf("%s%s = %s;", idt, name, c.varInit(p, &t)))
	}
	return lines
}

func (c *compileContext) block(p *shaderir.Program, topBlock, block *shaderir.Block, level int) []string {
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

	var expr func(e *shaderir.Expr) string
	expr = func(e *shaderir.Expr) string {
		switch e.Type {
		case shaderir.NumberExpr:
			return constantToNumberLiteral(e.Const)
		case shaderir.UniformVariable:
			return fmt.Sprintf("uniforms.U%d", e.Index)
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
				op = opString(e.Op)
			default:
				op = fmt.Sprintf("?(unexpected op: %d)", e.Op)
			}
			return fmt.Sprintf("%s(%s)", op, expr(&e.Exprs[0]))
		case shaderir.Binary:
			switch e.Op {
			case shaderir.VectorEqualOp:
				return fmt.Sprintf("all((%s) == (%s))", expr(&e.Exprs[0]), expr(&e.Exprs[1]))
			case shaderir.VectorNotEqualOp:
				return fmt.Sprintf("!all((%s) == (%s))", expr(&e.Exprs[0]), expr(&e.Exprs[1]))
			}
			return fmt.Sprintf("(%s) %s (%s)", expr(&e.Exprs[0]), opString(e.Op), expr(&e.Exprs[1]))
		case shaderir.Selection:
			return fmt.Sprintf("(%s) ? (%s) : (%s)", expr(&e.Exprs[0]), expr(&e.Exprs[1]), expr(&e.Exprs[2]))
		case shaderir.Call:
			callee := e.Exprs[0]
			var args []string
			if callee.Type != shaderir.BuiltinFuncExpr {
				if len(p.Uniforms) > 0 {
					args = append(args, "uniforms")
				}
				for i := 0; i < p.TextureCount; i++ {
					args = append(args, fmt.Sprintf("T%d", i))
				}
				args = append(args, "front_facing")
			}
			for _, exp := range e.Exprs[1:] {
				args = append(args, expr(&exp))
			}
			if callee.Type == shaderir.BuiltinFuncExpr && callee.BuiltinFunc == shaderir.TexelAt {
				switch p.Unit {
				case shaderir.Texels:
					return fmt.Sprintf("%s.sample(texture_sampler, %s)", args[0], strings.Join(args[1:], ", "))
				case shaderir.Pixels:
					return fmt.Sprintf("%s.read(static_cast<uint2>(%s))", args[0], strings.Join(args[1:], ", "))
				default:
					panic(fmt.Sprintf("msl: unexpected unit: %d", p.Unit))
				}
			}
			if callee.Type == shaderir.BuiltinFuncExpr && (callee.BuiltinFunc == shaderir.Min || callee.BuiltinFunc == shaderir.Max) {
				result := args[0]
				for i := 1; i < len(args); i++ {
					result = fmt.Sprintf("%s(%s, %s)", expr(&callee), result, args[i])
				}
				return result
			}
			if callee.Type == shaderir.BuiltinFuncExpr && callee.BuiltinFunc == shaderir.FrontFacing {
				return "(front_facing)"
			}
			return fmt.Sprintf("%s(%s)", expr(&callee), strings.Join(args, ", "))
		case shaderir.FieldSelector:
			return fmt.Sprintf("(%s).%s", expr(&e.Exprs[0]), expr(&e.Exprs[1]))
		case shaderir.Index:
			return fmt.Sprintf("(%s)[%s]", expr(&e.Exprs[0]), expr(&e.Exprs[1]))
		default:
			return fmt.Sprintf("?(unexpected expr: %d)", e.Type)
		}
	}

	for _, s := range block.Stmts {
		switch s.Type {
		case shaderir.ExprStmt:
			lines = append(lines, fmt.Sprintf("%s%s;", idt, expr(&s.Exprs[0])))
		case shaderir.BlockStmt:
			lines = append(lines, idt+"{")
			lines = append(lines, c.block(p, topBlock, s.Blocks[0], level+1)...)
			lines = append(lines, idt+"}")
		case shaderir.Assign:
			lines = append(lines, fmt.Sprintf("%s%s = %s;", idt, expr(&s.Exprs[0]), expr(&s.Exprs[1])))
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
			lines = append(lines, fmt.Sprintf("%sif (%s) {", idt, expr(&s.Exprs[0])))
			lines = append(lines, c.block(p, topBlock, s.Blocks[0], level+1)...)
			if len(s.Blocks) > 1 {
				lines = append(lines, fmt.Sprintf("%s} else {", idt))
				lines = append(lines, c.block(p, topBlock, s.Blocks[1], level+1)...)
			}
			lines = append(lines, fmt.Sprintf("%s}", idt))
		case shaderir.For:
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
					delta = fmt.Sprintf("%s += %s", v, constantToNumberLiteral(d))
				} else {
					d = constant.UnaryOp(token.SUB, d, 0)
					delta = fmt.Sprintf("%s -= %s", v, constantToNumberLiteral(d))
				}
			}
			var op string
			switch s.ForOp {
			case shaderir.LessThanOp, shaderir.LessThanEqualOp, shaderir.GreaterThanOp, shaderir.GreaterThanEqualOp, shaderir.EqualOp, shaderir.NotEqualOp:
				op = opString(s.ForOp)
			default:
				op = fmt.Sprintf("?(unexpected op: %d)", s.ForOp)
			}

			t := s.ForVarType
			init := constantToNumberLiteral(s.ForInit)
			end := constantToNumberLiteral(s.ForEnd)
			ts := typeString(&t, false)
			lines = append(lines, fmt.Sprintf("%sfor (%s %s = %s; %s %s %s; %s) {", idt, ts, v, init, v, op, end, delta))
			lines = append(lines, c.block(p, topBlock, s.Blocks[0], level+1)...)
			lines = append(lines, fmt.Sprintf("%s}", idt))
		case shaderir.Continue:
			lines = append(lines, idt+"continue;")
		case shaderir.Break:
			lines = append(lines, idt+"break;")
		case shaderir.Return:
			switch {
			case topBlock == p.VertexFunc.Block:
				lines = append(lines, fmt.Sprintf("%sreturn %s;", idt, vertexOut))
			case len(s.Exprs) == 0:
				lines = append(lines, idt+"return;")
			default:
				lines = append(lines, fmt.Sprintf("%sreturn %s;", idt, expr(&s.Exprs[0])))
			}
		case shaderir.Discard:
			// 'discard' is invoked only in the fragment shader entry point.
			lines = append(lines, idt+"discard_fragment();", idt+"return float4(0.0);")
		default:
			lines = append(lines, fmt.Sprintf("%s?(unexpected stmt: %d)", idt, s.Type))
		}
	}

	return lines
}
