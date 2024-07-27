package text

import (
	"github.com/Zyko0/Ebiary/atlas"
)

type glyphAtlas struct {
	atlas *atlas.Atlas
}

func newGlyphAtlas() *glyphAtlas {
	return &glyphAtlas{
		// Note: 128x128 is arbitrary, maybe a better value can be inferred
		// from the font size or something
		atlas: atlas.New(128, 128, nil),
	}
}

func (g *glyphAtlas) NewImage(w, h int) *atlas.Image {
	if img := g.atlas.NewImage(w, h); img != nil {
		return img
	}

	// Grow atlas
	old := g.atlas.Image()

	aw, ah := g.atlas.Bounds().Dx()*2, g.atlas.Bounds().Dy()*2
	g.atlas = atlas.New(aw, ah, nil)
	g.atlas.Image().DrawImage(old, nil)

	return g.NewImage(w, h)
}
