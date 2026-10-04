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
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func clipRectangle(x0, y0, x1, y1 float32) *vector.Path {
	var p vector.Path
	p.MoveTo(x0, y0)
	p.LineTo(x1, y0)
	p.LineTo(x1, y1)
	p.LineTo(x0, y1)
	p.Close()
	return &p
}

func clipCircle() *vector.Path {
	var p vector.Path
	p.Arc(24.375, 24.375, 17, 0, 2*math.Pi, vector.Clockwise)
	p.Close()
	return &p
}

func pathClip(paths ...*vector.Path) *vector.ClipSet {
	clip := &vector.ClipSet{}
	for _, path := range paths {
		clip.Children = append(clip.Children, &vector.PathClip{
			Path: path,
		})
	}
	return clip
}

func intersectionClip(children ...vector.Clip) *vector.ClipSet {
	return &vector.ClipSet{
		Operation: vector.ClipOperationIntersection,
		Children:  children,
	}
}

func clipRoots(clips []vector.Clip) vector.Clip {
	if len(clips) == 0 {
		return nil
	}
	return intersectionClip(clips...)
}

func TestDrawPathOptionsComparable(t *testing.T) {
	for _, clip := range []vector.Clip{
		nil,
		&vector.PathClip{},
		&vector.ClipSet{},
	} {
		op := vector.DrawPathOptions{
			Clip: clip,
		}
		copied := op
		if op != copied {
			t.Error("copied options must compare equal")
		}
		values := map[vector.DrawPathOptions]int{
			op: 1,
		}
		if got := values[copied]; got != 1 {
			t.Errorf("map lookup: got %d, want 1", got)
		}
	}
}

func TestClipViewport(t *testing.T) {
	const size = 256
	viewport := &vector.PathClip{
		Path: clipRectangle(0, 0, size, size),
	}
	for _, aa := range []bool{false, true} {
		for _, stroke := range []bool{false, true} {
			t.Run(fmt.Sprintf("aa=%t/stroke=%t", aa, stroke), func(t *testing.T) {
				got := ebiten.NewImage(size, size)
				defer got.Deallocate()
				want := ebiten.NewImage(size, size)
				defer want.Deallocate()
				for i := range 32 {
					var shape vector.Path
					transform := &vector.AddPathOptions{}
					transform.GeoM.Translate(float64(i%8)*36-16, float64(i/8)*72-16)
					shape.AddPath(clipCircle(), transform)
					op := &vector.DrawPathOptions{
						AntiAlias: aa,
					}
					op.ColorScale.ScaleAlpha(0.5)
					if stroke {
						vector.StrokePath(want, &shape, &vector.StrokeOptions{
							Width: 7,
						}, op)
						op.Clip = viewport
						vector.StrokePath(got, &shape, &vector.StrokeOptions{
							Width: 7,
						}, op)
					} else {
						vector.FillPath(want, &shape, nil, op)
						op.Clip = viewport
						vector.FillPath(got, &shape, nil, op)
					}
				}
				compareClipPixels(t, clipPixels(got), clipPixels(want))
			})
		}
	}
}

func BenchmarkClipViewport(b *testing.B) {
	const size = 512
	var paths [32]vector.Path
	for i := range paths {
		op := &vector.AddPathOptions{}
		op.GeoM.Translate(float64(i%8)*60, float64(i/8)*120)
		paths[i].AddPath(clipCircle(), op)
	}
	dst := ebiten.NewImage(size, size)
	defer dst.Deallocate()
	op := &vector.DrawPathOptions{
		AntiAlias: true,
		Clip: &vector.PathClip{
			Path: clipRectangle(0, 0, size, size),
		},
	}
	pixels := make([]byte, 4*size*size)
	for range 2 {
		dst.Clear()
		for i := range paths {
			vector.FillPath(dst, &paths[i], nil, op)
		}
		dst.ReadPixels(pixels)
	}
	b.ReportAllocs()
	for b.Loop() {
		dst.Clear()
		for i := range paths {
			vector.FillPath(dst, &paths[i], nil, op)
		}
		dst.ReadPixels(pixels)
	}
}

func clipPixels(dst *ebiten.Image) []byte {
	pixels := make([]byte, 4*dst.Bounds().Dx()*dst.Bounds().Dy())
	dst.ReadPixels(pixels)
	return pixels
}

func compareClipPixels(t *testing.T, got, want []byte) {
	t.Helper()
	if bytes.Equal(got, want) {
		return
	}
	var differences int
	for i := range got {
		if got[i] != want[i] {
			differences++
		}
	}
	t.Errorf("rendering differs in %d of %d channels", differences, len(got))
}

func TestClipIdentity(t *testing.T) {
	for _, antialias := range []bool{false, true} {
		for _, scale := range []int{1, 2, 4} {
			for _, rule := range []vector.FillRule{vector.FillRuleNonZero, vector.FillRuleEvenOdd} {
				t.Run(fmt.Sprintf("aa=%t/scale=%d/rule=%d", antialias, scale, rule), func(t *testing.T) {
					var shape vector.Path
					transform := &vector.AddPathOptions{}
					transform.GeoM.Scale(float64(scale), float64(scale))
					shape.AddPath(clipCircle(), transform)
					shape.AddPath(clipRectangle(18, 18, 30, 30), transform)
					fill := &vector.FillOptions{
						FillRule: rule,
					}
					op := &vector.DrawPathOptions{
						AntiAlias: antialias,
					}
					op.ColorScale.Scale(0.25, 0.5, 0.75, 0.75)
					want := ebiten.NewImage(48*scale, 48*scale)
					defer want.Deallocate()
					vector.FillPath(want, &shape, fill, op)
					expected := clipPixels(want)
					clip := &vector.ClipSet{
						Operation: vector.ClipOperationUnion,
						Children: []vector.Clip{
							&vector.PathClip{
								Path:     &shape,
								FillRule: rule,
							},
						},
					}
					for _, clips := range [][]vector.Clip{nil, {}, {clip}, {clip, clip}} {
						got := ebiten.NewImage(48*scale, 48*scale)
						op.Clip = clipRoots(clips)
						vector.FillPath(got, &shape, fill, op)
						compareClipPixels(t, clipPixels(got), expected)
						got.Deallocate()
					}
				})
			}
		}
	}
}

func TestClipComposition(t *testing.T) {
	left := clipRectangle(4, 4, 20, 36)
	right := clipRectangle(28, 4, 44, 36)
	top := clipRectangle(0, 0, 48, 20)
	bottom := clipRectangle(0, 20, 48, 48)
	circle := clipCircle()
	var disjoint, nested, opposite, nonzeroHole vector.Path
	disjoint.AddPath(left, nil)
	disjoint.AddPath(right, nil)
	nested.AddPath(clipRectangle(4, 4, 20, 20), nil)
	nested.AddPath(clipRectangle(28, 20, 44, 36), nil)
	opposite.MoveTo(4, 4)
	opposite.LineTo(4, 36)
	opposite.LineTo(20, 36)
	opposite.LineTo(20, 4)
	opposite.Close()
	nonzeroHole.AddPath(clipRectangle(0, 0, 48, 48), nil)
	nonzeroHole.AddPath(&opposite, nil)
	var hole vector.Path
	hole.AddPath(clipRectangle(4, 4, 44, 44), nil)
	hole.AddPath(clipRectangle(16, 16, 32, 32), nil)
	cases := []struct {
		name  string
		clips []vector.Clip
		want  *vector.Path
		rule  vector.FillRule
	}{
		{
			name: "empty",
			clips: []vector.Clip{
				&vector.ClipSet{
					Operation: vector.ClipOperationUnion,
				},
			},
			want: &vector.Path{},
		},
		{
			name:  "nil-element",
			clips: []vector.Clip{pathClip(nil)},
			want:  &vector.Path{},
		},
		{
			name:  "curve",
			clips: []vector.Clip{pathClip(circle)},
			want:  circle,
		},
		{
			name:  "disjoint-union",
			clips: []vector.Clip{pathClip(left, right)},
			want:  &disjoint,
		},
		{
			name:  "independent-winding",
			clips: []vector.Clip{pathClip(left, &opposite)},
			want:  left,
		},
		{
			name:  "intersection",
			clips: []vector.Clip{pathClip(left), pathClip(top)},
			want:  clipRectangle(4, 4, 20, 20),
		},
		{
			name:  "shared-boundary",
			clips: []vector.Clip{pathClip(clipRectangle(0, 0, 24.375, 48), clipRectangle(24.375, 0, 48, 48)), pathClip(circle)},
			want:  circle,
		},
		{
			name: "even-odd-hole",
			clips: []vector.Clip{
				&vector.ClipSet{
					Operation: vector.ClipOperationUnion,
					Children: []vector.Clip{
						&vector.PathClip{
							Path:     &hole,
							FillRule: vector.FillRuleEvenOdd,
						},
					},
				},
			},
			want: &hole,
			rule: vector.FillRuleEvenOdd,
		},
		{
			name:  "nonzero-hole",
			clips: []vector.Clip{pathClip(&nonzeroHole)},
			want:  &nonzeroHole,
		},
		{
			name: "mixed-fill-rules",
			clips: []vector.Clip{
				&vector.ClipSet{
					Operation: vector.ClipOperationUnion,
					Children: []vector.Clip{
						&vector.PathClip{
							Path:     &hole,
							FillRule: vector.FillRuleEvenOdd,
						},
						&vector.PathClip{
							Path:     &hole,
							FillRule: vector.FillRuleNonZero,
						},
					},
				},
			},
			want: &hole,
		},
		{
			name: "nested-before-union",
			clips: []vector.Clip{
				&vector.ClipSet{
					Operation: vector.ClipOperationUnion,
					Children: []vector.Clip{
						&vector.ClipSet{
							Operation: vector.ClipOperationIntersection,
							Children: []vector.Clip{
								&vector.PathClip{
									Path: left,
								},
								pathClip(top),
							},
						},
						&vector.ClipSet{
							Operation: vector.ClipOperationIntersection,
							Children: []vector.Clip{
								&vector.PathClip{
									Path: right,
								},
								pathClip(bottom),
							},
						},
					},
				},
			},
			want: &nested,
		},
	}
	for _, antialias := range []bool{false, true} {
		for _, tc := range cases {
			t.Run(fmt.Sprintf("%s/aa=%t", tc.name, antialias), func(t *testing.T) {
				got := ebiten.NewImage(48, 48)
				defer got.Deallocate()
				want := ebiten.NewImage(48, 48)
				defer want.Deallocate()
				vector.FillPath(want, tc.want, &vector.FillOptions{
					FillRule: tc.rule,
				}, &vector.DrawPathOptions{
					AntiAlias: antialias,
				})
				op := &vector.DrawPathOptions{
					AntiAlias: antialias,
					Clip:      clipRoots(tc.clips),
				}
				vector.FillPath(got, clipRectangle(0, 0, 48, 48), nil, op)
				compareClipPixels(t, clipPixels(got), clipPixels(want))
				slices.Reverse(tc.clips)
				for _, clip := range tc.clips {
					set := clip.(*vector.ClipSet)
					slices.Reverse(set.Children)
				}
				got.Clear()
				vector.FillPath(got, clipRectangle(0, 0, 48, 48), nil, op)
				compareClipPixels(t, clipPixels(got), clipPixels(want))
			})
		}
	}
}

func TestClipOperations(t *testing.T) {
	full := clipRectangle(0, 0, 48, 48)
	left := &vector.PathClip{
		Path: clipRectangle(4.375, 4.375, 20.375, 43.375),
	}
	right := &vector.PathClip{
		Path: clipRectangle(28.375, 4.375, 44.375, 43.375),
	}
	top := &vector.PathClip{
		Path: clipRectangle(0, 0, 48, 20.375),
	}
	bottom := &vector.PathClip{
		Path: clipRectangle(0, 28.375, 48, 48),
	}
	empty := &vector.ClipSet{}
	unrestricted := &vector.ClipSet{
		Operation: vector.ClipOperationIntersection,
	}
	var corners vector.Path
	for _, x := range []float32{4.375, 28.375} {
		for _, y := range []float32{4.375, 28.375} {
			corners.AddPath(clipRectangle(x, y, x+16, y+min(16, 43.375-y)), nil)
		}
	}
	var combined vector.Path
	combined.AddPath(left.Path, nil)
	combined.AddPath(right.Path, nil)
	cases := []struct {
		name string
		clip vector.Clip
		want *vector.Path
	}{
		{
			name: "nil-node",
		},
		{
			name: "empty-union",
			clip: empty,
		},
		{
			name: "empty-intersection",
			clip: unrestricted,
			want: full,
		},
		{
			name: "union-two-paths",
			clip: &vector.ClipSet{
				Children: []vector.Clip{
					&vector.PathClip{
						Path: left.Path,
					},
					right,
				},
			},
			want: &combined,
		},
		{
			name: "intersection-two-paths",
			clip: &vector.ClipSet{
				Operation: vector.ClipOperationIntersection,
				Children: []vector.Clip{
					&vector.PathClip{
						Path: left.Path,
					},
					top,
				},
			},
			want: clipRectangle(4.375, 4.375, 20.375, 20.375),
		},
		{
			name: "intersection-of-unions",
			clip: &vector.ClipSet{
				Operation: vector.ClipOperationIntersection,
				Children: []vector.Clip{
					&vector.ClipSet{
						Children: []vector.Clip{left, right},
					},
					&vector.ClipSet{
						Children: []vector.Clip{top, bottom},
					},
				},
			},
			want: &corners,
		},
		{
			name: "union-of-intersections",
			clip: &vector.ClipSet{
				Children: []vector.Clip{
					&vector.ClipSet{
						Operation: vector.ClipOperationIntersection,
						Children:  []vector.Clip{left, top},
					},
					&vector.ClipSet{
						Operation: vector.ClipOperationIntersection,
						Children:  []vector.Clip{left, bottom},
					},
					&vector.ClipSet{
						Operation: vector.ClipOperationIntersection,
						Children:  []vector.Clip{right, top},
					},
					&vector.ClipSet{
						Operation: vector.ClipOperationIntersection,
						Children:  []vector.Clip{right, bottom},
					},
				},
			},
			want: &corners,
		},
		{
			name: "union-nil-child",
			clip: &vector.ClipSet{
				Children: []vector.Clip{
					&vector.PathClip{
						Path: left.Path,
					},
					nil,
				},
			},
			want: left.Path,
		},
		{
			name: "intersection-nil-child",
			clip: &vector.ClipSet{
				Operation: vector.ClipOperationIntersection,
				Children: []vector.Clip{
					&vector.PathClip{
						Path: left.Path,
					},
					nil,
				},
			},
		},
		{
			name: "union-unrestricted-child",
			clip: &vector.ClipSet{
				Children: []vector.Clip{
					&vector.PathClip{
						Path: left.Path,
					},
					unrestricted,
				},
			},
			want: full,
		},
		{
			name: "intersection-unrestricted-child",
			clip: &vector.ClipSet{
				Operation: vector.ClipOperationIntersection,
				Children: []vector.Clip{
					&vector.PathClip{
						Path: left.Path,
					},
					unrestricted,
				},
			},
			want: left.Path,
		},
		{
			name: "shared-union-operand",
			clip: &vector.ClipSet{
				Children: []vector.Clip{left, left, right, right},
			},
			want: &combined,
		},
		{
			name: "shared-intersection-operand",
			clip: &vector.ClipSet{
				Operation: vector.ClipOperationIntersection,
				Children:  []vector.Clip{left, left, top, top},
			},
			want: clipRectangle(4.375, 4.375, 20.375, 20.375),
		},
	}
	for _, aa := range []bool{false, true} {
		for _, tc := range cases {
			t.Run(fmt.Sprintf("%s/aa=%t", tc.name, aa), func(t *testing.T) {
				got := ebiten.NewImage(48, 48)
				defer got.Deallocate()
				want := ebiten.NewImage(48, 48)
				defer want.Deallocate()
				op := &vector.DrawPathOptions{
					AntiAlias: aa,
				}
				vector.FillPath(want, tc.want, nil, op)
				op.Clip = clipRoots([]vector.Clip{tc.clip, tc.clip})
				vector.FillPath(got, full, nil, op)
				compareClipPixels(t, clipPixels(got), clipPixels(want))
			})
		}
	}
}

func TestClipInvalidEmptyIntersection(t *testing.T) {
	cycle := &vector.ClipSet{}
	cycle.Children = []vector.Clip{cycle}
	for _, invalid := range []vector.Clip{
		&vector.ClipSet{
			Operation: vector.ClipOperation(-1),
		},
		&vector.PathClip{
			FillRule: vector.FillRule(-1),
		},
		cycle,
		(*vector.PathClip)(nil),
		(*vector.ClipSet)(nil),
	} {
		func() {
			dst := ebiten.NewImage(48, 48)
			defer dst.Deallocate()
			defer func() {
				if recover() == nil {
					t.Error("invalid child of an empty intersection must panic")
				}
			}()
			vector.FillPath(dst, clipCircle(), nil, &vector.DrawPathOptions{
				Clip: intersectionClip(nil, invalid),
			})
		}()
	}
}

func TestClipWarmMixed(t *testing.T) {
	warm := ebiten.NewImage(512, 512)
	shape := clipRectangle(0, 0, 512, 512)
	vector.FillPath(warm, shape, nil, &vector.DrawPathOptions{
		AntiAlias: true,
		Clip: &vector.PathClip{
			Path: shape,
		},
	})
	clipPixels(warm)
	warm.Deallocate()
	for _, aa := range []bool{false, true} {
		t.Run(fmt.Sprintf("aa=%t", aa), func(t *testing.T) {
			got := ebiten.NewImage(256, 256)
			defer got.Deallocate()
			want := ebiten.NewImage(256, 256)
			defer want.Deallocate()
			for range 3 {
				got.Clear()
				want.Clear()
				for i := range 32 {
					var shape vector.Path
					transform := &vector.AddPathOptions{}
					transform.GeoM.Translate(float64(i%8)*28, float64(i/8)*56)
					shape.AddPath(clipCircle(), transform)
					op := &vector.DrawPathOptions{
						AntiAlias: aa,
					}
					op.ColorScale.Scale(0.25, 0.5, 0.75, 0.5)
					if i%2 == 0 {
						op.Clip = &vector.PathClip{
							Path: &shape,
						}
					}
					vector.FillPath(got, &shape, nil, op)
					op.Clip = nil
					vector.FillPath(want, &shape, nil, op)
				}
				compareClipPixels(t, clipPixels(got), clipPixels(want))
			}
		})
	}
}

func TestClipCheckStateReuse(t *testing.T) {
	for _, cyclic := range []bool{false, true} {
		t.Run(fmt.Sprintf("cyclic=%t", cyclic), func(t *testing.T) {
			shape := clipCircle()
			clip := &vector.ClipSet{}
			got := ebiten.NewImage(48, 48)
			defer got.Deallocate()
			want := ebiten.NewImage(48, 48)
			defer want.Deallocate()
			op := &vector.DrawPathOptions{
				AntiAlias: true,
				Clip:      clip,
			}
			if cyclic {
				clip.Children = []vector.Clip{clip}
				func() {
					defer func() {
						if recover() == nil {
							t.Error("cyclic clip must panic")
						}
					}()
					vector.FillPath(got, shape, nil, op)
				}()
			} else {
				vector.FillPath(got, shape, nil, op)
			}
			clip.Children = []vector.Clip{
				&vector.PathClip{
					Path: shape,
				},
			}
			vector.FillPath(got, shape, nil, op)
			op.Clip = nil
			vector.FillPath(want, shape, nil, op)
			compareClipPixels(t, clipPixels(got), clipPixels(want))
		})
	}
}

func TestClipPlanReuse(t *testing.T) {
	shape := clipRectangle(0, 0, 48, 48)
	union := pathClip(
		clipRectangle(4.375, 4.375, 20.375, 44.375),
		clipRectangle(20.375, 4.375, 40.375, 44.375),
	)
	var chain vector.Clip = &vector.PathClip{
		Path: clipRectangle(8.375, 8.375, 32.375, 32.375),
	}
	for range 32 {
		chain = intersectionClip(chain)
	}
	cases := []struct {
		clip vector.Clip
		want *vector.Path
	}{
		{
			clip: chain,
			want: clipRectangle(8.375, 8.375, 32.375, 32.375),
		},
		{
			clip: union,
			want: clipRectangle(4.375, 4.375, 40.375, 44.375),
		},
		{
			clip: intersectionClip(union, &vector.PathClip{
				Path: clipRectangle(0, 24.375, 48, 48),
			}),
			want: clipRectangle(4.375, 24.375, 40.375, 44.375),
		},
		{
			clip: intersectionClip(),
			want: shape,
		},
	}
	for _, aa := range []bool{false, true} {
		t.Run(fmt.Sprintf("aa=%t", aa), func(t *testing.T) {
			got := ebiten.NewImage(48, 48)
			defer got.Deallocate()
			want := ebiten.NewImage(48, 48)
			defer want.Deallocate()
			for iteration := range 3 {
				for i, tc := range cases {
					t.Run(fmt.Sprintf("iteration=%d/case=%d", iteration, i), func(t *testing.T) {
						got.Clear()
						want.Clear()
						vector.FillPath(got, shape, nil, &vector.DrawPathOptions{
							AntiAlias: aa,
							Clip:      tc.clip,
						})
						vector.FillPath(want, tc.want, nil, &vector.DrawPathOptions{
							AntiAlias: aa,
						})
						compareClipPixels(t, clipPixels(got), clipPixels(want))
					})
				}
			}
		})
	}
}

func TestClipTypedNil(t *testing.T) {
	for _, clip := range []vector.Clip{(*vector.PathClip)(nil), (*vector.ClipSet)(nil)} {
		for _, child := range []bool{false, true} {
			t.Run(fmt.Sprintf("%T/child=%t", clip, child), func(t *testing.T) {
				dst := ebiten.NewImage(48, 48)
				defer dst.Deallocate()
				defer func() {
					if recover() == nil {
						t.Error("typed nil clip must panic")
					}
				}()
				root := clip
				if child {
					root = &vector.ClipSet{
						Children: []vector.Clip{nil, clip},
					}
				}
				vector.FillPath(dst, clipCircle(), nil, &vector.DrawPathOptions{
					Clip: root,
				})
			})
		}
	}
}

func TestClipMissingStencil(t *testing.T) {
	shape := clipCircle()
	miss := &vector.PathClip{
		Path: clipRectangle(56, 0, 64, 64),
	}
	left := &vector.PathClip{
		Path: clipRectangle(0, 0, 24, 48),
	}
	for _, aa := range []bool{false, true} {
		for _, blend := range []ebiten.Blend{ebiten.BlendCopy, ebiten.BlendClear} {
			t.Run(fmt.Sprintf("aa=%t/blend=%v", aa, blend), func(t *testing.T) {
				got := ebiten.NewImage(64, 64)
				defer got.Deallocate()
				want := ebiten.NewImage(64, 64)
				defer want.Deallocate()
				got.Fill(color.White)
				want.Fill(color.White)
				partial := clipRectangle(0, 0, 24, 48)
				vector.FillPath(got, partial, nil, &vector.DrawPathOptions{
					AntiAlias: aa,
					Clip:      left,
					Blend:     blend,
				})
				vector.FillPath(want, partial, nil, &vector.DrawPathOptions{
					AntiAlias: aa,
					Blend:     blend,
				})
				for _, clip := range []vector.Clip{
					miss,
					intersectionClip(left, miss),
					&vector.ClipSet{
						Children: []vector.Clip{miss, miss},
					},
				} {
					vector.FillPath(got, shape, nil, &vector.DrawPathOptions{
						AntiAlias: aa,
						Clip:      clip,
						Blend:     blend,
					})
				}
				vector.FillPath(got, partial, nil, &vector.DrawPathOptions{
					AntiAlias: aa,
					Clip: &vector.ClipSet{
						Children: []vector.Clip{miss, left},
					},
					Blend: blend,
				})
				vector.FillPath(want, partial, nil, &vector.DrawPathOptions{
					AntiAlias: aa,
					Blend:     blend,
				})
				compareClipPixels(t, clipPixels(got), clipPixels(want))
			})
		}
	}
}

func TestClipMixedBlends(t *testing.T) {
	shape := clipCircle()
	shape.AddPath(clipRectangle(16, 16, 32, 32), nil)
	clip := &vector.PathClip{
		Path: clipRectangle(0, 0, 24, 64),
	}
	for _, aa := range []bool{false, true} {
		for _, rule := range []vector.FillRule{vector.FillRuleNonZero, vector.FillRuleEvenOdd} {
			for _, blend := range []ebiten.Blend{ebiten.BlendCopy, ebiten.BlendClear, ebiten.BlendSourceIn, ebiten.BlendDestinationIn} {
				t.Run(fmt.Sprintf("aa=%t/rule=%d/blend=%v", aa, rule, blend), func(t *testing.T) {
					got := ebiten.NewImage(64, 64)
					defer got.Deallocate()
					want := ebiten.NewImage(64, 64)
					defer want.Deallocate()
					got.Fill(color.White)
					want.Fill(color.White)
					for i := range 4 {
						op := &vector.DrawPathOptions{
							AntiAlias: aa,
							Blend:     blend,
						}
						op.ColorScale.Scale(0.25, 0.5, 0.75, 0.5)
						fill := &vector.FillOptions{
							FillRule: rule,
						}
						dst := want
						if i%2 == 0 {
							dst = want.SubImage(image.Rect(0, 0, 24, 64)).(*ebiten.Image)
							op.Clip = clip
						}
						vector.FillPath(got, shape, fill, op)
						op.Clip = nil
						vector.FillPath(dst, shape, fill, op)
					}
					compareClipPixels(t, clipPixels(got), clipPixels(want))
				})
			}
		}
	}
}

func wrappedClipChain(depth int, reverse bool) (vector.Clip, *vector.Path) {
	margin := float32(depth) + 0.375
	region := clipRectangle(margin, margin, 256-margin, 256-margin)
	var clip vector.Clip = &vector.PathClip{
		Path: region,
	}
	for level := depth - 1; level >= 0; level-- {
		margin := float32(level) + 0.375
		wrapper := &vector.ClipSet{
			Children: []vector.Clip{
				&vector.PathClip{
					Path: clipRectangle(margin, margin, 256-margin, 256-margin),
				},
			},
		}
		children := []vector.Clip{clip, wrapper}
		if reverse {
			children[0], children[1] = children[1], children[0]
		}
		clip = intersectionClip(children...)
	}
	strip := clipRectangle(4.375, 8.375, 12.375, 48.375)
	var expected vector.Path
	expected.AddPath(region, nil)
	expected.AddPath(strip, nil)
	return &vector.ClipSet{
		Children: []vector.Clip{
			clip,
			&vector.PathClip{
				Path: strip,
			},
		},
	}, &expected
}

func TestClipWrappedChain(t *testing.T) {
	for _, aa := range []bool{false, true} {
		for _, reverse := range []bool{false, true} {
			t.Run(fmt.Sprintf("aa=%t/reverse=%t", aa, reverse), func(t *testing.T) {
				clip, region := wrappedClipChain(32, reverse)
				got := ebiten.NewImage(256, 256)
				defer got.Deallocate()
				want := ebiten.NewImage(256, 256)
				defer want.Deallocate()
				vector.FillPath(want, region, nil, &vector.DrawPathOptions{
					AntiAlias: aa,
				})
				vector.FillPath(got, clipRectangle(0, 0, 256, 256), nil, &vector.DrawPathOptions{
					AntiAlias: aa,
					Clip:      clip,
				})
				compareClipPixels(t, clipPixels(got), clipPixels(want))
			})
		}
	}
}

func TestClipNodeSnapshot(t *testing.T) {
	for _, aa := range []bool{false, true} {
		t.Run(fmt.Sprintf("aa=%t", aa), func(t *testing.T) {
			full := clipRectangle(0, 0, 48, 48)
			left := &vector.PathClip{
				Path: clipRectangle(0, 0, 24.375, 48),
			}
			top := &vector.PathClip{
				Path: clipRectangle(0, 0, 48, 24.375),
			}
			root := intersectionClip(left, top)
			got := ebiten.NewImage(48, 48)
			defer got.Deallocate()
			want := ebiten.NewImage(48, 48)
			defer want.Deallocate()
			op := &vector.DrawPathOptions{
				AntiAlias: aa,
			}
			vector.FillPath(want, clipRectangle(0, 0, 24.375, 24.375), nil, op)
			op.Clip = clipRoots([]vector.Clip{root})
			vector.FillPath(got, full, nil, op)
			root.Operation = vector.ClipOperationUnion
			root.Children[0] = nil
			left.Path.Reset()
			top.Path.Reset()
			compareClipPixels(t, clipPixels(got), clipPixels(want))
		})
	}
}

func TestClipStrokeAndSnapshot(t *testing.T) {
	for _, aa := range []bool{false, true} {
		t.Run(fmt.Sprint(aa), func(t *testing.T) {
			path := clipCircle()
			stroke := &vector.StrokeOptions{
				Width: 3.5,
			}
			var outline vector.Path
			outline.AddStroke(path, &vector.AddStrokeOptions{
				StrokeOptions: *stroke,
			})
			want := ebiten.NewImage(48, 48)
			defer want.Deallocate()
			got := ebiten.NewImage(48, 48)
			defer got.Deallocate()
			bounds := image.Rect(10, 7, 40, 38)
			op := &vector.DrawPathOptions{
				AntiAlias: aa,
			}
			vector.StrokePath(want.SubImage(bounds).(*ebiten.Image), path, stroke, op)
			op.Clip = clipRoots([]vector.Clip{pathClip(&outline)})
			vector.StrokePath(got.SubImage(bounds).(*ebiten.Image), path, stroke, op)
			outline.Reset()
			path.Reset()
			set := op.Clip.(*vector.ClipSet).Children[0].(*vector.ClipSet)
			set.Children[0].(*vector.PathClip).Path = nil
			set.Children = nil
			compareClipPixels(t, clipPixels(got), clipPixels(want))
		})
	}
}

func TestClipEmptyBlendCopy(t *testing.T) {
	dst := ebiten.NewImage(48, 48)
	defer dst.Deallocate()
	dst.Fill(color.White)
	want := clipPixels(dst)
	vector.FillPath(dst, clipCircle(), nil, &vector.DrawPathOptions{
		AntiAlias: true,
		Blend:     ebiten.BlendCopy,
		Clip: clipRoots([]vector.Clip{
			&vector.ClipSet{
				Operation: vector.ClipOperationUnion,
			},
		}),
	})
	compareClipPixels(t, clipPixels(dst), want)
}

func TestClipEmptyInvalidFillRule(t *testing.T) {
	dst := ebiten.NewImage(48, 48)
	defer dst.Deallocate()
	defer func() {
		if recover() == nil {
			t.Error("invalid fill rule must panic")
		}
	}()
	vector.FillPath(dst, clipCircle(), &vector.FillOptions{
		FillRule: vector.FillRule(-1),
	}, &vector.DrawPathOptions{
		Clip: &vector.ClipSet{},
	})
	clipPixels(dst)
}

func TestClipInvalid(t *testing.T) {
	set := pathClip(clipCircle())
	cyclic := []vector.Clip{set}
	set.Children = append(set.Children, set)
	for _, clips := range [][]vector.Clip{
		cyclic,
		{
			&vector.ClipSet{
				Operation: vector.ClipOperation(-1),
			},
		},
		{
			&vector.ClipSet{
				Operation: vector.ClipOperationUnion,
				Children: []vector.Clip{
					&vector.PathClip{
						FillRule: vector.FillRule(-1),
					},
				},
			},
		},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Error("invalid clip must panic")
				}
			}()
			dst := ebiten.NewImage(48, 48)
			defer dst.Deallocate()
			vector.FillPath(dst, clipCircle(), nil, &vector.DrawPathOptions{
				Clip: clipRoots(clips),
			})
		}()
	}
}

func TestClipBatching(t *testing.T) {
	for _, origin := range []image.Point{image.Pt(0, 0), image.Pt(13, 9), image.Pt(-11, -7)} {
		t.Run(fmt.Sprint(origin), func(t *testing.T) {
			bounds := image.Rectangle{
				Min: origin,
				Max: origin.Add(image.Pt(64, 64)),
			}
			got := ebiten.NewImageWithOptions(bounds, nil)
			defer got.Deallocate()
			want := ebiten.NewImageWithOptions(bounds, nil)
			defer want.Deallocate()
			for i := range 12 {
				x := float32(origin.X + i*4)
				y := float32(origin.Y + i*3)
				shape := clipRectangle(x, y, x+14, y+12)
				clip := clipRectangle(x+3, y+2, x+11, y+10)
				op := &vector.DrawPathOptions{
					AntiAlias: true,
				}
				op.ColorScale.Scale(float32(i%3)/2, 0.5, 0.25, 0.5)
				if i%2 == 0 {
					vector.FillPath(want, clip, nil, op)
					op.Clip = clipRoots([]vector.Clip{pathClip(clip)})
				} else {
					vector.FillPath(want, shape, nil, op)
				}
				vector.FillPath(got, shape, nil, op)
				shape.Reset()
				clip.Reset()
			}
			copied := ebiten.NewImage(64, 64)
			defer copied.Deallocate()
			copied.DrawImage(got, nil)
			expected := clipPixels(want)
			compareClipPixels(t, clipPixels(got), expected)
			compareClipPixels(t, clipPixels(copied), expected)
		})
	}
}

func TestClipPartialStroke(t *testing.T) {
	for _, aa := range []bool{false, true} {
		t.Run(fmt.Sprintf("aa=%t", aa), func(t *testing.T) {
			got := ebiten.NewImage(48, 48)
			want := ebiten.NewImage(48, 48)
			bounds := image.Rect(0, 0, 24, 48)
			stroke := &vector.StrokeOptions{
				Width: 5,
			}
			op := &vector.DrawPathOptions{
				AntiAlias: aa,
			}
			vector.StrokePath(want.SubImage(bounds).(*ebiten.Image), clipCircle(), stroke, op)
			op.Clip = clipRoots([]vector.Clip{pathClip(clipRectangle(0, 0, 24, 48))})
			vector.StrokePath(got, clipCircle(), stroke, op)
			compareClipPixels(t, clipPixels(got), clipPixels(want))
			got.Deallocate()
			want.Deallocate()
		})
	}
}

func TestClipBatchedPassOrder(t *testing.T) {
	shape := clipRectangle(0, 0, 48, 48)
	ring := clipRectangle(4, 4, 44, 44)
	ring.AddPath(clipRectangle(16, 16, 32, 32), nil)
	cases := []struct {
		clips []vector.Clip
		want  *vector.Path
		rule  vector.FillRule
	}{
		{
			want: shape,
		},
		{
			clips: []vector.Clip{
				&vector.ClipSet{
					Operation: vector.ClipOperationUnion,
					Children: []vector.Clip{
						&vector.PathClip{
							Path:     ring,
							FillRule: vector.FillRuleEvenOdd,
						},
					},
				},
			},
			want: ring,
			rule: vector.FillRuleEvenOdd,
		},
		{
			clips: []vector.Clip{
				&vector.ClipSet{
					Operation: vector.ClipOperationUnion,
					Children: []vector.Clip{
						&vector.PathClip{
							Path: clipRectangle(0, 0, 24.375, 48),
						},
						&vector.PathClip{
							Path:     clipRectangle(24.375, 0, 48, 48),
							FillRule: vector.FillRuleEvenOdd,
						},
					},
				},
			},
			want: shape,
		},
		{
			clips: []vector.Clip{
				pathClip(clipRectangle(0, 0, 32.375, 48)),
				pathClip(clipRectangle(0, 0, 48, 32.375)),
			},
			want: clipRectangle(0, 0, 32.375, 32.375),
		},
		{
			clips: []vector.Clip{
				&vector.ClipSet{
					Operation: vector.ClipOperationUnion,
					Children: []vector.Clip{
						&vector.ClipSet{
							Operation: vector.ClipOperationIntersection,
							Children: []vector.Clip{
								&vector.PathClip{
									Path: clipRectangle(4, 4, 20, 36),
								},
								pathClip(clipRectangle(0, 0, 48, 20)),
							},
						},
					},
				},
			},
			want: clipRectangle(4, 4, 20, 20),
		},
		{
			clips: []vector.Clip{
				&vector.ClipSet{
					Operation: vector.ClipOperationUnion,
				},
			},
		},
	}
	for _, aa := range []bool{false, true} {
		for _, bounds := range []image.Rectangle{image.Rect(0, 0, 64, 64), image.Rect(7, 5, 55, 53)} {
			t.Run(fmt.Sprintf("aa=%t/bounds=%v", aa, bounds), func(t *testing.T) {
				got := ebiten.NewImage(64, 64)
				defer got.Deallocate()
				want := ebiten.NewImage(64, 64)
				defer want.Deallocate()
				for i := range 32 {
					tc := cases[i%len(cases)]
					op := &vector.DrawPathOptions{
						AntiAlias: aa,
					}
					op.ColorScale.Scale(float32(i%3)/2, float32(i%5)/4, 0.75, 0.5)
					vector.FillPath(want.SubImage(bounds).(*ebiten.Image), tc.want, &vector.FillOptions{
						FillRule: tc.rule,
					}, op)
					op.Clip = clipRoots(tc.clips)
					vector.FillPath(got.SubImage(bounds).(*ebiten.Image), shape, nil, op)
				}
				compareClipPixels(t, clipPixels(got), clipPixels(want))
			})
		}
	}
}

func BenchmarkClip(b *testing.B) {
	circle := clipCircle()
	var complexPath vector.Path
	for i := range 12 {
		op := &vector.AddPathOptions{}
		op.GeoM.Rotate(float64(i) * math.Pi / 6)
		op.GeoM.Translate(24, 24)
		complexPath.AddPath(circle, op)
	}
	nested := pathClip(circle)
	for range 4 {
		nested = &vector.ClipSet{
			Operation: vector.ClipOperationUnion,
			Children: []vector.Clip{
				&vector.ClipSet{
					Operation: vector.ClipOperationIntersection,
					Children: []vector.Clip{
						&vector.PathClip{
							Path: circle,
						},
						nested,
					},
				},
			},
		}
	}
	for _, tc := range []struct {
		name  string
		clips []vector.Clip
	}{
		{
			name: "none",
		},
		{
			name:  "simple",
			clips: []vector.Clip{pathClip(clipRectangle(0, 0, 24, 48))},
		},
		{
			name:  "complex",
			clips: []vector.Clip{pathClip(&complexPath)},
		},
		{
			name:  "nested",
			clips: []vector.Clip{nested},
		},
		{
			name:  "repeated",
			clips: []vector.Clip{pathClip(circle), pathClip(circle), pathClip(circle)},
		},
	} {
		b.Run(tc.name, func(b *testing.B) {
			dst := ebiten.NewImage(64, 64)
			defer dst.Deallocate()
			pixels := make([]byte, 4*64*64)
			op := &vector.DrawPathOptions{
				AntiAlias: true,
				Clip:      clipRoots(tc.clips),
			}
			vector.FillPath(dst, circle, nil, op)
			dst.ReadPixels(pixels)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				dst.Clear()
				vector.FillPath(dst, circle, nil, op)
				dst.ReadPixels(pixels)
			}
		})
	}
}

func TestClipTransformed(t *testing.T) {
	for _, aa := range []bool{false, true} {
		for _, scale := range []int{1, 2, 4} {
			t.Run(fmt.Sprintf("aa=%t/scale=%d", aa, scale), func(t *testing.T) {
				var shape vector.Path
				transform := &vector.AddPathOptions{}
				transform.GeoM.Rotate(math.Pi / 7)
				transform.GeoM.Translate(15.375, -2.125)
				transform.GeoM.Scale(float64(scale), float64(scale))
				shape.AddPath(clipRectangle(4, 4, 32, 32), transform)
				got := ebiten.NewImage(48*scale, 48*scale)
				defer got.Deallocate()
				want := ebiten.NewImage(48*scale, 48*scale)
				defer want.Deallocate()
				op := &vector.DrawPathOptions{
					AntiAlias: aa,
				}
				vector.FillPath(want, &shape, nil, op)
				op.Clip = clipRoots([]vector.Clip{pathClip(&shape)})
				vector.FillPath(got, clipRectangle(0, 0, float32(48*scale), float32(48*scale)), nil, op)
				compareClipPixels(t, clipPixels(got), clipPixels(want))
			})
		}
	}
}

func TestClipPointSampledOpacityGroup(t *testing.T) {
	const scale = 4
	const size = 48 * scale
	var circle, rectangle vector.Path
	transform := &vector.AddPathOptions{}
	transform.GeoM.Scale(scale, scale)
	circle.AddPath(clipCircle(), transform)
	rectangle.AddPath(clipRectangle(0, 0, 24.375, 48), transform)
	got := ebiten.NewImage(size, size)
	defer got.Deallocate()
	want := ebiten.NewImage(size, size)
	defer want.Deallocate()
	mask := ebiten.NewImage(size, size)
	defer mask.Deallocate()
	vector.FillPath(mask, &circle, nil, nil)
	for _, shape := range []*vector.Path{&circle, &rectangle} {
		op := &vector.DrawPathOptions{}
		op.ColorScale.ScaleAlpha(0.5)
		vector.FillPath(want, shape, nil, op)
		op.Clip = clipRoots([]vector.Clip{pathClip(&circle)})
		vector.FillPath(got, shape, nil, op)
	}
	maskOp := &ebiten.DrawImageOptions{}
	maskOp.Blend = ebiten.BlendDestinationIn
	want.DrawImage(mask, maskOp)
	compareClipPixels(t, clipPixels(got), clipPixels(want))
	gotResolved := ebiten.NewImage(48, 48)
	defer gotResolved.Deallocate()
	wantResolved := ebiten.NewImage(48, 48)
	defer wantResolved.Deallocate()
	resolve := &ebiten.DrawImageOptions{}
	resolve.GeoM.Scale(1.0/scale, 1.0/scale)
	resolve.ColorScale.ScaleAlpha(0.5)
	resolve.Filter = ebiten.FilterLinear
	resolve.DisableMipmaps = true
	gotResolved.DrawImage(got, resolve)
	wantResolved.DrawImage(want, resolve)
	compareClipPixels(t, clipPixels(gotResolved), clipPixels(wantResolved))
}

func TestClipAllBlends(t *testing.T) {
	blends := []struct {
		name  string
		blend ebiten.Blend
	}{
		{
			name:  "BlendSourceOver",
			blend: ebiten.BlendSourceOver,
		},
		{
			name:  "BlendClear",
			blend: ebiten.BlendClear,
		},
		{
			name:  "BlendCopy",
			blend: ebiten.BlendCopy,
		},
		{
			name:  "BlendDestination",
			blend: ebiten.BlendDestination,
		},
		{
			name:  "BlendDestinationOver",
			blend: ebiten.BlendDestinationOver,
		},
		{
			name:  "BlendSourceIn",
			blend: ebiten.BlendSourceIn,
		},
		{
			name:  "BlendDestinationIn",
			blend: ebiten.BlendDestinationIn,
		},
		{
			name:  "BlendSourceOut",
			blend: ebiten.BlendSourceOut,
		},
		{
			name:  "BlendDestinationOut",
			blend: ebiten.BlendDestinationOut,
		},
		{
			name:  "BlendSourceAtop",
			blend: ebiten.BlendSourceAtop,
		},
		{
			name:  "BlendDestinationAtop",
			blend: ebiten.BlendDestinationAtop,
		},
		{
			name:  "BlendXor",
			blend: ebiten.BlendXor,
		},
		{
			name:  "BlendLighter",
			blend: ebiten.BlendLighter,
		},
	}
	for _, tc := range blends {
		for _, aa := range []bool{false, true} {
			for _, rule := range []vector.FillRule{vector.FillRuleNonZero, vector.FillRuleEvenOdd} {
				t.Run(fmt.Sprintf("%s/aa=%t/rule=%d", tc.name, aa, rule), func(t *testing.T) {
					got := ebiten.NewImage(64, 64)
					defer got.Deallocate()
					want := ebiten.NewImage(64, 64)
					defer want.Deallocate()
					got.Fill(color.White)
					want.Fill(color.White)
					op := &vector.DrawPathOptions{
						AntiAlias: aa,
						Blend:     tc.blend,
					}
					op.ColorScale.Scale(0.25, 0.5, 0.75, 0.75)
					fill := &vector.FillOptions{
						FillRule: rule,
					}
					shape := clipCircle()
					vector.FillPath(want, shape, fill, op)
					op.Clip = clipRoots([]vector.Clip{pathClip(clipRectangle(0, 0, 64, 64))})
					vector.FillPath(got, shape, fill, op)
					compareClipPixels(t, clipPixels(got), clipPixels(want))
					if r, g, b, a := want.At(8, 8).RGBA(); r != 0xffff || g != 0xffff || b != 0xffff || a != 0xffff {
						t.Errorf("pixel outside the shape: got %v, want white", want.At(8, 8))
					}
				})
			}
		}
	}
}

func BenchmarkClipBatch(b *testing.B) {
	shape := clipCircle()
	dst := ebiten.NewImage(64, 64)
	defer dst.Deallocate()
	op := &vector.DrawPathOptions{
		AntiAlias: true,
		Clip:      clipRoots([]vector.Clip{pathClip(clipRectangle(0, 0, 24, 48))}),
	}
	pixels := make([]byte, 4*64*64)
	b.ReportAllocs()
	for b.Loop() {
		dst.Clear()
		for range 32 {
			vector.FillPath(dst, shape, nil, op)
		}
		dst.ReadPixels(pixels)
	}
}

func sharedClip(path *vector.Path, depth int) []vector.Clip {
	clips := []vector.Clip{pathClip(path)}
	for range depth {
		clips = []vector.Clip{
			&vector.ClipSet{
				Operation: vector.ClipOperationUnion,
				Children: []vector.Clip{
					&vector.ClipSet{
						Operation: vector.ClipOperationIntersection,
						Children: append([]vector.Clip{
							&vector.PathClip{
								Path: path,
							},
						}, clips...),
					},
					&vector.ClipSet{
						Operation: vector.ClipOperationIntersection,
						Children: append([]vector.Clip{
							&vector.PathClip{
								Path: path,
							},
						}, clips...),
					},
				},
			},
		}
	}
	return clips
}

func TestClipSharedGraph(t *testing.T) {
	for _, aa := range []bool{false, true} {
		t.Run(fmt.Sprintf("aa=%t", aa), func(t *testing.T) {
			shape := clipCircle()
			clips := sharedClip(shape, 32)
			got := ebiten.NewImage(48, 48)
			defer got.Deallocate()
			want := ebiten.NewImage(48, 48)
			defer want.Deallocate()
			op := &vector.DrawPathOptions{
				AntiAlias: aa,
			}
			vector.FillPath(want, shape, nil, op)
			op.Clip = clipRoots(clips)
			vector.FillPath(got, shape, nil, op)
			clips[0].(*vector.ClipSet).Children = nil
			shape.Reset()
			compareClipPixels(t, clipPixels(got), clipPixels(want))
		})
	}
}

func TestClipSharedGraphCycle(t *testing.T) {
	shape := clipCircle()
	clips := sharedClip(shape, 12)
	leaf := clips[0].(*vector.ClipSet)
	for range 12 {
		leaf = leaf.Children[0].(*vector.ClipSet).Children[1].(*vector.ClipSet)
	}
	leaf.Children = append(leaf.Children, clips[0])
	dst := ebiten.NewImage(48, 48)
	defer dst.Deallocate()
	defer func() {
		if recover() == nil {
			t.Error("cyclic shared clip must panic")
		}
	}()
	vector.FillPath(dst, shape, nil, &vector.DrawPathOptions{
		Clip: clipRoots(clips),
	})
}

func TestClipAlternatingSizes(t *testing.T) {
	for _, aa := range []bool{false, true} {
		t.Run(fmt.Sprintf("aa=%t", aa), func(t *testing.T) {
			for _, size := range []image.Point{image.Pt(192, 16), image.Pt(16, 192), image.Pt(192, 16), image.Pt(16, 192)} {
				got := ebiten.NewImage(size.X, size.Y)
				want := ebiten.NewImage(size.X, size.Y)
				shape := clipRectangle(0.375, 0.375, float32(size.X)-0.375, float32(size.Y)-0.375)
				op := &vector.DrawPathOptions{
					AntiAlias: aa,
				}
				vector.FillPath(want, shape, nil, op)
				op.Clip = clipRoots([]vector.Clip{pathClip(shape)})
				vector.FillPath(got, shape, nil, op)
				compareClipPixels(t, clipPixels(got), clipPixels(want))
				got.Deallocate()
				want.Deallocate()
			}
		})
	}
}

func TestClipSharedRestrictions(t *testing.T) {
	for _, aa := range []bool{false, true} {
		t.Run(fmt.Sprintf("aa=%t", aa), func(t *testing.T) {
			left := clipRectangle(4, 4, 20, 36)
			right := clipRectangle(28, 4, 44, 36)
			top := clipRectangle(0, 0, 48, 20)
			clips := []vector.Clip{
				&vector.ClipSet{
					Operation: vector.ClipOperationUnion,
					Children: []vector.Clip{
						&vector.PathClip{
							Path: left,
						},
						&vector.PathClip{
							Path: right,
						},
					},
				},
				pathClip(top),
			}
			set := clips[0].(*vector.ClipSet)
			for i, child := range set.Children {
				set.Children[i] = intersectionClip(child, clips[1])
			}
			var expected vector.Path
			expected.AddPath(clipRectangle(4, 4, 20, 20), nil)
			expected.AddPath(clipRectangle(28, 4, 44, 20), nil)
			got := ebiten.NewImage(48, 48)
			defer got.Deallocate()
			want := ebiten.NewImage(48, 48)
			defer want.Deallocate()
			op := &vector.DrawPathOptions{
				AntiAlias: aa,
			}
			vector.FillPath(want, &expected, nil, op)
			op.Clip = clipRoots(clips)
			vector.FillPath(got, clipRectangle(0, 0, 48, 48), nil, op)
			compareClipPixels(t, clipPixels(got), clipPixels(want))
		})
	}
}

func BenchmarkClipDraws(b *testing.B) {
	var paths [32]vector.Path
	var options [32]vector.DrawPathOptions
	for i := range paths {
		transform := &vector.AddPathOptions{}
		transform.GeoM.Translate(float64(i%8)*28, float64(i/8)*56)
		paths[i].AddPath(clipCircle(), transform)
		options[i].AntiAlias = true
		options[i].ColorScale.ScaleAlpha(0.5)
		options[i].Clip = clipRoots([]vector.Clip{pathClip(&paths[i])})
	}
	for _, tc := range []struct {
		name  string
		count int
		half  bool
	}{
		{
			name:  "Single",
			count: 1,
		},
		{
			name:  "All32",
			count: 32,
		},
		{
			name:  "Half32",
			count: 32,
			half:  true,
		},
	} {
		b.Run(tc.name, func(b *testing.B) {
			dst := ebiten.NewImage(256, 256)
			defer dst.Deallocate()
			pixels := make([]byte, 4*256*256)
			b.ReportAllocs()
			for b.Loop() {
				dst.Clear()
				for i := range tc.count {
					op := options[i]
					if tc.half && i%2 != 0 {
						op.Clip = nil
					}
					vector.FillPath(dst, &paths[i], nil, &op)
				}
				dst.ReadPixels(pixels)
			}
		})
	}
}

func TestClipRootDependencyOrder(t *testing.T) {
	for _, aa := range []bool{false, true} {
		t.Run(fmt.Sprintf("aa=%t", aa), func(t *testing.T) {
			clips := []vector.Clip{
				pathClip(clipRectangle(0, 0, 48, 24)),
				pathClip(clipRectangle(0, 0, 24, 48)),
			}
			clips[1] = intersectionClip(clips[1], clips[0])
			got := ebiten.NewImage(48, 48)
			defer got.Deallocate()
			want := ebiten.NewImage(48, 48)
			defer want.Deallocate()
			op := &vector.DrawPathOptions{
				AntiAlias: aa,
			}
			vector.FillPath(want, clipRectangle(0, 0, 24, 24), nil, op)
			op.Clip = clipRoots(clips)
			vector.FillPath(got, clipRectangle(0, 0, 48, 48), nil, op)
			compareClipPixels(t, clipPixels(got), clipPixels(want))
		})
	}
}

func TestClipLargeBatchOrder(t *testing.T) {
	const size = 512
	got := ebiten.NewImage(size, size)
	defer got.Deallocate()
	want := ebiten.NewImage(size, size)
	defer want.Deallocate()
	shape := clipRectangle(0, 0, size, size)
	for i := range 16 {
		op := &vector.DrawPathOptions{
			AntiAlias: true,
		}
		op.ColorScale.Scale(float32(i%3)/2, 0.5, float32(i%5)/4, 0.5)
		if i%2 == 0 {
			clip := clipRectangle(0, 0, float32(size-i*8), size)
			vector.FillPath(want, clip, nil, op)
			op.Clip = clipRoots([]vector.Clip{pathClip(clip)})
		} else {
			vector.FillPath(want, shape, nil, op)
		}
		vector.FillPath(got, shape, nil, op)
	}
	compareClipPixels(t, clipPixels(got), clipPixels(want))
}

func BenchmarkClipOversized(b *testing.B) {
	const size = 1300
	dst := ebiten.NewImage(size, size)
	defer dst.Deallocate()
	shape := clipRectangle(0, 0, size, size)
	clip := clipRectangle(0, 0, 650.375, size)
	op := &vector.DrawPathOptions{
		AntiAlias: true,
		Clip:      clipRoots([]vector.Clip{pathClip(clip)}),
	}
	pixels := make([]byte, 4*size*size)
	b.ReportAllocs()
	for b.Loop() {
		dst.Clear()
		vector.FillPath(dst, shape, nil, op)
		dst.ReadPixels(pixels)
	}
}

func TestClipOversized(t *testing.T) {
	const size = 1300
	got := ebiten.NewImage(size, size)
	defer got.Deallocate()
	want := ebiten.NewImage(size, size)
	defer want.Deallocate()
	shape := clipRectangle(0, 0, size, size)
	clip := clipRectangle(0, 0, 650.375, size)
	op := &vector.DrawPathOptions{
		AntiAlias: true,
	}
	vector.FillPath(want, clip, nil, op)
	expected := clipPixels(want)
	op.Clip = clipRoots([]vector.Clip{pathClip(clip)})
	for draw := range 2 {
		t.Run(fmt.Sprintf("draw=%d", draw), func(t *testing.T) {
			got.Clear()
			vector.FillPath(got, shape, nil, op)
			compareClipPixels(t, clipPixels(got), expected)
		})
	}
}

func TestClipRootChains(t *testing.T) {
	clips := []vector.Clip{
		pathClip(clipRectangle(0, 0, 40, 48)),
		pathClip(clipRectangle(0, 0, 48, 40)),
		pathClip(clipRectangle(8, 0, 48, 48), clipRectangle(8, 0, 48, 48)),
		pathClip(clipRectangle(0, 8, 48, 48), clipRectangle(0, 8, 48, 48)),
	}
	clips[1] = intersectionClip(clips[1], clips[0])
	third := clips[2].(*vector.ClipSet)
	third.Children[0] = intersectionClip(third.Children[0], clips[1])
	third.Children[1] = intersectionClip(third.Children[1], clips[0])
	fourth := clips[3].(*vector.ClipSet)
	for i, child := range fourth.Children {
		fourth.Children[i] = intersectionClip(child, clips[2])
	}
	for _, aa := range []bool{false, true} {
		for start := range 4 {
			t.Run(fmt.Sprintf("aa=%t/start=%d", aa, start), func(t *testing.T) {
				got := ebiten.NewImage(48, 48)
				defer got.Deallocate()
				want := ebiten.NewImage(48, 48)
				defer want.Deallocate()
				bottom := float32(48)
				if start < 2 {
					bottom = 40
				}
				op := &vector.DrawPathOptions{
					AntiAlias: aa,
				}
				vector.FillPath(want, clipRectangle(8, 8, 40, bottom), nil, op)
				op.Clip = clipRoots(clips[start:])
				vector.FillPath(got, clipRectangle(0, 0, 48, 48), nil, op)
				compareClipPixels(t, clipPixels(got), clipPixels(want))
			})
		}
	}
}

type clipPoolDraw struct {
	path  *vector.Path
	clips []vector.Clip
	color color.RGBA
	flush bool
}

func clipPoolDraws(pattern string) []clipPoolDraw {
	large := clipRectangle(0, 0, 1300, 1300)
	blue := color.RGBA{R: 70, G: 160, B: 240, A: 255}
	orange := color.RGBA{R: 240, G: 160, B: 70, A: 255}
	switch pattern {
	case "Mixed":
		small := clipRectangle(1400, 1400, 1448, 1448)
		return []clipPoolDraw{
			{
				path:  large,
				clips: []vector.Clip{pathClip(large)},
				color: blue,
			},
			{
				path:  small,
				clips: []vector.Clip{pathClip(small)},
				color: orange,
				flush: true,
			},
		}
	case "Nested":
		return []clipPoolDraw{
			{
				path: large,
				clips: []vector.Clip{
					&vector.ClipSet{
						Operation: vector.ClipOperationUnion,
						Children: []vector.Clip{
							&vector.PathClip{
								Path: large,
							},
							&vector.ClipSet{
								Operation: vector.ClipOperationIntersection,
								Children: []vector.Clip{
									&vector.PathClip{
										Path: large,
									},
									pathClip(large),
								},
							},
						},
					},
				},
				color: blue,
				flush: true,
			},
		}
	case "WideTall":
		wide := clipRectangle(0, 0, 1500, 64)
		tall := clipRectangle(0, 0, 64, 1500)
		return []clipPoolDraw{
			{
				path:  wide,
				clips: []vector.Clip{pathClip(wide)},
				color: blue,
				flush: true,
			},
			{
				path:  tall,
				clips: []vector.Clip{pathClip(tall)},
				color: orange,
				flush: true,
			},
		}
	default:
		return []clipPoolDraw{
			{
				path:  large,
				clips: []vector.Clip{pathClip(large)},
				color: blue,
				flush: true,
			},
		}
	}
}

func TestClipPoolPatterns(t *testing.T) {
	const size = 1500
	for _, pattern := range []string{"Mixed", "Nested", "WideTall"} {
		t.Run(pattern, func(t *testing.T) {
			got := ebiten.NewImage(size, size)
			defer got.Deallocate()
			want := ebiten.NewImage(size, size)
			defer want.Deallocate()
			gotPixels := make([]byte, 4*size*size)
			wantPixels := make([]byte, 4*size*size)
			draws := clipPoolDraws(pattern)
			for frame := range 3 {
				t.Run(fmt.Sprintf("frame=%d", frame), func(t *testing.T) {
					got.Clear()
					want.Clear()
					for _, draw := range draws {
						op := &vector.DrawPathOptions{
							AntiAlias: true,
						}
						op.ColorScale.ScaleWithColor(draw.color)
						vector.FillPath(want, draw.path, nil, op)
						op.Clip = clipRoots(draw.clips)
						vector.FillPath(got, draw.path, nil, op)
						if draw.flush {
							want.ReadPixels(wantPixels)
							got.ReadPixels(gotPixels)
							compareClipPixels(t, gotPixels, wantPixels)
						}
					}
				})
			}
		})
	}
}

func BenchmarkClipPoolPatterns(b *testing.B) {
	const size = 1500
	for _, pattern := range []string{"Mixed", "Nested", "WideTall", "Plain"} {
		b.Run(pattern, func(b *testing.B) {
			dst := ebiten.NewImage(size, size)
			defer dst.Deallocate()
			pixels := make([]byte, 4*size*size)
			draws := clipPoolDraws(pattern)
			b.ReportAllocs()
			for b.Loop() {
				dst.Clear()
				for _, draw := range draws {
					op := &vector.DrawPathOptions{
						AntiAlias: true,
						Clip:      clipRoots(draw.clips),
					}
					op.ColorScale.ScaleWithColor(draw.color)
					vector.FillPath(dst, draw.path, nil, op)
					if draw.flush {
						dst.ReadPixels(pixels)
					}
				}
			}
		})
	}
}

type clipCrossBatchDraw struct {
	path        *vector.Path
	clips       []vector.Clip
	destination int
	antialias   bool
	color       color.RGBA
}

func clipCrossBatchDraws(separate bool) []clipCrossBatchDraw {
	large := clipRectangle(0, 0, 1300, 1300)
	small := clipRectangle(1400, 1400, 1448, 1448)
	var destination int
	if separate {
		destination = 1
		small = clipRectangle(0, 0, 48, 48)
	}
	return []clipCrossBatchDraw{
		{
			path:      large,
			clips:     []vector.Clip{pathClip(large)},
			antialias: true,
			color:     color.RGBA{R: 70, G: 160, B: 240, A: 255},
		},
		{
			path:        small,
			clips:       []vector.Clip{pathClip(small)},
			destination: destination,
			antialias:   separate,
			color:       color.RGBA{R: 240, G: 160, B: 70, A: 255},
		},
	}
}

func TestClipCrossBatches(t *testing.T) {
	for _, tc := range []struct {
		name     string
		separate bool
	}{
		{
			name: "AAChange",
		},
		{
			name:     "Destinations",
			separate: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got, want [2]*ebiten.Image
			var gotPixels, wantPixels [2][]byte
			order := []int{0}
			if tc.separate {
				order = []int{1, 0}
			}
			for _, i := range order {
				size := 1500
				if i == 1 {
					size = 48
				}
				got[i] = ebiten.NewImage(size, size)
				defer got[i].Deallocate()
				want[i] = ebiten.NewImage(size, size)
				defer want[i].Deallocate()
				gotPixels[i] = make([]byte, 4*size*size)
				wantPixels[i] = make([]byte, 4*size*size)
			}
			draws := clipCrossBatchDraws(tc.separate)
			for frame := range 3 {
				t.Run(fmt.Sprintf("frame=%d", frame), func(t *testing.T) {
					for _, i := range order {
						got[i].Clear()
						want[i].Clear()
					}
					for _, draw := range draws {
						op := &vector.DrawPathOptions{
							AntiAlias: draw.antialias,
						}
						op.ColorScale.ScaleWithColor(draw.color)
						vector.FillPath(want[draw.destination], draw.path, nil, op)
						op.Clip = clipRoots(draw.clips)
						vector.FillPath(got[draw.destination], draw.path, nil, op)
					}
					for _, i := range order {
						want[i].ReadPixels(wantPixels[i])
						got[i].ReadPixels(gotPixels[i])
						compareClipPixels(t, gotPixels[i], wantPixels[i])
					}
				})
			}
		})
	}
}

func BenchmarkClipCrossBatches(b *testing.B) {
	for _, tc := range []struct {
		name     string
		separate bool
	}{
		{
			name: "AAChange",
		},
		{
			name:     "Destinations",
			separate: true,
		},
	} {
		b.Run(tc.name, func(b *testing.B) {
			var images [2]*ebiten.Image
			var pixels [2][]byte
			order := []int{0}
			if tc.separate {
				order = []int{1, 0}
			}
			for _, i := range order {
				size := 1500
				if i == 1 {
					size = 48
				}
				images[i] = ebiten.NewImage(size, size)
				defer images[i].Deallocate()
				pixels[i] = make([]byte, 4*size*size)
			}
			draws := clipCrossBatchDraws(tc.separate)
			b.ReportAllocs()
			for b.Loop() {
				for _, i := range order {
					images[i].Clear()
				}
				for _, draw := range draws {
					op := &vector.DrawPathOptions{
						AntiAlias: draw.antialias,
						Clip:      clipRoots(draw.clips),
					}
					op.ColorScale.ScaleWithColor(draw.color)
					vector.FillPath(images[draw.destination], draw.path, nil, op)
				}
				for _, i := range order {
					images[i].ReadPixels(pixels[i])
				}
			}
		})
	}
}

func narrowingClip(depth int, mixed bool) []vector.Clip {
	var clips []vector.Clip
	for level := depth; level >= 0; level-- {
		margin := float32(level) + 0.375
		element := &vector.ClipSet{
			Operation: vector.ClipOperationIntersection,
			Children: append([]vector.Clip{
				&vector.PathClip{
					Path: clipRectangle(margin, margin, 256-margin, 256-margin),
				},
			}, clips...),
		}
		elements := []vector.Clip{element}
		if mixed && level%5 == 0 {
			elements = append(elements, element)
		}
		clips = []vector.Clip{
			&vector.ClipSet{
				Operation: vector.ClipOperationUnion,
				Children:  elements,
			},
		}
	}
	return clips
}

func TestClipNestedOwnership(t *testing.T) {
	const size = 256
	for _, aa := range []bool{false, true} {
		for _, tc := range []struct {
			name  string
			clips []vector.Clip
			want  *vector.Path
		}{
			{
				name:  "Chain32",
				clips: narrowingClip(32, false),
				want:  clipRectangle(32.375, 32.375, 223.625, 223.625),
			},
			{
				name:  "MixedChain32",
				clips: narrowingClip(32, true),
				want:  clipRectangle(32.375, 32.375, 223.625, 223.625),
			},
			{
				name: "EmptyOperandWithChild",
				clips: []vector.Clip{
					&vector.ClipSet{
						Operation: vector.ClipOperationIntersection,
						Children: []vector.Clip{
							nil,
							&vector.PathClip{
								Path: clipRectangle(16, 16, 240, 240),
							},
						},
					},
				},
			},
			{
				name: "EmptyChild",
				clips: []vector.Clip{
					&vector.ClipSet{
						Operation: vector.ClipOperationUnion,
						Children: []vector.Clip{
							&vector.ClipSet{
								Operation: vector.ClipOperationIntersection,
								Children: []vector.Clip{
									&vector.PathClip{
										Path: clipRectangle(0, 0, size, size),
									},
									&vector.ClipSet{
										Operation: vector.ClipOperationUnion,
									},
								},
							},
						},
					},
				},
				want: &vector.Path{},
			},
		} {
			t.Run(fmt.Sprintf("%s/aa=%t", tc.name, aa), func(t *testing.T) {
				got := ebiten.NewImage(size, size)
				defer got.Deallocate()
				want := ebiten.NewImage(size, size)
				defer want.Deallocate()
				op := &vector.DrawPathOptions{
					AntiAlias: aa,
				}
				vector.FillPath(want, tc.want, nil, op)
				op.Clip = clipRoots(tc.clips)
				vector.FillPath(got, clipRectangle(0, 0, size, size), nil, op)
				compareClipPixels(t, clipPixels(got), clipPixels(want))
			})
		}
	}
}
