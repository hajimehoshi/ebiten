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
							Index: 4, // the varying variable
						},
						{
							Type:  shaderir.LocalVariable,
							Index: 1, // the 2nd attribute variable
						},
					},
				},
				{
					Type: shaderir.Assign,
					Exprs: []shaderir.Expr{
						{
							Type:  shaderir.LocalVariable,
							Index: 5, // gl_Position in GLSL
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
)

func defaultProgram() shaderir.Program {
	return shaderir.Program{
		Uniforms: []shaderir.Type{
			{Main: shaderir.Vec2},
		},
		Attributes: []shaderir.Type{
			{Main: shaderir.Vec2}, // Local var (0) in the vertex shader
			{Main: shaderir.Vec2}, // Local var (1) in the vertex shader
			{Main: shaderir.Vec4}, // Local var (2) in the vertex shader
			{Main: shaderir.Vec4}, // Local var (3) in the vertex shader
		},
		Varyings: []shaderir.Type{
			{Main: shaderir.Vec2}, // Local var (4) in the vertex shader, (0) in the fragment shader
		},
		VertexFunc: defaultVertexFunc,
	}
}

// ShaderProgramFill returns a shader intermediate representation to fill the frambuffer.
//
// Uniform variable's index and its value are:
//
//   0: the framebuffer size (Vec2)
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

	p := defaultProgram()
	p.FragmentFunc = shaderir.FragmentFunc{
		Block: shaderir.Block{
			Stmts: []shaderir.Stmt{
				{
					Type: shaderir.Assign,
					Exprs: []shaderir.Expr{
						{
							Type:  shaderir.LocalVariable,
							Index: 2,
						},
						clr,
					},
				},
			},
		},
	}

	return p
}

// ShaderProgramImages returns a shader intermediate representation to render the frambuffer with the given images.
//
// Uniform variables's indices and their values are:
//
//   0:    the framebuffer size (Vec2)
//   1:    the first images (Sampler2D)
//   3n-1: the (n+1)th image (Sampler2D)
//   3n:   the (n+1)th image's size (Vec2)
//   3n+1: the (n+1)th image's region (Vec4)
//
// The first image's size and region are represented in attribute variables.
//
// The size and region values are actually not used in this shader so far.
func ShaderProgramImages(imageNum int) shaderir.Program {
	if imageNum <= 0 {
		panic("testing: imageNum must be >= 1")
	}

	p := defaultProgram()

	for i := 0; i < imageNum; i++ {
		p.Uniforms = append(p.Uniforms, shaderir.Type{Main: shaderir.Sampler2D})
		if i > 0 {
			p.Uniforms = append(p.Uniforms, shaderir.Type{Main: shaderir.Vec2})
			p.Uniforms = append(p.Uniforms, shaderir.Type{Main: shaderir.Vec4})
		}
	}

	// In the fragment shader, local variables are:
	//
	//   0: Varying variables (vec2)
	//   1: gl_FragCoord
	//   2: gl_FragColor
	//   3: Actual local variables in the main function

	local := shaderir.Expr{
		Type:  shaderir.LocalVariable,
		Index: 3,
	}
	fragColor := shaderir.Expr{
		Type:  shaderir.LocalVariable,
		Index: 2,
	}
	texPos := shaderir.Expr{
		Type:  shaderir.LocalVariable,
		Index: 0,
	}

	var stmts []shaderir.Stmt
	for i := 0; i < imageNum; i++ {
		var rhs shaderir.Expr
		if i == 0 {
			rhs = shaderir.Expr{
				Type: shaderir.Call,
				Exprs: []shaderir.Expr{
					{
						Type:        shaderir.BuiltinFuncExpr,
						BuiltinFunc: shaderir.Texture2D,
					},
					{
						Type:  shaderir.UniformVariable,
						Index: 1,
					},
					texPos,
				},
			}
		} else {
			rhs = shaderir.Expr{
				Type: shaderir.Binary,
				Op:   shaderir.Add,
				Exprs: []shaderir.Expr{
					local,
					{
						Type: shaderir.Call,
						Exprs: []shaderir.Expr{
							{
								Type:        shaderir.BuiltinFuncExpr,
								BuiltinFunc: shaderir.Texture2D,
							},
							{
								Type:  shaderir.UniformVariable,
								Index: 3*i - 1,
							},
							texPos,
						},
					},
				},
			}
		}
		stmts = append(stmts, shaderir.Stmt{
			Type: shaderir.Assign,
			Exprs: []shaderir.Expr{
				local,
				rhs,
			},
		})
	}

	stmts = append(stmts, shaderir.Stmt{
		Type: shaderir.Assign,
		Exprs: []shaderir.Expr{
			fragColor,
			local,
		},
	})

	p.FragmentFunc = shaderir.FragmentFunc{
		Block: shaderir.Block{
			LocalVars: []shaderir.Type{
				{Main: shaderir.Vec4},
			},
			Stmts: stmts,
		},
	}

	return p
}
