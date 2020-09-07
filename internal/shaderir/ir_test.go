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

	. "github.com/hajimehoshi/ebiten/internal/shaderir"
	"github.com/hajimehoshi/ebiten/internal/shaderir/glsl"
	"github.com/hajimehoshi/ebiten/internal/shaderir/metal"
)

func block(localVars []Type, offset int, stmts ...Stmt) *Block {
	return &Block{
		LocalVars:           localVars,
		LocalVarIndexOffset: offset,
		Stmts:               stmts,
	}
}

func exprStmt(expr Expr) Stmt {
	return Stmt{
		Type:  ExprStmt,
		Exprs: []Expr{expr},
	}
}

func blockStmt(block *Block) Stmt {
	return Stmt{
		Type:   BlockStmt,
		Blocks: []*Block{block},
	}
}

func returnStmt(expr Expr) Stmt {
	return Stmt{
		Type:  Return,
		Exprs: []Expr{expr},
	}
}

func assignStmt(lhs Expr, rhs Expr) Stmt {
	return Stmt{
		Type:  Assign,
		Exprs: []Expr{lhs, rhs},
	}
}

func ifStmt(cond Expr, block *Block, elseBlock *Block) Stmt {
	return Stmt{
		Type:   If,
		Exprs:  []Expr{cond},
		Blocks: []*Block{block, elseBlock},
	}
}

func forStmt(t Type, index, init, end int, op Op, delta int, block *Block) Stmt {
	return Stmt{
		Type:        For,
		Blocks:      []*Block{block},
		ForVarType:  t,
		ForVarIndex: index,
		ForInit:     constant.MakeInt64(int64(init)),
		ForEnd:      constant.MakeInt64(int64(end)),
		ForOp:       op,
		ForDelta:    constant.MakeInt64(int64(delta)),
	}
}

func floatExpr(value float32) Expr {
	return Expr{
		Type:  NumberExpr,
		Const: constant.MakeFloat64(float64(value)),
	}
}

func uniformVariableExpr(index int) Expr {
	return Expr{
		Type:  UniformVariable,
		Index: index,
	}
}

func localVariableExpr(index int) Expr {
	return Expr{
		Type:  LocalVariable,
		Index: index,
	}
}

func builtinFuncExpr(f BuiltinFunc) Expr {
	return Expr{
		Type:        BuiltinFuncExpr,
		BuiltinFunc: f,
	}
}

func swizzlingExpr(swizzling string) Expr {
	return Expr{
		Type:      SwizzlingExpr,
		Swizzling: swizzling,
	}
}

func functionExpr(index int) Expr {
	return Expr{
		Type:  FunctionExpr,
		Index: index,
	}
}

func binaryExpr(op Op, exprs ...Expr) Expr {
	return Expr{
		Type:  Binary,
		Op:    op,
		Exprs: exprs,
	}
}

func selectionExpr(cond, a, b Expr) Expr {
	return Expr{
		Type:  Selection,
		Exprs: []Expr{cond, a, b},
	}
}

func callExpr(callee Expr, args ...Expr) Expr {
	return Expr{
		Type:  Call,
		Exprs: append([]Expr{callee}, args...),
	}
}

func fieldSelectorExpr(a, b Expr) Expr {
	return Expr{
		Type:  FieldSelector,
		Exprs: []Expr{a, b},
	}
}

func TestOutput(t *testing.T) {
	glslPrelude := glsl.FragmentPrelude + "\n"

	tests := []struct {
		Name    string
		Program Program
		GlslVS  string
		GlslFS  string
		Metal   string
	}{
		{
			Name:    "Empty",
			Program: Program{},
			GlslVS:  ``,
			GlslFS:  glsl.FragmentPrelude,
		},
		{
			Name: "Uniform",
			Program: Program{
				Uniforms: []Type{
					{Main: Float},
				},
			},
			GlslVS: `uniform float U0;`,
			GlslFS: glslPrelude + `
uniform float U0;`,
		},
		{
			Name: "UniformStruct",
			Program: Program{
				Uniforms: []Type{
					{
						Main: Struct,
						Sub: []Type{
							{Main: Float},
						},
					},
				},
			},
			GlslVS: `struct S0 {
	float M0;
};

uniform S0 U0;`,
			GlslFS: glslPrelude + `
struct S0 {
	float M0;
};

uniform S0 U0;`,
		},
		{
			Name: "Vars",
			Program: Program{
				Uniforms: []Type{
					{Main: Float},
				},
				Attributes: []Type{
					{Main: Vec2},
				},
				Varyings: []Type{
					{Main: Vec3},
				},
			},
			GlslVS: `uniform float U0;
attribute vec2 A0;
varying vec3 V0;`,
			GlslFS: glslPrelude + `
uniform float U0;
varying vec3 V0;`,
		},
		{
			Name: "Func",
			Program: Program{
				Funcs: []Func{
					{
						Index: 0,
					},
				},
			},
			GlslVS: `void F0(void);

void F0(void) {
}`,
			GlslFS: glslPrelude + `
void F0(void);

void F0(void) {
}`,
		},
		{
			Name: "FuncParams",
			Program: Program{
				Funcs: []Func{
					{
						Index: 0,
						InParams: []Type{
							{Main: Float},
							{Main: Vec2},
							{Main: Vec4},
						},
						OutParams: []Type{
							{Main: Mat4},
						},
					},
				},
			},
			GlslVS: `void F0(in float l0, in vec2 l1, in vec4 l2, out mat4 l3);

void F0(in float l0, in vec2 l1, in vec4 l2, out mat4 l3) {
}`,
			GlslFS: glslPrelude + `
void F0(in float l0, in vec2 l1, in vec4 l2, out mat4 l3);

void F0(in float l0, in vec2 l1, in vec4 l2, out mat4 l3) {
}`,
		},
		{
			Name: "FuncReturn",
			Program: Program{
				Funcs: []Func{
					{
						Index: 0,
						InParams: []Type{
							{Main: Float},
						},
						Return: Type{Main: Float},
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
			GlslVS: `float F0(in float l0);

float F0(in float l0) {
	return l0;
}`,
			GlslFS: glslPrelude + `
float F0(in float l0);

float F0(in float l0) {
	return l0;
}`,
		},
		{
			Name: "FuncLocals",
			Program: Program{
				Funcs: []Func{
					{
						Index: 0,
						InParams: []Type{
							{Main: Float},
						},
						OutParams: []Type{
							{Main: Float},
						},
						Block: block([]Type{
							{Main: Mat4},
							{Main: Mat4},
						}, 2),
					},
				},
			},
			GlslVS: `void F0(in float l0, out float l1);

void F0(in float l0, out float l1) {
	mat4 l2 = mat4(0);
	mat4 l3 = mat4(0);
}`,
			GlslFS: glslPrelude + `
void F0(in float l0, out float l1);

void F0(in float l0, out float l1) {
	mat4 l2 = mat4(0);
	mat4 l3 = mat4(0);
}`,
		},
		{
			Name: "FuncBlocks",
			Program: Program{
				Funcs: []Func{
					{
						Index: 0,
						InParams: []Type{
							{Main: Float},
						},
						OutParams: []Type{
							{Main: Float},
						},
						Block: block(
							[]Type{
								{Main: Mat4},
								{Main: Mat4},
							},
							2,
							blockStmt(
								block(
									[]Type{
										{Main: Mat4},
										{Main: Mat4},
									},
									4,
								),
							),
						),
					},
				},
			},
			GlslVS: `void F0(in float l0, out float l1);

void F0(in float l0, out float l1) {
	mat4 l2 = mat4(0);
	mat4 l3 = mat4(0);
	{
		mat4 l4 = mat4(0);
		mat4 l5 = mat4(0);
	}
}`,
			GlslFS: glslPrelude + `
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
			Program: Program{
				Funcs: []Func{
					{
						Index: 0,
						InParams: []Type{
							{Main: Float},
							{Main: Float},
						},
						OutParams: []Type{
							{Main: Float},
						},
						Block: block(
							nil,
							3,
							assignStmt(
								localVariableExpr(2),
								binaryExpr(
									Add,
									localVariableExpr(0),
									localVariableExpr(1),
								),
							),
						),
					},
				},
			},
			GlslVS: `void F0(in float l0, in float l1, out float l2);

void F0(in float l0, in float l1, out float l2) {
	l2 = (l0) + (l1);
}`,
			GlslFS: glslPrelude + `
void F0(in float l0, in float l1, out float l2);

void F0(in float l0, in float l1, out float l2) {
	l2 = (l0) + (l1);
}`,
		},
		{
			Name: "Selection",
			Program: Program{
				Funcs: []Func{
					{
						Index: 0,
						InParams: []Type{
							{Main: Bool},
							{Main: Float},
							{Main: Float},
						},
						OutParams: []Type{
							{Main: Float},
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
			GlslVS: `void F0(in bool l0, in float l1, in float l2, out float l3);

void F0(in bool l0, in float l1, in float l2, out float l3) {
	l3 = (l0) ? (l1) : (l2);
}`,
			GlslFS: glslPrelude + `
void F0(in bool l0, in float l1, in float l2, out float l3);

void F0(in bool l0, in float l1, in float l2, out float l3) {
	l3 = (l0) ? (l1) : (l2);
}`,
		},
		{
			Name: "Call",
			Program: Program{
				Funcs: []Func{
					{
						Index: 0,
						InParams: []Type{
							{Main: Float},
							{Main: Float},
						},
						OutParams: []Type{
							{Main: Vec2},
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
			GlslVS: `void F0(in float l0, in float l1, out vec2 l2);

void F0(in float l0, in float l1, out vec2 l2) {
	F1();
	l2 = F2(l0, l1);
}`,
			GlslFS: glslPrelude + `
void F0(in float l0, in float l1, out vec2 l2);

void F0(in float l0, in float l1, out vec2 l2) {
	F1();
	l2 = F2(l0, l1);
}`,
		},
		{
			Name: "BuiltinFunc",
			Program: Program{
				Funcs: []Func{
					{
						Index: 0,
						InParams: []Type{
							{Main: Float},
							{Main: Float},
						},
						OutParams: []Type{
							{Main: Float},
						},
						Block: block(
							nil,
							3,
							assignStmt(
								localVariableExpr(2),
								callExpr(
									builtinFuncExpr(Min),
									localVariableExpr(0),
									localVariableExpr(1),
								),
							),
						),
					},
				},
			},
			GlslVS: `void F0(in float l0, in float l1, out float l2);

void F0(in float l0, in float l1, out float l2) {
	l2 = min(l0, l1);
}`,
			GlslFS: glslPrelude + `
void F0(in float l0, in float l1, out float l2);

void F0(in float l0, in float l1, out float l2) {
	l2 = min(l0, l1);
}`,
		},
		{
			Name: "FieldSelector",
			Program: Program{
				Funcs: []Func{
					{
						Index: 0,
						InParams: []Type{
							{Main: Vec4},
						},
						OutParams: []Type{
							{Main: Vec2},
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
			GlslVS: `void F0(in vec4 l0, out vec2 l1);

void F0(in vec4 l0, out vec2 l1) {
	l1 = (l0).xz;
}`,
			GlslFS: glslPrelude + `
void F0(in vec4 l0, out vec2 l1);

void F0(in vec4 l0, out vec2 l1) {
	l1 = (l0).xz;
}`,
		},
		{
			Name: "If",
			Program: Program{
				Funcs: []Func{
					{
						Index: 0,
						InParams: []Type{
							{Main: Float},
							{Main: Float},
						},
						OutParams: []Type{
							{Main: Float},
						},
						Block: block(
							nil,
							3,
							ifStmt(
								binaryExpr(
									EqualOp,
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
			GlslVS: `void F0(in float l0, in float l1, out float l2);

void F0(in float l0, in float l1, out float l2) {
	if ((l0) == (0.0)) {
		l2 = l0;
	} else {
		l2 = l1;
	}
}`,
			GlslFS: glslPrelude + `
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
			Program: Program{
				Funcs: []Func{
					{
						Index: 0,
						InParams: []Type{
							{Main: Float},
							{Main: Float},
						},
						OutParams: []Type{
							{Main: Float},
						},
						Block: block(
							[]Type{
								{},
							},
							3,
							forStmt(
								Type{Main: Int},
								3,
								0,
								100,
								LessThanOp,
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
			GlslVS: `void F0(in float l0, in float l1, out float l2);

void F0(in float l0, in float l1, out float l2) {
	for (int l3 = 0; l3 < 100; l3++) {
		l2 = l0;
	}
}`,
			GlslFS: glslPrelude + `
void F0(in float l0, in float l1, out float l2);

void F0(in float l0, in float l1, out float l2) {
	for (int l3 = 0; l3 < 100; l3++) {
		l2 = l0;
	}
}`,
		},
		{
			Name: "For2",
			Program: Program{
				Funcs: []Func{
					{
						Index: 0,
						InParams: []Type{
							{Main: Float},
							{Main: Float},
						},
						OutParams: []Type{
							{Main: Float},
						},
						Block: block(
							[]Type{
								{},
							},
							3,
							forStmt(
								Type{Main: Int},
								3,
								0,
								100,
								LessThanOp,
								1,
								block(
									[]Type{
										{Main: Int},
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
			GlslVS: `void F0(in float l0, in float l1, out float l2);

void F0(in float l0, in float l1, out float l2) {
	for (int l3 = 0; l3 < 100; l3++) {
		int l4 = 0;
		l2 = l4;
	}
}`,
			GlslFS: glslPrelude + `
void F0(in float l0, in float l1, out float l2);

void F0(in float l0, in float l1, out float l2) {
	for (int l3 = 0; l3 < 100; l3++) {
		int l4 = 0;
		l2 = l4;
	}
}`,
			Metal: `#include <metal_stdlib>

using namespace metal;

constexpr sampler texture_sampler{filter::nearest};

void F0(float l0, float l1, thread float& l2);

void F0(float l0, float l1, thread float& l2) {
	for (int l3 = 0; l3 < 100; l3++) {
		int l4 = 0;
		l2 = l4;
	}
}`,
		},
		{
			Name: "For3",
			Program: Program{
				Funcs: []Func{
					{
						Index: 0,
						InParams: []Type{
							{Main: Float},
							{Main: Float},
						},
						OutParams: []Type{
							{Main: Float},
						},
						Block: block(
							[]Type{
								{},
								{},
							},
							3,
							forStmt(
								Type{Main: Int},
								3,
								0,
								100,
								LessThanOp,
								1,
								block(
									[]Type{
										{Main: Int},
									},
									4,
									assignStmt(
										localVariableExpr(2),
										localVariableExpr(4),
									),
								),
							),
							forStmt(
								Type{Main: Float},
								4,
								0,
								100,
								LessThanOp,
								1,
								block(
									[]Type{
										{Main: Int},
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
			GlslVS: `void F0(in float l0, in float l1, out float l2);

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
			GlslFS: glslPrelude + `
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
			Metal: `#include <metal_stdlib>

using namespace metal;

constexpr sampler texture_sampler{filter::nearest};

void F0(float l0, float l1, thread float& l2);

void F0(float l0, float l1, thread float& l2) {
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
			Program: Program{
				Uniforms: []Type{
					{Main: Float},
				},
				Attributes: []Type{
					{Main: Vec4},
					{Main: Float},
					{Main: Vec2},
				},
				Varyings: []Type{
					{Main: Float},
					{Main: Vec2},
				},
				VertexFunc: VertexFunc{
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
			GlslVS: `uniform float U0;
attribute vec4 A0;
attribute float A1;
attribute vec2 A2;
varying float V0;
varying vec2 V1;

void main(void) {
	gl_Position = A0;
	V0 = A1;
	V1 = A2;
}`,
			GlslFS: glslPrelude + `
uniform float U0;
varying float V0;
varying vec2 V1;`,
		},
		{
			Name: "FragmentFunc",
			Program: Program{
				Uniforms: []Type{
					{Main: Float},
				},
				Attributes: []Type{
					{Main: Vec4},
					{Main: Float},
					{Main: Vec2},
				},
				Varyings: []Type{
					{Main: Float},
					{Main: Vec2},
				},
				VertexFunc: VertexFunc{
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
				FragmentFunc: FragmentFunc{
					Block: block(
						[]Type{
							{Main: Float},
							{Main: Vec2},
						},
						3+1,
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
			GlslVS: `uniform float U0;
attribute vec4 A0;
attribute float A1;
attribute vec2 A2;
varying float V0;
varying vec2 V1;

void main(void) {
	gl_Position = A0;
	V0 = A1;
	V1 = A2;
}`,
			GlslFS: glslPrelude + `
uniform float U0;
varying float V0;
varying vec2 V1;

void main(void) {
	float l0 = float(0);
	vec2 l1 = vec2(0);
	gl_FragColor = gl_FragCoord;
	l0 = V0;
	l1 = V1;
}`,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			vs, fs := glsl.Compile(&tc.Program)
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
			m := metal.Compile(&tc.Program, "Vertex", "Fragment")
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
