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

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
)

type pixelsRecord struct {
	rect image.Rectangle
	pix  *graphics.ManagedBytes
}

func (p *pixelsRecord) readPixels(pixels []byte, region image.Rectangle, imageWidth, imageHeight int) {
	r := p.rect.Intersect(region.Intersect(image.Rect(0, 0, imageWidth, imageHeight)))
	if r.Empty() {
		return
	}

	dstBaseX := r.Min.X - region.Min.X
	dstBaseY := r.Min.Y - region.Min.Y
	lineWidth := 4 * r.Dx()
	if p.pix != nil {
		srcBaseX := r.Min.X - p.rect.Min.X
		srcBaseY := r.Min.Y - p.rect.Min.Y
		for j := 0; j < r.Dy(); j++ {
			dstX := 4 * ((dstBaseY+j)*region.Dx() + dstBaseX)
			srcX := 4 * ((srcBaseY+j)*p.rect.Dx() + srcBaseX)
			p.pix.Read(pixels[dstX:dstX+lineWidth], srcX, srcX+lineWidth)
		}
	} else {
		for j := 0; j < r.Dy(); j++ {
			dstX := 4 * ((dstBaseY+j)*region.Dx() + dstBaseX)
			for i := 0; i < lineWidth; i++ {
				pixels[i+dstX] = 0
			}
		}
	}
}

type pixelsRecords struct {
	records []*pixelsRecord
}

func (pr *pixelsRecords) addOrReplace(pixels *graphics.ManagedBytes, region image.Rectangle) {
	if pixels.Len() != 4*region.Dx()*region.Dy() {
		msg := fmt.Sprintf("restorable: len(pixels) must be 4*%d*%d = %d but %d", region.Dx(), region.Dy(), 4*region.Dx()*region.Dy(), pixels.Len())
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
	if region.Empty() {
		return
	}

	var n int
	var needsClear bool
	for _, r := range pr.records {
		if r.rect.In(region) {
			continue
		}
		if r.rect.Overlaps(region) {
			needsClear = true
		}
		pr.records[n] = r
		n++
	}
	for i := n; i < len(pr.records); i++ {
		pr.records[i] = nil
	}
	pr.records = pr.records[:n]
	if needsClear {
		pr.records = append(pr.records, &pixelsRecord{
			rect: region,
		})
	}
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
		if r.pix != nil {
			// Clone a ManagedBytes as the package graphicscommand has a different lifetime management.
			img.WritePixels(r.pix.Clone(), r.rect)
		} else {
			clearImage(img, r.rect)
		}
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

func (pr *pixelsRecords) dispose() {
	for _, r := range pr.records {
		if r.pix == nil {
			continue
		}
		// As the package graphicscommands already has cloned ManagedBytes objects, it is OK to release it.
		_, f := r.pix.GetAndRelease()
		f()
	}
}
