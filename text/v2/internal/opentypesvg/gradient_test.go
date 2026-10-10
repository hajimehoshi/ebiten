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
	"fmt"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/opentypesvg"
)

func gradientPaint(t *testing.T, defs string) opentypesvg.Paint {
	t.Helper()
	d, r := styles(t, svgStart+`<defs>`+defs+`</defs><rect fill="url(#g)" color="blue"/></svg>`, opentypesvg.ColorContext{
		Foreground: black,
	})
	return paintOf(t, r, computed(t, r, d.Root().Child(1), nil), false)
}

func TestGradientReferencesDefaultsAndOverrides(t *testing.T) {
	base := `<linearGradient id="base" x1="10%" gradientTransform="translate(.1 .2)" spreadMethod="reflect" color="red"><stop offset="0" stop-color="currentColor"/><stop offset="1" stop-color="blue"/></linearGradient>`
	derived := `<linearGradient id="g" href="#base" x2="50%"/>`
	for _, defs := range []string{base + derived, derived + base} {
		p := gradientPaint(t, defs)
		g := p.Gradient
		if p.Kind != opentypesvg.PaintGradient || g.StopCount() != 2 {
			t.Fatalf("gradient paint=%+v", p)
		}
		near(t, g.Start.X, .1)
		near(t, g.End.X, .5)
		near(t, g.End.Y, 0)
		if g.Spread != "reflect" || g.Radial || g.Solid || g.Disabled {
			t.Errorf("gradient=%+v", g)
		}
		if g.Stop(0).Color != red || g.Stop(1).Color != blue {
			t.Errorf("inherited stops=%+v,%+v", g.Stop(0), g.Stop(1))
		}
		mapped(t, g.Transform, opentypesvg.Point{}, opentypesvg.Point{X: 4, Y: 4})
		mapped(t, g.Transform, opentypesvg.Point{X: 1, Y: 1}, opentypesvg.Point{X: 44, Y: 24})
	}
	p := gradientPaint(t, base+`<linearGradient id="g" href="#base" spreadMethod="repeat" gradientTransform="scale(2)" color="blue"><stop offset=".25" stop-color="currentColor"/></linearGradient>`)
	if p.Gradient.StopCount() != 1 || p.Gradient.Stop(0).Color != blue || !p.Gradient.Solid || p.Gradient.Spread != "repeat" {
		t.Errorf("overrides=%+v", p.Gradient)
	}
	mapped(t, p.Gradient.Transform, opentypesvg.Point{X: 1, Y: 1}, opentypesvg.Point{X: 80, Y: 40})
}

func TestGradientStopSemantics(t *testing.T) {
	d, r := styles(t, svgStart+`<defs color="red" stop-color="blue" stop-opacity=".1">
 <style>.faded{stop-color:var(--color0,blue);stop-opacity:.5}</style>
 <linearGradient id="base" color-interpolation="linearRGB"><stop offset="-1"/><stop offset=".7" class="faded"/><stop offset=".3" stop-color="currentColor"/><stop offset="200%" stop-color="transparent"/></linearGradient>
 <linearGradient id="g" href="#base" color="blue" color-interpolation="sRGB"/>
 </defs><rect fill="url(#g)" fill-opacity=".4" opacity=".2"/></svg>`, opentypesvg.ColorContext{
		Foreground: black,
		Palette: []opentypesvg.RGBA{{
			G: 1,
			A: .5,
		}},
	})
	s := computed(t, r, d.Root().Child(1), nil)
	p := paintOf(t, r, s, false)
	g := p.Gradient
	if g.StopCount() != 4 {
		t.Fatalf("stops=%d", g.StopCount())
	}
	near(t, g.Stop(0).Offset, 0)
	near(t, g.Stop(1).Offset, .7)
	near(t, g.Stop(2).Offset, .7)
	near(t, g.Stop(3).Offset, 1)
	if g.Stop(0).Color != black {
		t.Errorf("non-inherited stop-color=%+v", g.Stop(0))
	}
	near(t, g.Stop(0).Color.A, 1)
	near(t, g.Stop(1).Color.A, .25)
	if g.Stop(2).Color != red {
		t.Errorf("source currentColor=%+v", g.Stop(2))
	}
	near(t, g.Stop(3).Color.A, 0)
	near(t, p.Opacity, .4)
	near(t, s.Opacity, .2)
	if g.Interpolation != "sRGB" {
		t.Errorf("gradient's own interpolation=%s", g.Interpolation)
	}
}

func TestGradientRadialUnitsAndFocalPoint(t *testing.T) {
	p := gradientPaint(t, `<radialGradient id="base" cx="25%"/><radialGradient id="g" href="#base" cx="75%"><stop stop-color="red"/><stop offset="1" stop-color="blue"/></radialGradient>`)
	g := p.Gradient
	near(t, g.Center.X, .75)
	near(t, g.Center.Y, .5)
	near(t, g.Radius, .5)
	if g.Focus != g.Center {
		t.Errorf("default focus=%v, center=%v", g.Focus, g.Center)
	}
	p = gradientPaint(t, `<radialGradient id="base" fx="10%" fy="50%"/><radialGradient id="g" href="#base" cx="50%" r="25%"><stop/><stop offset="1"/></radialGradient>`)
	near(t, p.Gradient.Focus.X, .25)
	near(t, p.Gradient.Focus.Y, .5)
	d, r := styles(t, svgStart+`<radialGradient id="g" gradientUnits="userSpaceOnUse" cx="50%" cy="50%" r="50%" fx="500" fy="25" gradientTransform="translate(2 3) scale(2)"><stop/><stop offset="1"/></radialGradient><rect fill="url(#g)"/></svg>`, opentypesvg.ColorContext{
		Foreground: black,
	})
	s := computed(t, r, d.Root().Child(1), nil)
	resolved, err := r.Paint(s, false, opentypesvg.PaintContext{
		Lengths: opentypesvg.LengthContext{
			Width:         100,
			Height:        50,
			PixelsPerInch: 96,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	g = resolved.Gradient
	near(t, g.Center.X, 50)
	near(t, g.Center.Y, 25)
	near(t, g.Radius, 39.528470752104745)
	near(t, g.Focus.X, 50+g.Radius)
	near(t, g.Focus.Y, 25)
	mapped(t, g.Transform, opentypesvg.Point{X: 1, Y: 1}, opentypesvg.Point{X: 4, Y: 5})
}

func TestGradientContextSnapshots(t *testing.T) {
	d, r := styles(t, svgStart+`<linearGradient id="g"><stop/><stop offset="1"/></linearGradient><rect fill="url(#g)"/></svg>`, opentypesvg.ColorContext{
		Foreground: black,
	})
	s := computed(t, r, d.Root().Child(1), nil)
	a, err := r.Paint(s, false, opentypesvg.PaintContext{
		Bounds: opentypesvg.ViewBox{
			X:      3,
			Y:      5,
			Width:  10,
			Height: 20,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := r.Paint(s, false, opentypesvg.PaintContext{
		Bounds: opentypesvg.ViewBox{
			Width:  40,
			Height: 60,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	mapped(t, a.Gradient.Transform, opentypesvg.Point{X: 1, Y: 1}, opentypesvg.Point{X: 13, Y: 25})
	mapped(t, b.Gradient.Transform, opentypesvg.Point{X: 1, Y: 1}, opentypesvg.Point{X: 40, Y: 60})
	zero, err := r.Paint(s, false, opentypesvg.PaintContext{
		Bounds: opentypesvg.ViewBox{
			Width:  0,
			Height: 60,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !zero.Gradient.Disabled {
		t.Error("zero bounding box enabled")
	}
	for _, defs := range []string{`<linearGradient id="g"/>`, `<linearGradient id="g"><stop/></linearGradient>`, `<linearGradient id="g" x2="0"><stop/><stop offset="1"/></linearGradient>`, `<radialGradient id="g" r="0"><stop/><stop offset="1"/></radialGradient>`} {
		g := gradientPaint(t, defs).Gradient
		if !g.Disabled && !g.Solid {
			t.Errorf("degenerate gradient=%+v", g)
		}
	}
}

func TestGradientFailuresAndFallbacks(t *testing.T) {
	for _, tc := range []struct {
		defs, paint string
		want        error
		fallback    bool
	}{
		{
			paint: "url(#missing)",
			want:  opentypesvg.ErrReference,
		},
		{
			paint:    "url(#missing) currentColor",
			fallback: true,
		},
		{
			defs:  `<linearGradient id="g" href="#g"/>`,
			paint: "url(#g)",
			want:  opentypesvg.ErrReferenceCycle,
		},
		{
			defs:  `<linearGradient id="g" href="#b"/><radialGradient id="b" href="#g"/>`,
			paint: "url(#g)",
			want:  opentypesvg.ErrReferenceCycle,
		},
		{
			defs:     `<linearGradient id="g" href="#g"/>`,
			paint:    "url(#g) currentColor",
			fallback: true,
		},
		{
			defs:  `<radialGradient id="g" r="-1"/>`,
			paint: "url(#g)",
			want:  opentypesvg.ErrStyle,
		},
		{
			defs:  `<linearGradient id="g" gradientUnits="screen"/>`,
			paint: "url(#g)",
			want:  opentypesvg.ErrStyle,
		},
		{
			defs:  `<linearGradient id="g" spreadMethod="bad"/>`,
			paint: "url(#g)",
			want:  opentypesvg.ErrStyle,
		},
		{
			defs:  `<linearGradient id="g" href="#missing"/>`,
			paint: "url(#g)",
			want:  opentypesvg.ErrReference,
		},
		{
			defs:  `<pattern id="g"/>`,
			paint: "url(#g) currentColor",
			want:  opentypesvg.ErrUnsupported,
		},
		{
			defs:  `<radialGradient id="g" fr="1"/>`,
			paint: "url(#g)",
			want:  opentypesvg.ErrUnsupported,
		},
		{
			defs:  `<path id="g"/>`,
			paint: "url(#g)",
			want:  opentypesvg.ErrReference,
		},
	} {
		t.Run(tc.defs+tc.paint, func(t *testing.T) {
			d, r := styles(t, svgStart+`<defs>`+tc.defs+`</defs><rect fill="`+tc.paint+`"/></svg>`, opentypesvg.ColorContext{
				Foreground: blue,
			})
			s := computed(t, r, d.Root().Child(1), nil)
			p, err := r.Paint(s, false, opentypesvg.PaintContext{})
			if tc.fallback {
				if err != nil || p.Color != blue || p.Kind != opentypesvg.PaintSolid {
					t.Errorf("fallback=%+v, %v", p, err)
				}
			} else if !errors.Is(err, tc.want) {
				t.Errorf("error=%v, want %v", err, tc.want)
			}
		})
	}
}

func TestGradientReferenceDepthAndSharedStops(t *testing.T) {
	var defs strings.Builder
	defs.WriteString(`<linearGradient id="g0"><stop stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient>`)
	for i := 1; i <= 256; i++ {
		fmt.Fprintf(&defs, `<linearGradient id="g%d" href="#g%d"/>`, i, i-1)
	}
	d, r := styles(t, svgStart+defs.String()+`<rect fill="url(#g255)"/><rect fill="url(#g256) red"/></svg>`, opentypesvg.ColorContext{
		Foreground: black,
	})
	p := paintOf(t, r, computed(t, r, d.Root().Child(257), nil), false)
	if p.Gradient.StopCount() != 2 || p.Gradient.Stop(0).Color != red {
		t.Errorf("shared stops=%+v", p)
	}
	s := computed(t, r, d.Root().Child(258), nil)
	if _, err := r.Paint(s, false, opentypesvg.PaintContext{}); !errors.Is(err, opentypesvg.ErrLimit) {
		t.Errorf("cached depth limit bypassed by fallback: %v", err)
	}
}

func TestGradientFocusedFixture(t *testing.T) {
	d, r := styles(t, string(read(t, "testdata/focused/gradient-reference.svg")), opentypesvg.ColorContext{
		Foreground: black,
	})
	glyph, err := d.SelectGlyph(1)
	if err != nil {
		t.Fatal(err)
	}
	parent := computed(t, r, glyph, nil)
	a := paintOf(t, r, computed(t, r, glyph.Child(0), &parent), false).Gradient
	b := paintOf(t, r, computed(t, r, glyph.Child(1), &parent), false).Gradient
	if a.Radial || a.Stop(0).Color != red || a.Stop(1).Color != blue {
		t.Errorf("linear=%+v", a)
	}
	if !b.Radial {
		t.Error("forward reference is not radial")
	}
	near(t, b.Center.X, 24)
	near(t, b.Center.Y, 32)
	near(t, b.Radius, 20)
	near(t, b.Stop(1).Color.A, .5)
}

func TestGradientZeroFocalRadius(t *testing.T) {
	for _, value := range []string{"0", "0%"} {
		g := gradientPaint(t, `<radialGradient id="g" fr="`+value+`"><stop stop-color="red"/></radialGradient>`).Gradient
		if !g.Radial || !g.Solid {
			t.Errorf("zero focal radius %s: %+v", value, g)
		}
	}
	for _, attr := range []string{`r="1em"`, `cx="1ex"`} {
		d, r := styles(t, svgStart+`<radialGradient id="g" `+attr+`/><rect fill="url(#g)"/></svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		s := computed(t, r, d.Root().Child(1), nil)
		if _, err := r.Paint(s, false, opentypesvg.PaintContext{}); !errors.Is(err, opentypesvg.ErrUnsupported) {
			t.Errorf("OpenType excludes relative font units %s: %v", attr, err)
		}
	}
}

func TestGradientEffectiveFocalRadius(t *testing.T) {
	for _, value := range []string{"0", "0%", "0px", "0cm", "0em", "0ex"} {
		g := gradientPaint(t, `<radialGradient id="b" fr="5"><stop stop-color="red"/></radialGradient><radialGradient id="g" href="#b" fr="`+value+`"/>`).Gradient
		if !g.Radial || !g.Solid {
			t.Errorf("overridden zero radius %s: %+v", value, g)
		}
	}
	d, r := styles(t, svgStart+`<radialGradient id="b" fr="5"><stop/></radialGradient><radialGradient id="g" href="#b"/><rect fill="url(#g)"/></svg>`, opentypesvg.ColorContext{
		Foreground: black,
	})
	s := computed(t, r, d.Root().Child(2), nil)
	if _, err := r.Paint(s, false, opentypesvg.PaintContext{}); !errors.Is(err, opentypesvg.ErrUnsupported) {
		t.Errorf("inherited nonzero radius: %v", err)
	}
}
