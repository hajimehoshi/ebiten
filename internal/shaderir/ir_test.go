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

package shaderir_test

import (
	"go/constant"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/glsl"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/msl"
)

func block(localVars []shaderir.Type, offset int, stmts ...shaderir.Stmt) *shaderir.Block {
	return &shaderir.Block{
		LocalVars:           localVars,
		LocalVarIndexOffset: offset,
		Stmts:               stmts,
	}
}

func exprStmt(expr shaderir.Expr) shaderir.Stmt {
	return shaderir.Stmt{
		Type:  shaderir.ExprStmt,
		Exprs: []shaderir.Expr{expr},
	}
}

func blockStmt(block *shaderir.Block) shaderir.Stmt {
	return shaderir.Stmt{
		Type:   shaderir.BlockStmt,
		Blocks: []*shaderir.Block{block},
	}
}

func returnStmt(expr shaderir.Expr) shaderir.Stmt {
	return shaderir.Stmt{
		Type:  shaderir.Return,
		Exprs: []shaderir.Expr{expr},
	}
}

func assignStmt(lhs shaderir.Expr, rhs shaderir.Expr) shaderir.Stmt {
	return shaderir.Stmt{
		Type:  shaderir.Assign,
		Exprs: []shaderir.Expr{lhs, rhs},
	}
}

func ifStmt(cond shaderir.Expr, block *shaderir.Block, elseBlock *shaderir.Block) shaderir.Stmt {
	return shaderir.Stmt{
		Type:   shaderir.If,
		Exprs:  []shaderir.Expr{cond},
		Blocks: []*shaderir.Block{block, elseBlock},
	}
}

func forStmt(t shaderir.Type, index, init, end int, op shaderir.Op, delta int, block *shaderir.Block) shaderir.Stmt {
	switch t.Main {
	case shaderir.Int:
		return shaderir.Stmt{
			Type:        shaderir.For,
			Blocks:      []*shaderir.Block{block},
			ForVarType:  t,
			ForVarIndex: index,
			ForInit:     constant.MakeInt64(int64(init)),
			ForEnd:      constant.MakeInt64(int64(end)),
			ForOp:       op,
			ForDelta:    constant.MakeInt64(int64(delta)),
		}
	case shaderir.Float:
		return shaderir.Stmt{
			Type:        shaderir.For,
			Blocks:      []*shaderir.Block{block},
			ForVarType:  t,
			ForVarIndex: index,
			ForInit:     constant.MakeFloat64(float64(init)),
			ForEnd:      constant.MakeFloat64(float64(end)),
			ForOp:       op,
			ForDelta:    constant.MakeFloat64(float64(delta)),
		}
	default:
		panic("not reached")
	}
}

func floatExpr(value float32) shaderir.Expr {
	return shaderir.Expr{
		Type:  shaderir.NumberExpr,
		Const: constant.MakeFloat64(float64(value)),
	}
}

func uniformVariableExpr(index int) shaderir.Expr {
	return shaderir.Expr{
		Type:  shaderir.UniformVariable,
		Index: index,
	}
}

func localVariableExpr(index int) shaderir.Expr {
	return shaderir.Expr{
		Type:  shaderir.LocalVariable,
		Index: index,
	}
}

func builtinFuncExpr(f shaderir.BuiltinFunc) shaderir.Expr {
	return shaderir.Expr{
		Type:        shaderir.BuiltinFuncExpr,
		BuiltinFunc: f,
	}
}

func swizzlingExpr(swizzling string) shaderir.Expr {
	return shaderir.Expr{
		Type:      shaderir.SwizzlingExpr,
		Swizzling: swizzling,
	}
}

func functionExpr(index int) shaderir.Expr {
	return shaderir.Expr{
		Type:  shaderir.FunctionExpr,
		Index: index,
	}
}

func binaryExpr(op shaderir.Op, exprs ...shaderir.Expr) shaderir.Expr {
	return shaderir.Expr{
		Type:  shaderir.Binary,
		Op:    op,
		Exprs: exprs,
	}
}

func selectionExpr(cond, a, b shaderir.Expr) shaderir.Expr {
	return shaderir.Expr{
		Type:  shaderir.Selection,
		Exprs: []shaderir.Expr{cond, a, b},
	}
}

func callExpr(callee shaderir.Expr, args ...shaderir.Expr) shaderir.Expr {
	return shaderir.Expr{
		Type:  shaderir.Call,
		Exprs: append([]shaderir.Expr{callee}, args...),
	}
}

func fieldSelectorExpr(a, b shaderir.Expr) shaderir.Expr {
	return shaderir.Expr{
		Type:  shaderir.FieldSelector,
		Exprs: []shaderir.Expr{a, b},
	}
}

func TestOutput(t *testing.T) {
	glslVertexPrelude := glsl.VertexPrelude(glsl.GLSLVersionDefault) + "\n"
	glslFragmentPrelude := glsl.FragmentPrelude(glsl.GLSLVersionDefault) + "\n"

	tests := []struct {
		Name    string
		Program shaderir.Program
		GlslVS  string
		GlslFS  string
		Metal   string
	}{
		{
			Name: "Empty",
			Program: shaderir.Program{
				Unit: shaderir.Pixels,
			},
			GlslVS: glsl.VertexPrelude(glsl.GLSLVersionDefault),
			GlslFS: glsl.FragmentPrelude(glsl.GLSLVersionDefault),
		},
		{
			Name: "Uniform",
			Program: shaderir.Program{
				Unit: shaderir.Pixels,
				Uniforms: []shaderir.Type{
					{Main: shaderir.Float},
				},
			},
			GlslVS: glslVertexPrelude + `
uniform float U0;`,
			GlslFS: glslFragmentPrelude + `
uniform float U0;`,
		},
		{
			Name: "UniformStruct",
			Program: shaderir.Program{
				Unit: shaderir.Pixels,
				Uniforms: []shaderir.Type{
					{
						Main: shaderir.Struct,
						Sub: []shaderir.Type{
							{Main: shaderir.Float},
						},
					},
				},
			},
			GlslVS: glslVertexPrelude + `
struct S0 {
	float M0;
};

uniform S0 U0;`,
			GlslFS: glslFragmentPrelude + `
struct S0 {
	float M0;
};

uniform S0 U0;`,
		},
		{
			Name: "Vars",
			Program: shaderir.Program{
				Unit: shaderir.Pixels,
				Uniforms: []shaderir.Type{
					{Main: shaderir.Float},
				},
				Attributes: []shaderir.Type{
					{Main: shaderir.Vec2},
				},
				Varyings: []shaderir.Type{
					{Main: shaderir.Vec3},
				},
			},
			GlslVS: glslVertexPrelude + `
uniform float U0;
in vec2 A0;
out vec3 V0;`,
			GlslFS: glslFragmentPrelude + `
uniform float U0;
in vec3 V0;`,
		},
		{
			Name: "Func",
			Program: shaderir.Program{
				Unit: shaderir.Pixels,
				Funcs: []shaderir.Func{
					{
						Index: 0,
					},
				},
			},
			GlslVS: glslVertexPrelude + `
void F0(void);

void F0(void) {
}`,
			GlslFS: glslFragmentPrelude + `
void F0(void);

void F0(void) {
}`,
		},
		{
			Name: "FuncParams",
			Program: shaderir.Program{
				Unit: shaderir.Pixels,
				Funcs: []shaderir.Func{
					{
						Index: 0,
						InParams: []shaderir.Type{
							{Main: shaderir.Float},
							{Main: shaderir.Vec2},
							{Main: shaderir.Vec4},
						},
						OutParams: []shaderir.Type{
							{Main: shaderir.Mat4},
						},
					},
				},
			},
			GlslVS: glslVertexPrelude + `
void F0(in float l0, in vec2 l1, in vec4 l2, out mat4 l3);

void F0(in float l0, in vec2 l1, in vec4 l2, out mat4 l3) {
}`,
			GlslFS: glslFragmentPrelude + `
void F0(in float l0, in vec2 l1, in vec4 l2, out mat4 l3);

void F0(in float l0, in vec2 l1, in vec4 l2, out mat4 l3) {
}`,
		},
		{
			Name: "FuncReturn",
			Program: shaderir.Program{
				Unit: shaderir.Pixels,
				Funcs: []shaderir.Func{
					{
						Index: 0,
						InParams: []shaderir.Type{
							{Main: shaderir.Float},
						},
						Return: shaderir.Type{Main: shaderir.Float},
						Block: block(
							nil,
							1,
							returnStmt(
								localVariableExpr(0),
							),
						),
					},
				},
			},
			GlslVS: glslVertexPrelude + `
float F0(in float l0);

float F0(in float l0) {
	return l0;
}`,
			GlslFS: glslFragmentPrelude + `
float F0(in float l0);

float F0(in float l0) {
	return l0;
}`,
		},
		{
			Name: "FuncLocals",
			Program: shaderir.Program{
				Unit: shaderir.Pixels,
				Funcs: []shaderir.Func{
					{
						Index: 0,
						InParams: []shaderir.Type{
							{Main: shaderir.Float},
						},
						OutParams: []shaderir.Type{
							{Main: shaderir.Float},
						},
						Block: block([]shaderir.Type{
							{Main: shaderir.Mat4},
							{Main: shaderir.Mat4},
						}, 2),
					},
				},
			},
			GlslVS: glslVertexPrelude + `
void F0(in float l0, out float l1);

void F0(in float l0, out float l1) {
	mat4 l2 = mat4(0);
	mat4 l3 = mat4(0);
}`,
			GlslFS: glslFragmentPrelude + `
void F0(in float l0, out float l1);

void F0(in float l0, out float l1) {
	mat4 l2 = mat4(0);
	mat4 l3 = mat4(0);
}`,
		},
		{
			Name: "FuncBlocks",
			Program: shaderir.Program{
				Unit: shaderir.Pixels,
				Funcs: []shaderir.Func{
					{
						Index: 0,
						InParams: []shaderir.Type{
							{Main: shaderir.Float},
						},
						OutParams: []shaderir.Type{
							{Main: shaderir.Float},
						},
						Block: block(
							[]shaderir.Type{
								{Main: shaderir.Mat4},
								{Main: shaderir.Mat4},
							},
							2,
							blockStmt(
								block(
									[]shaderir.Type{
										{Main: shaderir.Mat4},
										{Main: shaderir.Mat4},
									},
									4,
								),
							),
						),
					},
				},
			},
			GlslVS: glslVertexPrelude + `
void F0(in float l0, out float l1);

void F0(in float l0, out float l1) {
	mat4 l2 = mat4(0);
	mat4 l3 = mat4(0);
	{
		mat4 l4 = mat4(0);
		mat4 l5 = mat4(0);
	}
}`,
			GlslFS: glslFragmentPrelude + `
void F0(in float l0, out float l1);

void F0(in float l0, out float l1) {
	mat4 l2 = mat4(0);
	mat4 l3 = mat4(0);
	{
		mat4 l4 = mat4(0);
		mat4 l5 = mat4(0);
	}
}`,
		},
		{
			Name: "Add",
			Program: shaderir.Program{
				Unit: shaderir.Pixels,
				Funcs: []shaderir.Func{
					{
						Index: 0,
						InParams: []shaderir.Type{
							{Main: shaderir.Float},
							{Main: shaderir.Float},
						},
						OutParams: []shaderir.Type{
							{Main: shaderir.Float},
						},
						Block: block(
							nil,
							3,
							assignStmt(
								localVariableExpr(2),
								binaryExpr(
									shaderir.Add,
									localVariableExpr(0),
									localVariableExpr(1),
								),
							),
						),
					},
				},
			},
			GlslVS: glslVertexPrelude + `
void F0(in float l0, in float l1, out float l2);

void F0(in float l0, in float l1, out float l2) {
	l2 = (l0) + (l1);
}`,
			GlslFS: glslFragmentPrelude + `
void F0(in float l0, in float l1, out float l2);

void F0(in float l0, in float l1, out float l2) {
	l2 = (l0) + (l1);
}`,
		},
		{
			Name: "Selection",
			Program: shaderir.Program{
				Unit: shaderir.Pixels,
				Funcs: []shaderir.Func{
					{
						Index: 0,
						InParams: []shaderir.Type{
							{Main: shaderir.Bool},
							{Main: shaderir.Float},
							{Main: shaderir.Float},
						},
						OutParams: []shaderir.Type{
							{Main: shaderir.Float},
						},
						Block: block(
							nil,
							4,
							assignStmt(
								localVariableExpr(3),
								selectionExpr(
									localVariableExpr(0),
									localVariableExpr(1),
									localVariableExpr(2),
								),
							),
						),
					},
				},
			},
			GlslVS: glslVertexPrelude + `
void F0(in bool l0, in float l1, in float l2, out float l3);

void F0(in bool l0, in float l1, in float l2, out float l3) {
	l3 = (l0) ? (l1) : (l2);
}`,
			GlslFS: glslFragmentPrelude + `
void F0(in bool l0, in float l1, in float l2, out float l3);

void F0(in bool l0, in float l1, in float l2, out float l3) {
	l3 = (l0) ? (l1) : (l2);
}`,
		},
		{
			Name: "Call",
			Program: shaderir.Program{
				Unit: shaderir.Pixels,
				Funcs: []shaderir.Func{
					{
						Index: 0,
						InParams: []shaderir.Type{
							{Main: shaderir.Float},
							{Main: shaderir.Float},
						},
						OutParams: []shaderir.Type{
							{Main: shaderir.Vec2},
						},
						Block: block(
							nil,
							3,
							exprStmt(
								callExpr(
									functionExpr(1),
								),
							),
							assignStmt(
								localVariableExpr(2),
								callExpr(
									functionExpr(2),
									localVariableExpr(0),
									localVariableExpr(1),
								),
							),
						),
					},
				},
			},
			GlslVS: glslVertexPrelude + `
void F0(in float l0, in float l1, out vec2 l2);

void F0(in float l0, in float l1, out vec2 l2) {
	F1();
	l2 = F2(l0, l1);
}`,
			GlslFS: glslFragmentPrelude + `
void F0(in float l0, in float l1, out vec2 l2);

void F0(in float l0, in float l1, out vec2 l2) {
	F1();
	l2 = F2(l0, l1);
}`,
		},
		{
			Name: "BuiltinFunc",
			Program: shaderir.Program{
				Unit: shaderir.Pixels,
				Funcs: []shaderir.Func{
					{
						Index: 0,
						InParams: []shaderir.Type{
							{Main: shaderir.Float},
							{Main: shaderir.Float},
						},
						OutParams: []shaderir.Type{
							{Main: shaderir.Float},
						},
						Block: block(
							nil,
							3,
							assignStmt(
								localVariableExpr(2),
								callExpr(
									builtinFuncExpr(shaderir.Min),
									localVariableExpr(0),
									localVariableExpr(1),
								),
							),
						),
					},
				},
			},
			GlslVS: glslVertexPrelude + `
void F0(in float l0, in float l1, out float l2);

void F0(in float l0, in float l1, out float l2) {
	l2 = min(l0, l1);
}`,
			GlslFS: glslFragmentPrelude + `
void F0(in float l0, in float l1, out float l2);

void F0(in float l0, in float l1, out float l2) {
	l2 = min(l0, l1);
}`,
		},
		{
			Name: "FieldSelector",
			Program: shaderir.Program{
				Unit: shaderir.Pixels,
				Funcs: []shaderir.Func{
					{
						Index: 0,
						InParams: []shaderir.Type{
							{Main: shaderir.Vec4},
						},
						OutParams: []shaderir.Type{
							{Main: shaderir.Vec2},
						},
						Block: block(
							nil,
							2,
							assignStmt(
								localVariableExpr(1),
								fieldSelectorExpr(
									localVariableExpr(0),
									swizzlingExpr("xz"),
								),
							),
						),
					},
				},
			},
			GlslVS: glslVertexPrelude + `
void F0(in vec4 l0, out vec2 l1);

void F0(in vec4 l0, out vec2 l1) {
	l1 = (l0).xz;
}`,
			GlslFS: glslFragmentPrelude + `
void F0(in vec4 l0, out vec2 l1);

void F0(in vec4 l0, out vec2 l1) {
	l1 = (l0).xz;
}`,
		},
		{
			Name: "If",
			Program: shaderir.Program{
				Unit: shaderir.Pixels,
				Funcs: []shaderir.Func{
					{
						Index: 0,
						InParams: []shaderir.Type{
							{Main: shaderir.Float},
							{Main: shaderir.Float},
						},
						OutParams: []shaderir.Type{
							{Main: shaderir.Float},
						},
						Block: block(
							nil,
							3,
							ifStmt(
								binaryExpr(
									shaderir.EqualOp,
									localVariableExpr(0),
									floatExpr(0),
								),
								block(
									nil,
									3,
									assignStmt(
										localVariableExpr(2),
										localVariableExpr(0),
									),
								),
								block(
									nil,
									3,
									assignStmt(
										localVariableExpr(2),
										localVariableExpr(1),
									),
								),
							),
						),
					},
				},
			},
			GlslVS: glslVertexPrelude + `
void F0(in float l0, in float l1, out float l2);

void F0(in float l0, in float l1, out float l2) {
	if ((l0) == (0.0)) {
		l2 = l0;
	} else {
		l2 = l1;
	}
}`,
			GlslFS: glslFragmentPrelude + `
void F0(in float l0, in float l1, out float l2);

void F0(in float l0, in float l1, out float l2) {
	if ((l0) == (0.0)) {
		l2 = l0;
	} else {
		l2 = l1;
	}
}`,
		},
		{
			Name: "For",
			Program: shaderir.Program{
				Unit: shaderir.Pixels,
				Funcs: []shaderir.Func{
					{
						Index: 0,
						InParams: []shaderir.Type{
							{Main: shaderir.Float},
							{Main: shaderir.Float},
						},
						OutParams: []shaderir.Type{
							{Main: shaderir.Float},
						},
						Block: block(
							[]shaderir.Type{
								{},
							},
							3,
							forStmt(
								shaderir.Type{Main: shaderir.Int},
								3,
								0,
								100,
								shaderir.LessThanOp,
								1,
								block(
									nil,
									3,
									assignStmt(
										localVariableExpr(2),
										localVariableExpr(0),
									),
								),
							),
						),
					},
				},
			},
			GlslVS: glslVertexPrelude + `
void F0(in float l0, in float l1, out float l2);

void F0(in float l0, in float l1, out float l2) {
	for (int l3 = 0; l3 < 100; l3++) {
		l2 = l0;
	}
}`,
			GlslFS: glslFragmentPrelude + `
void F0(in float l0, in float l1, out float l2);

void F0(in float l0, in float l1, out float l2) {
	for (int l3 = 0; l3 < 100; l3++) {
		l2 = l0;
	}
}`,
		},
		{
			Name: "For2",
			Program: shaderir.Program{
				Unit: shaderir.Pixels,
				Funcs: []shaderir.Func{
					{
						Index: 0,
						InParams: []shaderir.Type{
							{Main: shaderir.Float},
							{Main: shaderir.Float},
						},
						OutParams: []shaderir.Type{
							{Main: shaderir.Float},
						},
						Block: block(
							[]shaderir.Type{
								{},
							},
							3,
							forStmt(
								shaderir.Type{Main: shaderir.Int},
								3,
								0,
								100,
								shaderir.LessThanOp,
								1,
								block(
									[]shaderir.Type{
										{Main: shaderir.Int},
									},
									4,
									assignStmt(
										localVariableExpr(2),
										localVariableExpr(4),
									),
								),
							),
						),
					},
				},
			},
			GlslVS: glslVertexPrelude + `
void F0(in float l0, in float l1, out float l2);

void F0(in float l0, in float l1, out float l2) {
	for (int l3 = 0; l3 < 100; l3++) {
		int l4 = 0;
		l2 = l4;
	}
}`,
			GlslFS: glslFragmentPrelude + `
void F0(in float l0, in float l1, out float l2);

void F0(in float l0, in float l1, out float l2) {
	for (int l3 = 0; l3 < 100; l3++) {
		int l4 = 0;
		l2 = l4;
	}
}`,
			Metal: msl.Prelude(shaderir.Pixels) + `

void F0(bool front_facing, float l0, float l1, thread float& l2);

void F0(bool front_facing, float l0, float l1, thread float& l2) {
	for (int l3 = 0; l3 < 100; l3++) {
		int l4 = 0;
		l2 = l4;
	}
}`,
		},
		{
			Name: "For3",
			Program: shaderir.Program{
				Unit: shaderir.Pixels,
				Funcs: []shaderir.Func{
					{
						Index: 0,
						InParams: []shaderir.Type{
							{Main: shaderir.Float},
							{Main: shaderir.Float},
						},
						OutParams: []shaderir.Type{
							{Main: shaderir.Float},
						},
						Block: block(
							[]shaderir.Type{
								{},
								{},
							},
							3,
							forStmt(
								shaderir.Type{Main: shaderir.Int},
								3,
								0,
								100,
								shaderir.LessThanOp,
								1,
								block(
									[]shaderir.Type{
										{Main: shaderir.Int},
									},
									4,
									assignStmt(
										localVariableExpr(2),
										localVariableExpr(4),
									),
								),
							),
							forStmt(
								shaderir.Type{Main: shaderir.Float},
								4,
								0,
								100,
								shaderir.LessThanOp,
								1,
								block(
									[]shaderir.Type{
										{Main: shaderir.Int},
									},
									5,
									assignStmt(
										localVariableExpr(2),
										localVariableExpr(5),
									),
								),
							),
						),
					},
				},
			},
			GlslVS: glslVertexPrelude + `
void F0(in float l0, in float l1, out float l2);

void F0(in float l0, in float l1, out float l2) {
	for (int l3 = 0; l3 < 100; l3++) {
		int l4 = 0;
		l2 = l4;
	}
	for (float l4 = 0.0; l4 < 100.0; l4++) {
		int l5 = 0;
		l2 = l5;
	}
}`,
			GlslFS: glslFragmentPrelude + `
void F0(in float l0, in float l1, out float l2);

void F0(in float l0, in float l1, out float l2) {
	for (int l3 = 0; l3 < 100; l3++) {
		int l4 = 0;
		l2 = l4;
	}
	for (float l4 = 0.0; l4 < 100.0; l4++) {
		int l5 = 0;
		l2 = l5;
	}
}`,
			Metal: msl.Prelude(shaderir.Pixels) + `

void F0(bool front_facing, float l0, float l1, thread float& l2);

void F0(bool front_facing, float l0, float l1, thread float& l2) {
	for (int l3 = 0; l3 < 100; l3++) {
		int l4 = 0;
		l2 = l4;
	}
	for (float l4 = 0.0; l4 < 100.0; l4++) {
		int l5 = 0;
		l2 = l5;
	}
}`,
		},
		{
			Name: "VertexFunc",
			Program: shaderir.Program{
				Unit: shaderir.Pixels,
				Uniforms: []shaderir.Type{
					{Main: shaderir.Float},
				},
				Attributes: []shaderir.Type{
					{Main: shaderir.Vec4},
					{Main: shaderir.Float},
					{Main: shaderir.Vec2},
				},
				Varyings: []shaderir.Type{
					{Main: shaderir.Float},
					{Main: shaderir.Vec2},
				},
				VertexFunc: shaderir.VertexFunc{
					Block: block(
						nil,
						4+1,
						assignStmt(
							localVariableExpr(3),
							localVariableExpr(0),
						),
						assignStmt(
							localVariableExpr(4),
							localVariableExpr(1),
						),
						assignStmt(
							localVariableExpr(5),
							localVariableExpr(2),
						),
					),
				},
			},
			GlslVS: glslVertexPrelude + `
uniform float U0;
in vec4 A0;
in float A1;
in vec2 A2;
out float V0;
out vec2 V1;

void main(void) {
	gl_Position = A0;
	V0 = A1;
	V1 = A2;
}`,
			GlslFS: glslFragmentPrelude + `
uniform float U0;
in float V0;
in vec2 V1;`,
		},
		{
			Name: "FragmentFunc",
			Program: shaderir.Program{
				Unit: shaderir.Pixels,
				Uniforms: []shaderir.Type{
					{Main: shaderir.Float},
				},
				Attributes: []shaderir.Type{
					{Main: shaderir.Vec4},
					{Main: shaderir.Float},
					{Main: shaderir.Vec2},
				},
				Varyings: []shaderir.Type{
					{Main: shaderir.Float},
					{Main: shaderir.Vec2},
				},
				VertexFunc: shaderir.VertexFunc{
					Block: block(
						nil,
						5+1,
						assignStmt(
							localVariableExpr(3),
							localVariableExpr(0),
						),
						assignStmt(
							localVariableExpr(4),
							localVariableExpr(1),
						),
						assignStmt(
							localVariableExpr(5),
							localVariableExpr(2),
						),
					),
				},
				FragmentFunc: shaderir.FragmentFunc{
					Block: block(
						[]shaderir.Type{
							{Main: shaderir.Vec4},
							{Main: shaderir.Float},
						},
						3,
						assignStmt(
							localVariableExpr(3),
							localVariableExpr(0),
						),
						assignStmt(
							localVariableExpr(4),
							localVariableExpr(1),
						),
						returnStmt(
							callExpr(
								builtinFuncExpr(shaderir.Vec4F),
								localVariableExpr(2),
								localVariableExpr(1),
								localVariableExpr(1),
							),
						),
					),
				},
			},
			GlslVS: glslVertexPrelude + `
uniform float U0;
in vec4 A0;
in float A1;
in vec2 A2;
out float V0;
out vec2 V1;

void main(void) {
	gl_Position = A0;
	V0 = A1;
	V1 = A2;
}`,
			GlslFS: glslFragmentPrelude + `
uniform float U0;
in float V0;
in vec2 V1;

vec4 F0(in vec4 l0, in float l1, in vec2 l2);

vec4 F0(in vec4 l0, in float l1, in vec2 l2) {
	vec4 l3 = vec4(0);
	float l4 = float(0);
	l3 = l0;
	l4 = l1;
	return vec4(l2, l1, l1);
}

void main(void) {
	fragColor = F0(gl_FragCoord, V0, V1);
}`,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			vs, fs := glsl.Compile(&tc.Program, glsl.GLSLVersionDefault)
			{
				got := vs
				want := tc.GlslVS + "\n"
				if got != want {
					t.Errorf("%s vertex: got: %s, want: %s", tc.Name, got, want)
				}
			}
			{
				got := fs
				want := tc.GlslFS + "\n"
				if got != want {
					t.Errorf("%s fragment: got: %s, want: %s", tc.Name, got, want)
				}
			}
			m := msl.Compile(&tc.Program)
			if tc.Metal != "" {
				got := m
				want := tc.Metal + "\n"
				if got != want {
					t.Errorf("%s metal: got: %s, want: %s", tc.Name, got, want)
				}
			}
		})
	}
}
