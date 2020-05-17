// Copyright 2018 The Ebiten Authors
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

package graphicscommand_test

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	. "github.com/hajimehoshi/ebiten/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/internal/shaderir"
	t "github.com/hajimehoshi/ebiten/internal/testing"
)

func TestMain(m *testing.M) {
	t.MainWithRunLoop(m)
}

func quadVertices(w, h float32) []float32 {
	return []float32{
		0, 0, 0, 0, 0, 0, w, h, 1, 1, 1, 1,
		w, 0, w, 0, 0, 0, w, h, 1, 1, 1, 1,
		0, w, 0, h, 0, 0, w, h, 1, 1, 1, 1,
		w, h, w, h, 0, 0, w, h, 1, 1, 1, 1,
	}
}

func TestClear(t *testing.T) {
	const w, h = 1024, 1024
	src := NewImage(w/2, h/2)
	dst := NewImage(w, h)

	vs := quadVertices(w/2, h/2)
	is := graphics.QuadIndices()
	dst.DrawTriangles(src, vs, is, nil, driver.CompositeModeClear, driver.FilterNearest, driver.AddressClampToZero)

	pix, err := dst.Pixels()
	if err != nil {
		t.Fatal(err)
	}
	for j := 0; j < h/2; j++ {
		for i := 0; i < w/2; i++ {
			idx := 4 * (i + w*j)
			got := color.RGBA{pix[idx], pix[idx+1], pix[idx+2], pix[idx+3]}
			want := color.RGBA{}
			if got != want {
				t.Errorf("dst.At(%d, %d) after DrawTriangles: got %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestReplacePixelsPartAfterDrawTriangles(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("ReplacePixels must panic but not")
		}
	}()
	const w, h = 32, 32
	clr := NewImage(w, h)
	src := NewImage(w/2, h/2)
	dst := NewImage(w, h)
	vs := quadVertices(w/2, h/2)
	is := graphics.QuadIndices()
	dst.DrawTriangles(clr, vs, is, nil, driver.CompositeModeClear, driver.FilterNearest, driver.AddressClampToZero)
	dst.DrawTriangles(src, vs, is, nil, driver.CompositeModeSourceOver, driver.FilterNearest, driver.AddressClampToZero)
	dst.ReplacePixels(make([]byte, 4), 0, 0, 1, 1)
}

func TestShader(t *testing.T) {
	if !IsGL() {
		t.Skip("shader is not implemented on non-GL environment")
	}

	const w, h = 16, 16
	clr := NewImage(w, h)
	dst := NewImage(w, h)
	vs := quadVertices(w, h)
	is := graphics.QuadIndices()
	dst.DrawTriangles(clr, vs, is, nil, driver.CompositeModeClear, driver.FilterNearest, driver.AddressClampToZero)

	mat := shaderir.Expr{
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
	pos := shaderir.Expr{
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
	red := shaderir.Expr{
		Type: shaderir.Call,
		Exprs: []shaderir.Expr{
			{
				Type:        shaderir.BuiltinFuncExpr,
				BuiltinFunc: shaderir.Vec4F,
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
				Float: 0,
			},
			{
				Type:  shaderir.FloatExpr,
				Float: 1,
			},
		},
	}
	s := NewShader(&shaderir.Program{
		Uniforms: []shaderir.Type{
			{Main: shaderir.Vec2},
		},
		Attributes: []shaderir.Type{
			{Main: shaderir.Vec2},
			{Main: shaderir.Vec2},
			{Main: shaderir.Vec4},
			{Main: shaderir.Vec4},
		},
		VertexFunc: shaderir.VertexFunc{
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
									mat,
									pos,
								},
							},
						},
					},
				},
			},
		},
		FragmentFunc: shaderir.FragmentFunc{
			Block: shaderir.Block{
				Stmts: []shaderir.Stmt{
					{
						Type: shaderir.Assign,
						Exprs: []shaderir.Expr{
							{
								Type:  shaderir.LocalVariable,
								Index: 1,
							},
							red,
						},
					},
				},
			},
		},
	})
	us := map[int]interface{}{
		0: []float32{w, h},
	}
	dst.DrawShader(s, vs, is, driver.CompositeModeSourceOver, us)

	pix, err := dst.Pixels()
	if err != nil {
		t.Fatal(err)
	}
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			idx := 4 * (i + w*j)
			got := color.RGBA{pix[idx], pix[idx+1], pix[idx+2], pix[idx+3]}
			want := color.RGBA{0xff, 0, 0, 0xff}
			if got != want {
				t.Errorf("dst.At(%d, %d) after DrawTriangles: got %v, want: %v", i, j, got, want)
			}
		}
	}
}
