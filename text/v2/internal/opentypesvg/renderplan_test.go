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
	"image/color"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/opentypesvg"
)

func prepare(t *testing.T, body string) *opentypesvg.RenderPlan {
	t.Helper()
	plan, err := opentypesvg.PrepareRender([]byte(`<svg xmlns="http://www.w3.org/2000/svg">`+body+`</svg>`), opentypesvg.RenderOptions{
		UnitsPerEm: 48,
		Width:      48,
		Height:     48,
		Samples:    4,
		Foreground: color.NRGBA{A: 255},
	})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestRenderPreparationErrors(t *testing.T) {
	tests := []struct {
		body string
		want error
	}{
		{
			body: `<path d="M0 0L4 4L"/>`,
			want: opentypesvg.ErrGeometry,
		},
		{
			body: `<image width="1" height="1" href="data:image/png;base64,AA=="/>`,
			want: opentypesvg.ErrImage,
		},
		{
			body: `<path d="M0 0H40" pathLength="10" stroke="red" stroke-dasharray="2 2"/>`,
			want: opentypesvg.ErrUnsupported,
		},
		{
			body: `<unknown/>`,
			want: opentypesvg.ErrUnsupported,
		},
		{
			body: `<svg><rect width="10" height="10"/></svg>`,
			want: opentypesvg.ErrUnsupported,
		},
		{
			body: `<defs><clipPath id="c" clip-path="url(#c)"><circle r="10"/></clipPath></defs><circle r="10" clip-path="url(#c)"/>`,
			want: opentypesvg.ErrReferenceCycle,
		},
		{
			body: `<use id="a" href="#a"/>`,
			want: opentypesvg.ErrReferenceCycle,
		},
		{
			body: `<rect width="10" height="10" transform="scale(1e100)"/>`,
			want: opentypesvg.ErrLimit,
		},
	}
	for _, tt := range tests {
		t.Run(tt.body, func(t *testing.T) {
			plan, err := opentypesvg.PrepareRender([]byte(`<svg>`+tt.body+`</svg>`), opentypesvg.RenderOptions{
				UnitsPerEm: 48,
				Width:      48,
				Height:     48,
				Samples:    4,
			})
			if !errors.Is(err, tt.want) {
				t.Errorf("error=%v,want %v", err, tt.want)
			}
			if plan != nil {
				t.Error("error returned a partial plan")
			}
		})
	}
}

func TestRenderPreparationPruningAndInheritance(t *testing.T) {
	p := prepare(t, `<defs><g fill="red" transform="translate(30)"><path id="tile" d="M0 0H8V8H0Z"/></g></defs><style>defs path {fill:blue}</style><g display="none"><unknown/><use href="#missing"/></g><g visibility="hidden"><use href="#tile" visibility="visible"/></g>`)
	if len(p.Draws) != 1 {
		t.Fatalf("draw count=%d,want 1", len(p.Draws))
	}
	d := p.Draws[0]
	if d.Paint.Color != [4]float32{0, 0, 1, 1} {
		t.Errorf("paint=%v", d.Paint.Color)
	}
	if d.Transform[4] != 0 {
		t.Errorf("unexpected source ancestor transform: %v", d.Transform)
	}
}

func TestRenderPreparationBounds(t *testing.T) {
	for _, tt := range []struct {
		axis string
		want [3]float32
	}{
		{
			axis: `x2="1" y2="0"`,
			want: [3]float32{1.0 / 160, 0, 0},
		},
		{
			axis: `x2="0" y2="1"`,
			want: [3]float32{0, 1.0 / 80, 0},
		},
	} {
		p := prepare(t, fmt.Sprintf(`<defs><linearGradient id="g" %s><stop stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient></defs><path d="M0 0Q20 40 40 0Z" fill="url(#g)" stroke="red" stroke-width="20"/>`, tt.axis))
		if len(p.Draws) != 2 {
			t.Fatalf("draw count=%d", len(p.Draws))
		}
		for i, want := range tt.want {
			if got := p.Draws[0].Paint.Linear[i]; math.Abs(float64(got-want)) > 1e-7 {
				t.Errorf("%s: linear coefficient %d=%v,want %v", tt.axis, i, got, want)
			}
		}
	}
}

func TestRenderPreparationLimits(t *testing.T) {
	options := opentypesvg.RenderOptions{
		UnitsPerEm: 48,
		Width:      48,
		Height:     48,
		Samples:    4,
	}
	for _, size := range []int{0, math.MaxInt, 4097} {
		options.Width = size
		want := opentypesvg.ErrLimit
		if size == 0 {
			want = opentypesvg.ErrGeometry
		}
		if _, err := opentypesvg.PrepareRender([]byte(`<svg/>`), options); !errors.Is(err, want) {
			t.Errorf("width %d: error=%v,want %v", size, err, want)
		}
	}
	options.Width = 48
	body := `<svg><path d="M0 0L40 0" stroke="red" stroke-dasharray=".00001 .00001"/></svg>`
	if _, err := opentypesvg.PrepareRender([]byte(body), options); !errors.Is(err, opentypesvg.ErrLimit) {
		t.Errorf("dash limit=%v", err)
	}
	body = `<svg><defs><linearGradient id="g">` + strings.Repeat(`<stop/>`, 65) + `</linearGradient></defs><rect width="10" height="10" fill="url(#g)"/></svg>`
	if _, err := opentypesvg.PrepareRender([]byte(body), options); !errors.Is(err, opentypesvg.ErrLimit) {
		t.Errorf("stop limit=%v", err)
	}
}

func FuzzRenderPreparation(f *testing.F) {
	f.Add(`<svg><defs><linearGradient id="g"><stop offset=".1" stop-color="red"/><stop offset=".9" stop-color="blue" stop-opacity=".5"/></linearGradient></defs><path d="M0 0H24V24Z" fill="url(#g)" stroke="lime" stroke-width="2" stroke-miterlimit="3"/></svg>`, 1.0)
	f.Add(`<svg><path d="M12 12L40 40" stroke="lime" stroke-width="10" stroke-linecap="square" stroke-dasharray="0 100"/></svg>`, 1.0)
	f.Add(`<svg><path d="M0 0Q20 40 40 0Z" fill="red"/></svg>`, 1.0)
	f.Add(`<svg><defs><clipPath id="c"><use href="#a"/></clipPath></defs><g id="a" clip-path="url(#c)"><circle r="4"/></g></svg>`, 1.0)
	f.Fuzz(func(t *testing.T, source string, scale float64) {
		if len(source) > 16384 {
			t.Skip()
		}
		plan, err := opentypesvg.PrepareRender([]byte(source), opentypesvg.RenderOptions{
			UnitsPerEm: 48,
			Width:      48,
			Height:     48,
			Samples:    4,
			Transform:  [6]float64{scale, 0, 0, scale, .375, .375},
		})
		if (plan != nil) != (err == nil) {
			t.Errorf("plan returned=%t error=%v", plan != nil, err)
		}
		if plan != nil {
			checkRenderPlanNumbers(t, plan)
		}
	})
}

func checkRenderPlanNumbers(t *testing.T, plan *opentypesvg.RenderPlan) {
	t.Helper()
	check := func(values ...float64) {
		t.Helper()
		for _, v := range values {
			if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > opentypesvg.MaxRenderCoordinate {
				t.Errorf("invalid rendering number %g", v)
			}
		}
	}
	commands := func(commands []opentypesvg.RenderCommand) {
		t.Helper()
		for _, c := range commands {
			check(c.X, c.Y, c.CX, c.CY, c.DX, c.DY)
		}
	}
	seen := map[*opentypesvg.RenderClip]bool{}
	var clip func(*opentypesvg.RenderClip)
	clip = func(c *opentypesvg.RenderClip) {
		if c == nil || seen[c] {
			return
		}
		seen[c] = true
		commands(c.Commands)
		check(c.Transform[:]...)
		for _, child := range c.Children {
			clip(child)
		}
	}
	for _, draw := range plan.Draws {
		commands(draw.Commands)
		check(draw.Transform[:]...)
		for _, v := range draw.Paint.Inverse {
			check(float64(v))
		}
		for _, v := range draw.Paint.Linear {
			check(float64(v))
		}
		for _, v := range []float64{draw.StrokeWidth, draw.MiterLimit} {
			if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
				t.Errorf("invalid stroke parameter %g", v)
			}
		}
		var previous float32
		for _, v := range draw.Paint.Offsets {
			if math.IsNaN(float64(v)) || v < 0 || v > 1 || v < previous {
				t.Errorf("invalid gradient offset %g after %g", v, previous)
			}
			previous = v
		}
		for _, stop := range draw.Paint.Stops {
			for _, v := range stop {
				if math.IsNaN(float64(v)) || v < 0 || v > 1 {
					t.Errorf("invalid gradient stop channel %g", v)
				}
			}
		}
		clip(draw.Clip)
	}
}

func TestRenderClipSubmissionBudget(t *testing.T) {
	data := `M0 0` + strings.Repeat(`L24 0L24 24L0 24L0 0`, 4000)
	for _, draws := range []int{1, 100, 400} {
		t.Run(fmt.Sprint(draws), func(t *testing.T) {
			source := []byte(`<svg><defs><path id="r" d="M0 0H24V24H0Z"/><clipPath id="c"><path d="` + data + `"/></clipPath></defs><g clip-path="url(#c)">` + strings.Repeat(`<use href="#r"/>`, draws) + `</g></svg>`)
			plan, err := opentypesvg.PrepareRender(source, opentypesvg.RenderOptions{
				UnitsPerEm: 48,
				Width:      48,
				Height:     48,
				Samples:    4,
			})
			if draws == 1 {
				if err != nil || plan == nil {
					t.Errorf("one clipped draw: plan=%v error=%v", plan, err)
				}
			} else if !errors.Is(err, opentypesvg.ErrLimit) || plan != nil {
				t.Errorf("%d clipped draws: plan returned=%t error=%v, want ErrLimit", draws, plan != nil, err)
			}
		})
	}
}

func TestRenderZeroOpacity(t *testing.T) {
	for _, body := range []string{`<g opacity="0"><rect width="10" height="10" fill="red"/></g>`, `<rect width="10" height="10" fill="red" stroke="blue" opacity="0"/>`, `<g opacity="0"><use href="#missing"/></g>`} {
		plan, err := opentypesvg.PrepareRender([]byte(`<svg>`+body+`</svg>`), opentypesvg.RenderOptions{
			UnitsPerEm: 48,
			Width:      48,
			Height:     48,
			Samples:    4,
		})
		if err != nil {
			t.Errorf("%s: %v", body, err)
			continue
		}
		if len(plan.Draws) != 0 {
			t.Errorf("%s: %d draws, want none", body, len(plan.Draws))
		}
	}
}

func TestRenderFlatLinearGradient(t *testing.T) {
	for _, fallback := range []string{"", " red"} {
		source := []byte(`<svg><defs><linearGradient id="g"><stop stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient></defs><path d="M0 10H40V10.0000001H0Z" fill="url(#g)` + fallback + `"/></svg>`)
		plan, err := opentypesvg.PrepareRender(source, opentypesvg.RenderOptions{
			UnitsPerEm: 48,
			Width:      48,
			Height:     48,
			Samples:    4,
		})
		if err != nil {
			t.Errorf("fallback %q: %v", fallback, err)
			continue
		}
		if len(plan.Draws) != 1 {
			t.Errorf("draws=%d,want 1", len(plan.Draws))
		}
	}
}

func TestRenderExcessiveGradientScale(t *testing.T) {
	for _, definition := range []string{
		`<linearGradient id="g" gradientUnits="userSpaceOnUse" x2=".000000000001"><stop/><stop offset="1"/></linearGradient>`,
		`<linearGradient id="g" x2="0" y2="1"><stop/><stop offset="1"/></linearGradient>`,
		`<radialGradient id="g"><stop/><stop offset="1"/></radialGradient>`,
	} {
		source := []byte(`<svg><defs>` + definition + `</defs><path d="M0 10H40V10.0000001H0Z" fill="url(#g) red"/></svg>`)
		plan, err := opentypesvg.PrepareRender(source, opentypesvg.RenderOptions{
			UnitsPerEm: 48,
			Width:      48,
			Height:     48,
			Samples:    4,
		})
		if !errors.Is(err, opentypesvg.ErrLimit) || plan != nil {
			t.Errorf("%s: plan returned=%t error=%v,want ErrLimit", definition, plan != nil, err)
		}
	}
}

func TestRenderStencilAreaBudget(t *testing.T) {
	for _, tt := range []struct {
		name        string
		count, size int
		clip        bool
	}{
		{
			name:  "many-clips",
			count: 11000,
			size:  256,
			clip:  true,
		},
		{
			name:  "clip-area",
			count: 200,
			size:  96,
			clip:  true,
		},
		{
			name:  "aggregate-draw-area",
			count: 200,
			size:  96,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rect := `<rect width="48" height="48" fill="red" fill-opacity=".01"/>`
			body := strings.Repeat(rect, tt.count)
			if tt.clip {
				body = `<defs><clipPath id="c">` + body + `</clipPath></defs><rect width="48" height="48" clip-path="url(#c)"/>`
			}
			scale := float64(tt.size) / 48
			plan, err := opentypesvg.PrepareRender([]byte(`<svg>`+body+`</svg>`), opentypesvg.RenderOptions{
				UnitsPerEm: 48,
				Width:      tt.size,
				Height:     tt.size,
				Samples:    8,
				Transform:  [6]float64{scale, 0, 0, scale, 0, 0},
			})
			if !errors.Is(err, opentypesvg.ErrLimit) || plan != nil {
				t.Errorf("plan returned=%t error=%v,want ErrLimit", plan != nil, err)
			}
		})
	}
}

func TestZeroOpacityBoundsDemand(t *testing.T) {
	for _, tt := range []struct {
		name, content string
		bounds        bool
		want          error
	}{
		{
			name:    "filter",
			content: `<g opacity="0" filter="url(#f)"><rect width="10" height="10"/></g>`,
		},
		{
			name:    "mask",
			content: `<rect opacity="0" width="10" height="10" mask="url(#m)"/>`,
		},
		{
			name:    "unsupported-unused",
			content: `<g opacity="0"><image/></g>`,
		},
		{
			name:    "unresolved-unused",
			content: `<g opacity="0"><use href="#missing"/></g>`,
		},
		{
			name:    "unresolved-needed",
			content: `<g opacity="0"><use href="#missing"/></g>`,
			bounds:  true,
			want:    opentypesvg.ErrReference,
		},
		{
			name:    "malformed-needed",
			content: `<g opacity="0"><path d="M0 0L"/></g>`,
			bounds:  true,
			want:    opentypesvg.ErrGeometry,
		},
		{
			name:    "unsupported-needed",
			content: `<g opacity="0"><unknown/></g>`,
			bounds:  true,
			want:    opentypesvg.ErrUnsupported,
		},
		{
			name:    "effects-ignored-for-bounds",
			content: `<g opacity="0" filter="url(#f)"><rect width="48" height="48" mask="url(#m)"/></g>`,
			bounds:  true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			attribute := ""
			if tt.bounds {
				attribute = `clip-path="url(#c)"`
			}
			source := []byte(`<svg><defs><filter id="f"/><mask id="m"/><clipPath id="c" clipPathUnits="objectBoundingBox"><rect width=".5" height="1"/></clipPath></defs><g ` + attribute + `>` + tt.content + `</g></svg>`)
			plan, err := opentypesvg.PrepareRender(source, opentypesvg.RenderOptions{
				UnitsPerEm: 48,
				Width:      48,
				Height:     48,
				Samples:    4,
			})
			if !errors.Is(err, tt.want) {
				t.Errorf("error=%v,want %v", err, tt.want)
			}
			if plan != nil && len(plan.Draws) != 0 {
				t.Errorf("%d draws,want none", len(plan.Draws))
			}
			if (plan != nil) != (err == nil) {
				t.Errorf("plan returned=%t error=%v", plan != nil, err)
			}
		})
	}
}

func TestRenderCorpusStencilBudget(t *testing.T) {
	for _, directory := range []string{"focused", "real"} {
		paths, err := filepath.Glob(filepath.Join("testdata", directory, "*.svg"))
		if err != nil || len(paths) == 0 {
			t.Fatalf("fixtures in %s: %v", directory, err)
		}
		for _, path := range paths {
			name := strings.TrimSuffix(filepath.Base(path), ".svg")
			for _, size := range []int{24, 96} {
				var offset float64
				if size == 96 {
					offset = .375
				}
				t.Run(fmt.Sprintf("%s/%d/%g", name, size, offset), func(t *testing.T) {
					options := opentypesvg.RenderOptions{
						GlyphID:    1,
						UnitsPerEm: 48,
						Width:      size,
						Height:     size,
						Samples:    8,
						Foreground: color.NRGBA{A: 255},
					}
					scale := float64(size) / 48
					tx, ty := offset, offset
					if directory == "real" {
						options.UnitsPerEm = 2048
						if _, err := fmt.Sscanf(strings.Split(name, "-")[1], "%d", &options.GlyphID); err != nil {
							t.Fatal(err)
						}
						scale = float64(size) / 2300
						tx += float64(size) * .04
						ty += float64(size) * .77
					}
					options.Transform = [6]float64{scale, 0, 0, scale, tx, ty}
					var want error
					plan, err := opentypesvg.PrepareRender(read(t, path), options)
					if !errors.Is(err, want) {
						t.Errorf("error=%v,want %v", err, want)
					}
					if (plan != nil) != (err == nil) {
						t.Errorf("plan returned=%t error=%v", plan != nil, err)
					}
				})
			}
		}
	}
}

func TestRenderStencilBoundsIntersection(t *testing.T) {
	source := []byte(`<svg><defs><clipPath id="c">` + strings.Repeat(`<rect width="48" height="48"/>`, 1000) + `</clipPath></defs><rect width="1" height="1" clip-path="url(#c)"/></svg>`)
	plan, err := opentypesvg.PrepareRender(source, opentypesvg.RenderOptions{
		UnitsPerEm: 48,
		Width:      96,
		Height:     96,
		Samples:    8,
		Transform:  [6]float64{2, 0, 0, 2, 0, 0},
	})
	if err != nil || plan == nil {
		t.Errorf("plan returned=%t error=%v", plan != nil, err)
	}
	source = []byte(`<svg>` + strings.Repeat(`<path d="M24 24H25" fill="none" stroke="red" stroke-width="1000"/>`, 20) + `</svg>`)
	plan, err = opentypesvg.PrepareRender(source, opentypesvg.RenderOptions{
		UnitsPerEm: 48,
		Width:      96,
		Height:     96,
		Samples:    8,
		Transform:  [6]float64{2, 0, 0, 2, 0, 0},
	})
	if err != nil || plan == nil {
		t.Fatalf("expanded strokes: %v", err)
	}
	var boundaries int
	for _, draw := range plan.Draws {
		if draw.FlushBefore {
			boundaries++
		}
	}
	if boundaries == 0 {
		t.Error("expanded strokes require a batch boundary")
	}
}

func TestRenderLargeRealRecords(t *testing.T) {
	paths, err := filepath.Glob("testdata/real/*.svg")
	if err != nil || len(paths) == 0 {
		t.Fatalf("real fixtures: %v", err)
	}
	for _, path := range paths {
		for _, size := range []int{256, 512} {
			t.Run(fmt.Sprintf("%s/%d", filepath.Base(path), size), func(t *testing.T) {
				samples := 8
				if size == 512 {
					samples = 4
				}
				options := opentypesvg.RenderOptions{
					UnitsPerEm: 2048,
					Width:      size,
					Height:     size,
					Samples:    samples,
					Foreground: color.NRGBA{A: 255},
				}
				if _, err := fmt.Sscanf(strings.Split(filepath.Base(path), "-")[1], "%d", &options.GlyphID); err != nil {
					t.Fatal(err)
				}
				scale := float64(size) / 2300
				options.Transform = [6]float64{scale, 0, 0, scale, float64(size) * .04, float64(size) * .77}
				plan, err := opentypesvg.PrepareRender(read(t, path), options)
				if err != nil || plan == nil {
					t.Errorf("plan returned=%t error=%v", plan != nil, err)
				}
			})
		}
	}
}

func TestRenderPaintAndRuleBoundaries(t *testing.T) {
	for _, kind := range []string{"gradient", "fill-rule"} {
		t.Run(kind, func(t *testing.T) {
			body := `<defs><linearGradient id="g"><stop stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient></defs>`
			for i := range 60 {
				paint, rule := "red", "nonzero"
				if i%2 == 1 {
					if kind == "gradient" {
						paint = "url(#g)"
					} else {
						rule = "evenodd"
					}
				}
				body += fmt.Sprintf(`<rect width="96" height="96" fill="%s" fill-rule="%s"/>`, paint, rule)
			}
			plan, err := opentypesvg.PrepareRender([]byte(`<svg>`+body+`</svg>`), opentypesvg.RenderOptions{
				UnitsPerEm: 96,
				Width:      96,
				Height:     96,
				Samples:    8,
			})
			if err != nil {
				t.Fatal(err)
			}
			for i, draw := range plan.Draws {
				want := kind == "fill-rule" && i > 0
				if draw.FlushBefore != want {
					t.Errorf("draw %d: flush=%t,want %t", i, draw.FlushBefore, want)
				}
			}
		})
	}
}

func TestRenderCulledRuleChangeBoundary(t *testing.T) {
	solid := strings.Repeat(`<rect width="96" height="96" fill="red"/>`, 13)
	body := `<defs><clipPath id="empty"/></defs>` + solid + `<rect width="96" height="96" fill-rule="evenodd" clip-path="url(#empty)"/>` + solid
	plan, err := opentypesvg.PrepareRender([]byte(`<svg>`+body+`</svg>`), opentypesvg.RenderOptions{
		UnitsPerEm: 96,
		Width:      96,
		Height:     96,
		Samples:    8,
	})
	if err != nil {
		t.Fatal(err)
	}
	var boundary bool
	for _, draw := range plan.Draws {
		boundary = boundary || draw.FlushBefore
	}
	if !boundary {
		t.Error("culled rule change must not join two oversized batches")
	}
}
