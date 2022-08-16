// Copyright 2022 The Ebiten Authors
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

package hlsl

import (
	"fmt"
	"go/constant"
	"go/token"
	"regexp"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

const (
	vsOut = "varyings"
)

type compileContext struct {
	structNames map[string]string
	structTypes []shaderir.Type
}

func (c *compileContext) structName(p *shaderir.Program, t *shaderir.Type) string {
	if t.Main != shaderir.Struct {
		panic("hlsl: the given type at structName must be a struct")
	}
	s := t.String()
	if n, ok := c.structNames[s]; ok {
		return n
	}
	n := fmt.Sprintf("S%d", len(c.structNames))
	if c.structNames == nil {
		c.structNames = map[string]string{}
	}
	c.structNames[s] = n
	c.structTypes = append(c.structTypes, *t)
	return n
}

const Prelude = `struct Varyings {
	float4 Position : SV_POSITION;
	float2 M0 : TEXCOORD0;
	float4 M1 : COLOR;
};

float mod(float x, float y) {
	return x - y * floor(x/y);
}

float2 mod(float2 x, float2 y) {
	return x - y * floor(x/y);
}

float3 mod(float3 x, float3 y) {
	return x - y * floor(x/y);
}

float4 mod(float4 x, float4 y) {
	return x - y * floor(x/y);
}

float2x2 float2x2FromScalar(float x) {
	return float2x2(x, 0, 0, x);
}

float3x3 float3x3FromScalar(float x) {
	return float3x3(x, 0, 0, 0, x, 0, 0, 0, x);
}

float4x4 float4x4FromScalar(float x) {
	return float4x4(x, 0, 0, 0, 0, x, 0, 0, 0, 0, x, 0, 0, 0, 0, x);
}`

func Compile(p *shaderir.Program) (string, []int) {
	c := &compileContext{}

	var lines []string
	lines = append(lines, strings.Split(Prelude, "\n")...)
	lines = append(lines, "", "{{.Structs}}")

	var offsets []int
	if len(p.Uniforms) > 0 {
		offsets = calculateMemoryOffsets(p.Uniforms)
		lines = append(lines, "")
		lines = append(lines, "cbuffer Uniforms : register(b0) {")
		for i, t := range p.Uniforms {
			// packingoffset is not mandatory, but this is useful to ensure the correct offset is used.
			offset := fmt.Sprintf("c%d", offsets[i]/boundaryInBytes)
			switch offsets[i] % boundaryInBytes {
			case 4:
				offset += ".y"
			case 8:
				offset += ".z"
			case 12:
				offset += ".w"
			}
			lines = append(lines, fmt.Sprintf("\t%s : packoffset(%s);", c.varDecl(p, &t, fmt.Sprintf("U%d", i)), offset))
		}
		lines = append(lines, "}")
	}
	if p.TextureNum > 0 {
		lines = append(lines, "")
		for i := 0; i < p.TextureNum; i++ {
			lines = append(lines, fmt.Sprintf("Texture2D T%[1]d : register(t%[1]d);", i))
		}
		lines = append(lines, "SamplerState samp : register(s0);")
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
		lines = append(lines, "Varyings VSMain(float2 A0 : POSITION, float2 A1 : TEXCOORD, float4 A2 : COLOR) {")
		lines = append(lines, fmt.Sprintf("\tVaryings %s;", vsOut))
		lines = append(lines, c.block(p, p.VertexFunc.Block, p.VertexFunc.Block, 0)...)
		if last := fmt.Sprintf("\treturn %s;", vsOut); lines[len(lines)-1] != last {
			lines = append(lines, last)
		}
		lines = append(lines, "}")
	}

	if p.FragmentFunc.Block != nil && len(p.FragmentFunc.Block.Stmts) > 0 {
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("float4 PSMain(Varyings %s) : SV_TARGET {", vsOut))
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
				stlines = append(stlines, fmt.Sprintf("\t%s;", c.varDecl(p, &st, fmt.Sprintf("M%d", j))))
			}
			stlines = append(stlines, "};")
		}
		st := strings.Join(stlines, "\n")
		ls = strings.ReplaceAll(ls, "{{.Structs}}", st)
	} else {
		ls = strings.ReplaceAll(ls, "{{.Structs}}", "")
	}

	nls := regexp.MustCompile(`\n\n+`)
	ls = nls.ReplaceAllString(ls, "\n\n")

	ls = strings.TrimSpace(ls) + "\n"

	return ls, offsets
}

func (c *compileContext) typ(p *shaderir.Program, t *shaderir.Type) (string, string) {
	switch t.Main {
	case shaderir.None:
		return "void", ""
	case shaderir.Struct:
		return c.structName(p, t), ""
	default:
		return typeString(t)
	}
}

func (c *compileContext) varDecl(p *shaderir.Program, t *shaderir.Type, varname string) string {
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

func (c *compileContext) varInit(p *shaderir.Program, t *shaderir.Type) string {
	switch t.Main {
	case shaderir.None:
		return "?(none)"
	case shaderir.Array:
		init := c.varInit(p, &t.Sub[0])
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
		return "0.0"
	default:
		t0, t1 := c.typ(p, t)
		panic(fmt.Sprintf("?(unexpected type: %s%s)", t0, t1))
	}
}

func (c *compileContext) function(p *shaderir.Program, f *shaderir.Func, prototype bool) []string {
	var args []string
	var idx int
	for _, t := range f.InParams {
		args = append(args, "in "+c.varDecl(p, &t, fmt.Sprintf("l%d", idx)))
		idx++
	}
	for _, t := range f.OutParams {
		args = append(args, "out "+c.varDecl(p, &t, fmt.Sprintf("l%d", idx)))
		idx++
	}
	argsstr := "void"
	if len(args) > 0 {
		argsstr = strings.Join(args, ", ")
	}

	t0, t1 := c.typ(p, &f.Return)
	sig := fmt.Sprintf("%s%s F%d(%s)", t0, t1, f.Index, argsstr)

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

func (c *compileContext) localVariableName(p *shaderir.Program, topBlock *shaderir.Block, idx int) string {
	switch topBlock {
	case p.VertexFunc.Block:
		na := len(p.Attributes)
		nv := len(p.Varyings)
		switch {
		case idx < na:
			return fmt.Sprintf("A%d", idx)
		case idx == na:
			return fmt.Sprintf("%s.Position", vsOut)
		case idx < na+nv+1:
			return fmt.Sprintf("%s.M%d", vsOut, idx-na-1)
		default:
			return fmt.Sprintf("l%d", idx-(na+nv+1))
		}
	case p.FragmentFunc.Block:
		nv := len(p.Varyings)
		switch {
		case idx == 0:
			return fmt.Sprintf("%s.Position", vsOut)
		case idx < nv+1:
			return fmt.Sprintf("%s.M%d", vsOut, idx-1)
		default:
			return fmt.Sprintf("l%d", idx-(nv+1))
		}
	default:
		return fmt.Sprintf("l%d", idx)
	}
}

func (c *compileContext) initVariable(p *shaderir.Program, topBlock, block *shaderir.Block, index int, decl bool, level int) []string {
	idt := strings.Repeat("\t", level+1)
	name := c.localVariableName(p, topBlock, index)
	t := p.LocalVariableType(topBlock, block, index)

	var lines []string
	switch t.Main {
	case shaderir.Array:
		if decl {
			lines = append(lines, fmt.Sprintf("%s%s;", idt, c.varDecl(p, &t, name)))
		}
		init := c.varInit(p, &t.Sub[0])
		for i := 0; i < t.Length; i++ {
			lines = append(lines, fmt.Sprintf("%s%s[%d] = %s;", idt, name, i, init))
		}
	case shaderir.None:
		// The type is None e.g., when the variable is a for-loop counter.
	default:
		if decl {
			lines = append(lines, fmt.Sprintf("%s%s = %s;", idt, c.varDecl(p, &t, name), c.varInit(p, &t)))
		} else {
			lines = append(lines, fmt.Sprintf("%s%s = %s;", idt, name, c.varInit(p, &t)))
		}
	}
	return lines
}

func (c *compileContext) block(p *shaderir.Program, topBlock, block *shaderir.Block, level int) []string {
	if block == nil {
		return nil
	}

	var lines []string
	for i := range block.LocalVars {
		lines = append(lines, c.initVariable(p, topBlock, block, block.LocalVarIndexOffset+i, true, level)...)
	}

	var expr func(e *shaderir.Expr) string
	expr = func(e *shaderir.Expr) string {
		switch e.Type {
		case shaderir.NumberExpr:
			return constantToNumberLiteral(e.ConstType, e.Const)
		case shaderir.UniformVariable:
			return fmt.Sprintf("U%d", e.Index)
		case shaderir.TextureVariable:
			return fmt.Sprintf("T%d", e.Index)
		case shaderir.LocalVariable:
			return c.localVariableName(p, topBlock, e.Index)
		case shaderir.StructMember:
			return fmt.Sprintf("M%d", e.Index)
		case shaderir.BuiltinFuncExpr:
			return c.builtinFuncString(e.BuiltinFunc)
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
				return fmt.Sprintf("all(%s == %s)", expr(&e.Exprs[0]), expr(&e.Exprs[1]))
			case shaderir.VectorNotEqualOp:
				return fmt.Sprintf("!all(%s == %s)", expr(&e.Exprs[0]), expr(&e.Exprs[1]))
			case shaderir.MatrixMul:
				// If either is a matrix, use the mul function.
				// Swap the order of the lhs and the rhs since matrices are row-major in HLSL.
				return fmt.Sprintf("mul(%s, %s)", expr(&e.Exprs[1]), expr(&e.Exprs[0]))
			}
			return fmt.Sprintf("(%s) %s (%s)", expr(&e.Exprs[0]), opString(e.Op), expr(&e.Exprs[1]))
		case shaderir.Selection:
			return fmt.Sprintf("(%s) ? (%s) : (%s)", expr(&e.Exprs[0]), expr(&e.Exprs[1]), expr(&e.Exprs[2]))
		case shaderir.Call:
			callee := e.Exprs[0]
			var args []string
			for _, exp := range e.Exprs[1:] {
				args = append(args, expr(&exp))
			}
			if callee.Type == shaderir.BuiltinFuncExpr {
				switch callee.BuiltinFunc {
				case shaderir.Vec2F, shaderir.Vec3F, shaderir.Vec4F:
					if len(args) == 1 {
						// Use casting. For example, `float4(1)` doesn't work.
						return fmt.Sprintf("(%s)(%s)", expr(&e.Exprs[0]), args[0])
					}
				case shaderir.Mat2F:
					if len(args) == 1 {
						// In HSLS, casting a scalar to a matrix initializes all the components.
						// There seems no easy way to have an identity matrix.
						return fmt.Sprintf("float2x2FromScalar(%s)", args[0])
					}
				case shaderir.Mat3F:
					if len(args) == 1 {
						return fmt.Sprintf("float3x3FromScalar(%s)", args[0])
					}
				case shaderir.Mat4F:
					if len(args) == 1 {
						return fmt.Sprintf("float4x4FromScalar(%s)", args[0])
					}
				case shaderir.Texture2DF:
					return fmt.Sprintf("%s.Sample(samp, %s)", args[0], strings.Join(args[1:], ", "))
				}
			}
			return fmt.Sprintf("%s(%s)", expr(&e.Exprs[0]), strings.Join(args, ", "))
		case shaderir.FieldSelector:
			return fmt.Sprintf("(%s).%s", expr(&e.Exprs[0]), expr(&e.Exprs[1]))
		case shaderir.Index:
			return fmt.Sprintf("(%s)[%s]", expr(&e.Exprs[0]), expr(&e.Exprs[1]))
		default:
			return fmt.Sprintf("?(unexpected expr: %d)", e.Type)
		}
	}

	idt := strings.Repeat("\t", level+1)
	for _, s := range block.Stmts {
		switch s.Type {
		case shaderir.ExprStmt:
			lines = append(lines, fmt.Sprintf("%s%s;", idt, expr(&s.Exprs[0])))
		case shaderir.BlockStmt:
			lines = append(lines, idt+"{")
			lines = append(lines, c.block(p, topBlock, s.Blocks[0], level+1)...)
			lines = append(lines, idt+"}")
		case shaderir.Assign:
			lhs := s.Exprs[0]
			rhs := s.Exprs[1]
			if lhs.Type == shaderir.LocalVariable {
				if t := p.LocalVariableType(topBlock, block, lhs.Index); t.Main == shaderir.Array {
					for i := 0; i < t.Length; i++ {
						lines = append(lines, fmt.Sprintf("%[1]s%[2]s[%[3]d] = %[4]s[%[3]d];", idt, expr(&lhs), i, expr(&rhs)))
					}
					continue
				}
			}
			lines = append(lines, fmt.Sprintf("%s%s = %s;", idt, expr(&lhs), expr(&rhs)))
		case shaderir.Init:
			lines = append(lines, c.initVariable(p, topBlock, block, s.InitIndex, false, level)...)
		case shaderir.If:
			lines = append(lines, fmt.Sprintf("%sif (%s) {", idt, expr(&s.Exprs[0])))
			lines = append(lines, c.block(p, topBlock, s.Blocks[0], level+1)...)
			if len(s.Blocks) > 1 {
				lines = append(lines, fmt.Sprintf("%s} else {", idt))
				lines = append(lines, c.block(p, topBlock, s.Blocks[1], level+1)...)
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

			v := c.localVariableName(p, topBlock, s.ForVarIndex)
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
				op = opString(s.ForOp)
			default:
				op = fmt.Sprintf("?(unexpected op: %d)", s.ForOp)
			}

			t := s.ForVarType
			init := constantToNumberLiteral(ct, s.ForInit)
			end := constantToNumberLiteral(ct, s.ForEnd)
			t0, t1 := typeString(&t)
			lines = append(lines, fmt.Sprintf("%sfor (%s %s%s = %s; %s %s %s; %s) {", idt, t0, v, t1, init, v, op, end, delta))
			lines = append(lines, c.block(p, topBlock, s.Blocks[0], level+1)...)
			lines = append(lines, fmt.Sprintf("%s}", idt))
		case shaderir.Continue:
			lines = append(lines, idt+"continue;")
		case shaderir.Break:
			lines = append(lines, idt+"break;")
		case shaderir.Return:
			switch {
			case topBlock == p.VertexFunc.Block:
				lines = append(lines, fmt.Sprintf("%sreturn %s;", idt, vsOut))
			case len(s.Exprs) == 0:
				lines = append(lines, idt+"return;")
			default:
				lines = append(lines, fmt.Sprintf("%sreturn %s;", idt, expr(&s.Exprs[0])))
			}
		case shaderir.Discard:
			// 'discard' is invoked only in the fragment shader entry point.
			lines = append(lines, idt+"discard;", idt+"return float4(0.0, 0.0, 0.0, 0.0);")
		default:
			lines = append(lines, fmt.Sprintf("%s?(unexpected stmt: %d)", idt, s.Type))
		}
	}

	return lines
}
