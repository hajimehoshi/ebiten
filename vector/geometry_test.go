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
	"image"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func TestStrokeCollinearQuad(t *testing.T) {
	for _, tc := range []struct {
		name              string
		control, end, tip image.Point
	}{
		{
			name:    "beyond end",
			control: image.Pt(120, 0),
			end:     image.Pt(90, 0),
			tip:     image.Pt(96, 0),
		},
		{
			name:    "before start",
			control: image.Pt(-30, 0),
			end:     image.Pt(90, 0),
			tip:     image.Pt(-6, 0),
		},
		{
			name:    "vertical",
			control: image.Pt(0, 120),
			end:     image.Pt(0, 90),
			tip:     image.Pt(0, 96),
		},
		{
			name:    "diagonal",
			control: image.Pt(120, 120),
			end:     image.Pt(90, 90),
			tip:     image.Pt(96, 96),
		},
		{
			name:    "between endpoints",
			control: image.Pt(30, 0),
			end:     image.Pt(90, 0),
			tip:     image.Pt(90, 0),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const offset = 40
			var curve, lines vector.Path
			curve.MoveTo(offset, offset)
			curve.QuadTo(float32(tc.control.X+offset), float32(tc.control.Y+offset), float32(tc.end.X+offset), float32(tc.end.Y+offset))
			lines.MoveTo(offset, offset)
			lines.LineTo(float32(tc.tip.X+offset), float32(tc.tip.Y+offset))
			if tc.tip != tc.end {
				lines.LineTo(float32(tc.end.X+offset), float32(tc.end.Y+offset))
			}
			for _, join := range []vector.LineJoin{vector.LineJoinMiter, vector.LineJoinBevel, vector.LineJoinRound} {
				for _, cap := range []vector.LineCap{vector.LineCapButt, vector.LineCapRound, vector.LineCapSquare} {
					op := &vector.StrokeOptions{
						Width:    8,
						LineJoin: join,
						LineCap:  cap,
					}
					got := ebiten.NewImage(180, 180)
					defer got.Deallocate()
					want := ebiten.NewImage(180, 180)
					defer want.Deallocate()
					vector.StrokePath(got, &curve, op, nil)
					vector.StrokePath(want, &lines, op, nil)
					gotPixels := make([]byte, 4*180*180)
					wantPixels := make([]byte, len(gotPixels))
					got.ReadPixels(gotPixels)
					want.ReadPixels(wantPixels)
					if !bytes.Equal(gotPixels, wantPixels) {
						t.Errorf("join %d, cap %d: stroked quadratic differs from the out-and-back lines", join, cap)
					}
				}
			}
		})
	}
}

func TestArePointsInRangeExtreme(t *testing.T) {
	for _, tc := range []struct {
		name     string
		p0, p1   vector.Point
		min, max float32
		want     bool
	}{
		{
			name: "overflowed distance and maximum",
			p1:   vector.Point{X: 2e20},
			max:  1e20,
		},
		{
			name: "overflowed minimum",
			p1:   vector.Point{X: 1e20},
			min:  2e20,
			max:  3e20,
		},
		{
			name: "overflowed difference",
			p0:   vector.Point{X: -3e38},
			p1:   vector.Point{X: 3e38},
			max:  math.MaxFloat32,
		},
		{
			name: "underflowed maximum",
			p1:   vector.Point{Y: 1e-30},
			max:  1e-31,
		},
		{
			name: "underflowed minimum",
			p1:   vector.Point{Y: 1e-31},
			min:  1e-30,
			max:  1e-29,
		},
		{
			name: "inclusive tiny bounds",
			p1:   vector.Point{Y: math.SmallestNonzeroFloat32},
			min:  math.SmallestNonzeroFloat32,
			max:  math.SmallestNonzeroFloat32,
			want: true,
		},
		{
			name: "squared negative allowances",
			p1:   vector.Point{X: 2},
			min:  -1,
			max:  -3,
			want: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := vector.ArePointsInRange(tc.p0, tc.p1, tc.min, tc.max); got != tc.want {
				t.Errorf("ArePointsInRange: got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsPointCloseToSegmentExtreme(t *testing.T) {
	for _, tc := range []struct {
		name      string
		p, p0, p1 vector.Point
		allow     float32
		want      bool
	}{
		{
			name:  "overflowed products",
			p:     vector.Point{Y: 1e20},
			p1:    vector.Point{X: 1e20},
			allow: 1,
		},
		{
			name:  "overflowed difference",
			p:     vector.Point{Y: 1},
			p0:    vector.Point{X: -3e38},
			p1:    vector.Point{X: 3e38},
			allow: 2,
			want:  true,
		},
		{
			name:  "coincident endpoints with overflow",
			p:     vector.Point{X: 2e20},
			allow: 1e20,
		},
		{
			name:  "underflowed products",
			p:     vector.Point{Y: 1e-30},
			p1:    vector.Point{X: 1e-30},
			allow: 1e-31,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := vector.IsPointCloseToSegment(tc.p, tc.p0, tc.p1, tc.allow); got != tc.want {
				t.Errorf("IsPointCloseToSegment: got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCrossingPointForTwoLinesExtreme(t *testing.T) {
	for _, tc := range []struct {
		name               string
		p00, p01, p10, p11 vector.Point
		want               vector.Point
	}{
		{
			name: "overflowed products",
			p01:  vector.Point{X: 1e20, Y: 1e20},
			p10:  vector.Point{Y: 1e20},
			p11:  vector.Point{X: 1e20},
			want: vector.Point{X: 5e19, Y: 5e19},
		},
		{
			name: "overflowed differences",
			p00:  vector.Point{X: -3e38, Y: -3e38},
			p01:  vector.Point{X: 3e38, Y: 3e38},
			p10:  vector.Point{X: -3e38, Y: 3e38},
			p11:  vector.Point{X: 3e38, Y: -3e38},
		},
		{
			name: "parallel",
			p01:  vector.Point{X: 1},
			p10:  vector.Point{Y: 1},
			p11:  vector.Point{X: 1, Y: 1},
			want: vector.Point{X: float32(math.NaN()), Y: float32(math.NaN())},
		},
		{
			name: "nearly parallel",
			p01:  vector.Point{X: 1},
			p10:  vector.Point{Y: 1},
			p11:  vector.Point{X: 1, Y: 1 + 1e-7},
			want: vector.Point{X: float32(math.NaN()), Y: float32(math.NaN())},
		},
		{
			name: "unrepresentable final intersection",
			p00:  vector.Point{X: 3e38},
			p01:  vector.Point{X: 3e38, Y: 1},
			p11:  vector.Point{X: 1, Y: 2},
			want: vector.Point{X: 3e38, Y: float32(math.Inf(1))},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := vector.CrossingPointForTwoLines(tc.p00, tc.p01, tc.p10, tc.p11)
			for _, pair := range [][2]float32{{got.X, tc.want.X}, {got.Y, tc.want.Y}} {
				if pair[0] != pair[1] && !(math.IsNaN(float64(pair[0])) && math.IsNaN(float64(pair[1]))) {
					t.Errorf("CrossingPointForTwoLines: got %v, want %v", got, tc.want)
					break
				}
			}
		})
	}
}

func TestArcToExtremeDirections(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		start, corner, end, want vector.Point
		radius                   float32
		tolerance                float64
	}{
		{
			name:      "overflowed distance check",
			corner:    vector.Point{Y: 2e20},
			end:       vector.Point{X: 4e20, Y: 2e20},
			want:      vector.Point{X: 1e20, Y: 2e20},
			radius:    1e20,
			tolerance: 1e14,
		},
		{
			name:      "overflowed direction difference",
			start:     vector.Point{X: -3e38},
			corner:    vector.Point{X: 3e38},
			end:       vector.Point{X: 3e38, Y: 2e38},
			want:      vector.Point{X: 3e38, Y: 1e38},
			radius:    1e38,
			tolerance: 1e32,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var p vector.Path
			p.MoveTo(tc.start.X, tc.start.Y)
			p.ArcTo(tc.corner.X, tc.corner.Y, tc.end.X, tc.end.Y, tc.radius)
			got, ok := vector.CurrentPosition(&p)
			if !ok {
				t.Fatal("ArcTo produced no position")
			}
			assertGeometryPoint(t, got, tc.want, tc.tolerance)
		})
	}
}

func TestStrokeExtremeDirections(t *testing.T) {
	for _, tc := range []struct {
		name             string
		start, end, want vector.Point
	}{
		{
			name: "overflowed length",
			end:  vector.Point{X: 3e38, Y: 3e38},
			want: vector.Point{X: -math.Sqrt2, Y: math.Sqrt2},
		},
		{
			name:  "overflowed difference",
			start: vector.Point{X: -3e38},
			end:   vector.Point{X: 3e38},
			want:  vector.Point{X: -3e38, Y: 2},
		},
		{
			name: "subnormal length",
			end:  vector.Point{X: math.SmallestNonzeroFloat32, Y: math.SmallestNonzeroFloat32},
			want: vector.Point{X: -math.Sqrt2, Y: math.Sqrt2},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var src vector.Path
			src.MoveTo(tc.start.X, tc.start.Y)
			src.LineTo(tc.end.X, tc.end.Y)
			var dst vector.Path
			dst.AddStroke(&src, &vector.AddStrokeOptions{
				StrokeOptions: vector.StrokeOptions{
					Width: 4,
				},
			})
			vertices, _ := dst.AppendVerticesAndIndicesForFilling(nil, nil)
			assertGeometryVertex(t, vertices, tc.want, 1e-6)
		})
	}
}

func TestStrokeQuadraticExtremeDirections(t *testing.T) {
	var src vector.Path
	src.MoveTo(-3e38, 0)
	src.QuadTo(3e38, 0, 0, 100)
	var dst vector.Path
	dst.AddStroke(&src, &vector.AddStrokeOptions{
		StrokeOptions: vector.StrokeOptions{
			Width: 4,
		},
	})
	vertices, _ := dst.AppendVerticesAndIndicesForFilling(nil, nil)
	assertGeometryVertex(t, vertices, vector.Point{X: -3e38, Y: 2}, 1e-6)
	assertGeometryVertex(t, vertices, vector.Point{Y: 98}, 1e-6)
}

func TestStrokeMiterIntersectionPrecision(t *testing.T) {
	for _, tc := range []struct {
		name                    string
		start, joint, end, want vector.Point
		width                   float32
		tolerance               float64
	}{
		{
			name:      "large joint coordinate",
			joint:     vector.Point{X: 1e8},
			end:       vector.Point{Y: -1e8},
			want:      vector.Point{X: 1e8 + 8, Y: 2},
			width:     4,
			tolerance: 1e-6,
		},
		{
			name:      "large width",
			start:     vector.Point{X: -1e20, Y: -1e20},
			end:       vector.Point{X: 1e20, Y: -1e20},
			want:      vector.Point{Y: 5e19 * math.Sqrt2},
			width:     1e20,
			tolerance: 1e13,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var src vector.Path
			src.MoveTo(tc.start.X, tc.start.Y)
			src.LineTo(tc.joint.X, tc.joint.Y)
			src.LineTo(tc.end.X, tc.end.Y)
			vertices, _ := src.AppendVerticesAndIndicesForStroke(nil, nil, &vector.StrokeOptions{
				Width:      tc.width,
				LineJoin:   vector.LineJoinMiter,
				MiterLimit: 4,
			})
			assertGeometryVertex(t, vertices, tc.want, tc.tolerance)
		})
	}
}

func TestStrokeSquareCapExtremeDirection(t *testing.T) {
	var src vector.Path
	src.MoveTo(0, 0)
	src.LineTo(3e38, 3e38)
	vertices, _ := src.AppendVerticesAndIndicesForStroke(nil, nil, &vector.StrokeOptions{
		Width:   4,
		LineCap: vector.LineCapSquare,
	})
	assertGeometryVertex(t, vertices, vector.Point{X: -2 * math.Sqrt2}, 1e-6)
	assertGeometryVertex(t, vertices, vector.Point{Y: -2 * math.Sqrt2}, 1e-6)
}

func assertGeometryVertex(t *testing.T, vertices []ebiten.Vertex, want vector.Point, tolerance float64) {
	t.Helper()
	var found bool
	for _, v := range vertices {
		if math.IsNaN(float64(v.DstX)) || math.IsNaN(float64(v.DstY)) || math.IsInf(float64(v.DstX), 0) || math.IsInf(float64(v.DstY), 0) {
			t.Errorf("non-finite vertex: (%g, %g)", v.DstX, v.DstY)
		}
		if math.Abs(float64(v.DstX)-float64(want.X)) <= tolerance && math.Abs(float64(v.DstY)-float64(want.Y)) <= tolerance {
			found = true
		}
	}
	if !found {
		t.Errorf("no vertex at %v (tolerance %g)", want, tolerance)
	}
}

func assertGeometryPoint(t *testing.T, got, want vector.Point, tolerance float64) {
	t.Helper()
	if !(math.Abs(float64(got.X)-float64(want.X)) <= tolerance) || !(math.Abs(float64(got.Y)-float64(want.Y)) <= tolerance) {
		t.Errorf("position: got %v, want %v (tolerance %g)", got, want, tolerance)
	}
}
