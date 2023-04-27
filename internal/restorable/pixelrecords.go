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

func (p *pixelsRecord) readPixels(pixels []byte, region image.Rectangle, imageWidth, imageHeight int) {
	r := p.rect.Intersect(region.Intersect(image.Rect(0, 0, imageWidth, imageHeight)))
	if r.Empty() {
		return
	}

	dstBaseX := r.Min.X - region.Min.X
	dstBaseY := r.Min.Y - region.Min.Y
	srcBaseX := r.Min.X - p.rect.Min.X
	srcBaseY := r.Min.Y - p.rect.Min.Y
	lineWidth := 4 * r.Dx()
	for j := 0; j < r.Dy(); j++ {
		dstX := 4 * ((dstBaseY+j)*region.Dx() + dstBaseX)
		srcX := 4 * ((srcBaseY+j)*p.rect.Dx() + srcBaseX)
		copy(pixels[dstX:dstX+lineWidth], p.pix[srcX:srcX+lineWidth])
	}
}

type pixelsRecords struct {
	records []*pixelsRecord
}

func (pr *pixelsRecords) addOrReplace(pixels []byte, region image.Rectangle) {
	if len(pixels) != 4*region.Dx()*region.Dy() {
		msg := fmt.Sprintf("restorable: len(pixels) must be 4*%d*%d = %d but %d", region.Dx(), region.Dy(), 4*region.Dx()*region.Dy(), len(pixels))
		if pixels == nil {
			msg += " (nil)"
		}
		panic(msg)
	}

	// Remove or update the duplicated records first.
	var n int
	for _, r := range pr.records {
		if r.rect.In(region) {
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
		rect: region,
		pix:  pixels,
	})
}

func (pr *pixelsRecords) clear(region image.Rectangle) {
	var n int
	for _, r := range pr.records {
		if r.rect.In(region) {
			continue
		}
		r.clearIfOverlapped(region)
		pr.records[n] = r
		n++
	}
	for i := n; i < len(pr.records); i++ {
		pr.records[i] = nil
	}
	pr.records = pr.records[:n]
}

func (pr *pixelsRecords) readPixels(pixels []byte, region image.Rectangle, imageWidth, imageHeight int) {
	for i := range pixels {
		pixels[i] = 0
	}
	for _, r := range pr.records {
		r.readPixels(pixels, region, imageWidth, imageHeight)
	}
}

func (pr *pixelsRecords) apply(img *graphicscommand.Image) {
	// TODO: Isn't this too heavy? Can we merge the operations?
	for _, r := range pr.records {
		img.WritePixels(r.pix, r.rect)
	}
}

func (pr *pixelsRecords) appendRegions(regions []image.Rectangle) []image.Rectangle {
	for _, r := range pr.records {
		if r.rect.Empty() {
			continue
		}
		regions = append(regions, r.rect)
	}
	return regions
}
