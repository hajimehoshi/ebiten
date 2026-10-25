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

package text_test

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func TestSVGProductionReference(t *testing.T) {
	executable, err := exec.LookPath("rsvg-convert")
	if err != nil {
		t.Skip("independent SVG comparison requires rsvg-convert")
	}
	for _, record := range []struct {
		name string
		gid  string
	}{
		{
			name: "noto-3330-3330",
			gid:  "glyph3330",
		},
		{
			name: "twemoji-839-839",
			gid:  "glyph839",
		},
	} {
		source, err := os.ReadFile(filepath.Join("internal", "opentypesvg", "testdata", "real", record.name+".svg"))
		if err != nil {
			t.Fatal(err)
		}
		// These standalone records have no root viewBox or inherited root paint.
		// Both fonts and the synthetic host use 2048 units per em.
		body := string(source)
		body = body[strings.IndexByte(body, '>')+1 : strings.LastIndex(body, "</svg>")]
		for _, size := range []float64{24, 96} {
			t.Run(fmt.Sprintf("%s/%g", record.name, size), func(t *testing.T) {
				face := &text.GoTextFace{
					Source: svgTestFont(t, strings.ReplaceAll(string(source), record.gid, "GLYPHA"), false),
					Size:   size,
				}
				input := "A"
				wantGlyphs := 1
				if size == 24 {
					// The preceding space supplies a fractional glyph origin through layout.
					input = " A"
					wantGlyphs = 2
				}
				glyphs := text.AppendGlyphs(nil, input, face, nil)
				if len(glyphs) != wantGlyphs {
					t.Fatalf("got %d glyphs, want %d", len(glyphs), wantGlyphs)
				}
				g := glyphs[len(glyphs)-1]
				if g.Image == nil || !g.Colored {
					t.Error("SVG was empty or fell back to an outline")
					return
				}
				const width, height = 128, 128
				// Fix the font origin independently of the renderer's calculated ink bounds.
				fraction := text.SVGAcceptanceOffset(face, g.OriginX+g.OriginOffsetX)
				if size == 24 && fraction == 0 {
					t.Fatal("small-size fixture requires a fractional glyph origin")
				}
				originX := 8 + math.Floor(g.OriginX+g.OriginOffsetX) + fraction
				originY := 8 + math.Floor(g.OriginY+g.OriginOffsetY)
				actual := ebiten.NewImage(width, height)
				defer actual.Deallocate()
				var op text.DrawOptions
				op.GeoM.Translate(8, 8)
				text.Draw(actual, input, face, &op)
				got := make([]byte, 4*width*height)
				actual.ReadPixels(got)
				public := ebiten.NewImage(width, height)
				defer public.Deallocate()
				var imageOp ebiten.DrawImageOptions
				imageOp.GeoM.Translate(g.X, g.Y)
				imageOp.GeoM.Concat(op.GeoM)
				public.DrawImage(g.Image, &imageOp)
				publicPixels := make([]byte, len(got))
				public.ReadPixels(publicPixels)
				if !bytes.Equal(got, publicPixels) {
					t.Error("Draw differs from public glyph image composition")
				}
				reference := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="%d" height="%d"><g transform="translate(%g %g) scale(%g)">%s</g></svg>`, width, height, originX, originY, size/2048, body)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				const referenceSamples = 8
				cmd := exec.CommandContext(ctx, executable, "-w", fmt.Sprint(width*referenceSamples), "-h", fmt.Sprint(height*referenceSamples))
				cmd.Stdin = strings.NewReader(reference)
				encoded, err := cmd.Output()
				if err != nil {
					t.Fatal(err)
				}
				decoded, err := png.Decode(bytes.NewReader(encoded))
				if err != nil {
					t.Fatal(err)
				}
				if decoded.Bounds() != image.Rect(0, 0, width*referenceSamples, height*referenceSamples) {
					t.Fatal("unexpected reference dimensions")
				}
				want := image.NewRGBA(image.Rect(0, 0, width, height))
				for y := range height {
					for x := range width {
						var sum [4]uint64
						for sy := range referenceSamples {
							for sx := range referenceSamples {
								r, g, b, a := decoded.At(x*referenceSamples+sx, y*referenceSamples+sy).RGBA()
								sum[0] += uint64(r)
								sum[1] += uint64(g)
								sum[2] += uint64(b)
								sum[3] += uint64(a)
							}
						}
						for c, value := range sum {
							want.Pix[4*(y*width+x)+c] = byte((value + referenceSamples*referenceSamples*257/2) / (referenceSamples * referenceSamples * 257))
						}
					}
				}
				var total, channels, maximum int
				for i := 0; i < len(got); i += 4 {
					if got[i+3] == 0 && want.Pix[i+3] == 0 {
						continue
					}
					for c := range 4 {
						delta := int(got[i+c]) - int(want.Pix[i+c])
						if delta < 0 {
							delta = -delta
						}
						total += delta
						maximum = max(maximum, delta)
						channels++
					}
				}
				if channels == 0 {
					t.Error("both images are empty")
					return
				}
				mean := float64(total) / float64(channels)
				t.Logf("native-AA versus librsvg: occupied RGBA mean %.3f, maximum %d", mean, maximum)
				// Different coverage algorithms need a tolerance at boundary pixels.
				// These single-paint records avoid coincident overpainting and group seams.
				if mean > 8 {
					t.Errorf("occupied mean channel error %.3f exceeds 8", mean)
				}
				if maximum > 64 {
					t.Errorf("maximum channel error %d exceeds 64", maximum)
				}
				if dir := os.Getenv("EBITENGINE_SVG_RENDER_REPORT_DIR"); dir != "" {
					for label, pixels := range map[string][]byte{"production": got, "reference": want.Pix} {
						path := filepath.Join(dir, fmt.Sprintf("production-%s-%g-%s.png", record.name, size, label))
						var output bytes.Buffer
						if err := png.Encode(&output, &image.RGBA{
							Pix:    pixels,
							Stride: 4 * width,
							Rect:   want.Bounds(),
						}); err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(path, output.Bytes(), 0644); err != nil {
							t.Fatal(err)
						}
					}
				}
			})
		}
	}
}
