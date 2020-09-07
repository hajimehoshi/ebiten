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
	"go/constant"

	"github.com/hajimehoshi/ebiten/internal/graphics"
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
						Type:      shaderir.NumberExpr,
						Const:     constant.MakeFloat64(2),
						ConstType: shaderir.ConstTypeFloat,
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
				Type:      shaderir.NumberExpr,
				Const:     constant.MakeFloat64(0),
				ConstType: shaderir.ConstTypeFloat,
			},
			{
				Type:      shaderir.NumberExpr,
				Const:     constant.MakeFloat64(0),
				ConstType: shaderir.ConstTypeFloat,
			},
			{
				Type:      shaderir.NumberExpr,
				Const:     constant.MakeFloat64(0),
				ConstType: shaderir.ConstTypeFloat,
			},
			{
				Type:      shaderir.NumberExpr,
				Const:     constant.MakeFloat64(0),
				ConstType: shaderir.ConstTypeFloat,
			},
			{
				Type: shaderir.Binary,
				Op:   shaderir.Div,
				Exprs: []shaderir.Expr{
					{
						Type:      shaderir.NumberExpr,
						Const:     constant.MakeFloat64(2),
						ConstType: shaderir.ConstTypeFloat,
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
				Type:      shaderir.NumberExpr,
				Const:     constant.MakeFloat64(0),
				ConstType: shaderir.ConstTypeFloat,
			},
			{
				Type:      shaderir.NumberExpr,
				Const:     constant.MakeFloat64(0),
				ConstType: shaderir.ConstTypeFloat,
			},
			{
				Type:      shaderir.NumberExpr,
				Const:     constant.MakeFloat64(0),
				ConstType: shaderir.ConstTypeFloat,
			},
			{
				Type:      shaderir.NumberExpr,
				Const:     constant.MakeFloat64(0),
				ConstType: shaderir.ConstTypeFloat,
			},
			{
				Type:      shaderir.NumberExpr,
				Const:     constant.MakeFloat64(1),
				ConstType: shaderir.ConstTypeFloat,
			},
			{
				Type:      shaderir.NumberExpr,
				Const:     constant.MakeFloat64(0),
				ConstType: shaderir.ConstTypeFloat,
			},
			{
				Type:      shaderir.NumberExpr,
				Const:     constant.MakeFloat64(-1),
				ConstType: shaderir.ConstTypeFloat,
			},
			{
				Type:      shaderir.NumberExpr,
				Const:     constant.MakeFloat64(-1),
				ConstType: shaderir.ConstTypeFloat,
			},
			{
				Type:      shaderir.NumberExpr,
				Const:     constant.MakeFloat64(0),
				ConstType: shaderir.ConstTypeFloat,
			},
			{
				Type:      shaderir.NumberExpr,
				Const:     constant.MakeFloat64(1),
				ConstType: shaderir.ConstTypeFloat,
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
				Type:      shaderir.NumberExpr,
				Const:     constant.MakeFloat64(0),
				ConstType: shaderir.ConstTypeFloat,
			},
			{
				Type:      shaderir.NumberExpr,
				Const:     constant.MakeFloat64(1),
				ConstType: shaderir.ConstTypeFloat,
			},
		},
	}
	defaultVertexFunc = shaderir.VertexFunc{
		Block: &shaderir.Block{
			LocalVarIndexOffset: 4 + 1,
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
							Index: 3, // gl_Position in GLSL
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
	p := shaderir.Program{
		Attributes: []shaderir.Type{
			{Main: shaderir.Vec2}, // Local var (0) in the vertex shader
			{Main: shaderir.Vec2}, // Local var (1) in the vertex shader
			{Main: shaderir.Vec4}, // Local var (2) in the vertex shader
		},
		Varyings: []shaderir.Type{
			{Main: shaderir.Vec2}, // Local var (4) in the vertex shader, (1) in the fragment shader
		},
		VertexFunc: defaultVertexFunc,
	}

	p.Uniforms = make([]shaderir.Type, graphics.PreservedUniformVariablesNum)
	// Destination texture size
	p.Uniforms[0] = shaderir.Type{Main: shaderir.Vec2}
	// Source texture sizes
	p.Uniforms[1] = shaderir.Type{
		Main:   shaderir.Array,
		Length: graphics.ShaderImageNum,
		Sub:    []shaderir.Type{{Main: shaderir.Vec2}},
	}
	// Source texture offsets
	p.Uniforms[2] = shaderir.Type{
		Main:   shaderir.Array,
		Length: graphics.ShaderImageNum - 1,
		Sub:    []shaderir.Type{{Main: shaderir.Vec2}},
	}
	// Source region origin
	p.Uniforms[3] = shaderir.Type{Main: shaderir.Vec2}
	// Source region size
	p.Uniforms[4] = shaderir.Type{Main: shaderir.Vec2}
	return p
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
				Type:      shaderir.NumberExpr,
				Const:     constant.MakeFloat64(float64(r) / 0xff),
				ConstType: shaderir.ConstTypeFloat,
			},
			{
				Type:      shaderir.NumberExpr,
				Const:     constant.MakeFloat64(float64(g) / 0xff),
				ConstType: shaderir.ConstTypeFloat,
			},
			{
				Type:      shaderir.NumberExpr,
				Const:     constant.MakeFloat64(float64(b) / 0xff),
				ConstType: shaderir.ConstTypeFloat,
			},
			{
				Type:      shaderir.NumberExpr,
				Const:     constant.MakeFloat64(float64(a) / 0xff),
				ConstType: shaderir.ConstTypeFloat,
			},
		},
	}

	p := defaultProgram()
	p.FragmentFunc = shaderir.FragmentFunc{
		Block: &shaderir.Block{
			LocalVarIndexOffset: 2 + 1,
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
//   0:  the framebuffer size (Vec2)
//
// The size and region values are actually not used in this shader so far.
func ShaderProgramImages(imageNum int) shaderir.Program {
	if imageNum <= 0 {
		panic("testing: imageNum must be >= 1")
	}

	p := defaultProgram()
	p.TextureNum = imageNum

	// In the fragment shader, local variables are:
	//
	//   0: gl_FragCoord
	//   1: Varying variables (vec2)
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
		Index: 1,
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
						BuiltinFunc: shaderir.Texture2DF,
					},
					{
						Type:  shaderir.TextureVariable,
						Index: 0,
					},
					texPos,
				},
			}
		} else {
			texPos2 := shaderir.Expr{
				Type: shaderir.Binary,
				Op:   shaderir.Add,
				Exprs: []shaderir.Expr{
					texPos,
					{
						Type: shaderir.Index,
						Exprs: []shaderir.Expr{
							{
								Type:  shaderir.UniformVariable,
								Index: graphics.TextureSourceOffsetsUniformVariableIndex,
							},
							{
								Type:      shaderir.NumberExpr,
								Const:     constant.MakeInt64(int64(i - 1)),
								ConstType: shaderir.ConstTypeInt,
							},
						},
					},
				},
			}
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
								BuiltinFunc: shaderir.Texture2DF,
							},
							{
								Type:  shaderir.TextureVariable,
								Index: i,
							},
							texPos2,
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
		Block: &shaderir.Block{
			LocalVars: []shaderir.Type{
				{Main: shaderir.Vec4},
			},
			LocalVarIndexOffset: 2 + 1,
			Stmts:               stmts,
		},
	}

	return p
}
