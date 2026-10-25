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
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/opentypesvg"
	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/opentypesvg/svgrender"
)

func TestIsolatedComposition(t *testing.T) {
	for _, samples := range []int{1, 4, 8} {
		for _, offset := range []float64{0, .375} {
			t.Run(fmt.Sprintf("%d/%g", samples, offset), func(t *testing.T) {
				op := options()
				op.Samples = samples
				op.Transform = [6]float64{1, 0, 0, 1, offset, offset}
				defs := `<defs><circle id="a" cx="24" cy="24" r="17"/><clipPath id="c"><use href="#a"/></clipPath><clipPath id="d"><rect width="30" height="48"/></clipPath></defs>`
				source := svg(defs + `<g clip-path="url(#d)" opacity=".5"><g clip-path="url(#c)" opacity=".5"><use href="#a" fill="red"/><circle cx="32" cy="24" r="17" fill="blue"/></g></g>`)
				got := render(t, source, op)
				// Opaque children choose the last covering paint at each point.
				// Render the same clipped ordering at native sample resolution,
				// apply the two opacity factors there, then average all samples.
				sampleOptions := op
				sampleOptions.Width *= samples
				sampleOptions.Height *= samples
				sampleOptions.Samples = 1
				sampleOptions.Transform = [6]float64{float64(samples), 0, 0, float64(samples), offset * float64(samples), offset * float64(samples)}
				reference := render(t, svg(defs+`<g clip-path="url(#d)"><g clip-path="url(#c)"><use href="#a" fill="red"/><circle cx="32" cy="24" r="17" fill="blue"/></g></g>`), sampleOptions)
				for i, v := range reference {
					reference[i] = byte((int(v) + 2) / 4)
				}
				want := averageSamples(reference, 48, samples)
				compareChannels(t, got, want, 1)

				identity, err := os.ReadFile("../testdata/focused/group-identity.svg")
				if err != nil {
					t.Fatal(err)
				}
				got = render(t, identity, op)
				want = render(t, svg(`<circle cx="24" cy="24" r="17" fill="blue" opacity=".5"/>`), op)
				compareChannels(t, got, want, 1)
			})
		}
	}
}

func TestIsolatedFillAndStroke(t *testing.T) {
	pixels := render(t, svg(`<rect x="8" y="8" width="24" height="24" fill="red" stroke="blue" stroke-width="8" opacity=".5"/>`), options())
	checkPixel(t, pixels, 48, 10, 16, color.RGBA{B: 128, A: 128})
	checkPixel(t, pixels, 48, 20, 20, color.RGBA{R: 128, A: 128})
}

func averageSamples(src []byte, size, samples int) []byte {
	dst := make([]byte, 4*size*size)
	for y := range size {
		for x := range size {
			for c := range 4 {
				var sum int
				for sy := range samples {
					for sx := range samples {
						sum += int(src[4*((y*samples+sy)*size*samples+x*samples+sx)+c])
					}
				}
				dst[4*(y*size+x)+c] = byte((sum + samples*samples/2) / (samples * samples))
			}
		}
	}
	return dst
}

func compareChannels(t *testing.T, got, want []byte, tolerance int) {
	t.Helper()
	var largest int
	for i, v := range got {
		d := int(v) - int(want[i])
		largest = max(largest, d, -d)
	}
	if largest > tolerance {
		t.Errorf("maximum channel error=%d, want <=%d", largest, tolerance)
	}
}

func TestEmbeddedImageMapping(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 128})
	img.SetNRGBA(1, 0, color.NRGBA{B: 255, A: 255})
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	href := "data:image/png;base64," + base64.StdEncoding.EncodeToString(encoded.Bytes())
	for _, aspect := range []string{"xMidYMid meet", "xMaxYMax slice", "none"} {
		t.Run(aspect, func(t *testing.T) {
			body := `<defs><clipPath id="c"><rect x="4" y="4" width="20" height="24"/></clipPath><image id="i" x="4" y="4" width="50%" height="50%" preserveAspectRatio="` + aspect + `" href="` + href + `"/></defs><g opacity=".5" transform="matrix(.8 .2 -.1 1 3.314 2.231)" clip-path="url(#c)"><use href="#i"/></g>`
			got := render(t, svg(body), options())
			// Geometry supplies binary viewport/clip membership. Image texel
			// selection uses the inverse affine mapping at each sample center.
			op := options()
			op.Width, op.Height, op.Samples = 192, 192, 1
			op.Transform = [6]float64{4, 0, 0, 4, 0, 0}
			mask := render(t, svg(`<defs><clipPath id="c"><rect x="4" y="4" width="20" height="24"/></clipPath></defs><g transform="matrix(.8 .2 -.1 1 3.314 2.231)" clip-path="url(#c)"><rect x="4" y="4" width="24" height="24" fill="white"/></g>`), op)
			for y := range 192 {
				for x := range 192 {
					i := 4 * (y*192 + x)
					covered := mask[i+3] != 0
					clear(mask[i : i+4])
					if !covered {
						continue
					}
					dx, dy := (float64(x)+.5)/4-3.314, (float64(y)+.5)/4-2.231
					localX, localY := (dx+.1*dy)/.82, (-.2*dx+.8*dy)/.82
					u, v := (localX-4)/12, (localY-4)/24
					if aspect == "xMidYMid meet" {
						v = (localY - 10) / 12
					}
					if aspect == "xMaxYMax slice" {
						u = 1 + (localX-4)/24
					}
					if u < 0 || u >= 2 || v < 0 || v >= 1 {
						continue
					}
					if u < 1 {
						mask[i], mask[i+3] = 64, 64
					} else {
						mask[i+2], mask[i+3] = 128, 128
					}
				}
			}
			want := averageSamples(mask, 48, 4)
			compareChannels(t, got, want, 1)
		})
	}
}

func TestCompositionFixturesAndReuse(t *testing.T) {
	var renderer svgrender.Renderer
	defer renderer.Deallocate()
	dst := ebiten.NewImage(56, 56)
	defer dst.Deallocate()
	view := dst.SubImage(image.Rect(4, 4, 52, 52)).(*ebiten.Image)
	for _, name := range []string{"embedded-png", "nested-groups", "embedded-jpeg", "group-identity", "embedded-png"} {
		source, err := os.ReadFile("../testdata/focused/" + name + ".svg")
		if err != nil {
			t.Fatal(err)
		}
		plan, err := opentypesvg.PrepareRender(source, options())
		if err != nil {
			t.Fatal(err)
		}
		view.Clear()
		if err := renderer.Draw(view, plan); err != nil {
			t.Fatal(err)
		}
		pixels := make([]byte, 48*48*4)
		view.ReadPixels(pixels)
		switch name {
		case "embedded-png":
			checkPixel(t, pixels, 48, 8, 12, color.RGBA{R: 255, A: 255})
			checkPixel(t, pixels, 48, 30, 12, color.RGBA{B: 255, A: 255})
		case "nested-groups":
			checkPixel(t, pixels, 48, 10, 24, color.RGBA{R: 64, A: 64})
			checkPixel(t, pixels, 48, 24, 24, color.RGBA{B: 64, A: 64})
			checkPixel(t, pixels, 48, 35, 24, color.RGBA{})
		case "group-identity":
			checkPixel(t, pixels, 48, 24, 24, color.RGBA{B: 128, A: 128})
		case "embedded-jpeg":
			checkPixel(t, pixels, 48, 12, 12, color.RGBA{R: 80, G: 160, B: 240, A: 255})
		}
	}
}

func TestInheritedPaintOpacityInGroup(t *testing.T) {
	source := svg(`<defs><linearGradient id="g"><stop stop-color="red"/><stop offset="1" stop-color="red"/></linearGradient><rect id="r" x="8" y="8" width="24" height="24"/></defs><g opacity=".5" fill-opacity=".5"><use href="#r" fill="url(#g)"/><rect x="16" y="16" width="24" height="24" fill="blue"/></g>`)
	pixels := render(t, source, options())
	checkPixel(t, pixels, 48, 12, 12, color.RGBA{R: 64, A: 64})
	checkPixel(t, pixels, 48, 20, 20, color.RGBA{R: 32, B: 64, A: 96})
}
