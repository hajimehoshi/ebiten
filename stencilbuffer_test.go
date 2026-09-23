// Copyright 2026 The Ebitengine Authors
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

package ebiten_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestImageDrawTrianglesWithStencilBufferOnEmptyDestination(t *testing.T) {
	src := ebiten.NewImage(1, 1)
	src.Fill(color.White)

	shader, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, src0Pos vec2, color vec4) vec4 {
	return color
}
`))
	if err != nil {
		t.Fatal(err)
	}

	vs := []ebiten.Vertex{{DstX: 0, DstY: 0}, {DstX: 1, DstY: 0}, {DstX: 0, DstY: 1}}
	is := []uint32{0, 1, 2}

	for _, tc := range []struct {
		name string
		draw func(dst *ebiten.Image)
	}{
		{
			name: "FillRule",
			draw: func(dst *ebiten.Image) {
				dst.DrawTriangles32(vs, is, src, &ebiten.DrawTrianglesOptions{FillRule: ebiten.FillRuleNonZero})
			},
		},
		{
			name: "AntiAlias",
			draw: func(dst *ebiten.Image) {
				dst.DrawTriangles32(vs, is, src, &ebiten.DrawTrianglesOptions{AntiAlias: true})
			},
		},
		{
			name: "ShaderFillRule",
			draw: func(dst *ebiten.Image) {
				dst.DrawTrianglesShader32(vs, is, shader, &ebiten.DrawTrianglesShaderOptions{FillRule: ebiten.FillRuleNonZero})
			},
		},
		{
			name: "ShaderAntiAlias",
			draw: func(dst *ebiten.Image) {
				dst.DrawTrianglesShader32(vs, is, shader, &ebiten.DrawTrianglesShaderOptions{AntiAlias: true})
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ebiten.ResetStencilBufferImagesForTesting()
			dst := ebiten.NewImage(1, 1).SubImage(image.Rectangle{}).(*ebiten.Image)
			tc.draw(dst)
		})
	}
}

func TestImageDrawTrianglesWithStencilBufferOnEmptyDestinationValidatesIndices(t *testing.T) {
	src := ebiten.NewImage(1, 1)
	shader, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, src0Pos vec2, color vec4) vec4 {
	return color
}
`))
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name string
		draw func(dst *ebiten.Image)
	}{
		{
			name: "DrawTriangles",
			draw: func(dst *ebiten.Image) {
				dst.DrawTriangles32(nil, []uint32{0}, src, &ebiten.DrawTrianglesOptions{AntiAlias: true})
			},
		},
		{
			name: "DrawTrianglesShader",
			draw: func(dst *ebiten.Image) {
				dst.DrawTrianglesShader32(nil, []uint32{0}, shader, &ebiten.DrawTrianglesShaderOptions{AntiAlias: true})
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dst := ebiten.NewImage(1, 1).SubImage(image.Rectangle{}).(*ebiten.Image)
			defer func() {
				if r := recover(); r == nil {
					t.Error("a draw with an invalid index count must panic")
				}
			}()
			tc.draw(dst)
		})
	}
}

func TestImageDrawTrianglesWithStencilBufferOnSubImage(t *testing.T) {
	whiteImage := ebiten.NewImage(3, 3)
	whiteImage.Fill(color.White)
	whiteSubImage := whiteImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)

	shader, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, src0Pos vec2, color vec4) vec4 {
	return vec4(1, 0, 0, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	const size = 20
	// The sub-image's origin is at least as large as its own size, so that a
	// draw lost by keeping the destination's coordinate space cannot
	// accidentally fall inside the sub-image.
	dstMin := image.Pt(40, 30)

	vertices := func(ox, oy int) []ebiten.Vertex {
		v := func(x, y int) ebiten.Vertex {
			return ebiten.Vertex{
				DstX:   float32(ox + x),
				DstY:   float32(oy + y),
				SrcX:   1,
				SrcY:   1,
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			}
		}
		return []ebiten.Vertex{
			v(2, 2),
			v(18, 2),
			v(2, 18),
			v(4, 4),
			v(10, 4),
			v(4, 10),
		}
	}
	// The second triangle is inside the first one with the same winding
	// order, so that FillRuleNonZero and FillRuleEvenOdd result in different
	// pixels.
	is := []uint32{0, 1, 2, 3, 4, 5}

	dt := func(options *ebiten.DrawTrianglesOptions) func(*ebiten.Image, int, int) {
		return func(dst *ebiten.Image, ox, oy int) {
			dst.DrawTriangles32(vertices(ox, oy), is, whiteSubImage, options)
		}
	}
	dts := func(options *ebiten.DrawTrianglesShaderOptions) func(*ebiten.Image, int, int) {
		return func(dst *ebiten.Image, ox, oy int) {
			options.Images[0] = whiteSubImage
			dst.DrawTrianglesShader32(vertices(ox, oy), is, shader, options)
		}
	}

	for _, tc := range []struct {
		name string
		draw func(dst *ebiten.Image, ox, oy int)
	}{
		{
			name: "FillRuleNonZero",
			draw: dt(&ebiten.DrawTrianglesOptions{
				FillRule: ebiten.FillRuleNonZero,
			}),
		},
		{
			name: "FillRuleEvenOdd",
			draw: dt(&ebiten.DrawTrianglesOptions{
				FillRule: ebiten.FillRuleEvenOdd,
			}),
		},
		{
			name: "AntiAlias",
			draw: dt(&ebiten.DrawTrianglesOptions{
				AntiAlias: true,
			}),
		},
		{
			name: "AntiAliasFillRuleNonZero",
			draw: dt(&ebiten.DrawTrianglesOptions{
				AntiAlias: true,
				FillRule:  ebiten.FillRuleNonZero,
			}),
		},
		{
			name: "AntiAliasBlendCopy",
			draw: dt(&ebiten.DrawTrianglesOptions{
				AntiAlias: true,
				Blend:     ebiten.BlendCopy,
			}),
		},
		{
			name: "ShaderFillRuleNonZero",
			draw: dts(&ebiten.DrawTrianglesShaderOptions{
				FillRule: ebiten.FillRuleNonZero,
			}),
		},
		{
			name: "ShaderFillRuleEvenOdd",
			draw: dts(&ebiten.DrawTrianglesShaderOptions{
				FillRule: ebiten.FillRuleEvenOdd,
			}),
		},
		{
			name: "ShaderAntiAlias",
			draw: dts(&ebiten.DrawTrianglesShaderOptions{
				AntiAlias: true,
			}),
		},
		{
			name: "ShaderAntiAliasFillRuleNonZero",
			draw: dts(&ebiten.DrawTrianglesShaderOptions{
				AntiAlias: true,
				FillRule:  ebiten.FillRuleNonZero,
			}),
		},
		{
			name: "ShaderAntiAliasBlendCopy",
			draw: dts(&ebiten.DrawTrianglesShaderOptions{
				AntiAlias: true,
				Blend:     ebiten.BlendCopy,
			}),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			background := color.RGBA{0, 0, 0, 0xff}

			dstPlain := ebiten.NewImage(size, size)
			dstPlain.Fill(background)

			base := ebiten.NewImage(dstMin.X+size, dstMin.Y+size)
			base.Fill(background)
			dstSub := base.SubImage(image.Rectangle{Min: dstMin, Max: dstMin.Add(image.Pt(size, size))}).(*ebiten.Image)

			tc.draw(dstPlain, 0, 0)
			tc.draw(dstSub, dstMin.X, dstMin.Y)

			for j := range size {
				for i := range size {
					want := dstPlain.At(i, j).(color.RGBA)
					got := dstSub.At(dstMin.X+i, dstMin.Y+j).(color.RGBA)
					if got != want {
						// Report only the first mismatch to keep the log readable.
						t.Errorf("pixel (%d, %d) of the sub-image: got %v, want %v", i, j, got, want)
						return
					}
				}
			}
		})
	}
}

func TestImageDrawTrianglesWithStencilBufferWithEmptyIndices(t *testing.T) {
	src := ebiten.NewImage(3, 3)
	src.Fill(color.White)

	shader, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, src0Pos vec2, color vec4) vec4 {
	return color
}
`))
	if err != nil {
		t.Fatal(err)
	}

	// BlendCopy with a FillRule draw must not clear the destination even if
	// there is nothing to draw.
	for _, tc := range []struct {
		name string
		draw func(dst *ebiten.Image)
	}{
		{
			name: "FillRuleBlendCopy",
			draw: func(dst *ebiten.Image) {
				dst.DrawTriangles32(nil, nil, src, &ebiten.DrawTrianglesOptions{
					FillRule:      ebiten.FillRuleNonZero,
					CompositeMode: ebiten.CompositeModeCustom,
					Blend:         ebiten.BlendCopy,
				})
			},
		},
		{
			name: "ShaderFillRuleBlendCopy",
			draw: func(dst *ebiten.Image) {
				dst.DrawTrianglesShader32(nil, nil, shader, &ebiten.DrawTrianglesShaderOptions{
					FillRule:      ebiten.FillRuleNonZero,
					CompositeMode: ebiten.CompositeModeCustom,
					Blend:         ebiten.BlendCopy,
				})
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := color.RGBA{0xff, 0, 0, 0xff}
			dst := ebiten.NewImage(3, 3)
			dst.Fill(want)

			// This must be a no-op.
			tc.draw(dst)

			for j := range 3 {
				for i := range 3 {
					if got := dst.At(i, j).(color.RGBA); got != want {
						t.Errorf("pixel (%d, %d): got %v, want %v", i, j, got, want)
					}
				}
			}
		})
	}
}

func TestImageDrawTrianglesWithStencilBufferWithEmptyIndicesStateChecks(t *testing.T) {
	disposedShader, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, src0Pos vec2, color vec4) vec4 {
	return color
}
`))
	if err != nil {
		t.Fatal(err)
	}
	disposedShader.Dispose()

	disposedImage := ebiten.NewImage(3, 3)
	disposedImage.Dispose()

	// The empty-indices check must be below the disposed checks, so that an
	// empty draw detects an invalid source in the same way as a non-empty
	// draw does.
	for _, tc := range []struct {
		name string
		draw func(dst *ebiten.Image)
	}{
		{
			name: "DisposedSourceImage",
			draw: func(dst *ebiten.Image) {
				dst.DrawTriangles32(nil, nil, disposedImage, &ebiten.DrawTrianglesOptions{FillRule: ebiten.FillRuleNonZero})
			},
		},
		{
			name: "DisposedShader",
			draw: func(dst *ebiten.Image) {
				dst.DrawTrianglesShader32(nil, nil, disposedShader, &ebiten.DrawTrianglesShaderOptions{FillRule: ebiten.FillRuleNonZero})
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dst := ebiten.NewImage(3, 3)
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("an empty draw on an invalid source must panic, but it did not")
				}
			}()
			tc.draw(dst)
		})
	}
}

func TestImageDrawTrianglesWithStencilBufferOnDisposedDestination(t *testing.T) {
	src := ebiten.NewImage(3, 3)
	src.Fill(color.White)

	shader, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, src0Pos vec2, color vec4) vec4 {
	return color
}
`))
	if err != nil {
		t.Fatal(err)
	}

	vs := []ebiten.Vertex{{DstX: 0, DstY: 0}, {DstX: 2, DstY: 0}, {DstX: 0, DstY: 2}}
	is := []uint32{0, 1, 2}

	for _, tc := range []struct {
		name string
		draw func(dst *ebiten.Image)
	}{
		{
			name: "FillRule",
			draw: func(dst *ebiten.Image) {
				dst.DrawTriangles32(vs, is, src, &ebiten.DrawTrianglesOptions{FillRule: ebiten.FillRuleNonZero})
			},
		},
		{
			name: "AntiAlias",
			draw: func(dst *ebiten.Image) {
				dst.DrawTriangles32(vs, is, src, &ebiten.DrawTrianglesOptions{AntiAlias: true})
			},
		},
		{
			name: "ShaderFillRule",
			draw: func(dst *ebiten.Image) {
				dst.DrawTrianglesShader32(vs, is, shader, &ebiten.DrawTrianglesShaderOptions{FillRule: ebiten.FillRuleNonZero})
			},
		},
		{
			name: "ShaderAntiAlias",
			draw: func(dst *ebiten.Image) {
				dst.DrawTrianglesShader32(vs, is, shader, &ebiten.DrawTrianglesShaderOptions{AntiAlias: true})
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dst := ebiten.NewImage(3, 3)
			dst.Dispose()
			// This must not panic.
			tc.draw(dst)
		})
	}
}
