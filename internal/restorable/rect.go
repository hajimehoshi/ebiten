// Copyright 2019 The Ebiten Authors
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

package restorable

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
)

type pixelsRecord struct {
	rect image.Rectangle
	pix  []byte
	mask []byte
}

func (p *pixelsRecord) clearIfOverlapped(rect image.Rectangle) {
	r := p.rect.Intersect(rect)
	ox := r.Min.X - p.rect.Min.X
	oy := r.Min.Y - p.rect.Min.Y
	w := p.rect.Dx()
	for j := 0; j < r.Dy(); j++ {
		for i := 0; i < r.Dx(); i++ {
			idx := 4 * ((oy+j)*w + ox + i)
			p.pix[idx] = 0
			p.pix[idx+1] = 0
			p.pix[idx+2] = 0
			p.pix[idx+3] = 0
		}
	}
}

func (p *pixelsRecord) merge(pix []byte, mask []byte) {
	if len(p.pix) != len(pix) {
		panic(fmt.Sprintf("restorable: len(p.pix) (%d) and len(pix) (%d) must match but not", len(p.pix), len(pix)))
	}

	if mask == nil {
		p.pix = pix
		return
	}

	if p.mask == nil {
		p.mask = mask
	}
	if len(p.mask) != len(mask) {
		panic(fmt.Sprintf("restorable: len(p.mask) (%d) and len(mask) (%d) must match but not", len(p.mask), len(mask)))
	}

	for i := 0; i < len(p.pix)/4; i++ {
		if mask[i/8]>>(i%8)&1 == 0 {
			continue
		}
		p.mask[i/8] |= 1 << (i % 8)
		copy(p.pix[4*i:4*(i+1)], pix[4*i:4*(i+1)])
	}
}

func (p *pixelsRecord) at(x, y int) (r, g, b, a byte, ok bool) {
	if !image.Pt(x, y).In(p.rect) {
		return 0, 0, 0, 0, false
	}
	idx := ((y-p.rect.Min.Y)*p.rect.Dx() + (x - p.rect.Min.X))
	if p.mask != nil && p.mask[idx/8]>>(idx%8)&1 == 0 {
		return 0, 0, 0, 0, false
	}
	return p.pix[4*idx], p.pix[4*idx+1], p.pix[4*idx+2], p.pix[4*idx+3], true
}

type pixelsRecords struct {
	records []*pixelsRecord
}

func (pr *pixelsRecords) addOrReplace(pixels []byte, mask []byte, x, y, width, height int) {
	if len(pixels) != 4*width*height {
		msg := fmt.Sprintf("restorable: len(pixels) must be 4*%d*%d = %d but %d", width, height, 4*width*height, len(pixels))
		if pixels == nil {
			msg += " (nil)"
		}
		panic(msg)
	}

	rect := image.Rect(x, y, x+width, y+height)
	if mask != nil {
		// If a mask is specified, try merging the records.
		var merged bool
		for idx := len(pr.records) - 1; idx >= 0; idx-- {
			if r := pr.records[idx]; r.rect.Overlaps(rect) {
				if r.rect == rect {
					r.merge(pixels, mask)
					merged = true
				}
				// If there is an overlap with other regions in the existing records,
				// give up modifying them and just add a record.
				break
			}
		}
		if !merged {
			pr.records = append(pr.records, &pixelsRecord{
				rect: rect,
				pix:  pixels,
				mask: mask,
			})
		}
		return
	}

	// Remove or update the duplicated records first.
	var n int
	for _, r := range pr.records {
		if r.rect.In(rect) {
			continue
		}
		pr.records[n] = r
		n++
	}
	for i := n; i < len(pr.records); i++ {
		pr.records[i] = nil
	}
	pr.records = pr.records[:n]

	// Add the new record.
	pr.records = append(pr.records, &pixelsRecord{
		rect: rect,
		pix:  pixels,
	})
}

func (pr *pixelsRecords) clear(x, y, width, height int) {
	newr := image.Rect(x, y, x+width, y+height)
	var n int
	for _, r := range pr.records {
		if r.rect.In(newr) {
			continue
		}
		r.clearIfOverlapped(newr)
		pr.records[n] = r
		n++
	}
	for i := n; i < len(pr.records); i++ {
		pr.records[i] = nil
	}
	pr.records = pr.records[:n]
}

func (pr *pixelsRecords) at(i, j int) (r, g, b, a byte, ok bool) {
	// Traverse the slice in the reversed order.
	for idx := len(pr.records) - 1; idx >= 0; idx-- {
		r, g, b, a, ok := pr.records[idx].at(i, j)
		if ok {
			return r, g, b, a, true
		}
	}

	return 0, 0, 0, 0, false
}

func (pr *pixelsRecords) apply(img *graphicscommand.Image) {
	// TODO: Isn't this too heavy? Can we merge the operations?
	for _, r := range pr.records {
		img.ReplacePixels(r.pix, r.mask, r.rect.Min.X, r.rect.Min.Y, r.rect.Dx(), r.rect.Dy())
	}
}
