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

package svgrender_test

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/internal/imagebridge"
	etesting "github.com/hajimehoshi/ebiten/v2/internal/testing"
	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/opentypesvg"
	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/opentypesvg/svgrender"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func TestMain(m *testing.M) {
	ebiten.SetWindowVisible(false)
	etesting.MainWithRunLoop(m)
}

func options() opentypesvg.RenderOptions {
	return opentypesvg.RenderOptions{
		GlyphID:    1,
		UnitsPerEm: 48,
		Width:      48,
		Height:     48,
		Samples:    4,
		Foreground: color.NRGBA{A: 255},
	}
}

func render(t *testing.T, source []byte, options opentypesvg.RenderOptions) []byte {
	t.Helper()
	plan, err := opentypesvg.PrepareRender(source, options)
	if err != nil {
		t.Fatal(err)
	}
	var renderer svgrender.Renderer
	defer renderer.Deallocate()
	dst := ebiten.NewImage(options.Width, options.Height)
	defer dst.Deallocate()
	if err := renderer.Draw(dst, plan); err != nil {
		t.Fatal(err)
	}
	pixels := make([]byte, 4*options.Width*options.Height)
	dst.ReadPixels(pixels)
	return pixels
}

func svg(body string) []byte {
	return []byte(`<svg xmlns="http://www.w3.org/2000/svg">` + body + `</svg>`)
}

func checkPixel(t *testing.T, pixels []byte, width, x, y int, want color.RGBA) {
	t.Helper()
	offset := 4 * (y*width + x)
	expected := []byte{want.R, want.G, want.B, want.A}
	for i, v := range expected {
		if d := int(pixels[offset+i]) - int(v); d < -1 || d > 1 {
			t.Errorf("pixel (%d,%d)=%v,want %v", x, y, pixels[offset:offset+4], expected)
			return
		}
	}
}

func TestFocusedFixtures(t *testing.T) {
	tests := []struct {
		name string
		x, y int
		want color.RGBA
	}{
		{
			name: "css-classes",
			x:    4,
			y:    4,
			want: color.RGBA{B: 128, A: 128},
		},
		{
			name: "evenodd-clip",
			x:    8,
			y:    8,
			want: color.RGBA{R: 255, A: 255},
		},
		{
			name: "evenodd-clip",
			x:    24,
			y:    24,
			want: color.RGBA{},
		},
		{
			name: "shared-context",
			x:    4,
			y:    4,
			want: color.RGBA{A: 255},
		},
		{
			name: "shared-context",
			x:    20,
			y:    4,
			want: color.RGBA{},
		},
		{
			name: "stroke-transform",
			x:    24,
			y:    16,
			want: color.RGBA{G: 128, A: 255},
		},
		{
			name: "stroke-transform",
			x:    24,
			y:    20,
			want: color.RGBA{},
		},
		{
			name: "restricted",
			x:    4,
			y:    4,
			want: color.RGBA{R: 255, A: 255},
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s/%d/%d", tt.name, tt.x, tt.y), func(t *testing.T) {
			source, err := os.ReadFile(filepath.Join("..", "testdata", "focused", tt.name+".svg"))
			if err != nil {
				t.Fatal(err)
			}
			pixels := render(t, source, options())
			checkPixel(t, pixels, 48, tt.x, tt.y, tt.want)
		})
	}
}

func TestClipIdentity(t *testing.T) {
	for _, tt := range []struct {
		samples int
		size    int
		offset  float64
	}{
		{
			samples: 1,
			size:    48,
		},
		{
			samples: 2,
			size:    16,
			offset:  .375,
		},
		{
			samples: 4,
			size:    96,
		},
		{
			samples: 8,
			size:    48,
			offset:  .375,
		},
	} {
		t.Run(fmt.Sprintf("%d/%d/%g", tt.samples, tt.size, tt.offset), func(t *testing.T) {
			op := options()
			op.Width = tt.size
			op.Height = tt.size
			op.Samples = tt.samples
			scale := float64(tt.size) / 48
			op.Transform = [6]float64{scale, 0, 0, scale, tt.offset, tt.offset}
			shape := `<circle cx="24" cy="24" r="17"/>`
			plain := render(t, svg(`<g fill="blue">`+shape+`</g>`), op)
			clipped := render(t, svg(`<defs><clipPath id="c">`+shape+`</clipPath><clipPath id="d" clip-path="url(#c)">`+shape+`</clipPath></defs><g fill="blue" clip-path="url(#d)">`+shape+`</g>`), op)
			var maxError int
			for i, v := range plain {
				maxError = max(maxError, int(math.Abs(float64(int(v)-int(clipped[i])))))
			}
			if maxError > 1 {
				t.Errorf("identity max error=%d", maxError)
			}
		})
	}
}

func TestGradients(t *testing.T) {
	tests := []struct {
		name, defs, fill string
		x, y             int
		want             color.RGBA
	}{
		{
			name: "linear",
			defs: `<linearGradient id="g"><stop stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient>`,
			fill: "url(#g)",
			x:    24,
			y:    24,
			want: color.RGBA{R: 125, B: 130, A: 255},
		},
		{
			name: "fallback",
			fill: "url(#missing) red",
			x:    24,
			y:    24,
			want: color.RGBA{R: 255, A: 255},
		},
		{
			name: "disabled",
			defs: `<linearGradient id="g"/>`,
			fill: "url(#g)",
			x:    24,
			y:    24,
			want: color.RGBA{},
		},
		{
			name: "solid",
			defs: `<linearGradient id="g"><stop stop-color="lime"/></linearGradient>`,
			fill: "url(#g)",
			x:    24,
			y:    24,
			want: color.RGBA{G: 255, A: 255},
		},
		{
			name: "repeat",
			defs: `<linearGradient id="g" gradientUnits="userSpaceOnUse" x2="16" spreadMethod="repeat"><stop stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient>`,
			fill: "url(#g)",
			x:    24,
			y:    24,
			want: color.RGBA{R: 120, B: 135, A: 255},
		},
		{
			name: "reflect",
			defs: `<linearGradient id="g" gradientUnits="userSpaceOnUse" x2="16" spreadMethod="reflect"><stop stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient>`,
			fill: "url(#g)",
			x:    24,
			y:    24,
			want: color.RGBA{R: 135, B: 120, A: 255},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := svg(`<defs>` + tt.defs + `</defs><rect width="48" height="48" fill="` + tt.fill + `"/>`)
			checkPixel(t, render(t, source, options()), 48, tt.x, tt.y, tt.want)
		})
	}
	a := `<linearGradient id="a" href="#b"/>`
	b := `<linearGradient id="b"><stop stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient>`
	shape := `<rect width="48" height="48" fill="url(#a)"/>`
	if !bytes.Equal(render(t, svg(a+b+shape), options()), render(t, svg(b+a+shape), options())) {
		t.Error("definition order changed pixels")
	}
}

func TestDashAndNonuniformStroke(t *testing.T) {
	p := render(t, svg(`<path d="M2 8H22" fill="none" stroke="lime" stroke-width="2" stroke-dasharray="4 4" transform="scale(2 1)"/>`), options())
	checkPixel(t, p, 48, 8, 8, color.RGBA{G: 255, A: 255})
	checkPixel(t, p, 48, 16, 8, color.RGBA{})
	checkPixel(t, p, 48, 8, 10, color.RGBA{})
}

func TestScratchReuseAndSubimage(t *testing.T) {
	plan, err := opentypesvg.PrepareRender(svg(`<rect width="48" height="48" fill="red"/>`), options())
	if err != nil {
		t.Fatal(err)
	}
	var renderer svgrender.Renderer
	defer renderer.Deallocate()
	dst := ebiten.NewImage(64, 64)
	defer dst.Deallocate()
	sub := dst.SubImage(image.Rect(8, 8, 56, 56)).(*ebiten.Image)
	for range 2 {
		dst.Clear()
		if err := renderer.Draw(sub, plan); err != nil {
			t.Fatal(err)
		}
	}
	pixels := make([]byte, 4*64*64)
	dst.ReadPixels(pixels)
	checkPixel(t, pixels, 64, 0, 0, color.RGBA{})
	checkPixel(t, pixels, 64, 8, 8, color.RGBA{R: 255, A: 255})
	checkPixel(t, pixels, 64, 55, 55, color.RGBA{R: 255, A: 255})
}

func TestResolveAllSamples(t *testing.T) {
	for _, samples := range []int{1, 2, 4, 8} {
		op := options()
		op.Width = 1
		op.Height = 1
		op.Samples = samples
		source := svg(fmt.Sprintf(`<rect width="%g" height="%g" fill="red"/>`, 1/float64(samples), 1/float64(samples)))
		alpha := byte(math.Round(255 / float64(samples*samples)))
		checkPixel(t, render(t, source, op), 1, 0, 0, color.RGBA{R: alpha, A: alpha})
	}
}

func TestGradientAnalyticCases(t *testing.T) {
	tests := []struct {
		name, definition, fill string
		x, y                   int
		want                   color.RGBA
	}{
		{
			name:       "linearRGB",
			definition: `<linearGradient id="g" color-interpolation="linearRGB"><stop stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient>`,
			fill:       "url(#g)",
			x:          24,
			y:          24,
			want:       color.RGBA{R: 186, B: 189, A: 255},
		},
		{
			name:       "singular-fallback",
			definition: `<linearGradient id="g" gradientTransform="scale(0)"><stop stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient>`,
			fill:       "url(#g) lime",
			x:          24,
			y:          24,
			want:       color.RGBA{G: 255, A: 255},
		},
		{
			name:       "duplicate-stops",
			definition: `<linearGradient id="g"><stop stop-color="red"/><stop offset=".5" stop-color="red"/><stop offset=".5" stop-color="blue"/><stop offset="1" stop-color="blue"/></linearGradient>`,
			fill:       "url(#g)",
			x:          24,
			y:          24,
			want:       color.RGBA{B: 255, A: 255},
		},
		{
			name:       "paint-alpha",
			definition: `<linearGradient id="g"><stop stop-color="red" stop-opacity=".5"/><stop offset="1" stop-color="blue" stop-opacity=".5"/></linearGradient>`,
			fill:       "url(#g)",
			x:          24,
			y:          24,
			want:       color.RGBA{R: 63, B: 65, A: 128},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := svg(`<defs>` + tt.definition + `</defs><rect width="48" height="48" fill="` + tt.fill + `"/>`)
			checkPixel(t, render(t, source, options()), 48, tt.x, tt.y, tt.want)
		})
	}
	for _, focus := range []float64{0, 12, 24} {
		t.Run(fmt.Sprintf("radial-focus-%g", focus), func(t *testing.T) {
			source := svg(fmt.Sprintf(`<defs><radialGradient id="g" gradientUnits="userSpaceOnUse" cx="24" cy="24" r="24" fx="%g" fy="24"><stop stop-color="white"/><stop offset="1" stop-color="black"/></radialGradient></defs><rect width="48" height="48" fill="url(#g)"/>`, focus))
			pixels := render(t, source, options())
			for _, pt := range []image.Point{{
				X: 12,
				Y: 12,
			}, {
				X: 24,
				Y: 24,
			}, {
				X: 35,
				Y: 24,
			}} {
				var sum float64
				for y := range 4 {
					for x := range 4 {
						dx, dy := float64(pt.X)+(float64(x)+.5)/4-focus, float64(pt.Y)+(float64(y)+.5)/4-24
						length := math.Hypot(dx, dy)
						ux, uy := dx/length, dy/length
						fx := focus - 24
						distance := -fx*ux + math.Sqrt(24*24-fx*fx*uy*uy)
						v := max(0, min(1, 1-length/distance))
						sum += v * 255
					}
				}
				v := byte(math.Round(sum / 16))
				checkPixel(t, pixels, 48, pt.X, pt.Y, color.RGBA{R: v, G: v, B: v, A: 255})
			}
		})
	}
}

func TestObjectBoundingBoxClipPercentages(t *testing.T) {
	source := `<defs><clipPath id="c" clipPathUnits="objectBoundingBox"><ellipse cx="%s" cy="%s" rx="%s" ry="%s"/></clipPath></defs><g transform="matrix(1 .2 .3 .75 2 4)" clip-path="url(#c)"><rect x="4" y="4" width="32" height="24" fill="lime"/></g>`
	a := render(t, svg(fmt.Sprintf(source, "50%", "50%", "50%", "50%")), options())
	b := render(t, svg(fmt.Sprintf(source, ".5", ".5", ".5", ".5")), options())
	if !bytes.Equal(a, b) {
		t.Error("percentage and normalized clips differ")
	}
}

func TestClipUnionAndNestedIntersection(t *testing.T) {
	body := `<defs><clipPath id="outer"><rect x="0" y="0" width="24" height="48"/></clipPath><clipPath id="inner" clip-path="url(#outer)"><rect x="4" y="4" width="8" height="8"/><rect x="20" y="20" width="16" height="8"/></clipPath></defs><rect width="48" height="48" fill="red" clip-path="url(#inner)"/>`
	pixels := render(t, svg(body), options())
	checkPixel(t, pixels, 48, 8, 8, color.RGBA{R: 255, A: 255})
	checkPixel(t, pixels, 48, 16, 16, color.RGBA{})
	checkPixel(t, pixels, 48, 22, 24, color.RGBA{R: 255, A: 255})
	checkPixel(t, pixels, 48, 28, 24, color.RGBA{})
}

func TestZeroLengthCapsAndDashes(t *testing.T) {
	for _, cap := range []string{"round", "square"} {
		pixels := render(t, svg(fmt.Sprintf(`<path d="M8 8L8 8" stroke="lime" stroke-width="4" stroke-linecap="%s"/>`, cap)), options())
		checkPixel(t, pixels, 48, 8, 8, color.RGBA{G: 255, A: 255})
		pixels = render(t, svg(fmt.Sprintf(`<path d="M8 8H40" stroke="lime" stroke-width="4" stroke-linecap="%s" stroke-dasharray="0 8"/>`, cap)), options())
		checkPixel(t, pixels, 48, 8, 8, color.RGBA{G: 255, A: 255})
		checkPixel(t, pixels, 48, 16, 8, color.RGBA{G: 255, A: 255})
		checkPixel(t, pixels, 48, 12, 8, color.RGBA{})
	}
}

func TestRootViewportAndStyleOnce(t *testing.T) {
	source := []byte(`<svg xmlns="http://www.w3.org/2000/svg" id="glyph1" width="48" height="48" viewBox="0 0 24 24" transform="translate(4)" fill="red" fill-opacity=".5"><rect width="4" height="4"/></svg>`)
	pixels := render(t, source, options())
	checkPixel(t, pixels, 48, 4, 2, color.RGBA{R: 128, A: 128})
	checkPixel(t, pixels, 48, 11, 2, color.RGBA{R: 128, A: 128})
	checkPixel(t, pixels, 48, 12, 2, color.RGBA{})
}

func TestPaletteAndForeground(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "testdata", "focused", "palette.svg"))
	if err != nil {
		t.Fatal(err)
	}
	op := options()
	op.Foreground = color.NRGBA{B: 255, A: 255}
	pixels := render(t, source, op)
	checkPixel(t, pixels, 48, 8, 8, color.RGBA{B: 255, A: 255})
	checkPixel(t, pixels, 48, 24, 8, color.RGBA{R: 255, A: 255})
	op.Palette = []color.NRGBA{{
		G: 255,
		A: 128,
	}}
	pixels = render(t, source, op)
	checkPixel(t, pixels, 48, 24, 8, color.RGBA{G: 128, A: 128})
}

func TestGradientClipIdentity(t *testing.T) {
	defs := `<defs><radialGradient id="g"><stop stop-color="red" stop-opacity=".5"/><stop offset="1" stop-color="blue" stop-opacity=".5"/></radialGradient><clipPath id="c"><circle cx="24" cy="24" r="17"/></clipPath></defs>`
	for _, samples := range []int{1, 4, 8} {
		op := options()
		op.Samples = samples
		op.Transform = [6]float64{1, 0, 0, 1, .375, .375}
		a := render(t, svg(defs+`<circle cx="24" cy="24" r="17" fill="url(#g)"/>`), op)
		b := render(t, svg(defs+`<circle cx="24" cy="24" r="17" fill="url(#g)" clip-path="url(#c)"/>`), op)
		for i, v := range a {
			if delta := int(v) - int(b[i]); delta < -1 || delta > 1 {
				t.Errorf("sample density %d, byte %d: %d != %d", samples, i, v, b[i])
				break
			}
		}
	}
}

func TestRealFontSharedGlyphSelection(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "testdata", "real", "noto-2-3.svg"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		glyph uint16
		gray  uint8
	}{{
		glyph: 2,
		gray:  66,
	}, {
		glyph: 3,
		gray:  224,
	}} {
		op := options()
		op.GlyphID = tt.glyph
		op.UnitsPerEm = 2048
		op.Transform = [6]float64{.02, 0, 0, .02, -5, 15}
		checkPixel(t, render(t, source, op), 48, 8, 8, color.RGBA{R: tt.gray, G: tt.gray, B: tt.gray, A: 255})
	}
}

func TestFillRules(t *testing.T) {
	for _, rule := range []string{"nonzero", "evenodd"} {
		pixels := render(t, svg(fmt.Sprintf(`<path d="M4 4H44V44H4ZM16 16H32V32H16Z" fill="red" fill-rule="%s"/>`, rule)), options())
		checkPixel(t, pixels, 48, 8, 8, color.RGBA{R: 255, A: 255})
		var want color.RGBA
		if rule == "nonzero" {
			want = color.RGBA{R: 255, A: 255}
		}
		checkPixel(t, pixels, 48, 24, 24, want)
	}
}

func TestClosedDashesAndOffset(t *testing.T) {
	path := `<path d="M8 8H40V40H8Z" fill="none" stroke="lime" stroke-width="4" stroke-linejoin="round" stroke-linecap="square" %s/>`
	plain := render(t, svg(fmt.Sprintf(path, "")), options())
	dashed := render(t, svg(fmt.Sprintf(path, `stroke-dasharray="1000 1"`)), options())
	if !bytes.Equal(plain, dashed) {
		t.Error("unbroken closed dash differs from solid stroke")
	}
	negative := render(t, svg(fmt.Sprintf(path, `stroke-dasharray="6 4" stroke-dashoffset="-3"`)), options())
	positive := render(t, svg(fmt.Sprintf(path, `stroke-dasharray="6 4" stroke-dashoffset="7"`)), options())
	if !bytes.Equal(negative, positive) {
		t.Error("equivalent dash phases differ")
	}
}

func TestPointCapStrokeWinding(t *testing.T) {
	for _, cap := range []string{"round", "square"} {
		for _, transform := range []string{"", "translate(48) scale(-1 1)"} {
			t.Run(cap+transform, func(t *testing.T) {
				source := svg(fmt.Sprintf(`<path d="M4 24H44M24 24L24 24" fill="none" stroke="lime" stroke-width="8" stroke-linecap="%s" transform="%s"/>`, cap, transform))
				checkPixel(t, render(t, source, options()), 48, 24, 24, color.RGBA{G: 255, A: 255})
				source = svg(fmt.Sprintf(`<path d="M4 24H44" fill="none" stroke="lime" stroke-width="6" stroke-linecap="%s" stroke-dasharray="0 4 8 4" transform="%s"/>`, cap, transform))
				pixels := render(t, source, options())
				for _, x := range []int{5, 6} {
					if transform != "" {
						x = 47 - x
					}
					checkPixel(t, pixels, 48, x, 24, color.RGBA{G: 255, A: 255})
				}
			})
		}
	}
}

func TestClipPathDisplay(t *testing.T) {
	for _, display := range []string{"", "display:none"} {
		source := svg(`<defs><clipPath id="c" style="` + display + `"><rect width="24" height="48"/><rect x="24" width="24" height="48" display="none"/></clipPath></defs><rect width="48" height="48" fill="red" clip-path="url(#c)"/>`)
		pixels := render(t, source, options())
		checkPixel(t, pixels, 48, 12, 24, color.RGBA{R: 255, A: 255})
		checkPixel(t, pixels, 48, 36, 24, color.RGBA{})
	}
}

func TestStaticAnimationElements(t *testing.T) {
	for _, element := range []string{"animate", "set", "animateMotion", "mpath", "animateColor", "animateTransform"} {
		for _, placement := range []string{"group", "shape", "clip"} {
			t.Run(element+"/"+placement, func(t *testing.T) {
				animation := fmt.Sprintf(`<%s attributeName="fill" to="blue" dur="1s"><unknown/></%s>`, element, element)
				var body string
				switch placement {
				case "group":
					body = `<g>` + animation + `<rect width="10" height="10" fill="red"/></g>`
				case "shape":
					body = `<rect width="10" height="10" fill="red">` + animation + `</rect>`
				case "clip":
					body = `<defs><clipPath id="c">` + animation + `<rect width="10" height="10"/></clipPath></defs><rect width="48" height="48" fill="red" clip-path="url(#c)"/>`
				}
				checkPixel(t, render(t, svg(body), options()), 48, 5, 5, color.RGBA{R: 255, A: 255})
			})
		}
	}
}

func TestSubpathAfterClose(t *testing.T) {
	for _, suffix := range []string{"L40 40", "L8 8", "Z"} {
		t.Run(suffix, func(t *testing.T) {
			path := `<path d="M8 8H40V40H8Z%s" fill="none" stroke="lime" stroke-width="3" stroke-dasharray="5 4" stroke-linecap="round"/>`
			a := render(t, svg(fmt.Sprintf(path, suffix)), options())
			b := render(t, svg(fmt.Sprintf(path, "M8 8"+suffix)), options())
			if !bytes.Equal(a, b) {
				t.Error("implicit and explicit subpath starts differ")
			}
		})
	}
	path := `<path d="M8 8H40V40H8Z%s" fill="none" stroke="lime" stroke-width="8" stroke-linecap="square"/>`
	a := render(t, svg(fmt.Sprintf(path, "L8 8")), options())
	b := render(t, svg(fmt.Sprintf(path, "M8 8L8 8")), options())
	if !bytes.Equal(a, b) {
		t.Error("point caps after close differ")
	}
}

func TestClosedPointCaps(t *testing.T) {
	for _, cap := range []string{"butt", "round", "square"} {
		source := svg(fmt.Sprintf(`<path d="M24 24z" fill="none" stroke="lime" stroke-width="8" stroke-linecap="%s"/>`, cap))
		var want color.RGBA
		if cap != "butt" {
			want = color.RGBA{G: 255, A: 255}
		}
		checkPixel(t, render(t, source, options()), 48, 24, 24, want)
	}
}

func TestDirectedSquareDash(t *testing.T) {
	source := svg(`<path d="M12 12L40 40" fill="none" stroke="lime" stroke-width="10" stroke-linecap="square" stroke-dasharray="0 100"/>`)
	pixels := render(t, source, options())
	checkPixel(t, pixels, 48, 17, 12, color.RGBA{G: 255, A: 255})
	checkPixel(t, pixels, 48, 16, 16, color.RGBA{})
}

func TestNonrenderingChildren(t *testing.T) {
	source := svg(`<rect width="10" height="10" fill="red"><rect x="20" width="10" height="10" fill="blue"/></rect>`)
	pixels := render(t, source, options())
	checkPixel(t, pixels, 48, 5, 5, color.RGBA{R: 255, A: 255})
	checkPixel(t, pixels, 48, 25, 5, color.RGBA{})
	for _, content := range []string{`<g><rect width="24" height="48"/></g>`, `<use href="#gg"/>`, `<use href="#indirect"/>`} {
		source = svg(`<defs><rect id="shape" width="24" height="48"/><use id="indirect" href="#shape"/><g id="gg"><rect width="24" height="48"/></g><clipPath id="c">` + content + `</clipPath></defs><rect width="48" height="48" fill="red" clip-path="url(#c)"/>`)
		checkPixel(t, render(t, source, options()), 48, 12, 24, color.RGBA{})
	}
	source = svg(`<defs><rect id="shape" width="24" height="48"/><clipPath id="c" opacity="0"><use href="#shape" opacity="0"/></clipPath></defs><rect width="48" height="48" fill="red" clip-path="url(#c)"/>`)
	checkPixel(t, render(t, source, options()), 48, 12, 24, color.RGBA{R: 255, A: 255})
}

func TestViewportOverflowFixture(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "testdata", "focused", "viewport-overflow.svg"))
	if err != nil {
		t.Fatal(err)
	}
	op := options()
	op.Transform = [6]float64{1, 0, 0, 1, 16, 16}
	checkPixel(t, render(t, source, op), 48, 13, 10, color.RGBA{R: 255, A: 255})
}

func TestGradientReferenceFixture(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "testdata", "focused", "gradient-reference.svg"))
	if err != nil {
		t.Fatal(err)
	}
	pixels := render(t, source, options())
	checkPixel(t, pixels, 48, 4, 8, color.RGBA{R: 252, B: 3, A: 255})
	checkPixel(t, pixels, 48, 43, 8, color.RGBA{R: 3, B: 252, A: 255})
	center := 4 * (32*48 + 24)
	edge := 4 * (32*48 + 42)
	if pixels[center] < 230 || pixels[center+3] < 245 {
		t.Errorf("radial center=%v", pixels[center:center+4])
	}
	if pixels[edge] >= pixels[center] || pixels[edge+3] >= pixels[center+3] {
		t.Errorf("radial edge=%v center=%v", pixels[edge:edge+4], pixels[center:center+4])
	}
}

func TestZeroOpacityAncestorBounds(t *testing.T) {
	for _, hidden := range []string{
		`<rect width="48" height="48" opacity="0"/>`,
		`<g opacity="0"><rect width="48" height="48"/></g>`,
		`<g opacity="0" filter="url(#f)"><use href="#shape"/></g>`,
	} {
		source := svg(`<defs><filter id="f"/><rect id="shape" width="48" height="48"/><clipPath id="c" clipPathUnits="objectBoundingBox"><rect width=".5" height="1"/></clipPath></defs><g clip-path="url(#c)">` + hidden + `<rect x="8" y="8" width="16" height="16" fill="red"/></g>`)
		pixels := render(t, source, options())
		checkPixel(t, pixels, 48, 12, 12, color.RGBA{R: 255, A: 255})
		checkPixel(t, pixels, 48, 20, 12, color.RGBA{R: 255, A: 255})
		checkPixel(t, pixels, 48, 4, 4, color.RGBA{})
	}
}

func TestBatchedSampleComposition(t *testing.T) {
	const count = 55
	source := svg(strings.Repeat(`<rect width="96" height="96" fill="red" fill-opacity=".5"/>`, count) + `<rect x=".375" width="96" height="96" fill="blue" fill-opacity=".5"/>`)
	op := options()
	op.Width = 96
	op.Height = 96
	op.UnitsPerEm = 96
	op.Samples = 8
	plan, err := opentypesvg.PrepareRender(source, op)
	if err != nil {
		t.Fatal(err)
	}
	var boundaries int
	for _, draw := range plan.Draws {
		if draw.FlushBefore {
			boundaries++
		}
	}
	if boundaries < 3 {
		t.Errorf("batch boundaries=%d,want at least 3", boundaries)
	}
	pixels := render(t, source, op)
	alpha := math.Round(255 * (1 - math.Pow(.5, count)))
	checkPixel(t, pixels, 96, 48, 48, color.RGBA{R: uint8(math.Round(alpha * .5)), B: 128, A: 255})
	checkPixel(t, pixels, 96, 0, 48, color.RGBA{R: uint8(math.Round(alpha * (3.0/8 + 5.0/16))), B: 80, A: 255})
	isolated := bytes.Replace(source, []byte("</svg>"), []byte("</g></svg>"), 1)
	isolated = bytes.Replace(isolated, []byte("<rect"), []byte(`<g opacity=".5"><rect`), 1)
	got := render(t, isolated, op)
	for i, v := range pixels {
		pixels[i] = byte((int(v) + 1) / 2)
	}
	compareChannels(t, got, pixels, 1)
}

func TestFlushSamplesUsage(t *testing.T) {
	samples := ebiten.NewImage(8, 8)
	defer samples.Deallocate()
	coverage := ebiten.NewImage(8, 8)
	defer coverage.Deallocate()
	coverage.Fill(color.RGBA{B: 255, A: 255})
	var path vector.Path
	path.MoveTo(0, 0)
	path.LineTo(8, 0)
	path.LineTo(8, 8)
	path.LineTo(0, 8)
	path.Close()
	var options vector.DrawPathOptions
	options.ColorScale.Scale(1, 0, 0, 1)
	vector.FillPath(samples, &path, nil, &options)
	var invoked bool
	bridge := imagebridge.Get[*ebiten.Image]()
	token := bridge.AddUsage(samples, func(*ebiten.Image) { invoked = true })
	defer bridge.RemoveUsage(samples, token)
	svgrender.FlushSamples(samples, coverage)
	if !invoked {
		t.Error("flush did not invoke sample-image usage callbacks")
	}
	pixels := make([]byte, 8*8*4)
	samples.ReadPixels(pixels)
	checkPixel(t, pixels, 8, 4, 4, color.RGBA{R: 255, A: 255})
}

func TestEmptyPlanPreservesDestination(t *testing.T) {
	plan, err := opentypesvg.PrepareRender(svg(""), options())
	if err != nil {
		t.Fatal(err)
	}
	dst := ebiten.NewImage(48, 48)
	defer dst.Deallocate()
	dst.Fill(color.RGBA{R: 255, A: 255})
	var renderer svgrender.Renderer
	defer renderer.Deallocate()
	if err := renderer.Draw(dst, plan); err != nil {
		t.Fatal(err)
	}
	pixels := make([]byte, 4*48*48)
	dst.ReadPixels(pixels)
	checkPixel(t, pixels, 48, 24, 24, color.RGBA{R: 255, A: 255})
}

func TestGradientPaintReuse(t *testing.T) {
	source := svg(`<defs><linearGradient id="red"><stop stop-color="red" stop-opacity=".5"/><stop offset="1" stop-color="red" stop-opacity=".5"/></linearGradient><linearGradient id="blue"><stop stop-color="blue" stop-opacity=".5"/><stop offset="1" stop-color="blue" stop-opacity=".5"/></linearGradient></defs><rect x="4" y="4" width="32" height="32" fill="url(#red)"/><rect x="12" y="12" width="32" height="32" fill="url(#blue)"/><rect x="40" width="8" height="8" fill="lime"/>`)
	pixels := render(t, source, options())
	checkPixel(t, pixels, 48, 8, 8, color.RGBA{R: 128, A: 128})
	checkPixel(t, pixels, 48, 24, 24, color.RGBA{R: 64, B: 128, A: 192})
	checkPixel(t, pixels, 48, 40, 40, color.RGBA{B: 128, A: 128})
	checkPixel(t, pixels, 48, 44, 4, color.RGBA{G: 255, A: 255})
	checkPixel(t, pixels, 48, 0, 0, color.RGBA{})
}

func TestGradientPaintDestinationSubimage(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "testdata", "focused", "gradient-reference.svg"))
	if err != nil {
		t.Fatal(err)
	}
	op := options()
	op.Transform = [6]float64{1, 0, 0, 1, .375, .375}
	expected := render(t, source, op)
	plan, err := opentypesvg.PrepareRender(source, op)
	if err != nil {
		t.Fatal(err)
	}
	parent := ebiten.NewImage(64, 64)
	defer parent.Deallocate()
	destination := parent.SubImage(image.Rect(8, 10, 56, 58)).(*ebiten.Image)
	var renderer svgrender.Renderer
	defer renderer.Deallocate()
	for range 2 {
		parent.Clear()
		if err := renderer.Draw(destination, plan); err != nil {
			t.Fatal(err)
		}
	}
	pixels := make([]byte, 4*48*48)
	destination.ReadPixels(pixels)
	if !bytes.Equal(pixels, expected) {
		t.Error("gradient painting into a destination subimage changed pixels")
	}
}

func TestGradientPaintTransformedStroke(t *testing.T) {
	source := svg(`<defs><linearGradient id="g" gradientUnits="userSpaceOnUse" x2="0" y2="24"><stop stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient></defs><path d="M2 16H22" fill="none" stroke="url(#g)" stroke-width="4" stroke-linecap="round" transform="scale(2 1)"/>`)
	pixels := render(t, source, options())
	checkPixel(t, pixels, 48, 24, 16, color.RGBA{R: 80, B: 175, A: 255})
	checkPixel(t, pixels, 48, 24, 17, color.RGBA{R: 69, B: 186, A: 255})
	checkPixel(t, pixels, 48, 24, 19, color.RGBA{})
}

func TestRendererPaintAllocationLifecycle(t *testing.T) {
	solid := svg(`<rect width="48" height="48" fill="red"/>`)
	gradient := svg(`<defs><linearGradient id="g"><stop stop-color="red"/><stop offset="1" stop-color="blue"/></linearGradient></defs><rect width="48" height="48" fill="url(#g)"/>`)
	var renderer svgrender.Renderer
	defer renderer.Deallocate()
	for _, tt := range []struct {
		name    string
		source  []byte
		size    int
		release bool
	}{
		{
			name:   "solid",
			source: solid,
			size:   48,
		},
		{
			name:   "gradient-same-size",
			source: gradient,
			size:   48,
		},
		{
			name:   "gradient-resized",
			source: gradient,
			size:   96,
		},
		{
			name:    "gradient-after-deallocate",
			source:  gradient,
			size:    48,
			release: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.release {
				renderer.Deallocate()
			}
			op := options()
			op.Width = tt.size
			op.Height = tt.size
			scale := float64(tt.size) / 48
			op.Transform = [6]float64{scale, 0, 0, scale, 0, 0}
			expected := render(t, tt.source, op)
			plan, err := opentypesvg.PrepareRender(tt.source, op)
			if err != nil {
				t.Fatal(err)
			}
			dst := ebiten.NewImage(tt.size, tt.size)
			defer dst.Deallocate()
			if err := renderer.Draw(dst, plan); err != nil {
				t.Fatal(err)
			}
			pixels := make([]byte, 4*tt.size*tt.size)
			dst.ReadPixels(pixels)
			if !bytes.Equal(pixels, expected) {
				t.Error("reused renderer differs from fresh renderer")
			}
		})
	}
}
