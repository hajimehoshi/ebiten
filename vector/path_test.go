// Copyright 2024 The Ebitengine Authors
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
	"image"
	"image/color"
	"math"
	"runtime"
	"sync"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func TestIsPointCloseToSegment(t *testing.T) {
	testCases := []struct {
		p     vector.Point
		p0    vector.Point
		p1    vector.Point
		allow float32
		want  bool
	}{
		{
			p:     vector.Point{0.5, 0.5},
			p0:    vector.Point{0, 0},
			p1:    vector.Point{1, 0},
			allow: 1,
			want:  true,
		},
		{
			p:     vector.Point{0.5, 1.5},
			p0:    vector.Point{0, 0},
			p1:    vector.Point{1, 0},
			allow: 1,
			want:  false,
		},
		{
			p:     vector.Point{0.5, 0.5},
			p0:    vector.Point{0, 0},
			p1:    vector.Point{1, 1},
			allow: 0,
			want:  true,
		},
		{
			p:     vector.Point{0, 1},
			p0:    vector.Point{0, 0},
			p1:    vector.Point{1, 1},
			allow: 0.7,
			want:  false,
		},
		{
			p:     vector.Point{0, 1},
			p0:    vector.Point{0, 0},
			p1:    vector.Point{1, 1},
			allow: 0.8,
			want:  true,
		},
		{
			// p0 and p1 are the same.
			p:     vector.Point{0, 1},
			p0:    vector.Point{0.5, 0.5},
			p1:    vector.Point{0.5, 0.5},
			allow: 0.7,
			want:  false,
		},
		{
			// p0 and p1 are the same.
			p:     vector.Point{0, 1},
			p0:    vector.Point{0.5, 0.5},
			p1:    vector.Point{0.5, 0.5},
			allow: 0.8,
			want:  true,
		},
	}
	for _, tc := range testCases {
		if got := vector.IsPointCloseToSegment(tc.p, tc.p0, tc.p1, tc.allow); got != tc.want {
			t.Errorf("got: %v, want: %v", got, tc.want)
		}
	}
}

func TestMoveToAndClose(t *testing.T) {
	var path vector.Path
	if _, ok := vector.CurrentPosition(&path); ok != false {
		t.Errorf("expected no last position, got one")
	}
	if got, want := vector.SubPathCount(&path), 0; got != want {
		t.Errorf("expected close count to be %d, got %d", want, got)
	}

	path.MoveTo(10, 20)
	if p, ok := vector.CurrentPosition(&path); p != (vector.Point{10, 20}) || !ok {
		t.Errorf("expected last position to be (10, 20), got %v", p)
	}
	if got, want := vector.SubPathCount(&path), 1; got != want {
		t.Errorf("expected close count to be %d, got %d", want, got)
	}

	path.MoveTo(30, 40)
	if p, ok := vector.CurrentPosition(&path); p != (vector.Point{30, 40}) || !ok {
		t.Errorf("expected last position to be (30, 40), got %v", p)
	}
	if got, want := vector.SubPathCount(&path), 1; got != want {
		t.Errorf("expected close count to be %d, got %d", want, got)
	}

	path.LineTo(50, 60)
	if p, ok := vector.CurrentPosition(&path); p != (vector.Point{50, 60}) || !ok {
		t.Errorf("expected last position to be (50, 60), got %v", p)
	}
	if got, want := vector.SubPathCount(&path), 1; got != want {
		t.Errorf("expected close count to be %d, got %d", want, got)
	}

	path.Close()
	if p, ok := vector.CurrentPosition(&path); p != (vector.Point{30, 40}) || !ok {
		t.Errorf("expected last position to be (30, 40) after close, got %v", p)
	}
	if got, want := vector.SubPathCount(&path), 1; got != want {
		t.Errorf("expected close count to be %d, got %d", want, got)
	}

	path.MoveTo(70, 80)
	if p, ok := vector.CurrentPosition(&path); p != (vector.Point{70, 80}) || !ok {
		t.Errorf("expected last position to be (70, 80), got %v", p)
	}
	if got, want := vector.SubPathCount(&path), 2; got != want {
		t.Errorf("expected close count to be %d, got %d", want, got)
	}

	path.LineTo(90, 100)
	if p, ok := vector.CurrentPosition(&path); p != (vector.Point{90, 100}) || !ok {
		t.Errorf("expected last position to be (50, 60), got %v", p)
	}
	if got, want := vector.SubPathCount(&path), 2; got != want {
		t.Errorf("expected close count to be %d, got %d", want, got)
	}

	// MoveTo without closing forces to create a new sub-path.
	// The previous sub-path is left unclosed.
	path.MoveTo(110, 120)
	if p, ok := vector.CurrentPosition(&path); p != (vector.Point{110, 120}) || !ok {
		t.Errorf("expected last position to be (70, 80), got %v", p)
	}
	if got, want := vector.SubPathCount(&path), 3; got != want {
		t.Errorf("expected close count to be %d, got %d", want, got)
	}
}

func TestAddPath(t *testing.T) {
	var path vector.Path
	path.MoveTo(10, 20)
	path.LineTo(30, 40)
	path.Close()

	op := &vector.AddPathOptions{}
	op.GeoM.Translate(100, 100)
	var path2 vector.Path
	path2.AddPath(&path, op)

	if p, ok := vector.CurrentPosition(&path); p != (vector.Point{10, 20}) || !ok {
		t.Errorf("expected last position to be (10, 20), got %v", p)
	}
	if got, want := vector.SubPathCount(&path), 1; got != want {
		t.Errorf("expected close count to be %d, got %d", want, got)
	}
	if p, ok := vector.CurrentPosition(&path2); p != (vector.Point{110, 120}) || !ok {
		t.Errorf("expected last position to be (110, 120), got %v", p)
	}
	if got, want := vector.SubPathCount(&path2), 1; got != want {
		t.Errorf("expected close count to be %d, got %d", want, got)
	}
}

func TestAddPathSelf(t *testing.T) {
	var path vector.Path
	path.MoveTo(10, 20)
	path.LineTo(30, 40)
	path.Close()

	op := &vector.AddPathOptions{}
	op.GeoM.Translate(100, 100)
	path.AddPath(&path, op)

	if p, ok := vector.CurrentPosition(&path); p != (vector.Point{110, 120}) || !ok {
		t.Errorf("expected last position to be (110, 120), got %v", p)
	}
	if got, want := vector.SubPathCount(&path), 2; got != want {
		t.Errorf("expected close count to be %d, got %d", want, got)
	}
}

func TestArcAndGeoM(t *testing.T) {
	if runtime.GOOS == "js" {
		t.Skip("the result might be flaky in this environment")
	}

	testCases := []struct {
		name     string
		geoM     ebiten.GeoM
		origPath vector.Path
		refPath  vector.Path
	}{
		{
			name: "identity",
			geoM: ebiten.GeoM{},
			origPath: func() (p vector.Path) {
				p.MoveTo(0, 0)
				p.ArcTo(16, 0, 16, 16, 16)
				return p
			}(),
			refPath: func() (p vector.Path) {
				p.MoveTo(0, 0)
				p.ArcTo(16, 0, 16, 16, 16)
				return p
			}(),
		},
		{
			name: "scale 2x",
			geoM: func() (geoM ebiten.GeoM) {
				geoM.Scale(2, 2)
				return geoM
			}(),
			origPath: func() (p vector.Path) {
				p.MoveTo(0, 0)
				p.ArcTo(8, 0, 8, 8, 8)
				return p
			}(),
			refPath: func() (p vector.Path) {
				p.MoveTo(0, 0)
				p.ArcTo(16, 0, 16, 16, 16)
				return p
			}(),
		},
		{
			name: "scale 256x",
			geoM: func() (geoM ebiten.GeoM) {
				geoM.Scale(256, 256)
				return geoM
			}(),
			origPath: func() (p vector.Path) {
				p.MoveTo(0, 0)
				p.ArcTo(1.0/16.0, 0, 1.0/16.0, 1.0/16.0, 1.0/16.0)
				return p
			}(),
			refPath: func() (p vector.Path) {
				p.MoveTo(0, 0)
				p.ArcTo(16, 0, 16, 16, 16)
				return p
			}(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			origDst := ebiten.NewImage(16, 16)
			defer origDst.Deallocate()
			refDst := ebiten.NewImage(16, 16)
			defer refDst.Deallocate()

			var path vector.Path
			op := &vector.AddPathOptions{}
			op.GeoM = tc.geoM
			path.AddPath(&tc.origPath, op)

			strokeOp := &vector.StrokeOptions{
				Width: 1,
			}
			// Do not use alpha blending, which can cause non-deterministic results.
			vector.StrokePath(origDst, &path, strokeOp, nil)
			vector.StrokePath(refDst, &tc.refPath, strokeOp, nil)

			for j := range 16 {
				for i := range 16 {
					got := origDst.At(i, j)
					want := refDst.At(i, j)
					if got != want {
						t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
					}
				}
			}
		})
	}
}

// Issue #3666
func TestArcHugeAngle(t *testing.T) {
	// For an angle this big, the float32 spacing is bigger than the angle by which an arc is split.
	// Splitting such an arc must still terminate.
	testCases := []struct {
		name       string
		startAngle float32
		sweep      float32
		dir        vector.Direction
	}{
		{
			name:       "clockwise",
			startAngle: 2e7,
			sweep:      2,
			dir:        vector.Clockwise,
		},
		{
			name:       "clockwise, small sweep",
			startAngle: 1 << 24,
			sweep:      1.6,
			dir:        vector.Clockwise,
		},
		{
			name:       "clockwise, full circle",
			startAngle: 1 << 24,
			sweep:      2 * math.Pi,
			dir:        vector.Clockwise,
		},
		{
			name:       "clockwise, negative angle",
			startAngle: -(1 << 25),
			sweep:      4,
			dir:        vector.Clockwise,
		},
		{
			name:       "clockwise, wider spacing",
			startAngle: 1 << 26,
			sweep:      8,
			dir:        vector.Clockwise,
		},
		{
			name:       "counterclockwise",
			startAngle: 2e7,
			sweep:      -2,
			dir:        vector.CounterClockwise,
		},
		{
			name:       "counterclockwise, negative angle",
			startAngle: -(1 << 25),
			sweep:      -4,
			dir:        vector.CounterClockwise,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			const (
				cx     = 100
				cy     = 100
				radius = 5
			)
			var p vector.Path
			p.Arc(cx, cy, radius, tc.startAngle, tc.startAngle+tc.sweep, tc.dir)

			// The angles are degenerate, so the shape is not specified, but it must be finite.
			// The control points of the approximated Bézier curves can reach out this far.
			const allow = 32 * radius
			bounds := p.Bounds()
			if want := image.Rect(cx-allow, cy-allow, cx+allow, cy+allow); !bounds.In(want) {
				t.Errorf("bounds: got: %v, want: a rectangle in %v", bounds, want)
			}
		})
	}
}

// Issue #3330
func TestFillPathSubImage(t *testing.T) {
	dst := ebiten.NewImage(16, 16)

	dst2 := dst.SubImage(image.Rect(0, 0, 8, 8)).(*ebiten.Image)
	var p vector.Path
	p.MoveTo(0, 0)
	p.LineTo(8, 0)
	p.LineTo(8, 8)
	p.LineTo(0, 8)
	p.Close()
	op := &vector.DrawPathOptions{}
	op.ColorScale.ScaleWithColor(color.White)
	op.AntiAlias = true
	vector.FillPath(dst2, &p, nil, op)
	if got, want := dst.At(5, 5), (color.RGBA{0xff, 0xff, 0xff, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := dst2.At(5, 5), (color.RGBA{0xff, 0xff, 0xff, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	dst3 := dst2.SubImage(image.Rect(4, 4, 8, 8)).(*ebiten.Image)
	var p2 vector.Path
	p2.MoveTo(4, 4)
	p2.LineTo(8, 4)
	p2.LineTo(8, 8)
	p2.LineTo(4, 8)
	p2.Close()
	op.ColorScale.Reset()
	op.ColorScale.ScaleWithColor(color.Black)
	vector.FillPath(dst3, &p2, nil, op)
	if got, want := dst.At(5, 5), (color.RGBA{0x00, 0x00, 0x00, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := dst2.At(5, 5), (color.RGBA{0x00, 0x00, 0x00, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := dst3.At(5, 5), (color.RGBA{0x00, 0x00, 0x00, 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestRaceConditionWithSubImage(t *testing.T) {
	const w, h = 16, 16
	src := ebiten.NewImage(w, h)

	var wg sync.WaitGroup
	for i := range h {
		for j := range w {
			wg.Go(func() {
				subImg := src.SubImage(image.Rect(i, j, i+1, j+1)).(*ebiten.Image)
				var p vector.Path
				p.MoveTo(0, 0)
				p.LineTo(w, 0)
				p.LineTo(w, h)
				p.LineTo(0, h)
				p.Close()
				op := &vector.DrawPathOptions{}
				op.ColorScale.ScaleWithColor(color.White)
				op.AntiAlias = true
				vector.FillPath(subImg, &p, nil, op)
				dst := ebiten.NewImage(w, h)
				dst.DrawImage(subImg, nil)
			})
		}
	}
	wg.Wait()
}

// Issue #3355
func TestFillPathSubImageAndImage(t *testing.T) {
	dst := ebiten.NewImage(200, 200)
	defer dst.Deallocate()
	for i := range 100 {
		var path vector.Path
		path.LineTo(0, 0)
		path.LineTo(0, 100)
		path.LineTo(100, 100)
		path.LineTo(100, 0)
		path.LineTo(0, 0)
		path.Close()
		drawOp := &vector.DrawPathOptions{}
		drawOp.ColorScale.ScaleWithColor(color.RGBA{255, 0, 0, 255})
		subDst := dst.SubImage(image.Rect(0, 0, 100, 100)).(*ebiten.Image)
		vector.FillPath(subDst, &path, nil, drawOp)
		drawOp.ColorScale.Reset()
		drawOp.ColorScale.ScaleWithColor(color.RGBA{0, 255, 0, 255})
		vector.FillPath(dst, &path, nil, drawOp)

		if got, want := dst.At(50, 50), (color.RGBA{0, 255, 0, 255}); got != want {
			t.Errorf("%d: got: %v, want: %v", i, got, want)
		}
	}
}

// Issue #3366
func TestFillPathFillRule(t *testing.T) {
	testCases := []struct {
		name     string
		fillRule vector.FillRule
		expected color.RGBA
	}{
		{
			name:     "evenOdd",
			fillRule: vector.FillRuleEvenOdd,
			expected: color.RGBA{0, 0, 0, 0},
		},
		{
			name:     "nonZero",
			fillRule: vector.FillRuleNonZero,
			expected: color.RGBA{0xff, 0xff, 0xff, 0xff},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dst := ebiten.NewImage(16, 16)
			defer dst.Deallocate()

			var p vector.Path
			p.MoveTo(0, 0)
			p.LineTo(16, 0)
			p.LineTo(16, 16)
			p.LineTo(0, 16)
			p.Close()
			p.MoveTo(4, 4)
			p.LineTo(12, 4)
			p.LineTo(12, 12)
			p.LineTo(4, 12)
			p.Close()
			fillOp := &vector.FillOptions{}
			fillOp.FillRule = tc.fillRule
			vector.FillPath(dst, &p, fillOp, nil)

			if got, want := dst.At(8, 8), tc.expected; got != want {
				t.Errorf("got: %v, want: %v", got, want)
			}
		})
	}
}

func TestBounds(t *testing.T) {
	nan := float32(math.NaN())
	inf := float32(math.Inf(1))

	testCases := []struct {
		name string
		path func(p *vector.Path)
		want image.Rectangle
	}{
		{
			name: "empty",
			path: func(p *vector.Path) {},
			want: image.Rectangle{},
		},
		{
			name: "moveTo only",
			path: func(p *vector.Path) {
				p.MoveTo(100, 50)
			},
			want: image.Rectangle{},
		},
		{
			name: "horizontal line",
			path: func(p *vector.Path) {
				p.MoveTo(100, 50)
				p.LineTo(200, 50)
			},
			want: image.Rect(100, 50, 200, 50),
		},
		{
			name: "vertical line",
			path: func(p *vector.Path) {
				p.MoveTo(50, 10)
				p.LineTo(50, 100)
			},
			want: image.Rect(50, 10, 50, 100),
		},
		{
			name: "diagonal line",
			path: func(p *vector.Path) {
				p.MoveTo(10.5, 20.5)
				p.LineTo(30.5, 40.5)
			},
			want: image.Rect(10, 20, 31, 41),
		},
		{
			name: "two horizontal lines",
			path: func(p *vector.Path) {
				p.MoveTo(100, 50)
				p.LineTo(200, 50)
				p.MoveTo(120, 80)
				p.LineTo(220, 80)
			},
			want: image.Rect(100, 50, 220, 80),
		},
		{
			name: "rectangle",
			path: func(p *vector.Path) {
				p.MoveTo(10, 20)
				p.LineTo(30, 20)
				p.LineTo(30, 40)
				p.LineTo(10, 40)
				p.Close()
			},
			want: image.Rect(10, 20, 30, 40),
		},
		{
			name: "nearly collinear arc",
			path: func(p *vector.Path) {
				p.MoveTo(676.47327, 1502.2303)
				p.ArcTo(257.7812, 1856.779, 1046.7478, 1188.3281, 100)
			},
			want: image.Rect(676, 1188, 1047, 1503),
		},
		{
			name: "infinite line",
			path: func(p *vector.Path) {
				p.MoveTo(0, 0)
				p.LineTo(inf, inf)
			},
			want: image.Rectangle{},
		},
		{
			name: "non-finite control point",
			path: func(p *vector.Path) {
				p.MoveTo(0, 0)
				p.QuadTo(nan, nan, 10, 10)
			},
			want: image.Rectangle{},
		},
		{
			name: "non-finite sub-path with a finite sub-path",
			path: func(p *vector.Path) {
				p.MoveTo(0, 0)
				p.LineTo(nan, nan)
				p.MoveTo(100, 100)
				p.LineTo(110, 110)
			},
			want: image.Rect(100, 100, 110, 110),
		},
		{
			name: "moveTo replacing a non-finite position",
			path: func(p *vector.Path) {
				p.MoveTo(nan, nan)
				p.MoveTo(0, 0)
				p.LineTo(10, 10)
			},
			want: image.Rect(0, 0, 10, 10),
		},
		{
			name: "addPath with a non-finite path",
			path: func(p *vector.Path) {
				var src vector.Path
				src.MoveTo(0, 0)
				src.LineTo(nan, nan)
				p.AddPath(&src, nil)
			},
			want: image.Rectangle{},
		},
		{
			name: "addPath with a non-finite geoM",
			path: func(p *vector.Path) {
				var src vector.Path
				src.MoveTo(0, 0)
				src.LineTo(10, 10)
				op := &vector.AddPathOptions{}
				op.GeoM.SetElement(0, 0, math.NaN())
				p.AddPath(&src, op)
			},
			want: image.Rectangle{},
		},
		{
			name: "addStroke with a non-finite geoM",
			path: func(p *vector.Path) {
				var src vector.Path
				src.MoveTo(0, 0)
				src.LineTo(10, 0)
				op := &vector.AddStrokeOptions{}
				op.StrokeOptions.Width = 2
				op.GeoM.SetElement(0, 0, math.NaN())
				p.AddStroke(&src, op)
			},
			want: image.Rectangle{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var p vector.Path
			tc.path(&p)
			if got, want := p.Bounds(), tc.want; got != want {
				t.Errorf("got: %v, want: %v", got, want)
			}
		})
	}
}

func TestBoundsConcurrency(t *testing.T) {
	var p vector.Path
	p.MoveTo(10, 20)
	p.LineTo(30, 40)
	p.QuadTo(50, 60, 70, 80)
	p.Close()

	// Bounds must not modify the path so that one path can be shared by multiple goroutines.
	const goroutineCount = 8
	got := make([]image.Rectangle, goroutineCount)
	var wg sync.WaitGroup
	for i := range goroutineCount {
		wg.Go(func() {
			got[i] = p.Bounds()
		})
	}
	wg.Wait()

	for i, b := range got {
		if want := image.Rect(10, 20, 70, 80); b != want {
			t.Errorf("%d: got: %v, want: %v", i, b, want)
		}
	}
}

func TestStrokeMiterJoinNearlyCollinear(t *testing.T) {
	var p vector.Path
	p.MoveTo(676.47327, 1502.2303)
	p.LineTo(257.7812, 1856.779)
	p.LineTo(1046.7478, 1188.3281)

	op := &vector.AddStrokeOptions{}
	op.StrokeOptions.LineJoin = vector.LineJoinMiter
	op.StrokeOptions.MiterLimit = 4
	op.StrokeOptions.Width = 1

	var sp vector.Path
	sp.AddStroke(&p, op)

	got := sp.Bounds()
	if got, want := got, image.Rect(257, 1187, 1048, 1858); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestQuadCuspIsKept(t *testing.T) {
	var p vector.Path
	p.MoveTo(0, 0)
	p.QuadTo(10, 10, 0, 0)

	// The cusp reaches (5, 5) at t=0.5.
	if got, want := p.Bounds(), image.Rect(0, 0, 5, 5); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}

	var d vector.Path
	d.MoveTo(5, 5)
	d.QuadTo(5, 5, 5, 5)
	if b := d.Bounds(); !b.Empty() {
		t.Errorf("Bounds of a single point: got %v, want empty", b)
	}
}

func TestStrokeQuadCusp(t *testing.T) {
	var p vector.Path
	p.MoveTo(0, 0)
	p.LineTo(100, 0)
	p.QuadTo(100, 50, 100, 0)
	p.LineTo(200, 0)

	// The cusp reaches (100, 25) at its midpoint, so it is equivalent to this out-and-back path.
	var l vector.Path
	l.MoveTo(0, 0)
	l.LineTo(100, 0)
	l.LineTo(100, 25)
	l.LineTo(100, 0)
	l.LineTo(200, 0)

	for _, lineJoin := range []vector.LineJoin{vector.LineJoinMiter, vector.LineJoinBevel, vector.LineJoinRound} {
		op := &vector.AddStrokeOptions{}
		op.StrokeOptions.Width = 10
		op.StrokeOptions.LineJoin = lineJoin

		var sp vector.Path
		sp.AddStroke(&p, op)
		var lp vector.Path
		lp.AddStroke(&l, op)

		// The stroked cusp must agree with the stroked out-and-back path exactly, including the joint at the tip.
		if got, want := vector.PathOperationsString(&sp), vector.PathOperationsString(&lp); got != want {
			t.Errorf("LineJoin %d: got:\n%v\nwant:\n%v", lineJoin, got, want)
		}
	}

	// The bounds of the cusp must be kept.
	if got, want := p.Bounds(), image.Rect(0, 0, 200, 25); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestStrokeTinyQuadCusp(t *testing.T) {
	// The midpoint of this cusp is rounded back to the start point in float32, so there is nothing to stroke.
	var p vector.Path
	p.MoveTo(1e7, 0)
	p.QuadTo(1e7+1, 0, 1e7, 0)

	op := &vector.AddStrokeOptions{}
	op.StrokeOptions.Width = 4

	var sp vector.Path
	sp.AddStroke(&p, op)
	if got, want := vector.SubPathCount(&sp), 0; got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestStrokeHugeQuadCusp(t *testing.T) {
	// The midpoint of this cusp must not overflow to infinity in float32.
	var p vector.Path
	p.MoveTo(3.0e38, 1)
	p.QuadTo(3.4e38, 1, 3.0e38, 1)

	op := &vector.AddStrokeOptions{}
	op.StrokeOptions.Width = 4

	var sp vector.Path
	sp.AddStroke(&p, op)

	// AddStroke must not modify the source path.
	if got, want := vector.PathOperationsString(&p), "MoveTo(3e+38, 1)\nQuadTo(3.4e+38, 1, 3e+38, 1)\n"; got != want {
		t.Errorf("got:\n%v\nwant:\n%v", got, want)
	}

	// The cusp is equivalent to this out-and-back path, whose midpoint (3.2e38, 1) doesn't overflow to infinity.
	var l vector.Path
	l.MoveTo(3.0e38, 1)
	l.LineTo(3.2e38, 1)
	l.LineTo(3.0e38, 1)

	var lp vector.Path
	lp.AddStroke(&l, op)

	// The stroked cusp must agree with the stroked out-and-back path exactly.
	if got, want := vector.PathOperationsString(&sp), vector.PathOperationsString(&lp); got != want {
		t.Errorf("got:\n%v\nwant:\n%v", got, want)
	}
}

func TestAddStrokeAllocs(t *testing.T) {
	testCases := []struct {
		name  string
		build func(p *vector.Path)
	}{
		{
			name: "no cusp",
			build: func(p *vector.Path) {
				p.MoveTo(0, 0)
				p.LineTo(100, 0)
				p.QuadTo(100, 50, 50, 50)
				p.LineTo(0, 0)
				p.Close()
			},
		},
		{
			name: "one cusp",
			build: func(p *vector.Path) {
				p.MoveTo(0, 0)
				p.LineTo(100, 0)
				p.QuadTo(100, 50, 100, 0)
				p.LineTo(200, 0)
			},
		},
		{
			name: "cusps in multiple sub-paths",
			build: func(p *vector.Path) {
				for i := range 3 {
					x := float32(i) * 100
					p.MoveTo(x, 0)
					p.QuadTo(x+50, 50, x, 0)
					p.LineTo(x+30, 30)
					p.QuadTo(x+60, 60, x+30, 30)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			op := &vector.AddStrokeOptions{}
			op.StrokeOptions.Width = 4

			var src vector.Path
			var dst vector.Path
			// Stroking must not allocate for a path that is reset and rebuilt repeatedly.
			if got := testing.AllocsPerRun(10, func() {
				src.Reset()
				tc.build(&src)
				dst.Reset()
				dst.AddStroke(&src, op)
			}); got != 0 {
				t.Errorf("allocations: got: %v, want: 0", got)
			}
		})
	}
}
