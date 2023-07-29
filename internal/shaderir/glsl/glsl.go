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
	"math"
	"regexp"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

type GLSLVersion int

const (
	GLSLVersionDefault GLSLVersion = iota
	GLSLVersionES300
)

// utilFunctions is GLSL utility functions for old GLSL versions.
const utilFunctions = `int modInt(int x, int y) {
	return x - y*(x/y);
}

ivec2 modInt(ivec2 x, int y) {
	return x - y*(x/y);
}

ivec3 modInt(ivec3 x, int y) {
	return x - y*(x/y);
}

ivec4 modInt(ivec4 x, int y) {
	return x - y*(x/y);
}

ivec2 modInt(ivec2 x, ivec2 y) {
	return x - y*(x/y);
}

ivec3 modInt(ivec3 x, ivec3 y) {
	return x - y*(x/y);
}

ivec4 modInt(ivec4 x, ivec4 y) {
	return x - y*(x/y);
}`

func VertexPrelude(version GLSLVersion) string {
	switch version {
	case GLSLVersionDefault:
		return `#version 150` + "\n\n" + utilFunctions
	case GLSLVersionES300:
		return `#version 300 es`
	}
	return ""
}

func FragmentPrelude(version GLSLVersion) string {
	var prefix string
	switch version {
	case GLSLVersionDefault:
		prefix = `#version 150` + "\n\n"
	case GLSLVersionES300:
		prefix = `#version 300 es` + "\n\n"
	}
	prelude := prefix + `#if defined(GL_ES)
precision highp float;
precision highp int;
#else
#define lowp
#define mediump
#define highp
#endif

out vec4 fragColor;`
	if version == GLSLVersionDefault {
		prelude += "\n\n" + utilFunctions
	}
	return prelude
}

type compileContext struct {
	version     GLSLVersion
	structNames map[string]string
	structTypes []shaderir.Type
	unit        shaderir.Unit
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

func Compile(p *shaderir.Program, version GLSLVersion) (vertexShader, fragmentShader string) {
	p = adjustProgram(p)

	c := &compileContext{
		version:     version,
		structNames: map[string]string{},
		unit:        p.Unit,
	}

	// Vertex func
	var vslines []string
	{
		vslines = append(vslines, strings.Split(VertexPrelude(version), "\n")...)
		vslines = append(vslines, "", "{{.Structs}}")
		if len(p.Uniforms) > 0 || p.TextureCount > 0 || len(p.Attributes) > 0 || len(p.Varyings) > 0 {
			vslines = append(vslines, "")
			for i, t := range p.Uniforms {
				vslines = append(vslines, fmt.Sprintf("uniform %s;", c.varDecl(p, &t, fmt.Sprintf("U%d", i))))
			}
			for i := 0; i < p.TextureCount; i++ {
				vslines = append(vslines, fmt.Sprintf("uniform sampler2D T%d;", i))
			}
			for i, t := range p.Attributes {
				vslines = append(vslines, fmt.Sprintf("in %s;", c.varDecl(p, &t, fmt.Sprintf("A%d", i))))
			}
			for i, t := range p.Varyings {
				vslines = append(vslines, fmt.Sprintf("out %s;", c.varDecl(p, &t, fmt.Sprintf("V%d", i))))
			}
		}

		var funcs []*shaderir.Func
		if p.VertexFunc.Block != nil {
			funcs = p.ReachableFuncsFromBlock(p.VertexFunc.Block)
		} else {
			// When a vertex entry point is not defined, allow to put all the functions. This is useful for testing.
			funcs = make([]*shaderir.Func, 0, len(p.Funcs))
			for _, f := range p.Funcs {
				f := f
				funcs = append(funcs, &f)
			}
		}
		if len(funcs) > 0 {
			vslines = append(vslines, "")
			for _, f := range funcs {
				vslines = append(vslines, c.function(p, f, true)...)
			}
			for _, f := range funcs {
				if len(vslines) > 0 && vslines[len(vslines)-1] != "" {
					vslines = append(vslines, "")
				}
				vslines = append(vslines, c.function(p, f, false)...)
			}
		}

		// Add a dummy function to just touch uniform array variable's elements (#1754).
		// Without this, the first elements of a uniform array might not be initialized correctly on some environments.
		var touchedUniforms []string
		for i, t := range p.Uniforms {
			if t.Main != shaderir.Array {
				continue
			}
			if t.Length <= 1 {
				continue
			}
			str := fmt.Sprintf("U%d[%d]", i, t.Length-1)
			switch t.Sub[0].Main {
			case shaderir.Vec2, shaderir.Vec3, shaderir.Vec4, shaderir.IVec2, shaderir.IVec3, shaderir.IVec4:
				str += ".x"
			case shaderir.Mat2, shaderir.Mat3, shaderir.Mat4:
				str += "[0][0]"
			}
			str = "float(" + str + ")"
			touchedUniforms = append(touchedUniforms, str)
		}

		var touchUniformsFunc []string
		if len(touchedUniforms) > 0 {
			touchUniformsFunc = append(touchUniformsFunc, "float touchUniforms() {")
			touchUniformsFunc = append(touchUniformsFunc, fmt.Sprintf("\treturn %s;", strings.Join(touchedUniforms, " + ")))
			touchUniformsFunc = append(touchUniformsFunc, "}")

		}

		if p.VertexFunc.Block != nil && len(p.VertexFunc.Block.Stmts) > 0 {
			if len(touchUniformsFunc) > 0 {
				vslines = append(vslines, "")
				vslines = append(vslines, touchUniformsFunc...)
			}
			vslines = append(vslines, "")
			vslines = append(vslines, "void main(void) {")
			if len(touchUniformsFunc) > 0 {
				vslines = append(vslines, "\ttouchUniforms();")
			}
			vslines = append(vslines, c.block(p, p.VertexFunc.Block, p.VertexFunc.Block, 0)...)
			vslines = append(vslines, "}")
		}
	}

	// Fragment func
	var fslines []string
	{
		fslines = append(fslines, strings.Split(FragmentPrelude(version), "\n")...)
		fslines = append(fslines, "", "{{.Structs}}")
		if len(p.Uniforms) > 0 || p.TextureCount > 0 || len(p.Varyings) > 0 {
			fslines = append(fslines, "")
			for i, t := range p.Uniforms {
				fslines = append(fslines, fmt.Sprintf("uniform %s;", c.varDecl(p, &t, fmt.Sprintf("U%d", i))))
			}
			for i := 0; i < p.TextureCount; i++ {
				fslines = append(fslines, fmt.Sprintf("uniform sampler2D T%d;", i))
			}
			for i, t := range p.Varyings {
				fslines = append(fslines, fmt.Sprintf("in %s;", c.varDecl(p, &t, fmt.Sprintf("V%d", i))))
			}
		}

		var funcs []*shaderir.Func
		if p.VertexFunc.Block != nil {
			funcs = p.ReachableFuncsFromBlock(p.FragmentFunc.Block)
		} else {
			// When a fragment entry point is not defined, allow to put all the functions. This is useful for testing.
			funcs = make([]*shaderir.Func, 0, len(p.Funcs))
			for _, f := range p.Funcs {
				f := f
				funcs = append(funcs, &f)
			}
		}
		if len(funcs) > 0 {
			fslines = append(fslines, "")
			for _, f := range funcs {
				fslines = append(fslines, c.function(p, f, true)...)
			}
			for _, f := range funcs {
				if len(fslines) > 0 && fslines[len(fslines)-1] != "" {
					fslines = append(fslines, "")
				}
				fslines = append(fslines, c.function(p, f, false)...)
			}
		}

		if p.FragmentFunc.Block != nil && len(p.FragmentFunc.Block.Stmts) > 0 {
			fslines = append(fslines, "")
			fslines = append(fslines, "void main(void) {")
			fslines = append(fslines, c.block(p, p.FragmentFunc.Block, p.FragmentFunc.Block, 0)...)
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
				stlines = append(stlines, fmt.Sprintf("\t%s;", c.varDecl(p, &st, fmt.Sprintf("M%d", j))))
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
	case shaderir.Float, shaderir.Vec2, shaderir.Vec3, shaderir.Vec4,
		shaderir.IVec2, shaderir.IVec3, shaderir.IVec4,
		shaderir.Mat2, shaderir.Mat3, shaderir.Mat4:
		return fmt.Sprintf("%s(0)", basicTypeString(t.Main))
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
			if e.Op == shaderir.ModOp && c.version == GLSLVersionDefault {
				// '%' is not defined.
				return fmt.Sprintf("modInt((%s), (%s))", expr(&e.Exprs[0]), expr(&e.Exprs[1]))
			}
			return fmt.Sprintf("(%s) %s (%s)", expr(&e.Exprs[0]), opString(e.Op), expr(&e.Exprs[1]))
		case shaderir.Selection:
			return fmt.Sprintf("(%s) ? (%s) : (%s)", expr(&e.Exprs[0]), expr(&e.Exprs[1]), expr(&e.Exprs[2]))
		case shaderir.Call:
			var args []string
			for _, exp := range e.Exprs[1:] {
				args = append(args, expr(&exp))
			}
			f := expr(&e.Exprs[0])
			if f == "texelFetch" {
				return fmt.Sprintf("%s(%s, ivec2(%s), 0)", f, args[0], args[1])
			}
			// Using parentheses at the callee is illegal.
			return fmt.Sprintf("%s(%s)", f, strings.Join(args, ", "))
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
			case topBlock == p.FragmentFunc.Block:
				lines = append(lines, fmt.Sprintf("%sfragColor = %s;", idt, expr(&s.Exprs[0])))
				// The 'return' statement is not required so far, as the fragment entrypoint has only one sentence so far. See adjustProgram implementation.
			case len(s.Exprs) == 0:
				lines = append(lines, idt+"return;")
			default:
				lines = append(lines, fmt.Sprintf("%sreturn %s;", idt, expr(&s.Exprs[0])))
			}
		case shaderir.Discard:
			// 'discard' is invoked only in the fragment shader entry point.
			lines = append(lines, idt+"discard;", idt+"return vec4(0.0);")
		default:
			lines = append(lines, fmt.Sprintf("%s?(unexpected stmt: %d)", idt, s.Type))
		}
	}

	return lines
}

func adjustProgram(p *shaderir.Program) *shaderir.Program {
	if p.FragmentFunc.Block == nil {
		return p
	}

	// Shallow-clone the program in order not to modify p itself.
	newP := *p

	// Create a new slice not to affect the original p.
	newP.Funcs = make([]shaderir.Func, len(p.Funcs))
	copy(newP.Funcs, p.Funcs)

	// Create a new function whose body is the same is the fragment shader's entry point.
	// The entry point will call this.
	// This indirect call is needed for these issues:
	// - Assignment to gl_FragColor doesn't work (#2245)
	// - There are some odd compilers that don't work with early returns and gl_FragColor (#2247)

	// Determine a unique index of the new function.
	var funcIdx int
	for _, f := range newP.Funcs {
		if funcIdx <= f.Index {
			funcIdx = f.Index + 1
		}
	}

	// For parameters of a fragment func, see the comment in internal/shaderir/program.go.
	inParams := make([]shaderir.Type, 1+len(newP.Varyings))
	inParams[0] = shaderir.Type{
		Main: shaderir.Vec4, // gl_FragCoord
	}
	copy(inParams[1:], newP.Varyings)

	newP.Funcs = append(newP.Funcs, shaderir.Func{
		Index:     funcIdx,
		InParams:  inParams,
		OutParams: nil,
		Return: shaderir.Type{
			Main: shaderir.Vec4,
		},
		Block: newP.FragmentFunc.Block,
	})

	// Create an AST to call the new function.
	call := []shaderir.Expr{
		{
			Type:  shaderir.FunctionExpr,
			Index: funcIdx,
		},
	}
	for i := 0; i < 1+len(newP.Varyings); i++ {
		call = append(call, shaderir.Expr{
			Type:  shaderir.LocalVariable,
			Index: i,
		})
	}

	// Replace the entry point with just calling the new function.
	stmts := []shaderir.Stmt{
		{
			// Return: This will be replaced with assignment to gl_FragColor.
			Type: shaderir.Return,
			Exprs: []shaderir.Expr{
				// The function call
				{
					Type:  shaderir.Call,
					Exprs: call,
				},
			},
		},
	}
	newP.FragmentFunc = shaderir.FragmentFunc{
		Block: &shaderir.Block{
			LocalVars:           nil,
			LocalVarIndexOffset: 1 + len(newP.Varyings) + 1,
			Stmts:               stmts,
		},
	}

	return &newP
}
