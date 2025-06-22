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
	"math"
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
	unit        shaderir.Unit
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

const utilFuncs = `float mod(float x, float y) {
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

func Compile(p *shaderir.Program) (vertexShader, pixelShader, vertexPrelude, pixelPrelude string) {
	offsets := UniformVariableOffsetsInDwords(p)

	c := &compileContext{
		unit: p.Unit,
	}

	appendPrelude := func(lines []string, vertex bool) []string {
		lines = append(lines, strings.Split(utilFuncs, "\n")...)

		lines = append(lines, "", "struct Varyings {")
		lines = append(lines, "\tfloat4 Position : SV_POSITION;")
		if len(p.Varyings) > 0 {
			for i, v := range p.Varyings {
				switch i {
				case 0:
					lines = append(lines, fmt.Sprintf("\tfloat2 M%d : TEXCOORD;", i))
				case 1:
					lines = append(lines, fmt.Sprintf("\tfloat4 M%d : COLOR0;", i))
				default:
					// Use COLOR[n] as a general purpose varying.
					if v.Main != shaderir.Vec4 {
						lines = append(lines, fmt.Sprintf("\t?(unexpected type: %s) M%d : COLOR%d;", v, i, i-1))
					} else {
						lines = append(lines, fmt.Sprintf("\tfloat4 M%d : COLOR%d;", i, i-1))
					}
				}
			}
		}
		if !vertex {
			lines = append(lines, "\tbool FrontFacing : SV_IsFrontFace;")
		}
		lines = append(lines, "};")
		return lines
	}

	var vslines, pslines []string
	vslines = appendPrelude(vslines, true)
	pslines = appendPrelude(pslines, false)

	vertexPrelude = strings.Join(vslines, "\n")
	pixelPrelude = strings.Join(pslines, "\n")

	appendGlobalVariables := func(lines []string) []string {
		lines = append(lines, "", "{{.Structs}}")

		if len(p.Uniforms) > 0 {
			lines = append(lines, "")
			lines = append(lines, "cbuffer Uniforms : register(b0) {")
			for i, t := range p.Uniforms {
				// packingoffset is not mandatory, but this is useful to ensure the correct offset is used.
				offset := fmt.Sprintf("c%d", offsets[i]/UniformVariableBoundaryInDwords)
				switch offsets[i] % UniformVariableBoundaryInDwords {
				case 1:
					offset += ".y"
				case 2:
					offset += ".z"
				case 3:
					offset += ".w"
				}
				lines = append(lines, fmt.Sprintf("\t%s : packoffset(%s);", c.varDecl(p, &t, fmt.Sprintf("U%d", i)), offset))
			}
			lines = append(lines, "}")
		}

		if p.TextureCount > 0 {
			lines = append(lines, "")
			for i := 0; i < p.TextureCount; i++ {
				lines = append(lines, fmt.Sprintf("Texture2D T%[1]d : register(t%[1]d);", i))
			}
			if c.unit == shaderir.Texels {
				lines = append(lines, "SamplerState samp : register(s0);")
			}
		}
		return lines
	}
	vslines = appendGlobalVariables(vslines)
	pslines = appendGlobalVariables(pslines)

	var vsfuncs []*shaderir.Func
	if p.VertexFunc.Block != nil {
		vsfuncs = p.ReachableFuncsFromBlock(p.VertexFunc.Block)
	} else {
		// Use all the functions for testing.
		vsfuncs = make([]*shaderir.Func, 0, len(p.Funcs))
		for _, f := range p.Funcs {
			f := f
			vsfuncs = append(vsfuncs, &f)
		}
	}
	if len(vsfuncs) > 0 {
		vslines = append(vslines, "")
		for _, f := range vsfuncs {
			vslines = append(vslines, c.function(p, f, true)...)
		}
		for _, f := range vsfuncs {
			if len(vslines) > 0 && vslines[len(vslines)-1] != "" {
				vslines = append(vslines, "")
			}
			vslines = append(vslines, c.function(p, f, false)...)
		}
	}
	if p.VertexFunc.Block != nil && len(p.VertexFunc.Block.Stmts) > 0 {
		vslines = append(vslines, "")
		var args []string
		for i, a := range p.Attributes {
			switch i {
			case 0:
				args = append(args, fmt.Sprintf("float2 A%d : POSITION", i))
			case 1:
				args = append(args, fmt.Sprintf("float2 A%d : TEXCOORD", i))
			case 2:
				args = append(args, fmt.Sprintf("float4 A%d : COLOR0", i))
			default:
				// Use COLOR[n] as a general purpose varying.
				if a.Main != shaderir.Vec4 {
					args = append(args, fmt.Sprintf("?(unexpected type: %s) A%d : COLOR%d", a, i, i-2))
				} else {
					args = append(args, fmt.Sprintf("float4 A%d : COLOR%d", i, i-2))
				}
			}
		}
		vslines = append(vslines, "Varyings VSMain("+strings.Join(args, ", ")+") {")
		vslines = append(vslines, fmt.Sprintf("\tVaryings %s;", vsOut))
		vslines = append(vslines, c.block(p, p.VertexFunc.Block, p.VertexFunc.Block, 0)...)
		if last := fmt.Sprintf("\treturn %s;", vsOut); vslines[len(vslines)-1] != last {
			vslines = append(vslines, last)
		}
		vslines = append(vslines, "}")
	}

	var psfuncs []*shaderir.Func
	if p.FragmentFunc.Block != nil {
		psfuncs = p.ReachableFuncsFromBlock(p.FragmentFunc.Block)
	} else {
		// Use all the functions for testing.
		psfuncs = make([]*shaderir.Func, 0, len(p.Funcs))
		for _, f := range p.Funcs {
			f := f
			psfuncs = append(psfuncs, &f)
		}
	}
	if len(psfuncs) > 0 {
		pslines = append(pslines, "")
		for _, f := range psfuncs {
			pslines = append(pslines, c.function(p, f, true)...)
		}
		for _, f := range psfuncs {
			if len(pslines) > 0 && pslines[len(pslines)-1] != "" {
				pslines = append(pslines, "")
			}
			pslines = append(pslines, c.function(p, f, false)...)
		}
	}
	if p.FragmentFunc.Block != nil && len(p.FragmentFunc.Block.Stmts) > 0 {
		pslines = append(pslines, "")
		pslines = append(pslines, fmt.Sprintf("float4 PSMain(Varyings %s) : SV_TARGET {", vsOut))
		pslines = append(pslines, c.block(p, p.FragmentFunc.Block, p.FragmentFunc.Block, 0)...)
		pslines = append(pslines, "}")
	}

	vertexShader = strings.Join(vslines, "\n")
	pixelShader = strings.Join(pslines, "\n")

	// Struct types are determined after converting the program.
	shaders := []string{vertexShader, pixelShader}
	for i, shader := range shaders {
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
			shader = strings.ReplaceAll(shader, "{{.Structs}}", st)
		} else {
			shader = strings.ReplaceAll(shader, "{{.Structs}}", "")
		}

		nls := regexp.MustCompile(`\n\n+`)
		shader = nls.ReplaceAllString(shader, "\n\n")

		shader = strings.TrimSpace(shader) + "\n"

		shaders[i] = shader
	}

	vertexShader = shaders[0]
	pixelShader = shaders[1]

	return
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
	case shaderir.Int, shaderir.IVec2, shaderir.IVec3, shaderir.IVec4:
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
			return constantToNumberLiteral(e.Const)
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
				return fmt.Sprintf("all((%s) == (%s))", expr(&e.Exprs[0]), expr(&e.Exprs[1]))
			case shaderir.VectorNotEqualOp:
				return fmt.Sprintf("!all((%s) == (%s))", expr(&e.Exprs[0]), expr(&e.Exprs[1]))
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
				case shaderir.Vec2F, shaderir.Vec3F, shaderir.Vec4F, shaderir.IVec2F, shaderir.IVec3F, shaderir.IVec4F:
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
				case shaderir.TexelAt:
					switch c.unit {
					case shaderir.Pixels:
						return fmt.Sprintf("%s.Load(int3(%s, 0))", args[0], strings.Join(args[1:], ", "))
					case shaderir.Texels:
						return fmt.Sprintf("%s.Sample(samp, %s)", args[0], strings.Join(args[1:], ", "))
					default:
						panic(fmt.Sprintf("hlsl: unexpected unit: %d", p.Unit))
					}
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
				return fmt.Sprintf("%s.FrontFacing", vsOut)
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
