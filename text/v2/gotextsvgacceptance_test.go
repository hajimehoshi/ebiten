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
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/font/opentype/tables"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/opentypesvg"
)

type svgAcceptanceOutcome struct {
	Glyph                        uint16
	Size                         float64
	Offset                       float64
	Outcome                      string
	Detail                       string `json:",omitempty"`
	Width, Height                int
	PreparationNS, RealizationNS int64
	Realized                     bool
}

type svgAcceptanceDocument struct {
	Offset, Length uint32
	DecodedBytes   int
	SHA256         string
	DecodeOutcome  string
	Detail         string `json:",omitempty"`
	Ranges         [][2]uint16
}

func svgOutcome(err error, empty bool) string {
	switch {
	case errors.Is(err, opentypesvg.ErrLimit):
		return "resource-limited"
	case errors.Is(err, opentypesvg.ErrUnsupported):
		return "unsupported"
	case err != nil:
		return "malformed"
	case empty:
		return "valid-empty"
	default:
		return "success"
	}
}

// TestSVGFontInventory inventories preparation, with an optional bounded GPU sample.
func TestSVGFontInventory(t *testing.T) {
	path := os.Getenv("EBITENGINE_SVG_ACCEPTANCE_FONT")
	if path == "" {
		t.Skip("set EBITENGINE_SVG_ACCEPTANCE_FONT; see internal/opentypesvg/testdata/README.md")
	}
	gpuLimit := 0
	if value := os.Getenv("EBITENGINE_SVG_ACCEPTANCE_GPU_GLYPHS"); value != "" {
		var err error
		gpuLimit, err = strconv.Atoi(value)
		if err != nil || gpuLimit < 0 || gpuLimit > 8 {
			t.Fatal("EBITENGINE_SVG_ACCEPTANCE_GPU_GLYPHS must be 0..8; run alone, without VMs, with -p 1")
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	loader, err := opentype.NewLoader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := loader.RawTable(opentype.MustNewTag("SVG "))
	if err != nil {
		t.Fatal(err)
	}
	table, _, err := tables.ParseSVG(raw)
	if err != nil {
		t.Fatal(err)
	}
	src, err := text.NewGoTextFaceSource(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	face := &text.GoTextFace{
		Source: src,
	}
	report := struct {
		FontSHA256                string
		Documents                 []svgAcceptanceDocument
		Outcomes                  []svgAcceptanceOutcome
		Counts                    map[string]int
		PublicGlyphs              int
		Phase                     string
		GPURequested, GPURealized int
		Complete                  bool
	}{
		FontSHA256:   fmt.Sprintf("%x", sha256.Sum256(data)),
		Counts:       make(map[string]int),
		Phase:        "svg-record-preparation",
		GPURequested: gpuLimit,
	}
	defer func() {
		if dir := os.Getenv("EBITENGINE_SVG_ACCEPTANCE_REPORT_DIR"); dir != "" {
			encoded, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				t.Error(err)
				return
			}
			if err := os.WriteFile(filepath.Join(dir, "full-font.json"), encoded, 0644); err != nil {
				t.Error(err)
			}
		}
	}()
	documentIndices := make(map[[2]uint32]int)
	for _, record := range table.SVGDocumentList.DocumentRecords {
		key := [2]uint32{uint32(record.SvgDocOffset), record.SvgDocLength}
		index, ok := documentIndices[key]
		if !ok {
			index = len(report.Documents)
			documentIndices[key] = index
			start, end := uint64(key[0]), uint64(key[0])+uint64(key[1])
			if end > uint64(len(table.SVGDocumentList.SVGRawData)) {
				t.Fatal("SVG record outside table")
			}
			source := table.SVGDocumentList.SVGRawData[start:end]
			decoded, err := opentypesvg.Decode(source)
			var detail string
			if err != nil {
				detail = err.Error()
			}
			report.Documents = append(report.Documents, svgAcceptanceDocument{
				Offset:        key[0],
				Length:        key[1],
				DecodedBytes:  len(decoded),
				SHA256:        fmt.Sprintf("%x", sha256.Sum256(source)),
				DecodeOutcome: svgOutcome(err, false),
				Detail:        detail,
			})
		}
		report.Documents[index].Ranges = append(report.Documents[index].Ranges, [2]uint16{uint16(record.StartGlyphID), uint16(record.EndGlyphID)})
	}
	// Traverse by document to exercise shared records without artificial cache churn.
	for _, document := range report.Documents {
		for _, size := range []float64{24, 96} {
			face.Size = size
			for _, glyphRange := range document.Ranges {
				for gid := int(glyphRange[0]); gid <= int(glyphRange[1]); gid++ {
					offset := text.SVGAcceptanceOffset(face, .375)
					start := time.Now()
					_, bounds, err := text.SVGAcceptanceGlyph(face, uint16(gid), offset, false)
					outcome := svgAcceptanceOutcome{
						Glyph:         uint16(gid),
						Size:          size,
						Offset:        offset,
						Outcome:       svgOutcome(err, bounds.Empty()),
						Width:         bounds.Dx(),
						Height:        bounds.Dy(),
						PreparationNS: time.Since(start).Nanoseconds(),
					}
					if err != nil {
						outcome.Detail = err.Error()
					}
					if err == nil && !bounds.Empty() && report.GPURealized < gpuLimit && bounds.Dx() <= 256 && bounds.Dy() <= 256 {
						start = time.Now()
						img, _, err := text.SVGAcceptanceGlyph(face, uint16(gid), offset, true)
						if err != nil || img == nil {
							t.Errorf("glyph %d: realization: %v", gid, err)
							outcome.Outcome = "realization-failed"
						}
						if img != nil {
							pixels := make([]byte, 4*bounds.Dx()*bounds.Dy())
							img.ReadPixels(pixels)
							outcome.RealizationNS = time.Since(start).Nanoseconds()
							outcome.Realized = true
							report.GPURealized++
						}
					}
					report.Outcomes = append(report.Outcomes, outcome)
					report.Counts[outcome.Outcome]++
				}
			}
		}
		t.Logf("document bytes=%d ranges=%d cumulative=%v", document.DecodedBytes, len(document.Ranges), report.Counts)
	}
	if input := os.Getenv("EBITENGINE_SVG_ACCEPTANCE_TEXT"); input != "" {
		if gpuLimit == 0 {
			t.Fatal("EBITENGINE_SVG_ACCEPTANCE_TEXT requires an explicit GPU sample limit")
		}
		face.Size = 24
		lazy := text.AppendLazyGlyphs(nil, input, face, nil)
		if len(lazy) == 0 || len(lazy) > 8 {
			t.Fatal("public sample must shape into 1..8 glyphs")
		}
		for _, glyph := range lazy {
			if glyph.ImageBounds.Dx() > 256 || glyph.ImageBounds.Dy() > 256 {
				t.Fatal("public sample image exceeds 256 pixels per axis")
			}
		}
		glyphs := text.AppendGlyphs(nil, input, face, nil)
		actual := ebiten.NewImage(256, 128)
		defer actual.Deallocate()
		expected := ebiten.NewImage(256, 128)
		defer expected.Deallocate()
		for _, glyph := range glyphs {
			img, bounds, err := text.SVGAcceptanceGlyph(face, uint16(glyph.GID), glyph.OriginX+glyph.OriginOffsetX, true)
			if err != nil || bounds.Empty() || img == nil || !glyph.Colored || glyph.Image != img {
				t.Errorf("public glyph %d did not select the expected nonempty SVG: %v", glyph.GID, err)
				continue
			}
			var op ebiten.DrawImageOptions
			op.GeoM.Translate(glyph.X+24.375, glyph.Y+48)
			expected.DrawImage(img, &op)
			report.PublicGlyphs++
		}
		var op text.DrawOptions
		op.GeoM.Translate(24.375, 48)
		text.Draw(actual, input, face, &op)
		got, want := make([]byte, 4*256*128), make([]byte, 4*256*128)
		actual.ReadPixels(got)
		expected.ReadPixels(want)
		if !bytes.Equal(got, want) {
			t.Error("Draw differs from public SVG glyph composition")
		}
	}
	report.Complete = true
	t.Logf("sha256=%s documents=%d outcomes=%v public=%d", report.FontSHA256, len(report.Documents), report.Counts, report.PublicGlyphs)
	if report.Counts["malformed"]+report.Counts["unsupported"]+report.Counts["resource-limited"]+report.Counts["realization-failed"] > 0 {
		t.Error("SVG preparation or sampled realization failed; inspect report")
	}
}

// BenchmarkSVGProduction measures synchronized draws with at most 16 iterations.
func BenchmarkSVGProduction(b *testing.B) {
	if os.Getenv("EBITENGINE_SVG_ACCEPTANCE_BENCHMARK") != "1" {
		b.Skip("opt-in: see internal/opentypesvg/testdata/README.md")
	}

	const document = `<svg viewBox="0 0 100 100"><defs><linearGradient id="g"><stop stop-color="currentColor"/><stop offset="1" stop-color="blue"/></linearGradient></defs><g id="GLYPHA"><circle cx="50" cy="50" r="45" fill="url(#g)"/></g></svg>`
	for _, workload := range []string{"cold", "warm", "layout", "colors", "large"} {
		b.Run(workload, func(b *testing.B) {
			face := &text.GoTextFace{
				Source: svgTestFont(b, document, false),
				Size:   96,
			}
			if workload == "large" {
				face.Size = 512
			}
			if b.N > 16 {
				b.Skip("use -benchtime=16x or less to bound cache misses")
			}
			var dst *ebiten.Image
			if workload != "layout" {
				dst = ebiten.NewImage(512, 512)
				defer dst.Deallocate()
			}
			if workload == "warm" || workload == "layout" {
				text.AppendLazyGlyphs(nil, "A", face, nil)
			}
			if workload == "warm" {
				text.AppendGlyphs(nil, "A", face, nil)
			}
			pixel := make([]byte, 4)
			b.ResetTimer()
			for iteration := 0; iteration < b.N; iteration++ {
				if workload == "cold" {
					face.Source = svgTestFont(b, document, false)
				}
				if workload == "colors" || workload == "large" {
					face.SVGForeground = color.NRGBA{R: byte(iteration), G: byte(iteration >> 8), A: 255}
				}
				if workload == "layout" {
					text.AppendLazyGlyphs(nil, "A", face, nil)
					continue
				}
				glyphs := text.AppendGlyphs(nil, "A", face, nil)
				if len(glyphs) != 1 || glyphs[0].Image == nil || !glyphs[0].Colored {
					b.Fatal("missing SVG glyph or outline fallback")
				}
				dst.Clear()
				text.Draw(dst, "A", face, nil)
				dst.SubImage(image.Rect(0, 0, 1, 1)).(*ebiten.Image).ReadPixels(pixel)
			}
		})
	}
}
