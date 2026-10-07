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

package opentypesvg_test

import (
	"errors"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/opentypesvg"
)

func path(t *testing.T, text string) opentypesvg.Path {
	t.Helper()
	p, err := opentypesvg.ParsePath(text)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func near(t *testing.T, got, want float64) {
	t.Helper()
	if math.IsNaN(got) || math.Abs(got-want) > 1e-9 {
		t.Errorf("got %g, want %g", got, want)
	}
}

func mapped(t *testing.T, m opentypesvg.Matrix, p, q opentypesvg.Point) {
	t.Helper()
	got, err := m.Apply(p)
	if err != nil {
		t.Errorf("Apply(%+v): %v", p, err)
		return
	}
	near(t, got.X, q.X)
	near(t, got.Y, q.Y)
}

func TestPathCommands(t *testing.T) {
	absolute := path(t, "M10 20 15 25 L20 30 H40 V50 C41 51 42 52 43 53 S44 54 45 55 Q46 56 47 57 T49 59 A4 5 30 0 1 60 70 Z l1 2")
	relative := path(t, "m10 20 5 5 l5 5 h20 v20 c1 1 2 2 3 3 s1 1 2 2 q1 1 2 2 t2 2 a4 5 30 0 1 11 11 z l1 2")
	if absolute.SegmentCount() != relative.SegmentCount() {
		t.Fatal("different segment counts")
	}
	for i := range absolute.SegmentCount() {
		if absolute.Segment(i) != relative.Segment(i) {
			t.Errorf("segment %d: %+v != %+v", i, absolute.Segment(i), relative.Segment(i))
		}
	}
	if got := absolute.Segment(6).Control1; got != (opentypesvg.Point{X: 44, Y: 54}) {
		t.Errorf("smooth cubic control: %+v", got)
	}
	if got := absolute.Segment(8).Control1; got != (opentypesvg.Point{X: 48, Y: 58}) {
		t.Errorf("smooth quadratic control: %+v", got)
	}
	if got := absolute.Segment(11).To; got != (opentypesvg.Point{X: 11, Y: 22}) {
		t.Errorf("close current point: %+v", got)
	}
	for _, text := range []string{"M0 0L2 3S4 5 6 7", "M0 0Q1 2 2 3S4 5 6 7", "M0 0C1 1 2 2 2 3T6 7", "M2 3zT6 7", "M2 3A2 2 0 0 0 2 3T6 7"} {
		p := path(t, text)
		seg := p.Segment(p.SegmentCount() - 1)
		if seg.Control1 != (opentypesvg.Point{X: 2, Y: 3}) {
			t.Errorf("%s: reflection not reset: %+v", text, seg)
		}
	}
	p := path(t, "M.6.5L1e1-2e-1 3.+4.")
	if p.SegmentCount() != 3 {
		t.Errorf("adjacent numbers: %d", p.SegmentCount())
	}
	near(t, p.Segment(1).To.Y, -.2)
	for _, text := range []string{"M1 2H3 4 5", "M1 2V3 4 5", "M1 2Q3 4 5 6 7 8 9 10", "M1 2C3 4 5 6 7 8 9 10 11 12 13 14", "M1 2T3 4 5 6", "M1 2S3 4 5 6 7 8 9 10", "M1 2A1 2 0 0 1 3 4 1 2 0 1 0 5 6"} {
		path(t, text)
	}
}

func TestPathArcs(t *testing.T) {
	p := path(t, "M0 0A-2 -3 390 011 2 a0 3 45 1 1 1 2 a5 5 0 1 1 0 0 t2 3")
	if p.SegmentCount() != 4 {
		t.Fatalf("segments: %d", p.SegmentCount())
	}
	arc := p.Segment(1)
	if arc.Kind != opentypesvg.ArcTo || arc.Radius != (opentypesvg.Point{X: 2, Y: 3}) || arc.Rotation != 30 || arc.LargeArc || !arc.Sweep {
		t.Errorf("arc: %+v", arc)
	}
	if got, want := p.Segment(2), (opentypesvg.Segment{
		Kind: opentypesvg.LineTo,
		To:   opentypesvg.Point{X: 2, Y: 4},
	}); got != want {
		t.Errorf("zero-radius arc = %+v, want %+v", got, want)
	}
	if p.Segment(3).Control1 != (opentypesvg.Point{X: 2, Y: 4}) {
		t.Error("omitted arc must reset reflection")
	}
}

func TestArcRadiusCorrection(t *testing.T) {
	p := path(t, "M0 0A1 1 0 0 1 10 0")
	if got := p.Segment(1).Radius; got != (opentypesvg.Point{X: 5, Y: 5}) {
		t.Errorf("corrected radii: %+v", got)
	}
}

func TestGeometryErrors(t *testing.T) {
	for _, text := range []string{"L0 0", "M", "M,0 0", "M0,,0", "M0 0,", "M0 0, L1 2", "M0 0Z1 2", "M0 0X1 2", "M0 0L1", "M0 0A1 2 0 2 0 3 4", "M0 0A1 2 0.01 3 4", "M0 0A1 2 0 +1 0 3 4", "M0 0LNaN 0", "M0 0L1e999 0", "M1e308 0l1e308 0", "M0 0\fL1 2"} {
		if _, err := opentypesvg.ParsePath(text); !errors.Is(err, opentypesvg.ErrGeometry) {
			t.Errorf("%q: %v", text, err)
		}
	}
	p, err := opentypesvg.ParsePath("M0 0L1 2 3")
	if !errors.Is(err, opentypesvg.ErrGeometry) || p.SegmentCount() != 2 {
		t.Errorf("valid prefix: %d, %v", p.SegmentCount(), err)
	}
	for _, text := range []string{"NaN", "Inf", "0x10", "1e", "1e999", "1e-999", "1 2", "1,2", "1px", ". ", "1\u00a0"} {
		if _, err := opentypesvg.ParseNumber(text); !errors.Is(err, opentypesvg.ErrGeometry) {
			t.Errorf("number %q: %v", text, err)
		}
	}
	for _, text := range []string{"+1", "-.5", "1.", "1e-3", " 1E+2\n"} {
		if _, err := opentypesvg.ParseNumber(text); err != nil {
			t.Errorf("%q: %v", text, err)
		}
	}
}

func TestTransforms(t *testing.T) {
	for _, tc := range []struct {
		text        string
		point, want opentypesvg.Point
	}{
		{
			text:  "translate(10 20) scale(2 3)",
			point: opentypesvg.Point{X: 1, Y: 2},
			want:  opentypesvg.Point{X: 12, Y: 26},
		},
		{
			text:  "scale(2),, translate(10)",
			point: opentypesvg.Point{X: 1, Y: 2},
			want:  opentypesvg.Point{X: 22, Y: 4},
		},
		{
			text:  "rotate(90 10 20)",
			point: opentypesvg.Point{X: 11, Y: 20},
			want:  opentypesvg.Point{X: 10, Y: 21},
		},
		{
			text:  "matrix(1,2,3,4,5,6)",
			point: opentypesvg.Point{X: 1, Y: 2},
			want:  opentypesvg.Point{X: 12, Y: 16},
		},
		{
			text:  "skewX(45) skewY(45)",
			point: opentypesvg.Point{X: 1, Y: 2},
			want:  opentypesvg.Point{X: 4, Y: 3},
		},
		{
			text:  "rotate(90)",
			point: opentypesvg.Point{X: 1},
			want:  opentypesvg.Point{Y: 1},
		},
		{
			text:  "scale(0)",
			point: opentypesvg.Point{X: 1, Y: 2},
		},
	} {
		m, err := opentypesvg.ParseTransform(tc.text)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			continue
		}
		mapped(t, m, tc.point, tc.want)
	}
	for _, text := range []string{"translate(1,)", "scale()", "rotate(1 2)", "matrix(1 2 3)", "skewX(90)", "scale(1), ", "scale(1e308) scale(1e308)", "foo(1)", "scale(1 2 3)", "translate(1,,2)", "scale(1) junk"} {
		if _, err := opentypesvg.ParseTransform(text); !errors.Is(err, opentypesvg.ErrGeometry) {
			t.Errorf("%q: %v", text, err)
		}
	}
	d := parse(t, `<svg><g transform="scale(99)"><use id="glyph1" x="3" y="4" transform="scale(2)"/></g></svg>`)
	e, err := d.SelectGlyph(1)
	if err != nil {
		t.Fatal(err)
	}
	m, err := e.LocalTransform(opentypesvg.LengthContext{})
	if err != nil {
		t.Fatal(err)
	}
	mapped(t, m, opentypesvg.Point{}, opentypesvg.Point{X: 6, Y: 8})
}

func TestLengths(t *testing.T) {
	c := opentypesvg.LengthContext{
		Width:         300,
		Height:        400,
		FontSize:      20,
		XHeight:       9,
		PixelsPerInch: 96,
	}
	for _, tc := range []struct {
		text string
		want float64
	}{
		{
			text: "1",
			want: 1,
		},
		{
			text: "1px",
			want: 1,
		},
		{
			text: "2em",
			want: 40,
		},
		{
			text: "2ex",
			want: 18,
		},
		{
			text: "1in",
			want: 96,
		},
		{
			text: "2.54cm",
			want: 96,
		},
		{
			text: "25.4mm",
			want: 96,
		},
		{
			text: "72pt",
			want: 96,
		},
		{
			text: "6pc",
			want: 96,
		},
		{
			text: "-2.5e1%",
			want: -75,
		},
	} {
		v, err := opentypesvg.ResolveLength(tc.text, c, opentypesvg.Horizontal)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			continue
		}
		near(t, v, tc.want)
	}
	v, err := opentypesvg.ResolveLength("100%", c, opentypesvg.Diagonal)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err == nil {
		near(t, v, 500/math.Sqrt2)
	}
	v, err = opentypesvg.ResolveLength("50%", c, opentypesvg.Vertical)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err == nil {
		near(t, v, 200)
	}
	for _, text := range []string{"1 px", "2rem", "2EM", "1e999px", "1%garbage"} {
		if _, err := opentypesvg.ResolveLength(text, c, opentypesvg.Horizontal); !errors.Is(err, opentypesvg.ErrGeometry) {
			t.Errorf("%s: %v", text, err)
		}
	}
	if _, err := opentypesvg.ResolveLength("1in", opentypesvg.LengthContext{}, opentypesvg.Horizontal); !errors.Is(err, opentypesvg.ErrGeometry) {
		t.Error(err)
	}
}

func TestShapes(t *testing.T) {
	c := opentypesvg.LengthContext{
		Width:  100,
		Height: 200,
	}
	for _, tc := range []struct {
		source string
		count  int
	}{
		{
			source: `<rect width="10" height="20"/>`,
			count:  5,
		},
		{
			source: `<rect width="10" height="20" rx="100"/>`,
			count:  10,
		},
		{
			source: `<rect width="10" height="20" ry="2"/>`,
			count:  10,
		},
		{
			source: `<rect width="10" height="20" rx="0" ry="2"/>`,
			count:  5,
		},
		{
			source: `<rect width="10"/>`,
		},
		{
			source: `<circle/>`,
		},
		{
			source: `<ellipse rx="2"/>`,
		},
		{
			source: `<circle r="2"/>`,
			count:  6,
		},
		{
			source: `<ellipse rx="2" ry="3"/>`,
			count:  6,
		},
		{
			source: `<line/>`,
			count:  2,
		},
		{
			source: `<polyline points="1,2 3,4"/>`,
			count:  2,
		},
		{
			source: `<polygon points="1,2 3,4 5,6"/>`,
			count:  4,
		},
		{
			source: `<polygon/>`,
		},
		{
			source: `<polyline points="1-2 3-4"/>`,
			count:  2,
		},
	} {
		p, err := parse(t, "<svg>"+tc.source+"</svg>").Root().Child(0).Geometry(c)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			continue
		}
		if p.SegmentCount() != tc.count {
			t.Errorf("%s: %d", tc.source, p.SegmentCount())
		}
	}
	p, err := parse(t, `<svg><rect x="10%" y="-5%" width="20" height="30" rx="100"/></svg>`).Root().Child(0).Geometry(c)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err == nil {
		if p.Segment(0).To != (opentypesvg.Point{X: 20, Y: -10}) || p.Segment(2).Radius != (opentypesvg.Point{X: 10, Y: 15}) {
			t.Errorf("rounded rectangle: %+v %+v", p.Segment(0), p.Segment(2))
		}
	}
	for _, source := range []string{`<rect width="-1"/>`, `<circle r="-1"/>`, `<ellipse rx="-1"/>`, `<rect rx="-1"/>`, `<polyline points="1 2 3"/>`, `<polygon points="1 2, "/>`, `<line x1="NaN"/>`, `<rect x="1e308" width="1e308" height="1"/>`} {
		if _, err := parse(t, "<svg>"+source+"</svg>").Root().Child(0).Geometry(c); !errors.Is(err, opentypesvg.ErrGeometry) {
			t.Errorf("%s: %v", source, err)
		}
	}
}

func TestViewport(t *testing.T) {
	box := opentypesvg.ViewBox{
		X:      10,
		Y:      -20,
		Width:  100,
		Height: 50,
	}
	for _, tc := range []struct {
		aspect string
		want   opentypesvg.Point
	}{
		{
			aspect: "",
			want:   opentypesvg.Point{Y: 50},
		},
		{
			aspect: "none",
			want:   opentypesvg.Point{},
		},
		{
			aspect: "xMinYMin meet",
			want:   opentypesvg.Point{},
		},
		{
			aspect: "xMaxYMax meet",
			want:   opentypesvg.Point{Y: 100},
		},
		{
			aspect: "xMidYMid slice",
			want:   opentypesvg.Point{X: -100},
		},
		{
			aspect: "defer xMaxYMin slice",
			want:   opentypesvg.Point{X: -200},
		},
	} {
		m, disabled, err := opentypesvg.ViewportMapping(box, 200, 200, tc.aspect)
		if err != nil || disabled {
			t.Errorf("%s: %v %v", tc.aspect, disabled, err)
			continue
		}
		mapped(t, m, opentypesvg.Point{X: 10, Y: -20}, tc.want)
	}
	for xi, x := range []string{"Min", "Mid", "Max"} {
		for yi, y := range []string{"Min", "Mid", "Max"} {
			for _, mode := range []string{"meet", "slice"} {
				m, _, err := opentypesvg.ViewportMapping(box, 200, 200, "x"+x+"Y"+y+" "+mode)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					continue
				}
				want := opentypesvg.Point{Y: float64(yi) * 50}
				if mode == "slice" {
					want = opentypesvg.Point{X: -float64(xi) * 100}
				}
				mapped(t, m, opentypesvg.Point{X: 10, Y: -20}, want)
			}
		}
	}
	for _, text := range []string{"0 0 -1 1", "0 0 1", "0 0 1 1,", "0 0 1e999 2", "0,0,,1,2"} {
		if _, err := opentypesvg.ParseViewBox(text); !errors.Is(err, opentypesvg.ErrGeometry) {
			t.Errorf("%s: %v", text, err)
		}
	}
	for _, aspect := range []string{"defer", "xmidYmid", "none bogus", "xMinYMin meet extra", "" + "\u00a0"} {
		if _, _, err := opentypesvg.ViewportMapping(box, 200, 200, aspect); !errors.Is(err, opentypesvg.ErrGeometry) {
			t.Errorf("%q: %v", aspect, err)
		}
	}
	d := parse(t, `<svg viewBox="0 1000 1000 1000" overflow="hidden"/>`)
	v, err := d.Viewport(1000, opentypesvg.LengthContext{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err == nil {
		mapped(t, v.Transform, opentypesvg.Point{X: 100, Y: 570}, opentypesvg.Point{X: 100, Y: -430})
	}

	d = parse(t, `<svg width="500" height="250"/>`)
	v, err = d.Viewport(1000, opentypesvg.LengthContext{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err == nil {
		mapped(t, v.Transform, opentypesvg.Point{X: 100, Y: -50}, opentypesvg.Point{X: 100, Y: -50})
		near(t, v.Lengths.Width, 500)
		near(t, v.Lengths.Height, 250)
	}

	d = parse(t, `<svg x="19" y="23" width="500"/>`)
	v, err = d.Viewport(1000, opentypesvg.LengthContext{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err == nil {
		if v.Transform != opentypesvg.Identity() {
			t.Errorf("no viewBox: %+v, want identity", v.Transform)
		}
		near(t, v.Lengths.Width, 500)
		near(t, v.Lengths.Height, 1000)
	}

	for _, source := range []string{`<svg width="0"/>`, `<svg viewBox="0 0 0 1"/>`} {
		v, err := parse(t, source).Viewport(1000, opentypesvg.LengthContext{})
		if err != nil || !v.Disabled {
			t.Errorf("disabled viewport: %+v %v", v, err)
		}
	}
}

func TestGeometryFixtures(t *testing.T) {
	files, err := filepath.Glob("testdata/*/*.svg")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			d := parse(t, string(read(t, file)))
			c := opentypesvg.LengthContext{
				Width:         2048,
				Height:        2048,
				FontSize:      2048,
				XHeight:       1024,
				PixelsPerInch: 96,
			}
			var visit func(*opentypesvg.Element)
			visit = func(e *opentypesvg.Element) {
				if _, err := e.Geometry(c); err != nil {
					t.Errorf("%s geometry: %v", e.Name().Local, err)
				}
				if _, err := e.LocalTransform(c); err != nil {
					t.Errorf("%s transform: %v", e.Name().Local, err)
				}
				for i := range e.ChildCount() {
					visit(e.Child(i))
				}
			}
			visit(d.Root())
			if _, err := d.Viewport(2048, c); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestGeometryLimits(t *testing.T) {
	for _, text := range []string{strings.Repeat(" ", (32<<20)+1), "M0 0" + strings.Repeat("L0 0", 1<<18)} {
		if _, err := opentypesvg.ParsePath(text); !errors.Is(err, opentypesvg.ErrLimit) {
			t.Errorf("limit: %v", err)
		}
	}
	if _, err := opentypesvg.ParseTransform(strings.Repeat("scale(1) ", (1<<18)+1)); !errors.Is(err, opentypesvg.ErrLimit) {
		t.Error(err)
	}
	d := parse(t, `<svg><polyline points="`+strings.Repeat("0 0 ", 1<<18)+`"/></svg>`)
	if _, err := d.Root().Child(0).Geometry(opentypesvg.LengthContext{}); !errors.Is(err, opentypesvg.ErrLimit) {
		t.Error(err)
	}
}

func FuzzPath(f *testing.F) {
	for _, text := range []string{"M0 0", "M.6.5C1 2 3 4 5 6s1 2 3 4z", "M0 0A1 2 30 0110 20", "M1e308 0l1e308 0"} {
		f.Add(text)
	}
	f.Fuzz(func(t *testing.T, text string) {
		p, _ := opentypesvg.ParsePath(text)
		for i := range p.SegmentCount() {
			s := p.Segment(i)
			for _, v := range []float64{s.To.X, s.To.Y, s.Control1.X, s.Control1.Y, s.Control2.X, s.Control2.Y, s.Radius.X, s.Radius.Y, s.Rotation} {
				if math.IsNaN(v) || math.IsInf(v, 0) {
					t.Error("nonfinite segment")
				}
			}
		}
	})
}

func FuzzCoordinates(f *testing.F) {
	for _, text := range []string{"translate(10,20)rotate(45)", "translate(1-2)", "scale(.5.5)", "1 2-3 4", "1.5.5 2 3", "translate(1 2) scale(3)", "1e2em", "0 0 20 30", "xMidYMid slice", "1,2 3,4"} {
		f.Add(text)
	}
	f.Fuzz(func(t *testing.T, text string) {
		m, err := opentypesvg.ParseTransform(text)
		if err == nil {
			if _, err := m.Apply(opentypesvg.Point{}); err != nil {
				t.Error(err)
			}
		}
		_, _ = opentypesvg.ParseNumber(text)
		_, _ = opentypesvg.ResolveLength(text, opentypesvg.LengthContext{
			Width:         200,
			Height:        100,
			FontSize:      20,
			XHeight:       10,
			PixelsPerInch: 96,
		}, opentypesvg.Diagonal)
		_, _ = opentypesvg.ParseViewBox(text)
		_, _, _ = opentypesvg.ViewportMapping(opentypesvg.ViewBox{
			Width:  10,
			Height: 20,
		}, 100, 200, text)
		escaped := strings.NewReplacer("&", "&amp;", "\"", "&quot;", "<", "&lt;").Replace(text)
		d, err := opentypesvg.Parse([]byte(`<svg><polygon points="` + escaped + `"/></svg>`))
		if err == nil {
			_, _ = d.Root().Child(0).Geometry(opentypesvg.LengthContext{})
		}
	})
}

func TestFixtureCoordinates(t *testing.T) {
	d := parse(t, string(read(t, "testdata/focused/stroke-transform.svg")))
	entry, err := d.SelectGlyph(1)
	if err != nil {
		t.Fatal(err)
	}
	m, err := entry.LocalTransform(opentypesvg.LengthContext{})
	if err != nil {
		t.Fatal(err)
	}
	p, err := entry.Child(0).Geometry(opentypesvg.LengthContext{})
	if err != nil {
		t.Fatal(err)
	}
	mapped(t, m, p.Segment(0).To, opentypesvg.Point{X: 12, Y: 16})
	mapped(t, m, p.Segment(1).To, opentypesvg.Point{X: 36, Y: 16})
	p, err = entry.Child(1).Geometry(opentypesvg.LengthContext{})
	if err != nil {
		t.Fatal(err)
	}
	if p.Segment(1).Kind != opentypesvg.QuadTo || p.Segment(2).Control1 != (opentypesvg.Point{X: 20, Y: 20}) {
		t.Error("quadratic geometry lost")
	}
	d = parse(t, string(read(t, "testdata/focused/shared-context.svg")))
	for _, gid := range []uint16{1, 2} {
		entry, err := d.SelectGlyph(gid)
		if err != nil {
			t.Fatal(err)
		}
		m, err := entry.LocalTransform(opentypesvg.LengthContext{})
		if err != nil {
			t.Fatal(err)
		}
		var x float64
		if gid == 2 {
			x = 16
		}
		mapped(t, m, opentypesvg.Point{}, opentypesvg.Point{X: x})
	}
	d = parse(t, string(read(t, "testdata/focused/viewport-overflow.svg")))
	v, err := d.Viewport(48, opentypesvg.LengthContext{})
	if err != nil {
		t.Fatal(err)
	}
	entry, err = d.SelectGlyph(1)
	if err != nil {
		t.Fatal(err)
	}
	p, err = entry.Child(0).Geometry(v.Lengths)
	if err != nil {
		t.Fatal(err)
	}
	mapped(t, v.Transform, p.Segment(0).To, opentypesvg.Point{X: -4, Y: -8})
	mapped(t, v.Transform, p.Segment(2).To, opentypesvg.Point{X: 12, Y: 8})
}

func TestViewportDimensions(t *testing.T) {
	for _, tc := range []struct {
		source         string
		origin, extent opentypesvg.Point
		width, height  float64
	}{
		{
			source: `<svg/>`,
			extent: opentypesvg.Point{X: 100, Y: 100},
			width:  100,
			height: 100,
		},
		{
			source: `<svg width="50%" height="25%" preserveAspectRatio="bad"/>`,
			extent: opentypesvg.Point{X: 50, Y: 25},
			width:  50,
			height: 25,
		},
		{
			source: `<svg width="30" height="40" viewBox="0 0 100 50"/>`,
			origin: opentypesvg.Point{Y: 12.5},
			extent: opentypesvg.Point{X: 30, Y: 27.5},
			width:  100,
			height: 50,
		},
		{
			source: `<svg viewBox="0 0 100 50" preserveAspectRatio="none"/>`,
			extent: opentypesvg.Point{X: 100, Y: 100},
			width:  100,
			height: 50,
		},
	} {
		v, err := parse(t, tc.source).Viewport(100, opentypesvg.LengthContext{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			continue
		}
		mapped(t, v.Transform, opentypesvg.Point{}, tc.origin)
		mapped(t, v.Transform, opentypesvg.Point{X: tc.width, Y: tc.height}, tc.extent)
		near(t, v.Lengths.Width, tc.width)
		near(t, v.Lengths.Height, tc.height)
	}
	for _, source := range []string{`<svg width="-1"/>`, `<svg height="1e999"/>`, `<svg viewBox="0 0 -1 2"/>`, `<svg viewBox="0 0 1e-320 1" preserveAspectRatio="none"/>`, `<svg viewBox="0 0 1 1" preserveAspectRatio="bad"/>`} {
		if _, err := parse(t, source).Viewport(100, opentypesvg.LengthContext{}); !errors.Is(err, opentypesvg.ErrGeometry) {
			t.Errorf("%s: %v", source, err)
		}
	}
	for _, upem := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, err := parse(t, `<svg/>`).Viewport(upem, opentypesvg.LengthContext{}); !errors.Is(err, opentypesvg.ErrGeometry) {
			t.Errorf("upem %g: %v", upem, err)
		}
	}
}

func TestFiniteArithmetic(t *testing.T) {
	for _, m := range []opentypesvg.Matrix{
		{
			A: math.Inf(1),
			D: 1,
		},
		{
			A: math.NaN(),
			D: 1,
		},
		{
			A: math.MaxFloat64,
			D: 1,
		},
	} {
		if _, err := m.Apply(opentypesvg.Point{X: 2}); !errors.Is(err, opentypesvg.ErrGeometry) {
			t.Errorf("matrix: %v", err)
		}
	}
	for _, c := range []opentypesvg.LengthContext{
		{
			Width:  -1,
			Height: 2,
		},
		{
			Width:  math.Inf(1),
			Height: 1,
		},
		{
			Width:  math.NaN(),
			Height: 1,
		},
	} {
		if _, err := opentypesvg.ResolveLength("50%", c, opentypesvg.Diagonal); !errors.Is(err, opentypesvg.ErrGeometry) {
			t.Errorf("context: %v", err)
		}
	}
	if _, err := opentypesvg.ParsePath("M0 0 A1e-320 1e-320 0 0 1 1e300 0"); !errors.Is(err, opentypesvg.ErrGeometry) {
		t.Error(err)
	}
}

func TestCompactTransforms(t *testing.T) {
	for _, tc := range []struct {
		source string
		want   opentypesvg.Matrix
	}{
		{
			source: "translate(10,20)rotate(45)",
			want: opentypesvg.Matrix{
				A: math.Sqrt(.5),
				B: math.Sqrt(.5),
				C: -math.Sqrt(.5),
				D: math.Sqrt(.5),
				E: 10,
				F: 20,
			},
		},
		{
			source: "translate(0)scale(1)",
			want: opentypesvg.Matrix{
				A: 1,
				D: 1,
			},
		},
		{
			source: "translate(1-2)",
			want: opentypesvg.Matrix{
				A: 1,
				D: 1,
				E: 1,
				F: -2,
			},
		},
		{
			source: "matrix(1 0 0-1 0 0)",
			want: opentypesvg.Matrix{
				A: 1,
				D: -1,
			},
		},
		{
			source: "scale(.5.5)",
			want: opentypesvg.Matrix{
				A: .5,
				D: .5,
			},
		},
	} {
		got, err := opentypesvg.ParseTransform(tc.source)
		if err != nil {
			t.Errorf("%s: %v", tc.source, err)
			continue
		}
		near(t, got.A, tc.want.A)
		near(t, got.B, tc.want.B)
		near(t, got.C, tc.want.C)
		near(t, got.D, tc.want.D)
		near(t, got.E, tc.want.E)
		near(t, got.F, tc.want.F)
	}
}

func TestCompactPoints(t *testing.T) {
	for _, tc := range []struct {
		source string
		want   []opentypesvg.Point
		err    error
	}{
		{
			source: "1 2-3 4",
			want:   []opentypesvg.Point{{X: 1, Y: 2}, {X: -3, Y: 4}},
		},
		{
			source: "1,2-3,4",
			want:   []opentypesvg.Point{{X: 1, Y: 2}, {X: -3, Y: 4}},
		},
		{
			source: "1.5.5 2 3",
			want:   []opentypesvg.Point{{X: 1.5, Y: .5}, {X: 2, Y: 3}},
		},
		{
			source: "1 2-3",
			want:   []opentypesvg.Point{{X: 1, Y: 2}},
			err:    opentypesvg.ErrGeometry,
		},
		{
			source: "1 2-3 4, ",
			want:   []opentypesvg.Point{{X: 1, Y: 2}, {X: -3, Y: 4}},
			err:    opentypesvg.ErrGeometry,
		},
		{
			source: "1,,2",
			err:    opentypesvg.ErrGeometry,
		},
	} {
		p, err := parse(t, `<svg><polyline points="`+tc.source+`"/></svg>`).Root().Child(0).Geometry(opentypesvg.LengthContext{})
		if !errors.Is(err, tc.err) {
			t.Errorf("%q: %v, want %v", tc.source, err, tc.err)
		}
		if p.SegmentCount() != len(tc.want) {
			t.Errorf("%q: %d segments, want %d", tc.source, p.SegmentCount(), len(tc.want))
			continue
		}
		for i, point := range tc.want {
			kind := opentypesvg.LineTo
			if i == 0 {
				kind = opentypesvg.MoveTo
			}
			want := opentypesvg.Segment{
				Kind: kind,
				To:   point,
			}
			if got := p.Segment(i); got != want {
				t.Errorf("%q segment %d: %+v, want %+v", tc.source, i, got, want)
			}
		}
	}
}

func TestShapeSegments(t *testing.T) {
	for _, tc := range []struct {
		source string
		want   []opentypesvg.Segment
	}{
		{
			source: `<circle cx="3" cy="4" r="2"/>`,
			want: []opentypesvg.Segment{
				{
					Kind: opentypesvg.MoveTo,
					To:   opentypesvg.Point{X: 5, Y: 4},
				},
				{
					Kind:   opentypesvg.ArcTo,
					To:     opentypesvg.Point{X: 3, Y: 6},
					Radius: opentypesvg.Point{X: 2, Y: 2},
					Sweep:  true,
				},
				{
					Kind:   opentypesvg.ArcTo,
					To:     opentypesvg.Point{X: 1, Y: 4},
					Radius: opentypesvg.Point{X: 2, Y: 2},
					Sweep:  true,
				},
				{
					Kind:   opentypesvg.ArcTo,
					To:     opentypesvg.Point{X: 3, Y: 2},
					Radius: opentypesvg.Point{X: 2, Y: 2},
					Sweep:  true,
				},
				{
					Kind:   opentypesvg.ArcTo,
					To:     opentypesvg.Point{X: 5, Y: 4},
					Radius: opentypesvg.Point{X: 2, Y: 2},
					Sweep:  true,
				},
				{
					Kind: opentypesvg.ClosePath,
					To:   opentypesvg.Point{X: 5, Y: 4},
				},
			},
		},
		{
			source: `<ellipse cx="3" cy="4" rx="2" ry="3"/>`,
			want: []opentypesvg.Segment{
				{
					Kind: opentypesvg.MoveTo,
					To:   opentypesvg.Point{X: 5, Y: 4},
				},
				{
					Kind:   opentypesvg.ArcTo,
					To:     opentypesvg.Point{X: 3, Y: 7},
					Radius: opentypesvg.Point{X: 2, Y: 3},
					Sweep:  true,
				},
				{
					Kind:   opentypesvg.ArcTo,
					To:     opentypesvg.Point{X: 1, Y: 4},
					Radius: opentypesvg.Point{X: 2, Y: 3},
					Sweep:  true,
				},
				{
					Kind:   opentypesvg.ArcTo,
					To:     opentypesvg.Point{X: 3, Y: 1},
					Radius: opentypesvg.Point{X: 2, Y: 3},
					Sweep:  true,
				},
				{
					Kind:   opentypesvg.ArcTo,
					To:     opentypesvg.Point{X: 5, Y: 4},
					Radius: opentypesvg.Point{X: 2, Y: 3},
					Sweep:  true,
				},
				{
					Kind: opentypesvg.ClosePath,
					To:   opentypesvg.Point{X: 5, Y: 4},
				},
			},
		},
		{
			source: `<rect x="1" y="2" width="10" height="20" rx="2" ry="3"/>`,
			want: []opentypesvg.Segment{
				{
					Kind: opentypesvg.MoveTo,
					To:   opentypesvg.Point{X: 3, Y: 2},
				},
				{
					Kind: opentypesvg.LineTo,
					To:   opentypesvg.Point{X: 9, Y: 2},
				},
				{
					Kind:   opentypesvg.ArcTo,
					To:     opentypesvg.Point{X: 11, Y: 5},
					Radius: opentypesvg.Point{X: 2, Y: 3},
					Sweep:  true,
				},
				{
					Kind: opentypesvg.LineTo,
					To:   opentypesvg.Point{X: 11, Y: 19},
				},
				{
					Kind:   opentypesvg.ArcTo,
					To:     opentypesvg.Point{X: 9, Y: 22},
					Radius: opentypesvg.Point{X: 2, Y: 3},
					Sweep:  true,
				},
				{
					Kind: opentypesvg.LineTo,
					To:   opentypesvg.Point{X: 3, Y: 22},
				},
				{
					Kind:   opentypesvg.ArcTo,
					To:     opentypesvg.Point{X: 1, Y: 19},
					Radius: opentypesvg.Point{X: 2, Y: 3},
					Sweep:  true,
				},
				{
					Kind: opentypesvg.LineTo,
					To:   opentypesvg.Point{X: 1, Y: 5},
				},
				{
					Kind:   opentypesvg.ArcTo,
					To:     opentypesvg.Point{X: 3, Y: 2},
					Radius: opentypesvg.Point{X: 2, Y: 3},
					Sweep:  true,
				},
				{
					Kind: opentypesvg.ClosePath,
					To:   opentypesvg.Point{X: 3, Y: 2},
				},
			},
		},
		{
			source: `<rect x="1" y="2" width="10" height="20"/>`,
			want: []opentypesvg.Segment{
				{
					Kind: opentypesvg.MoveTo,
					To:   opentypesvg.Point{X: 1, Y: 2},
				},
				{
					Kind: opentypesvg.LineTo,
					To:   opentypesvg.Point{X: 11, Y: 2},
				},
				{
					Kind: opentypesvg.LineTo,
					To:   opentypesvg.Point{X: 11, Y: 22},
				},
				{
					Kind: opentypesvg.LineTo,
					To:   opentypesvg.Point{X: 1, Y: 22},
				},
				{
					Kind: opentypesvg.ClosePath,
					To:   opentypesvg.Point{X: 1, Y: 2},
				},
			},
		},
	} {
		p, err := parse(t, "<svg>"+tc.source+"</svg>").Root().Child(0).Geometry(opentypesvg.LengthContext{})
		if err != nil {
			t.Errorf("%s: %v", tc.source, err)
			continue
		}
		if p.SegmentCount() != len(tc.want) {
			t.Errorf("%s: %d segments, want %d", tc.source, p.SegmentCount(), len(tc.want))
			continue
		}
		for i, want := range tc.want {
			if got := p.Segment(i); got != want {
				t.Errorf("%s segment %d: %+v, want %+v", tc.source, i, got, want)
			}
		}
	}
}
