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

package vector_test

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func paintCompare(t *testing.T, a, b []byte, tolerance int) {
	t.Helper()
	var maximum, index int
	for i, v := range a {
		d := int(v) - int(b[i])
		if d < 0 {
			d = -d
		}
		if d > maximum {
			maximum, index = d, i
		}
	}
	if maximum > tolerance {
		t.Errorf("maximum error %d at byte %d: got %d, want %d", maximum, index, a[index], b[index])
	}
}

func TestImagePaint(t *testing.T) {
	cpu := image.NewRGBA(image.Rect(0, 0, 12, 10))
	for y := range 10 {
		for x := range 12 {
			cpu.SetRGBA(x, y, color.RGBA{R: uint8(x * 9), G: uint8(y * 11), B: 23, A: 128})
		}
	}
	parent := ebiten.NewImageFromImage(cpu)
	defer parent.Deallocate()
	src := parent.SubImage(image.Rect(3, 2, 9, 8)).(*ebiten.Image)
	for _, filter := range []ebiten.Filter{ebiten.FilterNearest, ebiten.FilterLinear} {
		for _, aa := range []bool{false, true} {
			for _, rule := range []vector.FillRule{vector.FillRuleNonZero, vector.FillRuleEvenOdd} {
				t.Run(fmt.Sprintf("filter=%d/aa=%t/rule=%d", filter, aa, rule), func(t *testing.T) {
					var matrix ebiten.GeoM
					matrix.Scale(6, 4.5)
					matrix.Rotate(0.19)
					matrix.Translate(13.375, 8.375)
					base := clipRectangle(0, 0, 6, 6)
					base.AddPath(clipRectangle(2, 2, 4, 4), nil)
					var path vector.Path
					path.AddPath(base, &vector.AddPathOptions{
						GeoM: matrix,
					})
					identity := &vector.PathClip{
						Path:     &path,
						FillRule: rule,
					}
					nested := intersectionClip(identity, intersectionClip(identity, &vector.PathClip{
						Path: clipRectangle(0, 0, 128, 128),
					}))
					var results [][]byte
					for _, clip := range []vector.Clip{nil, identity, nested} {
						dst := ebiten.NewImage(128, 128)
						vector.FillPath(dst, &path, &vector.FillOptions{
							FillRule: rule,
						}, &vector.DrawPathOptions{
							AntiAlias: aa,
							Clip:      clip,
							Paint: &vector.ImagePaint{
								Image:  src,
								GeoM:   matrix,
								Filter: filter,
							},
						})
						results = append(results, clipPixels(dst))
						dst.Deallocate()
					}
					for _, result := range results[1:] {
						paintCompare(t, result, results[0], 1)
					}
					mask, reference := ebiten.NewImage(128, 128), ebiten.NewImage(128, 128)
					defer mask.Deallocate()
					defer reference.Deallocate()
					vector.FillPath(mask, &path, &vector.FillOptions{
						FillRule: rule,
					}, &vector.DrawPathOptions{
						AntiAlias: aa,
					})
					imageOptions := &ebiten.DrawImageOptions{
						GeoM:   matrix,
						Filter: filter,
					}
					reference.DrawImage(src, imageOptions)
					m, r := clipPixels(mask), clipPixels(reference)
					for i := range r {
						r[i] = byte((int(r[i])*int(m[i/4*4+3]) + 127) / 255)
					}
					inverse := matrix
					inverse.Invert()
					for y := range 128 {
						for x := range 128 {
							px, py := inverse.Apply(float64(x)+0.5, float64(y)+0.5)
							if px < 0.6 || py < 0.6 || px > 5.4 || py > 5.4 {
								continue
							}
							i := 4 * (y*128 + x)
							if filter == ebiten.FilterNearest {
								if math.Abs(px-math.Round(px)) < 0.03 || math.Abs(py-math.Round(py)) < 0.03 {
									continue
								}
								c := cpu.RGBAAt(3+int(px), 2+int(py))
								values := []byte{c.R, c.G, c.B, c.A}
								for j := range 4 {
									r[i+j] = byte((int(values[j])*int(m[i+3]) + 127) / 255)
								}
							}
							paintCompare(t, results[0][i:i+4], r[i:i+4], 2)
						}
					}
					if rule == vector.FillRuleEvenOdd {
						x, y := matrix.Apply(3, 3)
						i := 4 * (int(y)*128 + int(x))
						if results[0][i+3] != 0 {
							t.Errorf("hole alpha = %d", results[0][i+3])
						}
					}
				})
			}
		}
	}
}

func TestImagePaintLifetime(t *testing.T) {
	for _, operation := range []string{"clear", "write", "deallocate", "dispose", "vector"} {
		t.Run(operation, func(t *testing.T) {
			src := ebiten.NewImage(32, 32)
			defer src.Deallocate()
			src.Fill(color.RGBA{R: 255, A: 255})
			sub := src.RecyclableSubImage(image.Rect(7, 5, 23, 21))
			a, b := ebiten.NewImage(32, 32), ebiten.NewImage(32, 32)
			defer a.Deallocate()
			defer b.Deallocate()
			paint := &vector.ImagePaint{
				Image: sub,
			}
			var options vector.DrawPathOptions
			options.Paint = paint
			for _, dst := range []*ebiten.Image{a, b} {
				vector.FillPath(dst, clipRectangle(0, 0, 16, 16), nil, &options)
			}
			paint.Image = nil
			paint.GeoM.Translate(99, 99)
			sub.Recycle()
			view := src.RecyclableSubImage(image.Rect(0, 0, 1, 1))
			defer view.Recycle()
			switch operation {
			case "clear":
				src.Clear()
			case "write":
				src.WritePixels(make([]byte, 32*32*4))
			case "deallocate":
				src.Deallocate()
			case "dispose":
				src.Dispose()
			case "vector":
				op := &vector.DrawPathOptions{}
				op.ColorScale.ScaleWithColor(color.RGBA{B: 255, A: 255})
				vector.FillPath(src, clipRectangle(0, 0, 32, 32), nil, op)
			}
			for _, dst := range []*ebiten.Image{a, b} {
				if got := dst.At(8, 8).(color.RGBA); got != (color.RGBA{R: 255, A: 255}) {
					t.Errorf("captured paint = %v", got)
				}
			}
		})
	}
}

func TestImagePaintDependenciesAndOrder(t *testing.T) {
	a, b, c := ebiten.NewImage(32, 32), ebiten.NewImage(32, 32), ebiten.NewImage(32, 32)
	defer a.Deallocate()
	defer b.Deallocate()
	defer c.Deallocate()
	red := &vector.DrawPathOptions{}
	red.ColorScale.ScaleWithColor(color.RGBA{R: 255, A: 255})
	path := clipRectangle(0, 0, 32, 32)
	vector.FillPath(a, path, nil, red)
	vector.FillPath(b, path, nil, &vector.DrawPathOptions{
		Paint: &vector.ImagePaint{
			Image: a,
		},
	})
	vector.FillPath(c, path, nil, &vector.DrawPathOptions{
		Paint: &vector.ImagePaint{
			Image: b,
		},
	})
	vector.FillPath(a, path, nil, &vector.DrawPathOptions{
		Paint: &vector.ImagePaint{
			Image: c,
		},
	})
	b.Clear()
	c.Clear()
	if got := a.At(5, 5).(color.RGBA); got != (color.RGBA{R: 255, A: 255}) {
		t.Errorf("dependency chain = %v", got)
	}

	blue := ebiten.NewImage(32, 32)
	defer blue.Deallocate()
	blue.Fill(color.RGBA{B: 128, A: 128})
	vector.FillPath(a, path, nil, &vector.DrawPathOptions{
		Paint: &vector.ImagePaint{
			Image: blue,
		},
	})
	vector.FillPath(a, path, nil, red)
	vector.FillPath(a, path, nil, &vector.DrawPathOptions{
		Paint: &vector.ImagePaint{
			Image: blue,
		},
	})
	if got := a.At(5, 5).(color.RGBA); got != (color.RGBA{R: 127, B: 128, A: 255}) {
		t.Errorf("painter order = %v", got)
	}
}

func TestImagePaintBatchOrder(t *testing.T) {
	src := ebiten.NewImage(64, 32)
	defer src.Deallocate()
	left := src.SubImage(image.Rect(0, 0, 32, 32)).(*ebiten.Image)
	right := src.SubImage(image.Rect(32, 0, 64, 32)).(*ebiten.Image)
	left.Fill(color.RGBA{R: 128, A: 128})
	right.Fill(color.RGBA{B: 128, A: 128})
	got, want := ebiten.NewImage(64, 64), ebiten.NewImage(64, 64)
	defer got.Deallocate()
	defer want.Deallocate()
	for i := range 41 {
		path := clipRectangle(float32(i%9), float32(i%7), 60, 60)
		source := left
		solid := &vector.DrawPathOptions{
			AntiAlias: true,
		}
		solid.ColorScale.ScaleWithColor(color.RGBA{R: 128, A: 128})
		if i%2 != 0 {
			source = right
			solid.ColorScale.Reset()
			solid.ColorScale.ScaleWithColor(color.RGBA{B: 128, A: 128})
		}
		var matrix ebiten.GeoM
		matrix.Scale(2, 2)
		op := &vector.DrawPathOptions{
			AntiAlias: true,
			Paint: &vector.ImagePaint{
				Image: source,
				GeoM:  matrix,
			},
		}
		if i%3 == 0 {
			op.Clip = &vector.PathClip{
				Path: path,
			}
			solid.Clip = op.Clip
		}
		vector.FillPath(got, path, nil, op)
		vector.FillPath(want, path, nil, solid)
	}
	paintCompare(t, clipPixels(got), clipPixels(want), 1)
}

func TestImagePaintStrokeAndDestinationSubimage(t *testing.T) {
	cpu := image.NewRGBA(image.Rect(0, 0, 48, 48))
	for y := range 48 {
		for x := range 48 {
			cpu.SetRGBA(x, y, color.RGBA{R: uint8(x * 5), G: uint8(y * 5), B: 30, A: 255})
		}
	}
	for _, filter := range []ebiten.Filter{ebiten.FilterNearest, ebiten.FilterLinear} {
		t.Run(fmt.Sprint(filter), func(t *testing.T) {
			src := ebiten.NewImageFromImage(cpu)
			defer src.Deallocate()
			parent := ebiten.NewImageWithOptions(image.Rect(4, 3, 68, 67), nil)
			defer parent.Deallocate()
			referenceParent := ebiten.NewImageWithOptions(parent.Bounds(), nil)
			defer referenceParent.Deallocate()
			dst := parent.RecyclableSubImage(image.Rect(7, 5, 55, 53))
			defer dst.Recycle()
			reference := referenceParent.RecyclableSubImage(dst.Bounds())
			defer reference.Recycle()
			var matrix ebiten.GeoM
			matrix.Scale(2, 2)
			matrix.Translate(2.25, 3.25)
			var line vector.Path
			line.MoveTo(8, 12)
			line.LineTo(36, 12)
			vector.StrokePath(dst, &line, &vector.StrokeOptions{
				Width:   6,
				LineCap: vector.LineCapRound,
			}, &vector.DrawPathOptions{
				Paint: &vector.ImagePaint{
					Image:  src,
					GeoM:   matrix,
					Filter: filter,
				},
			})
			reference.DrawImage(src, &ebiten.DrawImageOptions{
				GeoM:   matrix,
				Filter: filter,
			})
			src.Clear()
			actual, expected := clipPixels(dst), clipPixels(reference)
			inverse := matrix
			inverse.Invert()
			for y := 10; y < 14; y++ {
				for x := 12; x < 33; x++ {
					px, py := inverse.Apply(float64(x)+0.5, float64(y)+0.5)
					if filter == ebiten.FilterNearest && (math.Abs(px-math.Round(px)) < 1e-6 || math.Abs(py-math.Round(py)) < 1e-6) {
						continue
					}
					at := 4 * ((y-dst.Bounds().Min.Y)*dst.Bounds().Dx() + x - dst.Bounds().Min.X)
					paintCompare(t, actual[at:at+4], expected[at:at+4], 1)
				}
			}
			if got := dst.At(24, 18).(color.RGBA); got.A != 0 {
				t.Errorf("outside stroke = %v, want transparent", got)
			}
		})
	}
}

func TestImagePaintSourceSwitchAfterSolid(t *testing.T) {
	a, b := ebiten.NewImage(24, 8), ebiten.NewImage(24, 8)
	defer a.Deallocate()
	defer b.Deallocate()
	a.Fill(color.RGBA{R: 255, A: 255})
	b.Fill(color.RGBA{B: 255, A: 255})
	dst := ebiten.NewImage(24, 8)
	defer dst.Deallocate()
	vector.FillPath(dst, clipRectangle(0, 0, 24, 8), nil, &vector.DrawPathOptions{
		Paint: &vector.ImagePaint{
			Image: a,
		},
	})
	var solid vector.DrawPathOptions
	solid.ColorScale.ScaleWithColor(color.RGBA{G: 255, A: 255})
	vector.FillPath(dst, clipRectangle(0, 0, 16, 8), nil, &solid)
	vector.FillPath(dst, clipRectangle(0, 0, 8, 8), nil, &vector.DrawPathOptions{
		Paint: &vector.ImagePaint{
			Image: b,
		},
	})
	for _, tc := range []struct {
		x    int
		want color.RGBA
	}{
		{
			x:    4,
			want: color.RGBA{B: 255, A: 255},
		},
		{
			x:    12,
			want: color.RGBA{G: 255, A: 255},
		},
		{
			x:    20,
			want: color.RGBA{R: 255, A: 255},
		},
	} {
		if got := dst.At(tc.x, 4).(color.RGBA); got != tc.want {
			t.Errorf("pixel (%d, 4) after switching image source across a solid fill = %v, want %v", tc.x, got, tc.want)
		}
	}
}

func TestImagePaintSharedSourceDestinationFlush(t *testing.T) {
	src := ebiten.NewImage(16, 16)
	defer src.Deallocate()
	src.Fill(color.RGBA{R: 255, A: 255})
	a, b := ebiten.NewImage(16, 16), ebiten.NewImage(16, 16)
	defer a.Deallocate()
	defer b.Deallocate()
	options := &vector.DrawPathOptions{
		Paint: &vector.ImagePaint{
			Image: src,
		},
	}
	path := clipRectangle(0, 0, 16, 16)
	vector.FillPath(a, path, nil, options)
	vector.FillPath(b, path, nil, options)
	if got := a.At(8, 8).(color.RGBA); got != (color.RGBA{R: 255, A: 255}) {
		t.Errorf("first consumer = %v, want red", got)
	}
	src.Fill(color.RGBA{B: 255, A: 255})
	if got := b.At(8, 8).(color.RGBA); got != (color.RGBA{R: 255, A: 255}) {
		t.Errorf("second consumer after reading first and modifying source = %v, want original red", got)
	}
}

func TestImagePaintSamplingAndBlending(t *testing.T) {
	for _, filter := range []ebiten.Filter{ebiten.FilterNearest, ebiten.FilterLinear} {
		for _, blend := range []ebiten.Blend{ebiten.BlendSourceOver, ebiten.BlendCopy} {
			parent := ebiten.NewImage(16, 16)
			parent.Fill(color.RGBA{G: 255, A: 255})
			src := parent.SubImage(image.Rect(3, 5, 4, 6)).(*ebiten.Image)
			src.Fill(color.RGBA{R: 128, A: 128})
			dst := ebiten.NewImage(8, 8)
			dst.Fill(color.RGBA{B: 255, A: 255})
			var matrix ebiten.GeoM
			matrix.Translate(3.25, 3)
			op := &vector.DrawPathOptions{
				Blend: blend,
				Paint: &vector.ImagePaint{
					Image:  src,
					GeoM:   matrix,
					Filter: filter,
				},
			}
			op.ColorScale.ScaleAlpha(0.5)
			vector.FillPath(dst, clipRectangle(1, 1, 7, 7), nil, op)
			pixels := clipPixels(dst)
			for y := range 8 {
				for x := range 8 {
					var alpha float64
					if y == 3 {
						if filter == ebiten.FilterNearest && x == 3 {
							alpha = 64
						}
						if filter == ebiten.FilterLinear {
							if x == 3 {
								alpha = 48
							}
							if x == 4 {
								alpha = 16
							}
						}
					}
					want := []byte{byte(alpha), 0, 0, byte(alpha)}
					if x < 1 || y < 1 || x >= 7 || y >= 7 {
						want = []byte{0, 0, 255, 255}
					} else if blend == ebiten.BlendSourceOver {
						want[2] = 255 - byte(alpha)
						want[3] = 255
					}
					at := 4 * (y*8 + x)
					paintCompare(t, pixels[at:at+4], want, 1)
				}
			}
			dst.Deallocate()
			parent.Deallocate()
		}
	}

	src := ebiten.NewImage(1, 1)
	defer src.Deallocate()
	src.Fill(color.RGBA{R: 128, G: 64, B: 32, A: 128})
	dst := ebiten.NewImage(1, 1)
	defer dst.Deallocate()
	options := &vector.DrawPathOptions{
		Paint: &vector.ImagePaint{
			Image: src,
		},
	}
	options.ColorScale.Scale(0.25, 0.5, 0.75, 0.5)
	vector.FillPath(dst, clipRectangle(0, 0, 1, 1), nil, options)
	if got := dst.At(0, 0).(color.RGBA); got != (color.RGBA{R: 32, G: 32, B: 24, A: 64}) {
		t.Errorf("per-channel color scale = %v, want (32, 32, 24, 64)", got)
	}
}

func TestImagePaintValidation(t *testing.T) {
	src, dst := ebiten.NewImage(16, 16), ebiten.NewImage(16, 16)
	defer src.Deallocate()
	defer dst.Deallocate()
	var singular ebiten.GeoM
	singular.Scale(0, 1)
	var infinite ebiten.GeoM
	infinite.Translate(math.Inf(1), 0)
	var huge ebiten.GeoM
	huge.Scale(1e30, 1e30)
	cases := []struct {
		name  string
		paint vector.Paint
	}{
		{
			name:  "typed nil",
			paint: (*vector.ImagePaint)(nil),
		},
		{
			name:  "nil source",
			paint: &vector.ImagePaint{},
		},
		{
			name: "filter",
			paint: &vector.ImagePaint{
				Image:  src,
				Filter: ebiten.FilterPixelated,
			},
		},
		{
			name: "singular",
			paint: &vector.ImagePaint{
				Image: src,
				GeoM:  singular,
			},
		},
		{
			name: "infinite",
			paint: &vector.ImagePaint{
				Image: src,
				GeoM:  infinite,
			},
		},
		{
			name: "huge",
			paint: &vector.ImagePaint{
				Image: src,
				GeoM:  huge,
			},
		},
		{
			name: "self",
			paint: &vector.ImagePaint{
				Image: dst,
			},
		},
		{
			name: "alias",
			paint: &vector.ImagePaint{
				Image: dst.SubImage(image.Rect(1, 1, 8, 8)).(*ebiten.Image),
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			op := &vector.DrawPathOptions{
				Paint: tt.paint,
			}
			vector.FillPath(dst, nil, nil, op)
			vector.FillPath(dst, &vector.Path{}, nil, op)
			defer func() {
				if recover() == nil {
					t.Error("invalid paint did not panic")
				}
			}()
			vector.FillPath(dst, clipRectangle(0, 0, 16, 16), nil, op)
		})
	}
}

func TestImagePaintDoesNotRetainDestination(t *testing.T) {
	if !waitForEmptyFillPathsStates(t) {
		t.Skip("other pending fills were not released")
	}
	src := ebiten.NewImage(16, 16)
	defer src.Deallocate()
	func() {
		dst := ebiten.NewImage(16, 16)
		vector.FillPath(dst, clipRectangle(0, 0, 16, 16), nil, &vector.DrawPathOptions{
			Paint: &vector.ImagePaint{
				Image: src,
			},
		})
	}()
	if !waitForEmptyFillPathsStates(t) {
		t.Error("abandoned painted destination was retained")
	}
	src.Fill(color.RGBA{R: 255, A: 255})
	if got := src.At(0, 0).(color.RGBA); got.R != 255 {
		t.Errorf("source = %v", got)
	}
}

func ExampleImagePaint() {
	texture := ebiten.NewImage(16, 16)
	texture.Fill(color.RGBA{R: 255, A: 255})
	var path vector.Path
	path.Arc(32, 32, 24, 0, 2*math.Pi, vector.Clockwise)
	path.Close()
	paint := &vector.ImagePaint{
		Image:  texture,
		Filter: ebiten.FilterLinear,
	}
	paint.GeoM.Scale(4, 4)
	dst := ebiten.NewImage(64, 64)
	vector.FillPath(dst, &path, nil, &vector.DrawPathOptions{
		Paint:     paint,
		AntiAlias: true,
	})
}

func TestImagePaintEmptySource(t *testing.T) {
	source := ebiten.NewImageWithOptions(image.Rect(-10, -10, 10, 10), nil)
	defer source.Deallocate()
	source.Fill(color.RGBA{R: 255, A: 255})
	src := source.SubImage(image.Rect(30, 30, 40, 40)).(*ebiten.Image)
	dst := ebiten.NewImage(16, 16)
	defer dst.Deallocate()
	dst.Fill(color.RGBA{B: 255, A: 255})
	vector.FillPath(dst, clipRectangle(4, 4, 12, 12), nil, &vector.DrawPathOptions{
		Paint: &vector.ImagePaint{
			Image: src,
		},
		Blend: ebiten.BlendCopy,
	})
	if got := dst.At(8, 8).(color.RGBA); got != (color.RGBA{}) {
		t.Errorf("empty source inside path = %v", got)
	}
	if got := dst.At(0, 0).(color.RGBA); got != (color.RGBA{B: 255, A: 255}) {
		t.Errorf("outside path = %v", got)
	}
}
