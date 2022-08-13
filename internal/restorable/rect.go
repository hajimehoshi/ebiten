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

func (p *pixelsRecord) readPixels(pixels []byte, x, y, width, height, imageWidth, imageHeight int) {
	r := p.rect.Intersect(image.Rect(x, y, x+width, y+height)).Intersect(image.Rect(0, 0, imageWidth, imageHeight))
	if r.Empty() {
		return
	}

	dstBaseX := r.Min.X - x
	dstBaseY := r.Min.Y - y
	srcBaseX := r.Min.X - p.rect.Min.X
	srcBaseY := r.Min.Y - p.rect.Min.Y
	lineWidth := 4 * r.Dx()
	for j := 0; j < r.Dy(); j++ {
		dstX := 4 * ((dstBaseY+j)*width + dstBaseX)
		srcX := 4 * ((srcBaseY+j)*p.rect.Dx() + srcBaseX)
		copy(pixels[dstX:dstX+lineWidth], p.pix[srcX:srcX+lineWidth])
	}
}

type pixelsRecords struct {
	records []*pixelsRecord
}

func (pr *pixelsRecords) addOrReplace(pixels []byte, x, y, width, height int) {
	if len(pixels) != 4*width*height {
		msg := fmt.Sprintf("restorable: len(pixels) must be 4*%d*%d = %d but %d", width, height, 4*width*height, len(pixels))
		if pixels == nil {
			msg += " (nil)"
		}
		panic(msg)
	}

	rect := image.Rect(x, y, x+width, y+height)

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

func (pr *pixelsRecords) readPixels(pixels []byte, x, y, width, height, imageWidth, imageHeight int) {
	for i := range pixels {
		pixels[i] = 0
	}
	for _, r := range pr.records {
		r.readPixels(pixels, x, y, width, height, imageWidth, imageHeight)
	}
}

func (pr *pixelsRecords) apply(img *graphicscommand.Image) {
	// TODO: Isn't this too heavy? Can we merge the operations?
	for _, r := range pr.records {
		img.WritePixels(r.pix, r.rect.Min.X, r.rect.Min.Y, r.rect.Dx(), r.rect.Dy())
	}
}
