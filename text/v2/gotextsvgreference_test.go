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
	"crypto/sha256"
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

func svgReferenceRenderer(t *testing.T) string {
	t.Helper()
	executable, err := exec.LookPath("rsvg-convert")
	if err != nil {
		t.Skip("independent SVG comparison requires rsvg-convert")
	}
	return executable
}

func svgReferenceBody(t *testing.T, name string) string {
	t.Helper()
	source, err := os.ReadFile(filepath.Join("internal", "opentypesvg", "testdata", "real", name+".svg"))
	if err != nil {
		t.Fatal(err)
	}
	// These fixtures have no root viewBox or inherited root paint.
	value := string(source)
	return value[strings.IndexByte(value, '>')+1 : strings.LastIndex(value, "</svg>")]
}

func TestSVGProductionReference(t *testing.T) {
	executable := svgReferenceRenderer(t)
	for _, record := range []struct{ name, gid string }{
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
		body := svgReferenceBody(t, record.name)
		for _, size := range []float64{24, 96} {
			t.Run(fmt.Sprintf("%s/%g", record.name, size), func(t *testing.T) {
				face := &text.GoTextFace{
					Source: svgTestFont(t, strings.ReplaceAll(string(source), record.gid, "GLYPHA"), false),
					Size:   size,
				}
				input := "A"
				if size == 24 {
					input = " A"
				}
				// The synthetic host uses 2048 units per em.
				compareSVGTextReference(t, executable, face, input, body, 2048, 64)
			})
		}
	}
}

func TestSVGRealFontReference(t *testing.T) {
	path := os.Getenv("EBITENGINE_SVG_ACCEPTANCE_FONT")
	if path == "" {
		t.Skip("set EBITENGINE_SVG_ACCEPTANCE_FONT to a pinned Noto or Twemoji font")
	}
	executable := svgReferenceRenderer(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var name, input string
	var unitsPerEm float64
	fontSHA256 := fmt.Sprintf("%x", sha256.Sum256(data))
	switch fontSHA256 {
	case "7fb39738ab18f10612d6f4595e2e8e47a0afdf34738460442d99cd0c344a4d90":
		name, input, unitsPerEm = "noto-3682-3682", " 🔥", 1024
	case "42d4b2a827c9ed557dff88c2f4723cb93985003a81dee76797324d66d04a57b8":
		name, input, unitsPerEm = "twemoji-1355-1355", " 🪟", 2048
	default:
		t.Fatalf("font SHA-256 %s does not match a pinned reference version", fontSHA256)
	}
	src, err := text.NewGoTextFaceSource(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	body := svgReferenceBody(t, name)
	for _, size := range []float64{24, 96} {
		t.Run(fmt.Sprintf("%s/%g", name, size), func(t *testing.T) {
			face := &text.GoTextFace{
				Source: src,
				Size:   size,
			}
			compareSVGTextReference(t, executable, face, input, body, unitsPerEm, 96)
		})
	}
}

func compareSVGTextReference(t *testing.T, executable string, face *text.GoTextFace, input, body string, unitsPerEm float64, maxChannelError int) {
	t.Helper()
	wantGlyphs := 1
	if strings.HasPrefix(input, " ") {
		wantGlyphs++
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
	const width, height = 160, 160
	// Fix the font origin independently of the renderer's calculated ink bounds.
	fraction := text.SVGAcceptanceOffset(face, g.OriginX+g.OriginOffsetX)
	if face.Size == 24 && fraction == 0 {
		t.Fatal("small-size fixture requires a fractional glyph origin")
	}
	originX := 24 + fraction
	originY := 128.0
	expectedSVG, _, err := text.SVGAcceptanceGlyph(face, uint16(g.GID), g.OriginX+g.OriginOffsetX, true)
	if err != nil || expectedSVG != g.Image {
		t.Errorf("SVG selection failed for gid %d: %v", g.GID, err)
		return
	}

	actual := ebiten.NewImage(width, height)
	defer actual.Deallocate()
	var op text.DrawOptions
	op.GeoM.Translate(24-math.Floor(g.OriginX+g.OriginOffsetX), 128-math.Floor(g.OriginY+g.OriginOffsetY))
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
	reference := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="%d" height="%d"><g transform="translate(%g %g) scale(%g)">%s</g></svg>`, width, height, originX, originY, face.Size/unitsPerEm, body)
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
	// Native coverage quantization and coincident layer edges produce local
	// differences even when the glyph's shapes and paints agree.
	if mean > 8 {
		t.Errorf("occupied mean channel error %.3f exceeds 8", mean)
	}
	if maximum > maxChannelError {
		t.Errorf("maximum channel error %d exceeds %d", maximum, maxChannelError)
	}
	if dir := os.Getenv("EBITENGINE_SVG_RENDER_REPORT_DIR"); dir != "" {
		for label, pixels := range map[string][]byte{"production": got, "reference": want.Pix} {
			path := filepath.Join(dir, fmt.Sprintf("%s-%s.png", strings.ReplaceAll(t.Name(), "/", "-"), label))
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
}
