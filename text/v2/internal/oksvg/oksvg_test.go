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

package oksvg_test

import (
	"fmt"
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/srwiley/rasterx"

	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/oksvg"
)

// Issue #2649
func TestDrawTransformedUserSpaceGradient(t *testing.T) {
	const svg = `<svg xmlns="http://www.w3.org/2000/svg">
<defs><radialGradient id="g" cx="500" cy="500" r="500" gradientUnits="userSpaceOnUse">
<stop offset="0" stop-color="#FFFFFF"/><stop offset="1" stop-color="#000000"/>
</radialGradient></defs>
<rect x="0" y="0" width="1000" height="1000" fill="url(#g)"/>
</svg>`

	icon, err := oksvg.ReadIconStream(strings.NewReader(svg))
	if err != nil {
		t.Fatal(err)
	}

	const w, h = 100, 100
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	scanner := rasterx.NewScannerGV(w, h, img, img.Bounds())
	dasher := rasterx.NewDasher(w, h, scanner)
	icon.Transform = rasterx.Matrix2D{A: 0.1, D: 0.1}
	icon.Draw(dasher, 1)

	// The gradient center (500, 500) in user space maps to (50, 50).
	if r, _, _, _ := img.At(50, 50).RGBA(); r>>8 < 0xc0 {
		t.Errorf("center pixel is not near-white: got R=0x%02x, want >= 0xc0", r>>8)
	}
	// (950, 500) in user space maps to (95, 50), near the gradient edge.
	if r, _, _, _ := img.At(95, 50).RGBA(); r>>8 >= 0x40 {
		t.Errorf("edge pixel is not near-black: got R=0x%02x, want < 0x40", r>>8)
	}
}

// Issue #3662
func TestScaleWithOneArgument(t *testing.T) {
	const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100">
<g transform="scale(0.5)"><rect x="0" y="0" width="100" height="100" fill="#FF0000"/></g>
</svg>`

	icon, err := oksvg.ReadIconStream(strings.NewReader(svg))
	if err != nil {
		t.Fatal(err)
	}

	const w, h = 100, 100
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	scanner := rasterx.NewScannerGV(w, h, img, img.Bounds())
	dasher := rasterx.NewDasher(w, h, scanner)
	icon.Draw(dasher, 1)

	// The rectangle covers (0, 0)-(50, 50) after the scale.
	for _, tc := range []struct {
		x, y  int
		alpha bool
	}{
		{x: 10, y: 10, alpha: true},
		{x: 60, y: 10, alpha: false},
		{x: 10, y: 60, alpha: false},
		{x: 60, y: 60, alpha: false},
	} {
		_, _, _, a := img.At(tc.x, tc.y).RGBA()
		if got, want := a > 0, tc.alpha; got != want {
			t.Errorf("pixel at (%d, %d): opaque = %t, want %t", tc.x, tc.y, got, want)
		}
	}
}

// Issue #3659
func TestReadIconStreamCyclicUse(t *testing.T) {
	testCases := []struct {
		name string
		svg  string
	}{
		{
			name: "self",
			svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">
<defs><g id="a"><use href="#a"/></g></defs>
<use href="#a"/>
</svg>`,
		},
		{
			name: "mutual",
			svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">
<defs><g id="a"><use href="#b"/></g><g id="b"><use href="#a"/></g></defs>
<use href="#a"/>
</svg>`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			icon, err := oksvg.ReadIconStream(strings.NewReader(tc.svg))
			if err != nil {
				t.Fatal(err)
			}
			if got, want := len(icon.SVGPaths), 0; got != want {
				t.Errorf("len(icon.SVGPaths): got: %d, want: %d", got, want)
			}
		})
	}
}

// Issue #3659
func TestReadIconStreamNestedUse(t *testing.T) {
	var svg strings.Builder
	svg.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><defs>`)
	svg.WriteString(`<g id="g0"><path d="M0 0 L10 0 L10 10 Z"/></g>`)
	// Each definition uses the previous one twice, so #g20 covers 2^20 paths.
	const n = 20
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&svg, `<g id="g%d"><use href="#g%d"/><use href="#g%d"/></g>`, i, i-1, i-1)
	}
	fmt.Fprintf(&svg, `</defs><use href="#g%d"/></svg>`, n)

	icon, err := oksvg.ReadIconStream(strings.NewReader(svg.String()))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(icon.SVGPaths), 1<<n; got >= want {
		t.Errorf("len(icon.SVGPaths): got: %d, want: < %d", got, want)
	}
}

func TestReadIconStreamUse(t *testing.T) {
	const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10">
<defs><g id="a"><path d="M0 0 L10 0 L10 10 Z"/></g></defs>
<use href="#a"/>
<use href="#a" x="1" y="2"/>
</svg>`

	icon, err := oksvg.ReadIconStream(strings.NewReader(svg))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(icon.SVGPaths), 2; got != want {
		t.Errorf("len(icon.SVGPaths): got: %d, want: %d", got, want)
	}
}

func TestParseSVGColor(t *testing.T) {
	for _, tc := range []struct {
		colorStr string
		want     color.Color
	}{
		{colorStr: "hsl(0,100%,50%)", want: color.NRGBA{0xff, 0, 0, 0xff}},
		{colorStr: "hsl(120, 100%, 50%)", want: color.NRGBA{0, 0xff, 0, 0xff}},
		{colorStr: "rgb(50%,0,0)", want: color.NRGBA{0x7f, 0, 0, 0xff}},
	} {
		got, err := oksvg.ParseSVGColor(tc.colorStr)
		if err != nil {
			t.Errorf("ParseSVGColor(%q): %v", tc.colorStr, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseSVGColor(%q): got: %v, want: %v", tc.colorStr, got, tc.want)
		}
	}
}

// Issue #3665
func TestParseSVGColorEmptyComponent(t *testing.T) {
	// A color with an empty component must be rejected instead of crashing.
	for _, colorStr := range []string{
		"hsl(,,)",
		"hsl(0,,)",
		"hsl(0,50%,)",
		"hsl(0,,50%)",
		"rgb(,,)",
		"rgb(0,,0)",
		"rgb(0,0,)",
	} {
		if _, err := oksvg.ParseSVGColor(colorStr); err == nil {
			t.Errorf("ParseSVGColor(%q) must return an error but did not", colorStr)
		}
	}
}
