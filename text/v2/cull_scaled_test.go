package text_test

import (
	"bytes"
	"testing"

	"github.com/hajimehoshi/bitmapfont/v4"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// TestDrawScaledAgainstAppendGlyphs compares text.Draw with an unculled reference rendering built
// from AppendGlyphs. Draw's offscreen-glyph culling must not drop any glyph that the reference
// renders, especially with a scaled GeoM where a glyph's extent on dst is larger than its extent
// in text space.
func TestDrawScaledAgainstAppendGlyphs(t *testing.T) {
	face := text.NewGoXFace(bitmapfont.Face)
	m := face.Metrics()
	txt := "AAAA\nBBBB\nCCCC\nDDDD\nEEEE\nFFFF\nGGGG\nHHHH"
	layout := &text.LayoutOptions{LineSpacing: m.HAscent + m.HDescent}

	const w, h = 120, 120
	for _, scale := range []float64{1, 2, 4} {
		for _, ty := range []float64{-400, -300, -200, -100, -50, 0, 50, 80, 95} {
			dstDraw := ebiten.NewImage(w, h)
			defer dstDraw.Deallocate()
			dstRef := ebiten.NewImage(w, h)
			defer dstRef.Deallocate()

			geoM := ebiten.GeoM{}
			geoM.Scale(scale, scale)
			geoM.Translate(10, ty)

			op := &text.DrawOptions{}
			op.GeoM = geoM
			op.LayoutOptions = *layout
			text.Draw(dstDraw, txt, face, op)

			for _, g := range text.AppendGlyphs(nil, txt, face, layout) {
				if g.Image == nil {
					continue
				}
				gop := &ebiten.DrawImageOptions{}
				gop.GeoM.Translate(g.X, g.Y)
				gop.GeoM.Concat(geoM)
				dstRef.DrawImage(g.Image, gop)
			}

			bufDraw := make([]byte, 4*w*h)
			bufRef := make([]byte, 4*w*h)
			dstDraw.ReadPixels(bufDraw)
			dstRef.ReadPixels(bufRef)
			if !bytes.Equal(bufDraw, bufRef) {
				t.Errorf("scale: %g, ty: %g: Draw differs from AppendGlyphs-based rendering", scale, ty)
			}
		}
	}
}
