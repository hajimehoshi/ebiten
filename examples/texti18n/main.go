// Copyright 2023 The Ebitengine Authors
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

// This example is a demonstration to render languages that cannot be rendered with the `text` package.
// We plan to provide a useful API to render them more easily (#2454). Stay tuned!

package main

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	"image/draw"
	"log"
	"math"

	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/opentype/api"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
)

//go:embed NotoSansArabic-Regular.ttf
var arabicTTF []byte

var arabicOut shaping.Output

func init() {
	face, err := font.ParseTTF(bytes.NewReader(arabicTTF))
	if err != nil {
		log.Fatal(err)
	}
	runes := []rune("لمّا كان الاعتراف بالكرامة المتأصلة في جميع")
	input := shaping.Input{
		Text:      runes,
		RunStart:  0,
		RunEnd:    len(runes),
		Direction: di.DirectionRTL,
		Face:      face,
		Size:      fixed.I(24),
		Script:    language.Arabic,
		Language:  "ar",
	}
	arabicOut = (&shaping.HarfbuzzShaper{}).Shape(input)
}

//go:embed NotoSansDevanagari-Regular.ttf
var devanagariTTF []byte

var devanagariOut shaping.Output

func init() {
	face, err := font.ParseTTF(bytes.NewReader(devanagariTTF))
	if err != nil {
		log.Fatal(err)
	}
	runes := []rune("चूंकि मानव परिवार के सभी सदस्यों के जन्मजात गौरव और समान")
	input := shaping.Input{
		Text:      runes,
		RunStart:  0,
		RunEnd:    len(runes),
		Direction: di.DirectionLTR,
		Face:      face,
		Size:      fixed.I(24),
		Script:    language.Devanagari,
		Language:  "hi",
	}
	devanagariOut = (&shaping.HarfbuzzShaper{}).Shape(input)
}

//go:embed NotoSansThai-Regular.ttf
var thaiTTF []byte

var thaiOut shaping.Output

func init() {
	face, err := font.ParseTTF(bytes.NewReader(thaiTTF))
	if err != nil {
		log.Fatal(err)
	}
	runes := []rune("โดยที่การไม่นำพาและการหมิ่นในคุณค่าของสิทธิมนุษยชน")
	input := shaping.Input{
		Text:      runes,
		RunStart:  0,
		RunEnd:    len(runes),
		Direction: di.DirectionLTR,
		Face:      face,
		Size:      fixed.I(24),
		Script:    language.Thai,
		Language:  "th",
	}
	thaiOut = (&shaping.HarfbuzzShaper{}).Shape(input)
}

var japaneseOut shaping.Output

func init() {
	const japanese = language.Script(('j' << 24) | ('p' << 16) | ('a' << 8) | 'n')

	face, err := font.ParseTTF(bytes.NewReader(fonts.MPlus1pRegular_ttf))
	if err != nil {
		log.Fatal(err)
	}
	runes := []rune("ラーメン。")
	input := shaping.Input{
		Text:      runes,
		RunStart:  0,
		RunEnd:    len(runes),
		Direction: di.DirectionTTB,
		Face:      face,
		Size:      fixed.I(24),
		Script:    japanese,
		Language:  "ja",
	}
	japaneseOut = (&shaping.HarfbuzzShaper{}).Shape(input)
}

var (
	whiteImage = ebiten.NewImage(3, 3)

	// whiteSubImage is an internal sub image of whiteImage.
	// Use whiteSubImage at DrawTriangles instead of whiteImage in order to avoid bleeding edges.
	whiteSubImage = whiteImage.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
)

func init() {
	whiteImage.Fill(color.White)
}

const (
	screenWidth  = 640
	screenHeight = 480
)

type Game struct {
	vertices []ebiten.Vertex
	indices  []uint16

	glyphCache map[glyphCacheKey]glyphCacheValue
}

type glyphCacheKey struct {
	output  *shaping.Output // TODO: This should be a font.Face instead of shaping.Output.
	glyphID api.GID
	origin  fixed.Point26_6
}

type glyphCacheValue struct {
	image *ebiten.Image
	point image.Point
}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.drawGlyphs(screen, &arabicOut, 20, 100)
	g.drawGlyphs(screen, &devanagariOut, 20, 150)
	g.drawGlyphs(screen, &thaiOut, 20, 200)
	g.drawGlyphs(screen, &japaneseOut, 20, 250)
}

func fixed26_6ToFloat32(x fixed.Int26_6) float32 {
	return float32(x>>6) + (float32(x&(1<<6-1)) / (1 << 6))
}

func float32ToFixed26_6(x float32) fixed.Int26_6 {
	i := float32(math.Floor(float64(x)))
	return (fixed.Int26_6(i) << 6) + fixed.Int26_6((x-i)*(1<<6))
}

func (g *Game) drawGlyphs(dst *ebiten.Image, output *shaping.Output, originX, originY float32) {
	g.vertices = g.vertices[:0]
	g.indices = g.indices[:0]

	scale := fixed26_6ToFloat32(output.Size) / float32(output.Face.Font.Upem())

	orig := fixed.Point26_6{
		X: float32ToFixed26_6(originX),
		Y: float32ToFixed26_6(originY),
	}
	for _, glyph := range output.Glyphs {
		key := glyphCacheKey{
			output:  output,
			glyphID: glyph.GlyphID,
			origin:  orig,
		}

		v, ok := g.glyphCache[key]
		if !ok {
			data := output.Face.GlyphData(glyph.GlyphID).(api.GlyphOutline)
			if len(data.Segments) > 0 {
				segs := make([]api.Segment, len(data.Segments))
				for i, seg := range data.Segments {
					segs[i] = seg
					for j := range seg.Args {
						segs[i].Args[j].X *= scale
						segs[i].Args[j].Y *= scale
						segs[i].Args[j].Y *= -1
					}
				}
				v.image, v.point = segmentsToImage(segs, orig)
			}
			if g.glyphCache == nil {
				g.glyphCache = map[glyphCacheKey]glyphCacheValue{}
			}
			g.glyphCache[key] = v
		}

		if v.image != nil {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(v.point.X), float64(v.point.Y))
			dst.DrawImage(v.image, op)
		}

		orig = orig.Add(fixed.Point26_6{X: glyph.XAdvance, Y: glyph.YAdvance * -1})
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Text I18N (Ebitengine Demo)")
	if err := ebiten.RunGame(&Game{}); err != nil {
		log.Fatal(err)
	}
}

func segmentsToRect(segs []api.Segment) fixed.Rectangle26_6 {
	if len(segs) == 0 {
		return fixed.Rectangle26_6{}
	}

	minX := float32(math.Inf(1))
	minY := float32(math.Inf(1))
	maxX := float32(math.Inf(-1))
	maxY := float32(math.Inf(-1))

	for _, seg := range segs {
		n := 1
		switch seg.Op {
		case api.SegmentOpQuadTo:
			n = 2
		case api.SegmentOpCubeTo:
			n = 3
		}
		for i := 0; i < n; i++ {
			x := seg.Args[i].X
			y := seg.Args[i].Y
			if minX > x {
				minX = x
			}
			if minY > y {
				minY = y
			}
			if maxX < x {
				maxX = x
			}
			if maxY < y {
				maxY = y
			}
		}
	}

	return fixed.Rectangle26_6{
		Min: fixed.Point26_6{
			X: float32ToFixed26_6(minX),
			Y: float32ToFixed26_6(minY),
		},
		Max: fixed.Point26_6{
			X: float32ToFixed26_6(maxX),
			Y: float32ToFixed26_6(maxY),
		},
	}
}

func segmentsToImage(segs []api.Segment, orig fixed.Point26_6) (*ebiten.Image, image.Point) {
	dBounds := segmentsToRect(segs).Add(orig)
	dr := image.Rect(
		dBounds.Min.X.Floor(),
		dBounds.Min.Y.Floor(),
		dBounds.Max.X.Ceil(),
		dBounds.Max.Y.Ceil(),
	)
	biasX := fixed26_6ToFloat32(orig.X) - float32(dr.Min.X)
	biasY := fixed26_6ToFloat32(orig.Y) - float32(dr.Min.Y)

	width, height := dr.Dx(), dr.Dy()
	if width <= 0 || height <= 0 {
		return nil, image.Point{}
	}

	rast := vector.NewRasterizer(width, height)
	rast.DrawOp = draw.Src
	for _, seg := range segs {
		switch seg.Op {
		case api.SegmentOpMoveTo:
			rast.MoveTo(seg.Args[0].X+biasX, seg.Args[0].Y+biasY)
		case api.SegmentOpLineTo:
			rast.LineTo(seg.Args[0].X+biasX, seg.Args[0].Y+biasY)
		case api.SegmentOpQuadTo:
			rast.QuadTo(
				seg.Args[0].X+biasX, seg.Args[0].Y+biasY,
				seg.Args[1].X+biasX, seg.Args[1].Y+biasY,
			)
		case api.SegmentOpCubeTo:
			rast.CubeTo(
				seg.Args[0].X+biasX, seg.Args[0].Y+biasY,
				seg.Args[1].X+biasX, seg.Args[1].Y+biasY,
				seg.Args[2].X+biasX, seg.Args[2].Y+biasY,
			)
		}
	}

	dst := image.NewAlpha(image.Rect(0, 0, width, height))
	rast.Draw(dst, dst.Bounds(), image.Opaque, image.Point{})
	return ebiten.NewImageFromImage(dst), dr.Min
}
