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
	"slices"
	"strings"
	"testing"

	"golang.org/x/image/colornames"

	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/opentypesvg"
)

var black = opentypesvg.RGBA{A: 1}
var red = opentypesvg.RGBA{R: 1, A: 1}
var blue = opentypesvg.RGBA{B: 1, A: 1}

func styles(t *testing.T, source string, colors opentypesvg.ColorContext) (*opentypesvg.Document, *opentypesvg.StyleResolver) {
	t.Helper()
	d := parse(t, source)
	r, err := newStyleResolver(d, colors)
	if err != nil {
		t.Fatal(err)
	}
	return d, r
}

func computed(t *testing.T, r *opentypesvg.StyleResolver, e *opentypesvg.Element, parent *opentypesvg.Style) opentypesvg.Style {
	t.Helper()
	s, err := r.Compute(e, parent)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func paintOf(t *testing.T, r *opentypesvg.StyleResolver, s opentypesvg.Style, stroke bool) opentypesvg.Paint {
	t.Helper()
	p, err := r.Paint(s, stroke, opentypesvg.PaintContext{
		Lengths: opentypesvg.LengthContext{
			Width:         48,
			Height:        48,
			PixelsPerInch: 96,
		},
		Bounds: opentypesvg.ViewBox{
			Width:  40,
			Height: 20,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestStyleCascade(t *testing.T) {
	for _, tc := range []struct {
		name, css, attributes string
		want                  opentypesvg.RGBA
	}{
		{
			name:       "source order",
			css:        ".a{fill:red}.b{fill:blue}",
			attributes: `class="a b"`,
			want:       blue,
		},
		{
			name:       "token order",
			css:        ".a{fill:red}.b{fill:blue}",
			attributes: `class="b a"`,
			want:       blue,
		},
		{
			name:       "repeated class",
			css:        ".a{fill:red}.b{fill:blue}",
			attributes: `class="b a b"`,
			want:       blue,
		},
		{
			name:       "specificity",
			css:        "#x{fill:blue}.a.a{fill:red}",
			attributes: `class="a"`,
			want:       blue,
		},
		{
			name:       "presentation",
			css:        "*{fill:blue}",
			attributes: `fill="red"`,
			want:       blue,
		},
		{
			name:       "inline",
			css:        "#x{fill:red}",
			attributes: `style="fill:blue"`,
			want:       blue,
		},
		{
			name:       "important",
			css:        "rect{fill:blue!important}",
			attributes: `style="fill:red"`,
			want:       blue,
		},
		{
			name:       "inline important",
			css:        "#x{fill:red!important}",
			attributes: `style="fill:blue!important"`,
			want:       blue,
		},
		{
			name: "declaration order",
			css:  "rect{fill:red;fill:blue}",
			want: blue,
		},
		{
			name: "invalid later",
			css:  "rect{fill:blue;fill:nonsense}",
			want: blue,
		},
		{
			name: "important order",
			css:  "rect{fill:blue!important;fill:red}",
			want: blue,
		},
		{
			name:       "selector list",
			css:        "#x,.a{fill:blue}.a{fill:red}",
			attributes: `class="a"`,
			want:       blue,
		},
		{
			name: "comments",
			css:  "/*start*/ rect /*gap*/ {fill:/*color*/blue}",
			want: blue,
		},
		{
			name:       "compound",
			css:        "rect.a.b#x{fill:blue}.a{fill:red}",
			attributes: `class="a b"`,
			want:       blue,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, r := styles(t, svgStart+`<style>`+tc.css+`</style><rect id="x" `+tc.attributes+`/></svg>`, opentypesvg.ColorContext{
				Foreground: black,
			})
			e, err := d.Resolve("#x")
			if err != nil {
				t.Fatal(err)
			}
			p := paintOf(t, r, computed(t, r, e, nil), false)
			if p.Kind != opentypesvg.PaintSolid || p.Color != tc.want {
				t.Errorf("paint=%+v, want %v", p, tc.want)
			}
		})
	}
}

func TestStyleUseInheritanceAndSelectors(t *testing.T) {
	d, r := styles(t, svgStart+`<style>.source > g .tile{stroke:blue} use .tile{fill:red}</style>
 <g class="source" fill="red" color="red" opacity=".25" transform="scale(7)">
  <g><g><g id="glyph1" class="tile" fill="currentColor"/></g></g>
 </g><use id="instance" color="blue" opacity=".5"/></svg>`, opentypesvg.ColorContext{
		Foreground: black,
	})
	glyph, err := d.SelectGlyph(1)
	if err != nil {
		t.Fatal(err)
	}
	instance, err := d.Resolve("#instance")
	if err != nil {
		t.Fatal(err)
	}
	parent := computed(t, r, instance, nil)
	s := computed(t, r, glyph, &parent)
	if got := paintOf(t, r, s, false).Color; got != blue {
		t.Errorf("instance fill=%v", got)
	}
	if got := paintOf(t, r, s, true).Color; got != blue {
		t.Errorf("source selector stroke=%v", got)
	}
	near(t, s.Opacity, 1)
	near(t, parent.Opacity, .5)
	transform, err := glyph.LocalTransform(opentypesvg.LengthContext{})
	if err != nil {
		t.Fatal(err)
	}
	if transform != opentypesvg.Identity() {
		t.Errorf("source transform inherited: %+v", transform)
	}
	if got := paintOf(t, r, computed(t, r, glyph, nil), false).Color; got != black {
		t.Errorf("selected fill=%v", got)
	}
}

func TestStyleInheritanceDefaultsAndOpacity(t *testing.T) {
	d, r := styles(t, svgStart+`<g fill="currentColor" color="red" fill-opacity=".4" stroke-opacity=".3" opacity=".5" visibility="hidden" display="none" stop-color="red" stop-opacity=".2">
 <rect color="blue"/><rect opacity="inherit" display="inherit" fill="initial" fill-opacity="unset" visibility="visible"/>
 <rect style="fill:inherit; color:inherit; stroke:initial"/>
 </g></svg>`, opentypesvg.ColorContext{
		Foreground: black,
	})
	parent := computed(t, r, d.Root().Child(0), nil)
	a := computed(t, r, d.Root().Child(0).Child(0), &parent)
	near(t, a.Opacity, 1)
	near(t, a.FillOpacity, .4)
	near(t, a.StrokeOpacity, .3)
	near(t, a.StopOpacity, 1)
	if !a.Display || a.Visible {
		t.Errorf("display/visibility=%+v", a)
	}
	if got := paintOf(t, r, a, false).Color; got != blue {
		t.Errorf("inherited currentColor=%v", got)
	}
	b := computed(t, r, d.Root().Child(0).Child(1), &parent)
	near(t, b.Opacity, .5)
	near(t, b.FillOpacity, .4)
	if b.Display || !b.Visible {
		t.Errorf("explicit inheritance=%+v", b)
	}
	if got := paintOf(t, r, b, false).Color; got != black {
		t.Errorf("initial fill=%v", got)
	}
	c := computed(t, r, d.Root().Child(0).Child(2), &parent)
	if got := paintOf(t, r, c, false).Color; got != red {
		t.Errorf("explicit inherit=%v", got)
	}
	if got := paintOf(t, r, c, true).Kind; got != opentypesvg.PaintNone {
		t.Errorf("stroke initial=%v", got)
	}
}

func TestStylePaletteSnapshotAndAlpha(t *testing.T) {
	colors := opentypesvg.ColorContext{
		Foreground: blue,
		Palette: []opentypesvg.RGBA{{
			G: 1,
			A: .5,
		}},
	}
	d, r := styles(t, svgStart+`<g fill="var(--color0, red)" fill-opacity=".4" opacity=".25"><rect/><rect fill="var(--color1,currentColor)"/><rect fill="var(--color00,red)"/><rect style="fill:blue; fill:var(--missing)"/></g></svg>`, colors)
	colors.Palette[0] = red
	parent := computed(t, r, d.Root().Child(0), nil)
	child := computed(t, r, d.Root().Child(0).Child(0), &parent)
	p := paintOf(t, r, child, false)
	near(t, p.Color.A, .5)
	near(t, p.Opacity, .4)
	near(t, p.Color.A*p.Opacity, .2)
	near(t, child.FillOpacity, .4)
	near(t, child.Opacity, 1)
	if p.Color.G != 1 {
		t.Errorf("palette snapshot changed: %+v", p.Color)
	}
	fallback := computed(t, r, d.Root().Child(0).Child(1), &parent)
	if got := paintOf(t, r, fallback, false).Color; got != blue {
		t.Errorf("currentColor fallback=%v", got)
	}
	padded := computed(t, r, d.Root().Child(0).Child(2), &parent)
	if got := paintOf(t, r, padded, false).Color; got != red {
		t.Errorf("padded palette index=%v", got)
	}
	missing := computed(t, r, d.Root().Child(0).Child(3), &parent)
	if got := paintOf(t, r, missing, false).Color; got != p.Color {
		t.Errorf("computed invalid variable=%v", got)
	}
}

func TestStyleColors(t *testing.T) {
	for name, c := range colornames.Map {
		d, r := styles(t, svgStart+`<rect fill="`+name+`"/></svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		got := paintOf(t, r, computed(t, r, d.Root().Child(0), nil), false).Color
		want := opentypesvg.RGBA{R: float64(c.R) / 255, G: float64(c.G) / 255, B: float64(c.B) / 255, A: 1}
		if got != want {
			t.Errorf("%s: %v != %v", name, got, want)
		}
	}
	for _, tc := range []struct {
		text string
		want opentypesvg.RGBA
	}{
		{
			text: "#f00",
			want: red,
		},
		{
			text: "#FF0000",
			want: red,
		},
		{
			text: "rgb(1000%,-1,0)",
			want: red,
		},
		{
			text: "rgb(50%,0,0)",
			want: opentypesvg.RGBA{R: .5, A: 1},
		},
		{
			text: "hsl(120,100%,50%)",
			want: opentypesvg.RGBA{G: 1, A: 1},
		},
		{
			text: "hsl(-240,100%,50%)",
			want: opentypesvg.RGBA{G: 1, A: 1},
		},
		{
			text: "rgba(255,0,0,.5)",
			want: opentypesvg.RGBA{R: 1, A: .5},
		},
		{
			text: "transparent",
			want: opentypesvg.RGBA{},
		},
		{
			text: "var(--absent,var(--color0,blue))",
			want: blue,
		},
	} {
		d, r := styles(t, svgStart+`<rect fill="`+tc.text+`"/></svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		if got := paintOf(t, r, computed(t, r, d.Root().Child(0), nil), false).Color; got != tc.want {
			t.Errorf("%s: %v != %v", tc.text, got, tc.want)
		}
	}
	for _, value := range []string{"rgb(,,)", "hsl(0,,50%)", "#12345", "rgb(1,2,)", "hsl(0,50,50)", "rgba(1,2,3,NaN)"} {
		d, r := styles(t, svgStart+`<rect style="fill:blue;fill:`+value+`"/></svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		if got := paintOf(t, r, computed(t, r, d.Root().Child(0), nil), false).Color; got != blue {
			t.Errorf("invalid %s replaced valid declaration: %v", value, got)
		}
	}
}

func TestStyleStrokeAndClip(t *testing.T) {
	d, r := styles(t, svgStart+`<defs><clipPath id="clip"/></defs><rect fill-rule="evenodd" clip-rule="evenodd" stroke-width="10%" stroke-linecap="round" stroke-linejoin="bevel" stroke-miterlimit="7" stroke-dasharray="1, 2 3" stroke-dashoffset="-2px" clip-path="url('#clip')"/></svg>`, opentypesvg.ColorContext{
		Foreground: black,
	})
	s := computed(t, r, d.Root().Child(1), nil)
	stroke, err := r.Stroke(s, opentypesvg.LengthContext{
		Width:         100,
		Height:        100,
		PixelsPerInch: 96,
	})
	if err != nil {
		t.Fatal(err)
	}
	near(t, stroke.Width, 10)
	near(t, stroke.Offset, -2)
	if !slices.Equal(stroke.Dash, []float64{1, 2, 3, 1, 2, 3}) {
		t.Errorf("dash=%v", stroke.Dash)
	}
	if s.FillRule != "evenodd" || s.ClipRule != "evenodd" || s.LineCap != "round" || s.LineJoin != "bevel" || s.MiterLimit != 7 {
		t.Errorf("stroke properties=%+v", s)
	}
	clip, err := r.Clip(s)
	if err != nil {
		t.Fatal(err)
	}
	if clip != d.Root().Child(0).Child(0) {
		t.Error("wrong clip")
	}
}

func TestStyleUnsupportedAndLimits(t *testing.T) {
	for _, css := range []string{"rect:hover{fill:red}", "[x]{fill:red}", "rect + rect{fill:red}", "@media all{rect{fill:red}}", ".a\\:b{fill:red}"} {
		d := parse(t, svgStart+`<style>`+css+`</style></svg>`)
		_, err := newStyleResolver(d, opentypesvg.ColorContext{
			Foreground: black,
		})
		if !errors.Is(err, opentypesvg.ErrUnsupported) && !errors.Is(err, opentypesvg.ErrStyle) {
			t.Errorf("%s: %v", css, err)
		}
	}
	d := parse(t, svgStart+`<style>`+strings.Repeat(".a{fill:red}", opentypesvg.MaxCSSRules+1)+`</style></svg>`)
	if _, err := newStyleResolver(d, opentypesvg.ColorContext{
		Foreground: black,
	}); !errors.Is(err, opentypesvg.ErrLimit) {
		t.Errorf("rule budget: %v", err)
	}
	d, r := styles(t, svgStart+`<rect fill="`+strings.Repeat("x", opentypesvg.MaxStyleValue+1)+`"/></svg>`, opentypesvg.ColorContext{
		Foreground: black,
	})
	if _, err := r.Compute(d.Root().Child(0), nil); !errors.Is(err, opentypesvg.ErrLimit) {
		t.Errorf("value budget: %v", err)
	}
	d, r = styles(t, svgStart+`<style>`+strings.Repeat(".absent{fill:red}", opentypesvg.MaxCSSRules)+`</style>`+strings.Repeat(`<rect class="`+strings.Repeat("a", 1024)+`"/>`, 8)+`</svg>`, opentypesvg.ColorContext{
		Foreground: black,
	})
	var exhausted bool
	for i := 1; i < d.Root().ChildCount(); i++ {
		_, err := r.Compute(d.Root().Child(i), nil)
		if errors.Is(err, opentypesvg.ErrLimit) {
			exhausted = true
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if !exhausted {
		t.Error("aggregate matching work was unbounded")
	}
}

func TestStyleFixtureClasses(t *testing.T) {
	d, r := styles(t, string(read(t, "testdata/focused/css-classes.svg")), opentypesvg.ColorContext{
		Foreground: black,
	})
	glyph, err := d.SelectGlyph(1)
	if err != nil {
		t.Fatal(err)
	}
	parent := computed(t, r, glyph, nil)
	for i := 0; i < glyph.ChildCount(); i++ {
		p := paintOf(t, r, computed(t, r, glyph.Child(i), &parent), false)
		if p.Color != blue {
			t.Errorf("child %d: %v", i, p.Color)
		}
		near(t, p.Opacity, .5)
	}
}

func TestStyleRealFontFixtures(t *testing.T) {
	files, err := filepath.Glob("testdata/real/*.svg")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			d, r := styles(t, string(read(t, file)), opentypesvg.ColorContext{
				Foreground: black,
			})
			var visit func(*opentypesvg.Element, *opentypesvg.Style)
			visit = func(e *opentypesvg.Element, parent *opentypesvg.Style) {
				s := computed(t, r, e, parent)
				for _, stroke := range []bool{false, true} {
					p := paintOf(t, r, s, stroke)
					if p.Kind == opentypesvg.PaintGradient {
						for i := 0; i < p.Gradient.StopCount(); i++ {
							stop := p.Gradient.Stop(i)
							if math.IsNaN(stop.Color.A) || stop.Color.A < 0 || stop.Color.A > 1 {
								t.Errorf("invalid stop: %+v", stop)
							}
						}
					}
				}
				if _, err := r.Clip(s); err != nil {
					t.Error(err)
				}
				for i := 0; i < e.ChildCount(); i++ {
					visit(e.Child(i), &s)
				}
			}
			visit(d.Root(), nil)
		})
	}
}

func TestStyleCommentsAndKeywordCase(t *testing.T) {
	d, r := styles(t, svgStart+`<style>.a/**/.b{FILL:BLUE !IMPORTANT}</style><rect class="a b" style="fill:red"/><rect fill="URL(#missing) NONE"/></svg>`, opentypesvg.ColorContext{
		Foreground: black,
	})
	if got := paintOf(t, r, computed(t, r, d.Root().Child(1), nil), false).Color; got != blue {
		t.Errorf("compound comment=%v", got)
	}
	if got := paintOf(t, r, computed(t, r, d.Root().Child(2), nil), false).Kind; got != opentypesvg.PaintNone {
		t.Errorf("case-insensitive fallback=%v", got)
	}
}

func TestStyleSharedGlyphInstances(t *testing.T) {
	d, r := styles(t, string(read(t, "testdata/real/noto-2-3.svg")), opentypesvg.ColorContext{
		Foreground: blue,
	})
	for _, tc := range []struct {
		glyph uint16
		gray  float64
	}{
		{
			glyph: 2,
			gray:  float64(0x42) / 255,
		},
		{
			glyph: 3,
			gray:  float64(0xe0) / 255,
		},
	} {
		glyph, err := d.SelectGlyph(tc.glyph)
		if err != nil {
			t.Fatal(err)
		}
		parent := computed(t, r, glyph, nil)
		use := glyph.Child(1)
		useStyle := computed(t, r, use, &parent)
		ref, _ := use.Attribute("http://www.w3.org/1999/xlink", "href")
		target, err := d.Resolve(ref)
		if err != nil {
			t.Fatal(err)
		}
		p := paintOf(t, r, computed(t, r, target, &useStyle), false)
		want := opentypesvg.RGBA{R: tc.gray, G: tc.gray, B: tc.gray, A: 1}
		if p.Color != want {
			t.Errorf("glyph %d shared path color=%v, want %v", tc.glyph, p.Color, want)
		}
	}
	d, r = styles(t, string(read(t, "testdata/focused/shared-context.svg")), opentypesvg.ColorContext{
		Foreground: blue,
	})
	for _, gid := range []uint16{1, 2} {
		glyph, err := d.SelectGlyph(gid)
		if err != nil {
			t.Fatal(err)
		}
		if got := paintOf(t, r, computed(t, r, glyph, nil), false).Color; got != black {
			t.Errorf("glyph %d inherited source color: %v", gid, got)
		}
	}
}

func TestStyleInvalidValuesAndReferences(t *testing.T) {
	for _, value := range []string{"-1 2", "1,,2", "1,", "", "0 0"} {
		d, r := styles(t, svgStart+`<rect stroke-dasharray="`+value+`"/></svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		s := computed(t, r, d.Root().Child(0), nil)
		stroke, err := r.Stroke(s, opentypesvg.LengthContext{
			Width:  10,
			Height: 20,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(stroke.Dash) != 0 {
			t.Errorf("invalid or zero dash %q=%v", value, stroke.Dash)
		}
	}
	for _, attrs := range []string{`fill="url(https://example.invalid/g) red"`, `stroke-width="1em"`, `mask="url(#m)"`} {
		d, r := styles(t, svgStart+`<rect `+attrs+`/></svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		if _, err := r.Compute(d.Root().Child(0), nil); !errors.Is(err, opentypesvg.ErrUnsupported) {
			t.Errorf("%s: %v", attrs, err)
		}
	}
	for _, ref := range []string{"missing", "wrong"} {
		d, r := styles(t, svgStart+`<rect id="wrong" clip-path="url(#`+ref+`)"/></svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		s := computed(t, r, d.Root().Child(0), nil)
		if _, err := r.Clip(s); !errors.Is(err, opentypesvg.ErrReference) {
			t.Errorf("clip %s: %v", ref, err)
		}
	}
}

func TestStyleSharedStylesheetOrder(t *testing.T) {
	d, r := styles(t, svgStart+`<style>.a{fill:red}</style><g id="glyph1"><rect class="a"/></g><defs><style>.a{fill:blue}</style></defs></svg>`, opentypesvg.ColorContext{
		Foreground: black,
	})
	glyph, err := d.SelectGlyph(1)
	if err != nil {
		t.Fatal(err)
	}
	parent := computed(t, r, glyph, nil)
	if got := paintOf(t, r, computed(t, r, glyph.Child(0), &parent), false).Color; got != blue {
		t.Errorf("shared later stylesheet=%v", got)
	}
	for _, css := range []string{".a{fill:r/**/ed}", ".a{fill:r/**//**/ed}", ".a{fill:rgb/**/(1,2,3)}"} {
		d := parse(t, svgStart+`<style>`+css+`</style></svg>`)
		if _, err := newStyleResolver(d, opentypesvg.ColorContext{
			Foreground: black,
		}); !errors.Is(err, opentypesvg.ErrUnsupported) {
			t.Errorf("split token %s: %v", css, err)
		}
	}
}

func newStyleResolver(d *opentypesvg.Document, colors opentypesvg.ColorContext) (*opentypesvg.StyleResolver, error) {
	sheet, err := opentypesvg.NewStylesheet(d)
	if err != nil {
		return nil, err
	}
	return opentypesvg.NewStyleResolver(sheet, colors)
}

func TestStyleBoundedNestedVariables(t *testing.T) {
	value := strings.Repeat("var(--missing,", 32) + "red" + strings.Repeat(")", 32)
	d, r := styles(t, svgStart+`<rect fill="`+value+`"/></svg>`, opentypesvg.ColorContext{
		Foreground: black,
	})
	if _, err := r.Compute(d.Root().Child(0), nil); !errors.Is(err, opentypesvg.ErrLimit) {
		t.Errorf("variable depth: %v", err)
	}
}

func TestStyleIgnoredProperties(t *testing.T) {
	for _, declarations := range []string{
		"font-family:serif", "shape-rendering:crispEdges", "mix-blend-mode:normal", "filter:none", "paint-order:normal", "enable-background:accumulate", "enable-background:new 0 0 64 64", "color-profile:srgb", "unknown-property:whatever",
	} {
		for _, source := range []string{
			svgStart + `<style>.unused{font-size:12px} rect{fill:blue;` + declarations + `}</style><rect/></svg>`,
			svgStart + `<rect style="fill:blue;` + declarations + `"/></svg>`,
		} {
			d, r := styles(t, source, opentypesvg.ColorContext{
				Foreground: black,
			})
			e := d.Root().Child(d.Root().ChildCount() - 1)
			if got := paintOf(t, r, computed(t, r, e, nil), false).Color; got != blue {
				t.Errorf("ignored %s changed fill: %v", declarations, got)
			}
		}
	}
	for _, attrs := range []string{`font-family="serif"`, `shape-rendering="crispEdges"`, `paint-order="normal"`, `enable-background="accumulate"`, `color-profile="srgb"`} {
		d, r := styles(t, svgStart+`<rect fill="blue" `+attrs+`/></svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		if got := paintOf(t, r, computed(t, r, d.Root().Child(0), nil), false).Color; got != blue {
			t.Errorf("ignored %s changed fill: %v", attrs, got)
		}
	}
	d, r := styles(t, `<svg xmlns="http://www.w3.org/2000/svg" enable-background="new 0 0 64 64"><linearGradient id="g"><stop stop-color="red"/></linearGradient><rect fill="url(#g)"/></svg>`, opentypesvg.ColorContext{
		Foreground: black,
	})
	g := paintOf(t, r, computed(t, r, d.Root().Child(1), nil), false).Gradient
	if g.StopCount() != 1 {
		t.Fatal("expected one gradient stop")
	}
	if got := g.Stop(0).Color; got != red {
		t.Errorf("Illustrator root gradient=%v", got)
	}
}

func TestStyleUnsupportedEffectsCascade(t *testing.T) {
	for _, tc := range []struct{ name, noop, effect string }{
		{
			name:   "filter",
			noop:   "none",
			effect: "url(#effect)",
		},
		{
			name:   "mask",
			noop:   "none",
			effect: "url(#effect)",
		},
		{
			name:   "marker",
			noop:   "none",
			effect: "url(#effect)",
		},
		{
			name:   "marker-start",
			noop:   "none",
			effect: "url(#effect)",
		},
		{
			name:   "marker-mid",
			noop:   "none",
			effect: "url(#effect)",
		},
		{
			name:   "marker-end",
			noop:   "none",
			effect: "url(#effect)",
		},
		{
			name:   "vector-effect",
			noop:   "none",
			effect: "non-scaling-stroke",
		},
		{
			name:   "paint-order",
			noop:   "normal",
			effect: "stroke fill",
		},
		{
			name:   "mix-blend-mode",
			noop:   "normal",
			effect: "multiply",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, value := range []string{tc.noop, "inherit", "initial", "unset", tc.effect} {
				for _, source := range []string{
					svgStart + `<path ` + tc.name + `="` + value + `"/></svg>`,
					svgStart + `<path style="` + tc.name + `:` + value + `"/></svg>`,
					svgStart + `<style>path{` + tc.name + `:` + value + `}</style><path/></svg>`,
				} {
					d, r := styles(t, source, opentypesvg.ColorContext{
						Foreground: black,
					})
					_, err := r.Compute(d.Root().Child(d.Root().ChildCount()-1), nil)
					if value == tc.effect && !(tc.name == "marker" && strings.Contains(source, `<path marker=`)) {
						if !errors.Is(err, opentypesvg.ErrUnsupported) {
							t.Errorf("effect %s: %v", source, err)
						}
					} else if err != nil {
						t.Errorf("no-op %s: %v", source, err)
					}
				}
			}
			source := svgStart + `<style>.unused{` + tc.name + `:` + tc.effect + `} path{` + tc.name + `:` + tc.effect + `}</style><path style="` + tc.name + `:` + tc.noop + `"/></svg>`
			d, r := styles(t, source, opentypesvg.ColorContext{
				Foreground: black,
			})
			parent := computed(t, r, d.Root().Child(1), nil)
			d2, r2 := styles(t, svgStart+`<path style="`+tc.name+`:inherit"/></svg>`, opentypesvg.ColorContext{
				Foreground: black,
			})
			if _, err := r2.Compute(d2.Root().Child(0), &parent); err != nil {
				t.Errorf("inherited no-op: %v", err)
			}
		})
	}
	for _, declarations := range []string{"marker:url(#x);marker:none", "marker:url(#x);marker-start:none;marker-mid:none;marker-end:none", "marker-start:url(#x);marker:initial"} {
		d, r := styles(t, svgStart+`<path style="`+declarations+`"/></svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		if _, err := r.Compute(d.Root().Child(0), nil); err != nil {
			t.Errorf("marker cascade %s: %v", declarations, err)
		}
	}
	for _, source := range []string{svgStart + `<style>.unused{--color0:red}</style></svg>`, svgStart + `<path style="--color0:red"/></svg>`} {
		d := parse(t, source)
		r, err := newStyleResolver(d, opentypesvg.ColorContext{
			Foreground: black,
		})
		if err == nil {
			_, err = r.Compute(d.Root().Child(0), nil)
		}
		if !errors.Is(err, opentypesvg.ErrUnsupported) {
			t.Errorf("custom property definition: %v", err)
		}
	}
}

func TestStyleVariableFallbackTiming(t *testing.T) {
	for _, fallback := range []string{"nonsense", "red, blue", "var(--other,red,blue)"} {
		for _, hasPalette := range []bool{false, true} {
			colors := opentypesvg.ColorContext{
				Foreground: black,
			}
			if hasPalette {
				colors.Palette = []opentypesvg.RGBA{{
					G: 1,
					A: .5,
				}}
			}
			d, r := styles(t, svgStart+`<g fill="red"><rect style="fill:blue;fill:var(--color0,`+fallback+`)"/></g></svg>`, colors)
			parent := computed(t, r, d.Root().Child(0), nil)
			child := computed(t, r, d.Root().Child(0).Child(0), &parent)
			got := paintOf(t, r, child, false).Color
			want := red
			if hasPalette {
				want = colors.Palette[0]
			}
			if got != want {
				t.Errorf("fallback %q, palette %t: %v != %v", fallback, hasPalette, got, want)
			}
		}
	}
}

func TestStylesheetReuseAndMetadata(t *testing.T) {
	d := parse(t, svgStart+`<style media="screen" type="text/css; charset=utf-8">rect{fill:var(--color0,currentColor)}</style><rect/></svg>`)
	sheet, err := opentypesvg.NewStylesheet(d)
	if err != nil {
		t.Fatal(err)
	}
	for _, colors := range []opentypesvg.ColorContext{
		{
			Foreground: blue,
		},
		{
			Foreground: red,
			Palette: []opentypesvg.RGBA{{
				G: 1,
				A: .5,
			}},
		},
		{
			Foreground: red,
		},
	} {
		r, err := opentypesvg.NewStyleResolver(sheet, colors)
		if err != nil {
			t.Fatal(err)
		}
		p := paintOf(t, r, computed(t, r, d.Root().Child(1), nil), false)
		want := colors.Foreground
		if len(colors.Palette) > 0 {
			want = colors.Palette[0]
		}
		if p.Color != want {
			t.Errorf("shared stylesheet color=%v, want %v", p.Color, want)
		}
	}
	for _, source := range []string{
		svgStart + `<style>@media all{rect{fill:red}}</style></svg>`,
		svgStart + `<style>@import "test.css";</style></svg>`,
		svgStart + `<style media="print">rect{fill:red}</style></svg>`,
		svgStart + `<style type="text/plain">rect{fill:red}</style></svg>`,
	} {
		d := parse(t, source)
		if _, err := opentypesvg.NewStylesheet(d); !errors.Is(err, opentypesvg.ErrUnsupported) {
			t.Errorf("unsupported stylesheet metadata/rule: %v", err)
		}
	}
}

func TestStylePaletteFixture(t *testing.T) {
	for _, hasPalette := range []bool{false, true} {
		colors := opentypesvg.ColorContext{
			Foreground: blue,
		}
		if hasPalette {
			colors.Palette = []opentypesvg.RGBA{{
				G: 1,
				A: .5,
			}}
		}
		d, r := styles(t, string(read(t, "testdata/focused/palette.svg")), colors)
		glyph, err := d.SelectGlyph(1)
		if err != nil {
			t.Fatal(err)
		}
		parent := computed(t, r, glyph, nil)
		a := paintOf(t, r, computed(t, r, glyph.Child(0), &parent), false)
		b := paintOf(t, r, computed(t, r, glyph.Child(1), &parent), false)
		if a.Color != blue {
			t.Errorf("foreground=%v", a.Color)
		}
		want := red
		if hasPalette {
			want = colors.Palette[0]
		}
		if b.Color != want {
			t.Errorf("palette %t: %v != %v", hasPalette, b.Color, want)
		}
		near(t, a.Opacity, 1)
		near(t, b.Opacity, 1)
	}
}

func TestStyleIndependentResourceLimits(t *testing.T) {
	d := parse(t, svgStart+`<rect/></svg>`)
	sheet, err := opentypesvg.NewStylesheet(d)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := opentypesvg.NewStyleResolver(sheet, opentypesvg.ColorContext{
		Foreground: black,
		Palette:    make([]opentypesvg.RGBA, (1<<16)+1),
	}); !errors.Is(err, opentypesvg.ErrLimit) {
		t.Errorf("palette entry limit: %v", err)
	}
	for _, count := range []int{1024, 1025} {
		d, r := styles(t, svgStart+`<rect stroke-dasharray="`+strings.Repeat("1 ", count)+`"/></svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		s, err := r.Compute(d.Root().Child(0), nil)
		if count > 1024 {
			if !errors.Is(err, opentypesvg.ErrLimit) {
				t.Errorf("dash entry limit: %v", err)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		stroke, err := r.Stroke(s, opentypesvg.LengthContext{})
		if err != nil {
			t.Fatal(err)
		}
		if len(stroke.Dash) != count {
			t.Errorf("dash count=%d, want %d", len(stroke.Dash), count)
		}
	}
	for _, depth := range []int{31, 32} {
		value := "var(--color0," + strings.Repeat("f(", depth) + "red" + strings.Repeat(")", depth) + ")"
		d := parse(t, svgStart+`<rect fill="`+value+`"/></svg>`)
		r, err := newStyleResolver(d, opentypesvg.ColorContext{
			Foreground: black,
			Palette:    []opentypesvg.RGBA{blue},
		})
		if err != nil {
			t.Fatal(err)
		}
		s, err := r.Compute(d.Root().Child(0), nil)
		if depth == 32 {
			if !errors.Is(err, opentypesvg.ErrLimit) {
				t.Errorf("CSS nesting limit: %v", err)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if got := paintOf(t, r, s, false).Color; got != blue {
			t.Errorf("unused nested fallback changed paint: %v", got)
		}
	}
}

func TestDeferredGeometrySources(t *testing.T) {
	for _, source := range []string{
		`<rect style="fill:blue;transform:translate(10px,0)"/>`,
		`<style>rect{transform:rotate(45deg)}</style><rect fill="blue"/>`,
		`<rect fill="blue" transform="rotate(45)" transform-origin="center"/>`,
		`<style>rect{width:10px;height:10px}</style><rect fill="blue"/>`,
		`<rect transform-box="fill-box"/>`,
	} {
		d, r := styles(t, svgStart+source+`</svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		if _, err := r.Compute(d.Root().Child(d.Root().ChildCount()-1), nil); !errors.Is(err, opentypesvg.ErrUnsupported) {
			t.Errorf("deferred geometry %s: %v", source, err)
		}
	}
	for _, tc := range []struct{ name, value string }{
		{
			name:  "x",
			value: "10",
		}, {
			name:  "y",
			value: "10",
		}, {
			name:  "width",
			value: "10",
		}, {
			name:  "height",
			value: "10",
		},
		{
			name:  "cx",
			value: "10",
		}, {
			name:  "cy",
			value: "10",
		}, {
			name:  "r",
			value: "10",
		}, {
			name:  "rx",
			value: "10",
		}, {
			name:  "ry",
			value: "10",
		},
		{
			name:  "d",
			value: "path('M0 0L1 1')",
		},
	} {
		d, r := styles(t, svgStart+`<rect style="`+tc.name+`:`+tc.value+`"/></svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		if _, err := r.Compute(d.Root().Child(0), nil); !errors.Is(err, opentypesvg.ErrUnsupported) {
			t.Errorf("CSS %s: %v", tc.name, err)
		}
	}
	d, r := styles(t, svgStart+`<rect x="10" y="10" width="10" height="10" transform="translate(10 0)"/><path d="M0 0L1 1"/><circle cx="10" cy="10" r="10"/><ellipse rx="10" ry="10"/></svg>`, opentypesvg.ColorContext{
		Foreground: black,
	})
	for i := 0; i < d.Root().ChildCount(); i++ {
		computed(t, r, d.Root().Child(i), nil)
	}
	m, err := d.Root().Child(0).LocalTransform(opentypesvg.LengthContext{})
	if err != nil {
		t.Fatal(err)
	}
	mapped(t, m, opentypesvg.Point{}, opentypesvg.Point{X: 10})
	for _, declaration := range []string{"transform:none", "transform-origin:0px 0%", "transform-box:view-box", "width:auto", "x:0", "d:none"} {
		d, r := styles(t, svgStart+`<rect style="`+declaration+`"/></svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		if _, err := r.Compute(d.Root().Child(0), nil); err != nil {
			t.Errorf("initial %s: %v", declaration, err)
		}
	}
}

func TestDeferredEffectsDisplayAndInvalidValues(t *testing.T) {
	for _, display := range []string{"none", "inline"} {
		d, r := styles(t, svgStart+`<g display="`+display+`" filter="url(#f)"><rect/></g></svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		s, err := r.Compute(d.Root().Child(0), nil)
		if display == "none" {
			if err != nil || s.Display {
				t.Errorf("hidden effect: display=%t, err=%v", s.Display, err)
			}
		} else if !errors.Is(err, opentypesvg.ErrUnsupported) {
			t.Errorf("visible effect: %v", err)
		}
	}
	for _, attrs := range []string{`filter="garbage"`, `style="mix-blend-mode:nonsense"`, `style="filter:none;filter:garbage"`, `style="mix-blend-mode:normal;mix-blend-mode:nonsense"`} {
		d, r := styles(t, svgStart+`<rect `+attrs+`/></svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		if _, err := r.Compute(d.Root().Child(0), nil); err != nil {
			t.Errorf("invalid declaration %s: %v", attrs, err)
		}
	}
	d, r := styles(t, svgStart+`<rect style="mix-blend-mode:multiply;mix-blend-mode:nonsense"/></svg>`, opentypesvg.ColorContext{
		Foreground: black,
	})
	if _, err := r.Compute(d.Root().Child(0), nil); !errors.Is(err, opentypesvg.ErrUnsupported) {
		t.Errorf("invalid declaration displaced earlier effect: %v", err)
	}
	for _, value := range []string{"fill", "fill stroke", "fill stroke markers", "fill markers stroke", "markers fill stroke", "markers", "stroke", "markers stroke fill", "fill fill"} {
		d, r := styles(t, svgStart+`<rect paint-order="`+value+`"/></svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		_, err := r.Compute(d.Root().Child(0), nil)
		if value == "stroke" || value == "markers stroke fill" {
			if !errors.Is(err, opentypesvg.ErrUnsupported) {
				t.Errorf("reversed order %s: %v", value, err)
			}
		} else if err != nil {
			t.Errorf("no-op/invalid order %s: %v", value, err)
		}
	}
}

func TestMarkerApplicabilityAndAttributeOrder(t *testing.T) {
	d, r := styles(t, svgStart+`<g marker-end="url(#m)"><rect fill="blue"/><path/><line/><polyline/><polygon/></g></svg>`, opentypesvg.ColorContext{
		Foreground: black,
	})
	parent := computed(t, r, d.Root().Child(0), nil)
	for i := 0; i < d.Root().Child(0).ChildCount(); i++ {
		e := d.Root().Child(0).Child(i)
		_, err := r.Compute(e, &parent)
		if i == 0 {
			if err != nil {
				t.Errorf("rect marker: %v", err)
			}
		} else if !errors.Is(err, opentypesvg.ErrUnsupported) {
			t.Errorf("%s inherited marker: %v", e.Name().Local, err)
		}
	}
	for _, attrs := range []string{
		`marker="url(#m)" marker-start="none" marker-mid="none" marker-end="none"`,
		`marker-start="none" marker-mid="none" marker-end="none" marker="url(#m)"`,
		`marker="url(#m)"`,
	} {
		d, r := styles(t, svgStart+`<path `+attrs+`/></svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		if _, err := r.Compute(d.Root().Child(0), nil); err != nil {
			t.Errorf("attribute order %s: %v", attrs, err)
		}
	}
}

func TestVariableTrailingTokens(t *testing.T) {
	for _, palette := range [][]opentypesvg.RGBA{nil, {blue}} {
		d, r := styles(t, svgStart+`<g fill="red"><rect style="fill:blue;fill:var(--color0, red) blue"/></g></svg>`, opentypesvg.ColorContext{
			Foreground: black,
			Palette:    palette,
		})
		parent := computed(t, r, d.Root().Child(0), nil)
		s := computed(t, r, d.Root().Child(0).Child(0), &parent)
		if got := paintOf(t, r, s, false).Color; got != red {
			t.Errorf("trailing tokens revived an earlier declaration: %v", got)
		}
	}
}

func TestCSSInitialGeometryOverridesAttributes(t *testing.T) {
	for _, source := range []string{
		`<style>rect{transform:none}</style><rect fill="blue" transform="rotate(45)"/>`,
		`<style>rect{x:0}</style><rect fill="blue" x="5" width="1" height="1"/>`,
		`<style>rect{width:auto}</style><rect fill="blue" width="10" height="1"/>`,
		`<rect style="fill:blue;transform:unset" transform="rotate(45)"/>`,
		`<rect style="transform:inherit" transform="rotate(45)"/>`,
		`<rect style="transform:initial" transform="rotate(45)"/>`,
		`<path d="M0 0L1 1" style="d:none"/>`,
	} {
		d, r := styles(t, svgStart+source+`</svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		if _, err := r.Compute(d.Root().Child(d.Root().ChildCount()-1), nil); !errors.Is(err, opentypesvg.ErrUnsupported) {
			t.Errorf("CSS overrides geometry in %s: %v", source, err)
		}
	}
	for _, source := range []string{
		`<style>rect{transform:none}</style><rect fill="blue"/>`,
		`<rect style="transform:none"/>`,
		`<rect display="none" transform="rotate(45)" style="transform:initial"/>`,
	} {
		d, r := styles(t, svgStart+source+`</svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		if _, err := r.Compute(d.Root().Child(d.Root().ChildCount()-1), nil); err != nil {
			t.Errorf("no visible override in %s: %v", source, err)
		}
	}
}

func TestDeferredEffectValueForms(t *testing.T) {
	for _, declaration := range []string{
		"filter:url(#a) url(#b)", "filter:url(#a) blur(2px)", "filter:blur(2px) url(#a) contrast(50%) url(#b)",
		"filter:BLUR(2px)", "transform:translatex(10px)", "transform:Rotate(45deg)", "transform-origin:CENTER",
		"mask:url(#m) no-repeat", "mask-image:url(#m)",
		"clip-path:circle(50%)", "clip-path:ellipse()", "clip-path:inset(1px)", "clip-path:polygon(0 0,1px 0,1px 1px)", "clip-path:path('M0 0L1 1')",
		"clip-path:fill-box", "clip-path:CIRCLE(50%) border-box", "clip-path:content-box inset(1px)",
	} {
		d, r := styles(t, svgStart+`<rect style="`+declaration+`"/></svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		if _, err := r.Compute(d.Root().Child(0), nil); !errors.Is(err, opentypesvg.ErrUnsupported) {
			t.Errorf("valid deferred declaration %s: %v", declaration, err)
		}
	}
	for _, declaration := range []string{
		"filter:none", "transform-box:VIEW-BOX", "mask:none", "mask-image:none", "mask-mode:match-source",
		"mask-repeat:repeat", "mask-position:0% 0%", "mask-clip:border-box", "mask-origin:border-box", "mask-size:auto", "mask-composite:add", "mask-type:luminance",
		"filter:unknown(2px)", "clip-path:unknown(50%)", "clip-path:unknown-box",
	} {
		d, r := styles(t, svgStart+`<rect style="`+declaration+`"/></svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		if _, err := r.Compute(d.Root().Child(0), nil); err != nil {
			t.Errorf("initial or invalid declaration %s: %v", declaration, err)
		}
	}
	for _, attrs := range []string{`filter="blur(2px"`, `mask="url('#m)"`, `clip-path="circle(50%"`} {
		d, r := styles(t, svgStart+`<rect `+attrs+`/></svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		if _, err := r.Compute(d.Root().Child(0), nil); err != nil {
			t.Errorf("broken presentation declaration %s: %v", attrs, err)
		}
	}
	d, r := styles(t, svgStart+`<rect display="none" style="filter:url(#a) blur(2px);clip-path:circle(50%);mask:url(#m) no-repeat"/></svg>`, opentypesvg.ColorContext{
		Foreground: black,
	})
	hidden := computed(t, r, d.Root().Child(0), nil)
	if hidden.Display {
		t.Error("hidden deferred effects lost display:none")
	}
}

func TestStyleLargeSimpleGlyphBudget(t *testing.T) {
	const count = 200000
	d, r := styles(t, svgStart+`<g id="glyph1">`+strings.Repeat(`<rect/>`, count)+`</g></svg>`, opentypesvg.ColorContext{
		Foreground: black,
	})
	glyph, err := d.SelectGlyph(1)
	if err != nil {
		t.Fatal(err)
	}
	parent := computed(t, r, glyph, nil)
	for i := 0; i < glyph.ChildCount(); i++ {
		s, err := r.Compute(glyph.Child(i), &parent)
		if err != nil {
			t.Errorf("simple instance %d of %d: %v", i, count, err)
			return
		}
		if !s.Display {
			t.Errorf("simple instance %d is hidden", i)
		}
	}
}

func TestStyleInheritedValueWorkBudget(t *testing.T) {
	value := "rgb(" + strings.Repeat(" ", 16000) + "255,0,0)"
	d, r := styles(t, svgStart+`<g fill="`+value+`"><rect/></g></svg>`, opentypesvg.ColorContext{
		Foreground: black,
	})
	parent := computed(t, r, d.Root().Child(0), nil)
	for range 2048 {
		_, err := r.Compute(d.Root().Child(0).Child(0), &parent)
		if errors.Is(err, opentypesvg.ErrLimit) {
			return
		}
		if err != nil {
			t.Errorf("inherited value: %v", err)
			return
		}
	}
	t.Error("repeated parsing of a long inherited value did not exhaust the work budget")
}

func TestInheritedStrokeResolutionWorkBudget(t *testing.T) {
	dash := strings.TrimSpace(strings.Repeat("1.00000000000000 ", 960))
	d, r := styles(t, svgStart+`<g stroke="red" stroke-dasharray="`+dash+`"><path/></g></svg>`, opentypesvg.ColorContext{
		Foreground: black,
	})
	parent := computed(t, r, d.Root().Child(0), nil)
	for i := range 2048 {
		child, err := r.Compute(d.Root().Child(0).Child(0), &parent)
		if err != nil {
			t.Errorf("computing child %d before stroke resolution: %v", i, err)
			return
		}
		_, err = r.Stroke(child, opentypesvg.LengthContext{})
		if errors.Is(err, opentypesvg.ErrLimit) {
			return
		}
		if err != nil {
			t.Errorf("stroke resolution: %v", err)
			return
		}
	}
	t.Error("repeated parsing of inherited stroke values did not exhaust the work budget")
}

func TestAdjacentEffectFunctions(t *testing.T) {
	for _, declaration := range []string{
		"transform:rotate(45deg)scale(2)",
		"filter:blur(2px)sepia(1)",
		"filter:url(#a)url(#b)",
		"clip-path:circle(50%)fill-box",
	} {
		d, r := styles(t, svgStart+`<rect style="`+declaration+`"/></svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		if _, err := r.Compute(d.Root().Child(0), nil); !errors.Is(err, opentypesvg.ErrUnsupported) {
			t.Errorf("adjacent components %s: %v", declaration, err)
		}
	}
	for _, declaration := range []string{"filter:blur(2px)unknown(1)", "transform:rotate(45deg)unknown(2)", "clip-path:circle(50%)unknown-box"} {
		d, r := styles(t, svgStart+`<rect style="`+declaration+`"/></svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		if _, err := r.Compute(d.Root().Child(0), nil); err != nil {
			t.Errorf("unknown adjacent component %s: %v", declaration, err)
		}
	}
}

func TestMaskImageCascadeAndInertLonghands(t *testing.T) {
	for _, declaration := range []string{
		"fill:blue;mask-size:contain",
		"fill:blue;mask-image:url(#m);mask:none",
		"mask-mode:alpha", "mask-repeat:no-repeat", "mask-position:center", "mask-clip:content-box", "mask-origin:content-box", "mask-size:cover", "mask-composite:subtract", "mask-type:alpha",
	} {
		d, r := styles(t, svgStart+`<rect style="`+declaration+`"/></svg>`, opentypesvg.ColorContext{
			Foreground: black,
		})
		if _, err := r.Compute(d.Root().Child(0), nil); err != nil {
			t.Errorf("inert mask property %s: %v", declaration, err)
		}
	}
	for _, declarations := range []string{"mask:none;mask-image:url(#m)", "mask-image:url(#m);mask:none"} {
		for _, source := range []string{
			svgStart + `<rect style="` + declarations + `"/></svg>`,
			svgStart + `<style>rect{` + declarations + `}</style><rect/></svg>`,
		} {
			d, r := styles(t, source, opentypesvg.ColorContext{
				Foreground: black,
			})
			_, err := r.Compute(d.Root().Child(d.Root().ChildCount()-1), nil)
			if strings.HasSuffix(declarations, "mask:none") {
				if err != nil {
					t.Errorf("mask reset: %v", err)
				}
			} else if !errors.Is(err, opentypesvg.ErrUnsupported) {
				t.Errorf("effective mask image: %v", err)
			}
		}
	}
}

func TestSparseLocalStyleInheritance(t *testing.T) {
	d, r := styles(t, svgStart+`<g display="none" opacity=".25" filter="url(#f)" style="mask:url(#m)"><rect/><rect filter="inherit"/><rect style="mask:inherit"/><rect display="inherit" opacity="inherit"/></g></svg>`, opentypesvg.ColorContext{
		Foreground: black,
	})
	parent := computed(t, r, d.Root().Child(0), nil)
	ordinary := computed(t, r, d.Root().Child(0).Child(0), &parent)
	if !ordinary.Display {
		t.Error("display inherited without an explicit declaration")
	}
	near(t, ordinary.Opacity, 1)
	for _, i := range []int{1, 2} {
		if _, err := r.Compute(d.Root().Child(0).Child(i), &parent); !errors.Is(err, opentypesvg.ErrUnsupported) {
			t.Errorf("explicitly inherited hidden effect %d: %v", i, err)
		}
	}
	hidden := computed(t, r, d.Root().Child(0).Child(3), &parent)
	if hidden.Display {
		t.Error("explicitly inherited display lost its value")
	}
	near(t, hidden.Opacity, .25)
}

func TestSparseStyleCacheValueLimit(t *testing.T) {
	const declarations = "fill:blue;stroke:none;fill-rule:nonzero;clip-rule:nonzero;stroke-width:1;stroke-linecap:butt;stroke-linejoin:miter;stroke-miterlimit:4;stroke-dasharray:none;stroke-dashoffset:0;fill-opacity:1;stroke-opacity:1;opacity:1;stop-color:black;stop-opacity:1;clip-path:none;display:inline;visibility:visible;color-interpolation:sRGB"
	d, r := styles(t, svgStart+`<style>rect{`+declarations+`}</style><g>`+strings.Repeat(`<rect/>`, 60000)+`</g></svg>`, opentypesvg.ColorContext{
		Foreground: black,
	})
	parent := computed(t, r, d.Root().Child(1), nil)
	for i := 0; i < d.Root().Child(1).ChildCount(); i++ {
		_, err := r.Compute(d.Root().Child(1).Child(i), &parent)
		if errors.Is(err, opentypesvg.ErrLimit) {
			return
		}
		if err != nil {
			t.Errorf("style cache pressure: %v", err)
			return
		}
	}
	t.Error("caching more than the retained-value budget succeeded")
}

func TestStyleCacheKindsHaveIndependentEntryLimits(t *testing.T) {
	const count = 262138
	d, r := styles(t, svgStart+`<linearGradient id="g"><stop stop-color="red"/></linearGradient><g id="glyph1">`+strings.Repeat(`<rect fill="url(#g)"/>`, count)+`</g></svg>`, opentypesvg.ColorContext{
		Foreground: black,
	})
	glyph, err := d.SelectGlyph(1)
	if err != nil {
		t.Fatal(err)
	}
	parent := computed(t, r, glyph, nil)
	var last opentypesvg.Style
	for i := 0; i < glyph.ChildCount(); i++ {
		last, err = r.Compute(glyph.Child(i), &parent)
		if err != nil {
			t.Errorf("instance %d: %v", i, err)
			return
		}
	}
	paint, err := r.Paint(last, false, opentypesvg.PaintContext{
		Bounds: opentypesvg.ViewBox{
			Width:  1,
			Height: 1,
		},
	})
	if err != nil {
		t.Errorf("resolving a gradient after caching %d elements: %v", count, err)
		return
	}
	if paint.Kind != opentypesvg.PaintGradient || paint.Gradient.StopCount() != 1 {
		t.Errorf("paint=%+v, want a gradient with one stop", paint)
		return
	}
	if got := paint.Gradient.Stop(0).Color; got != red {
		t.Errorf("gradient color=%v, want %v", got, red)
	}
}

func TestCSSDeclarationValueLimit(t *testing.T) {
	value := strings.Repeat("x", opentypesvg.MaxStyleValue+1)
	for _, source := range []string{
		svgStart + `<style>rect{fill:` + value + `}</style><rect/></svg>`,
		svgStart + `<rect style="fill:` + value + `"/></svg>`,
	} {
		d := parse(t, source)
		r, err := newStyleResolver(d, opentypesvg.ColorContext{
			Foreground: black,
		})
		if err == nil {
			_, err = r.Compute(d.Root().Child(d.Root().ChildCount()-1), nil)
		}
		if !errors.Is(err, opentypesvg.ErrLimit) {
			t.Errorf("oversized CSS declaration: %v", err)
		}
	}
}

func TestColorValueLimitAcrossDeclarationSources(t *testing.T) {
	for _, size := range []int{opentypesvg.MaxStyleValue, opentypesvg.MaxStyleValue + 1} {
		value := "rgb(" + strings.Repeat(" ", size-len("rgb(255,0,0)")) + "255,0,0)"
		for i, source := range []string{
			svgStart + `<rect fill="` + value + `"/></svg>`,
			svgStart + `<rect style="fill:` + value + `"/></svg>`,
			svgStart + `<rect style="fill:` + value + ` !important"/></svg>`,
			svgStart + `<style>rect{fill:` + value + `}</style><rect/></svg>`,
			svgStart + `<style>rect{fill:` + value + ` !important}</style><rect/></svg>`,
		} {
			d := parse(t, source)
			r, err := newStyleResolver(d, opentypesvg.ColorContext{
				Foreground: black,
			})
			var s opentypesvg.Style
			if err == nil {
				s, err = r.Compute(d.Root().Child(d.Root().ChildCount()-1), nil)
			}
			if size > opentypesvg.MaxStyleValue {
				if !errors.Is(err, opentypesvg.ErrLimit) {
					t.Errorf("size %d, source %d: %v", size, i, err)
				}
				continue
			}
			if err != nil {
				t.Errorf("size %d, source %d: %v", size, i, err)
				continue
			}
			paint := paintOf(t, r, s, false)
			if paint.Kind != opentypesvg.PaintSolid || paint.Color != red {
				t.Errorf("size %d, source %d: paint=%+v, want red", size, i, paint)
			}
		}
	}
}
