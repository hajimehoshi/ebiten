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

package text

import (
	"image"
	"runtime"
	"sync/atomic"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/hajimehoshi/ebiten/v2"
)

var currentStdFaceID uint64

func nextStdFaceID() uint64 {
	return atomic.AddUint64(&currentStdFaceID, 1)
}

var _ Face = (*StdFace)(nil)

// StdFace is a Face implementation for a semi-standard font.Face (golang.org/x/image/font).
// StdFace is useful to transit from existing codebase with text v1, or to use some bitmap fonts defined as font.Face.
// StdFace must not be copied by value.
type StdFace struct {
	f *faceWithCache

	id uint64

	addr *StdFace
}

// NewStdFace creates a new StdFace from a semi-standard font.Face.
func NewStdFace(face font.Face) *StdFace {
	s := &StdFace{
		f: &faceWithCache{
			f: face,
		},
		id: nextStdFaceID(),
	}
	s.addr = s
	runtime.SetFinalizer(s, theGlyphImageCache.clear)
	return s
}

func (s *StdFace) copyCheck() {
	if s.addr != s {
		panic("text: illegal use of non-zero StdFace copied by value")
	}
}

// Metrics implelements Face.
func (s *StdFace) Metrics() Metrics {
	s.copyCheck()

	m := s.f.Metrics()
	return Metrics{
		Height:   fixed26_6ToFloat64(m.Height),
		HAscent:  fixed26_6ToFloat64(m.Ascent),
		HDescent: fixed26_6ToFloat64(m.Descent),
	}
}

// UnsafeInternal implements Face.
func (s *StdFace) UnsafeInternal() any {
	s.copyCheck()
	return s.f.f
}

// faceCacheKey implements Face.
func (s *StdFace) faceCacheKey() faceCacheKey {
	return faceCacheKey(s.id)
}

// advance implements Face.
func (s *StdFace) advance(text string) float64 {
	return fixed26_6ToFloat64(font.MeasureString(s.f, text))
}

// appendGlyphs implements Face.
func (s *StdFace) appendGlyphs(glyphs []Glyph, text string, originX, originY float64) []Glyph {
	s.copyCheck()

	origin := fixed.Point26_6{
		X: float64ToFixed26_6(originX),
		Y: float64ToFixed26_6(originY),
	}
	prevR := rune(-1)

	for i, r := range text {
		if prevR >= 0 {
			origin.X += s.f.Kern(prevR, r)
		}
		img, imgX, imgY, a := s.glyphImage(r, origin)
		if img != nil {
			// Adjust the position to the integers.
			// The current glyph images assume that they are rendered on integer positions so far.
			glyphs = append(glyphs, Glyph{
				Rune:         r,
				IndexInBytes: i,
				Image:        img,
				X:            imgX,
				Y:            imgY,
			})
		}
		origin.X += a
		prevR = r
	}

	return glyphs
}

func (s *StdFace) glyphImage(r rune, origin fixed.Point26_6) (*ebiten.Image, float64, float64, fixed.Int26_6) {
	b, a, _ := s.f.GlyphBounds(r)
	offset := fixed.Point26_6{
		X: (adjustOffsetGranularity(origin.X) + b.Min.X) & ((1 << 6) - 1),
		Y: (fixed.I(origin.Y.Floor()) + b.Min.Y) & ((1 << 6) - 1),
	}
	key := glyphImageCacheKey{
		rune:    r,
		xoffset: offset.X,
		// yoffset is always an integer, so this doesn't have to be a key.
	}
	img := theGlyphImageCache.getOrCreate(s, key, func() *ebiten.Image {
		return s.glyphImageImpl(r, offset)
	})
	imgX := fixed26_6ToFloat64(origin.X + b.Min.X - offset.X)
	imgY := fixed26_6ToFloat64(origin.Y + b.Min.Y - offset.Y)
	return img, imgX, imgY, a
}

func (s *StdFace) glyphImageImpl(r rune, offset fixed.Point26_6) *ebiten.Image {
	b, _, _ := s.f.GlyphBounds(r)
	w, h := (b.Max.X - b.Min.X).Ceil(), (b.Max.Y - b.Min.Y).Ceil()
	if w == 0 || h == 0 {
		return nil
	}

	if b.Min.X&((1<<6)-1) != 0 {
		w++
	}
	if b.Min.Y&((1<<6)-1) != 0 {
		h++
	}
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))

	d := font.Drawer{
		Dst:  rgba,
		Src:  image.White,
		Face: s.f,
	}

	x, y := -b.Min.X, -b.Min.Y
	x += offset.X
	y += offset.Y
	d.Dot = fixed.Point26_6{X: x, Y: y}
	d.DrawString(string(r))

	return ebiten.NewImageFromImage(rgba)
}

// direction implelements Face.
func (s *StdFace) direction() Direction {
	return DirectionLeftToRight
}

// Metrics implelements Face.
func (s *StdFace) private() {
}
