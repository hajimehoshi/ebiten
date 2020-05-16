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
	"testing"

	. "github.com/hajimehoshi/ebiten/internal/shaderir"
)

func block(localVars []Type, stmts ...Stmt) Block {
	return Block{
		LocalVars: localVars,
		Stmts:     stmts,
	}
}

func blockStmt(block Block) Stmt {
	return Stmt{
		Type:   BlockStmt,
		Blocks: []Block{block},
	}
}

func assignStmt(lhs Expr, rhs Expr) Stmt {
	return Stmt{
		Type:  Assign,
		Exprs: []Expr{lhs, rhs},
	}
}

func ifStmt(cond Expr, block Block, elseBlock Block) Stmt {
	return Stmt{
		Type:   If,
		Exprs:  []Expr{cond},
		Blocks: []Block{block, elseBlock},
	}
}

func forStmt(init, end int, op Op, delta int, block Block) Stmt {
	return Stmt{
		Type:     For,
		Blocks:   []Block{block},
		ForInit:  init,
		ForEnd:   end,
		ForOp:    op,
		ForDelta: delta,
	}
}

func floatExpr(value float32) Expr {
	return Expr{
		Type:  FloatExpr,
		Float: value,
	}
}

func varNameExpr(vt VariableType, index int) Expr {
	return Expr{
		Type: VarName,
		Variable: Variable{
			Type:  vt,
			Index: index,
		},
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

func TestOutput(t *testing.T) {
	tests := []struct {
		Name    string
		Program Program
		Glsl    string
	}{
		{
			Name:    "Empty",
			Program: Program{},
			Glsl:    ``,
		},
		{
			Name: "Uniform",
			Program: Program{
				Uniforms: []Type{
					{Main: Float},
				},
			},
			Glsl: `uniform float U0;`,
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
			Glsl: `struct S0 {
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
			Glsl: `uniform float U0;
attribute vec2 A0;
varying vec3 V0;`,
		},
		{
			Name: "Func",
			Program: Program{
				Funcs: []Func{
					{
						Name: "F0",
					},
				},
			},
			Glsl: `void F0(void) {
}`,
		},
		{
			Name: "FuncParams",
			Program: Program{
				Funcs: []Func{
					{
						Name: "F0",
						InParams: []Type{
							{Main: Float},
							{Main: Vec2},
							{Main: Vec4},
						},
						InOutParams: []Type{
							{Main: Mat2},
						},
						OutParams: []Type{
							{Main: Mat4},
						},
					},
				},
			},
			Glsl: `void F0(in float l0, in vec2 l1, in vec4 l2, inout mat2 l3, out mat4 l4) {
}`,
		},
		{
			Name: "FuncLocals",
			Program: Program{
				Funcs: []Func{
					{
						Name: "F0",
						InParams: []Type{
							{Main: Float},
						},
						InOutParams: []Type{
							{Main: Float},
						},
						OutParams: []Type{
							{Main: Float},
						},
						Block: block([]Type{
							{Main: Mat4},
							{Main: Mat4},
						}),
					},
				},
			},
			Glsl: `void F0(in float l0, inout float l1, out float l2) {
	mat4 l3;
	mat4 l4;
}`,
		},
		{
			Name: "FuncBlocks",
			Program: Program{
				Funcs: []Func{
					{
						Name: "F0",
						InParams: []Type{
							{Main: Float},
						},
						InOutParams: []Type{
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
							blockStmt(
								block(
									[]Type{
										{Main: Mat4},
										{Main: Mat4},
									},
								),
							),
						),
					},
				},
			},
			Glsl: `void F0(in float l0, inout float l1, out float l2) {
	mat4 l3;
	mat4 l4;
	{
		mat4 l5;
		mat4 l6;
	}
}`,
		},
		{
			Name: "Add",
			Program: Program{
				Funcs: []Func{
					{
						Name: "F0",
						InParams: []Type{
							{Main: Float},
							{Main: Float},
						},
						OutParams: []Type{
							{Main: Float},
						},
						Block: block(
							nil,
							assignStmt(
								varNameExpr(Local, 2),
								binaryExpr(
									Add,
									varNameExpr(Local, 0),
									varNameExpr(Local, 1),
								),
							),
						),
					},
				},
			},
			Glsl: `void F0(in float l0, in float l1, out float l2) {
	l2 = (l0) + (l1);
}`,
		},
		{
			Name: "Selection",
			Program: Program{
				Funcs: []Func{
					{
						Name: "F0",
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
							assignStmt(
								varNameExpr(Local, 3),
								selectionExpr(
									varNameExpr(Local, 0),
									varNameExpr(Local, 1),
									varNameExpr(Local, 2),
								),
							),
						),
					},
				},
			},
			Glsl: `void F0(in bool l0, in float l1, in float l2, out float l3) {
	l3 = (l0) ? (l1) : (l2);
}`,
		},
		{
			Name: "If",
			Program: Program{
				Funcs: []Func{
					{
						Name: "F0",
						InParams: []Type{
							{Main: Float},
							{Main: Float},
						},
						OutParams: []Type{
							{Main: Float},
						},
						Block: block(
							nil,
							ifStmt(
								binaryExpr(
									Equal,
									varNameExpr(Local, 0),
									floatExpr(0),
								),
								block(
									nil,
									assignStmt(
										varNameExpr(Local, 2),
										varNameExpr(Local, 0),
									),
								),
								block(
									nil,
									assignStmt(
										varNameExpr(Local, 2),
										varNameExpr(Local, 1),
									),
								),
							),
						),
					},
				},
			},
			Glsl: `void F0(in float l0, in float l1, out float l2) {
	if ((l0) == (0.000000000e+00)) {
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
						Name: "F0",
						InParams: []Type{
							{Main: Float},
							{Main: Float},
						},
						OutParams: []Type{
							{Main: Float},
						},
						Block: block(
							nil,
							forStmt(
								0,
								100,
								LessThan,
								1,
								block(
									nil,
									assignStmt(
										varNameExpr(Local, 2),
										varNameExpr(Local, 0),
									),
								),
							),
						),
					},
				},
			},
			Glsl: `void F0(in float l0, in float l1, out float l2) {
	for (int l3 = 0; l3 < 100; l3++) {
		l2 = l0;
	}
}`,
		},
	}
	for _, tc := range tests {
		got := tc.Program.Glsl()
		want := tc.Glsl + "\n"
		if got != want {
			t.Errorf("%s: got: %s, want: %s", tc.Name, got, want)
		}
	}
}
