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
	"bytes"
	"encoding/xml"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/opentypesvg"
)

func checkPaint(t *testing.T, p opentypesvg.Paint) {
	t.Helper()
	check := func(value float64) {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 1 {
			t.Errorf("invalid color/opacity: %g", value)
		}
	}
	check(p.Opacity)
	for _, v := range []float64{p.Color.R, p.Color.G, p.Color.B, p.Color.A} {
		check(v)
	}
	var previous float64
	for i := 0; i < p.Gradient.StopCount(); i++ {
		stop := p.Gradient.Stop(i)
		check(stop.Offset)
		if stop.Offset < previous {
			t.Error("decreasing stops")
		}
		previous = stop.Offset
		for _, v := range []float64{stop.Color.R, stop.Color.G, stop.Color.B, stop.Color.A} {
			check(v)
		}
	}
}

func FuzzStyles(f *testing.F) {
	for _, source := range []string{
		svgStart + `<style>.a{fill:blue;fill-opacity:.5}</style><g id="glyph1" class="a a"><rect/></g></svg>`,
		svgStart + `<linearGradient id="base"><stop offset="0" stop-color="red"/><stop offset="1" stop-color="var(--color0,blue)"/></linearGradient><radialGradient id="g" href="#base"/><rect fill="url(#g)"/></svg>`,
		svgStart + `<linearGradient id="g" href="#g"/><rect fill="url(#g) red"/></svg>`,
		svgStart + `<style>g > .a .b{fill:red!important}</style><g><g class="a"><rect class="b"/></g></g></svg>`,
	} {
		f.Add(source)
	}
	f.Fuzz(func(t *testing.T, source string) {
		if len(source) > 1<<16 {
			t.Skip()
		}
		d, err := opentypesvg.Parse([]byte(source))
		if err != nil {
			return
		}
		r, err := newStyleResolver(d, opentypesvg.ColorContext{
			Foreground: blue,
			Palette: []opentypesvg.RGBA{{
				G: 1,
				A: .5,
			}},
		})
		if err != nil {
			return
		}
		var visit func(*opentypesvg.Element, *opentypesvg.Style)
		visit = func(e *opentypesvg.Element, parent *opentypesvg.Style) {
			s, err := r.Compute(e, parent)
			if err != nil {
				return
			}
			for _, stroke := range []bool{false, true} {
				p, err := r.Paint(s, stroke, opentypesvg.PaintContext{
					Lengths: opentypesvg.LengthContext{
						Width:         100,
						Height:        50,
						PixelsPerInch: 96,
					},
					Bounds: opentypesvg.ViewBox{
						Width:  20,
						Height: 30,
					},
				})
				if err == nil {
					checkPaint(t, p)
				}
			}
			_, _ = r.Clip(s)
			_, _ = r.Stroke(s, opentypesvg.LengthContext{
				Width:         100,
				Height:        50,
				PixelsPerInch: 96,
			})
			for i := 0; i < e.ChildCount(); i++ {
				visit(e.Child(i), &s)
			}
		}
		visit(d.Root(), nil)
		if e, err := d.SelectGlyph(1); err == nil {
			visit(e, nil)
		}
	})
}

func FuzzStyleValues(f *testing.F) {
	for _, value := range []string{"rgb(50%,0,0)", "hsl(120,100%,50%)", "var(--color0,red)", "var(--absent,var(--color1,currentColor))", "url(#g) blue", "url('#missing') none", "rgba(255,0,0,.5)", "rgb(,,)", "1,2,3", "inherit"} {
		f.Add(".a{opacity:.5}", value)
	}
	f.Fuzz(func(t *testing.T, css, value string) {
		if len(css)+len(value) > 1<<15 {
			t.Skip()
		}
		escape := func(text string) string { var b bytes.Buffer; _ = xml.EscapeText(&b, []byte(text)); return b.String() }
		d, err := opentypesvg.Parse([]byte(svgStart + `<style>` + escape(css) + `</style><linearGradient id="g"><stop/><stop offset="1"/></linearGradient><rect class="a a" fill="` + escape(value) + `" stroke-dasharray="` + escape(value) + `"/></svg>`))
		if err != nil {
			return
		}
		r, err := newStyleResolver(d, opentypesvg.ColorContext{
			Foreground: blue,
			Palette: []opentypesvg.RGBA{{
				R: 1,
				A: .5,
			}},
		})
		if err != nil {
			return
		}
		s, err := r.Compute(d.Root().Child(2), nil)
		if err != nil {
			return
		}
		p, err := r.Paint(s, false, opentypesvg.PaintContext{
			Bounds: opentypesvg.ViewBox{
				Width:  1,
				Height: 1,
			},
		})
		if err == nil {
			checkPaint(t, p)
		}
		_, _ = r.Stroke(s, opentypesvg.LengthContext{
			Width:         100,
			Height:        100,
			PixelsPerInch: 96,
		})
	})
}
