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
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func referencePixels(t *testing.T, source []byte, size int) []byte {
	t.Helper()
	executable, err := exec.LookPath("rsvg-convert")
	if err != nil {
		t.Skip("independent comparison requires rsvg-convert")
	}
	const scale = 8
	cmd := exec.Command(executable, "-w", fmt.Sprint(size*scale), "-h", fmt.Sprint(size*scale))
	cmd.Stdin = bytes.NewReader(source)
	output, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(output))
	if err != nil {
		t.Fatal(err)
	}
	pixels := make([]byte, 4*size*size)
	for y := range size {
		for x := range size {
			var sum [4]uint64
			for sy := range scale {
				for sx := range scale {
					r, g, b, a := img.At(x*scale+sx, y*scale+sy).RGBA()
					sum[0] += uint64(r)
					sum[1] += uint64(g)
					sum[2] += uint64(b)
					sum[3] += uint64(a)
				}
			}
			for c, v := range sum {
				pixels[4*(y*size+x)+c] = byte((v + scale*scale*257/2) / (scale * scale * 257))
			}
		}
	}
	return pixels
}

func TestIndependentReference(t *testing.T) {
	sources := map[string][]byte{
		"isolated-stroke": svg(`<g opacity=".5"><rect x="8" y="8" width="24" height="24" fill="red" stroke="blue" stroke-width="8" opacity=".5"/><circle cx="32" cy="32" r="10" fill="lime"/></g>`),
		"joins":           svg(`<path d="M4 30L12 6L20 30" fill="none" stroke="red" stroke-width="4" stroke-linejoin="miter" stroke-miterlimit="8"/><path d="M24 30L32 6L40 30" fill="none" stroke="blue" stroke-width="4" stroke-linejoin="miter" stroke-miterlimit="1" stroke-linecap="square"/>`),
		"curves-stroke":   svg(`<path d="M3 35C3 2 35 3 40 22Q25 38 8 12" fill="none" stroke="#40A080" stroke-width="2" transform="matrix(1 .1 .2 .75 -2 2)" stroke-linejoin="round" stroke-linecap="round"/>`),
		"elliptical-arc":  svg(`<path d="M8 8A14 8 35 1 1 40 35L30 10Z" fill="red" stroke="blue" stroke-width="1"/>`),
		"dash":            svg(`<path d="M3 35C3 2 35 3 40 22" fill="none" stroke="blue" stroke-width="2" stroke-dasharray="3 2 1" stroke-linecap="round"/>`),
		"focal":           svg(`<defs><radialGradient id="g" fx="15%" fy="30%"><stop stop-color="white"/><stop offset="1" stop-color="blue" stop-opacity=".3"/></radialGradient></defs><circle cx="24" cy="24" r="20" fill="url(#g)"/>`),
		"box-clip":        svg(`<defs><clipPath id="c" clipPathUnits="objectBoundingBox"><circle cx=".5" cy=".5" r=".5"/></clipPath></defs><g clip-path="url(#c)" transform="translate(2 4)"><rect x="4" y="4" width="32" height="24" fill="lime"/></g>`),
	}
	for _, name := range []string{"group-identity", "nested-groups", "embedded-jpeg"} {
		source, err := os.ReadFile(filepath.Join("..", "testdata", "focused", name+".svg"))
		if err != nil {
			t.Fatal(err)
		}
		sources[name] = source
	}
	for _, name := range []string{"noto-3330-3330", "noto-3682-3682", "twemoji-1338-1338", "twemoji-1355-1355"} {
		source, err := os.ReadFile(filepath.Join("..", "testdata", "real", name+".svg"))
		if err != nil {
			t.Fatal(err)
		}
		sources[name] = source
	}
	for name, source := range sources {
		for _, size := range []int{24, 96} {
			if size == 24 && name != "joins" && name != "noto-3330-3330" {
				continue
			}
			var offset float64
			if size == 96 {
				offset = .375
			}
			t.Run(fmt.Sprintf("%s/%d/%g", name, size, offset), func(t *testing.T) {
				op := options()
				op.Samples = 8
				op.Width = size
				op.Height = size
				scale := float64(size) / 48
				real := strings.HasPrefix(name, "noto-") || strings.HasPrefix(name, "twemoji-")
				var tx, ty float64
				if real {
					op.UnitsPerEm = 2048
					if _, err := fmt.Sscanf(strings.Split(name, "-")[1], "%d", &op.GlyphID); err != nil {
						t.Fatal(err)
					}
					scale = float64(size) / 2300
					tx = float64(size) * .04
					ty = float64(size) * .77
				}
				tx += offset
				ty += offset
				op.Transform = [6]float64{scale, 0, 0, scale, tx, ty}
				actual := render(t, source, op)
				text := string(source)
				text = text[strings.Index(text, "<svg"):]
				body := text[strings.IndexByte(text, '>')+1 : strings.LastIndex(text, "</svg>")]
				reference := []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="%d" height="%d"><g transform="matrix(%g 0 0 %g %g %g)">%s</g></svg>`, size, size, scale, scale, tx, ty, body))
				expected := referencePixels(t, reference, size)
				var sum, count, maximum, localErrors int
				for i := 0; i < len(actual); i += 4 {
					if actual[i+3] == 0 && expected[i+3] == 0 {
						continue
					}
					var pixelMaximum int
					for c := range 4 {
						delta := int(math.Abs(float64(int(actual[i+c]) - int(expected[i+c]))))
						sum += delta
						pixelMaximum = max(pixelMaximum, delta)
						maximum = max(maximum, delta)
						count++
					}
					if pixelMaximum > 32 {
						localErrors++
					}
				}
				mean := float64(sum) / float64(max(1, count))
				t.Logf("occupied premultiplied RGBA mean error %.3f; max %d; pixels above 32: %d", mean, maximum, localErrors)
				if localErrors > 0 {
					t.Errorf("%d pixels exceed local channel-error limit 32", localErrors)
				}
				if mean > 5 {
					t.Errorf("reference mean error %.3f exceeds 5", mean)
				}
				if dir := os.Getenv("SVG_RENDER_REPORT_DIR"); dir != "" {
					for label, pixels := range map[string][]byte{"actual": actual, "reference": expected} {
						file, err := os.Create(filepath.Join(dir, fmt.Sprintf("%s-%d-%g-%s.png", name, size, offset, label)))
						if err != nil {
							t.Fatal(err)
						}
						err = png.Encode(file, &image.RGBA{Pix: pixels, Stride: 4 * size, Rect: image.Rect(0, 0, size, size)})
						closeErr := file.Close()
						if err != nil {
							t.Fatal(err)
						}
						if closeErr != nil {
							t.Fatal(closeErr)
						}
					}
				}
			})
		}
	}
}
