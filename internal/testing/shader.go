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

package testing

import (
	"github.com/hajimehoshi/ebiten/internal/shaderir"
)

var (
	projectionMatrix = shaderir.Expr{
		Type: shaderir.Call,
		Exprs: []shaderir.Expr{
			{
				Type:        shaderir.BuiltinFuncExpr,
				BuiltinFunc: shaderir.Mat4F,
			},
			{
				Type: shaderir.Binary,
				Op:   shaderir.Div,
				Exprs: []shaderir.Expr{
					{
						Type:  shaderir.FloatExpr,
						Float: 2,
					},
					{
						Type: shaderir.FieldSelector,
						Exprs: []shaderir.Expr{
							{
								Type:  shaderir.UniformVariable,
								Index: 0,
							},
							{
								Type:      shaderir.SwizzlingExpr,
								Swizzling: "x",
							},
						},
					},
				},
			},
			{
				Type:  shaderir.FloatExpr,
				Float: 0,
			},
			{
				Type:  shaderir.FloatExpr,
				Float: 0,
			},
			{
				Type:  shaderir.FloatExpr,
				Float: 0,
			},
			{
				Type:  shaderir.FloatExpr,
				Float: 0,
			},
			{
				Type: shaderir.Binary,
				Op:   shaderir.Div,
				Exprs: []shaderir.Expr{
					{
						Type:  shaderir.FloatExpr,
						Float: 2,
					},
					{
						Type: shaderir.FieldSelector,
						Exprs: []shaderir.Expr{
							{
								Type:  shaderir.UniformVariable,
								Index: 0,
							},
							{
								Type:      shaderir.SwizzlingExpr,
								Swizzling: "y",
							},
						},
					},
				},
			},
			{
				Type:  shaderir.FloatExpr,
				Float: 0,
			},
			{
				Type:  shaderir.FloatExpr,
				Float: 0,
			},
			{
				Type:  shaderir.FloatExpr,
				Float: 0,
			},
			{
				Type:  shaderir.FloatExpr,
				Float: 0,
			},
			{
				Type:  shaderir.FloatExpr,
				Float: 1,
			},
			{
				Type:  shaderir.FloatExpr,
				Float: 0,
			},
			{
				Type:  shaderir.FloatExpr,
				Float: -1,
			},
			{
				Type:  shaderir.FloatExpr,
				Float: -1,
			},
			{
				Type:  shaderir.FloatExpr,
				Float: 0,
			},
			{
				Type:  shaderir.FloatExpr,
				Float: 1,
			},
		},
	}
	vertexPosition = shaderir.Expr{
		Type: shaderir.Call,
		Exprs: []shaderir.Expr{
			{
				Type:        shaderir.BuiltinFuncExpr,
				BuiltinFunc: shaderir.Vec4F,
			},
			{
				Type:  shaderir.LocalVariable,
				Index: 0,
			},
			{
				Type:  shaderir.FloatExpr,
				Float: 0,
			},
			{
				Type:  shaderir.FloatExpr,
				Float: 1,
			},
		},
	}
	defaultVertexFunc = shaderir.VertexFunc{
		Block: shaderir.Block{
			Stmts: []shaderir.Stmt{
				{
					Type: shaderir.Assign,
					Exprs: []shaderir.Expr{
						{
							Type:  shaderir.LocalVariable,
							Index: 4,
						},
						{
							Type: shaderir.Binary,
							Op:   shaderir.Mul,
							Exprs: []shaderir.Expr{
								projectionMatrix,
								vertexPosition,
							},
						},
					},
				},
			},
		},
	}
	defaultProgram = shaderir.Program{
		Uniforms: []shaderir.Type{
			{Main: shaderir.Vec2},
		},
		Attributes: []shaderir.Type{
			{Main: shaderir.Vec2},
			{Main: shaderir.Vec2},
			{Main: shaderir.Vec4},
			{Main: shaderir.Vec4},
		},
		VertexFunc: defaultVertexFunc,
	}
)

// ShaderProgramFill returns a shader intermediate representation to fill the frambuffer.
//
// Uniform variables:
//
//   0. the framebuffer size (vec2)
func ShaderProgramFill(r, g, b, a byte) shaderir.Program {
	clr := shaderir.Expr{
		Type: shaderir.Call,
		Exprs: []shaderir.Expr{
			{
				Type:        shaderir.BuiltinFuncExpr,
				BuiltinFunc: shaderir.Vec4F,
			},
			{
				Type:  shaderir.FloatExpr,
				Float: float32(r) / 0xff,
			},
			{
				Type:  shaderir.FloatExpr,
				Float: float32(g) / 0xff,
			},
			{
				Type:  shaderir.FloatExpr,
				Float: float32(b) / 0xff,
			},
			{
				Type:  shaderir.FloatExpr,
				Float: float32(a) / 0xff,
			},
		},
	}

	p := defaultProgram
	p.FragmentFunc = shaderir.FragmentFunc{
		Block: shaderir.Block{
			Stmts: []shaderir.Stmt{
				{
					Type: shaderir.Assign,
					Exprs: []shaderir.Expr{
						{
							Type:  shaderir.LocalVariable,
							Index: 1,
						},
						clr,
					},
				},
			},
		},
	}

	return p
}
