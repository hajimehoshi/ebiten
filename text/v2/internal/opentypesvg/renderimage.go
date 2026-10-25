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

package opentypesvg

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"net/url"
	"strings"
)

const (
	maxImages           = 256
	maxImageBytes       = 4 << 20
	maxTotalImageBytes  = 16 << 20
	maxImagePixels      = 4 << 20
	maxTotalImagePixels = 8 << 20
)

var errImage = errors.New("opentypesvg: invalid embedded image")

func (b *renderBuilder) decodeImage(href string) (*image.RGBA, error) {
	if img := b.images[href]; img != nil {
		return img, nil
	}
	if len(href) > maxImageBytes || len(b.images) >= maxImages {
		return nil, errLimit
	}
	if !strings.HasPrefix(href, "data:") {
		return nil, errUnsupported
	}
	metadata, payload, ok := strings.Cut(href[5:], ",")
	if !ok {
		return nil, errImage
	}
	mime, encoded := strings.CutSuffix(metadata, ";base64")
	var format string
	switch mime {
	case "image/png":
		format = "png"
	case "image/jpeg":
		format = "jpeg"
	default:
		return nil, errUnsupported
	}
	if len(href) > maxTotalImageBytes-b.imageBytes {
		return nil, errLimit
	}
	raw, err := url.PathUnescape(payload)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errImage, err)
	}
	data := []byte(raw)
	if encoded {
		data, err = base64.StdEncoding.DecodeString(raw)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", errImage, err)
		}
	}
	config, actual, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errImage, err)
	}
	if actual != format {
		return nil, errImage
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > MaxSampleDimension || config.Height > MaxSampleDimension || config.Width > maxImagePixels/config.Height {
		return nil, errLimit
	}
	pixels := config.Width * config.Height
	if pixels > maxTotalImagePixels-b.imagePixels {
		return nil, errLimit
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errImage, err)
	}
	img := image.NewRGBA(image.Rect(0, 0, config.Width, config.Height))
	draw.Draw(img, img.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
	if b.images == nil {
		b.images = map[string]*image.RGBA{}
	}
	b.images[href] = img
	b.imagePixels += pixels
	b.imageBytes += len(href)
	return img, nil
}

func imageRectangle(x, y, width, height float64) path {
	return path{
		segments: []segment{
			{
				Kind: moveTo,
				To:   point{X: x, Y: y},
			},
			{
				Kind: lineTo,
				To:   point{X: x + width, Y: y},
			},
			{
				Kind: lineTo,
				To:   point{X: x + width, Y: y + height},
			},
			{
				Kind: lineTo,
				To:   point{X: x, Y: y + height},
			},
			{
				Kind: closePath,
				To:   point{X: x, Y: y},
			},
		},
	}
}

func (b *renderBuilder) prepareImage(n *renderNode) error {
	var values [4]float64
	for i, name := range []string{"x", "y", "width", "height"} {
		axis := horizontal
		if i%2 != 0 {
			axis = vertical
		}
		v, err := n.element.length(name, n.lengths, axis)
		if err != nil {
			return err
		}
		values[i] = v
	}
	x, y, w, h := values[0], values[1], values[2], values[3]
	if w < 0 || h < 0 {
		return errGeometry
	}
	if w == 0 || h == 0 {
		return nil
	}
	n.geometry = imageRectangle(x, y, w, h)
	if n.boundsOnly || !n.style.Visible {
		return nil
	}
	href, ok := n.element.attribute("", "href")
	if !ok {
		href, _ = n.element.attribute(xlinkNamespace, "href")
	}
	img, err := b.decodeImage(href)
	if err != nil {
		return err
	}
	aspect, _ := n.element.attribute("", "preserveAspectRatio")
	mapping, _, err := viewportMapping(viewBox{
		Width:  float64(img.Bounds().Dx()),
		Height: float64(img.Bounds().Dy()),
	}, w, h, aspect)
	if err != nil {
		return err
	}
	mapping.E += x
	mapping.F += y
	n.image, n.imageMapping = img, mapping
	return nil
}

func (b *renderBuilder) emitImage(n *renderNode, transform matrix, clip *RenderClip) error {
	mapping, err := transform.mul(n.imageMapping)
	if err != nil {
		return err
	}
	det := mapping.A*mapping.D - mapping.B*mapping.C
	inverse := [6]float64{mapping.D / det, -mapping.B / det, -mapping.C / det, mapping.A / det, (mapping.C*mapping.F - mapping.D*mapping.E) / det, (mapping.B*mapping.E - mapping.A*mapping.F) / det}
	for _, v := range inverse {
		if !isFinite(v) || math.Abs(v) > maxRenderCoordinate {
			return errLimit
		}
	}
	commands, err := b.convertPath(n.geometry, 1)
	if err != nil {
		return err
	}
	d := RenderDraw{
		Commands:       commands,
		Transform:      transform.values(),
		Clip:           clip,
		Image:          n.image,
		ImageTransform: mapping.values(),
	}
	if err := validateRenderDraw(d); err != nil {
		return err
	}
	if err := validateRenderDraw(RenderDraw{
		Transform: mapping.values(),
	}); err != nil {
		return err
	}
	area := b.plan.Width * b.plan.Height * b.plan.Samples * b.plan.Samples
	if area > maxRenderSampleWork-b.sampleWork {
		return errLimit
	}
	b.sampleWork += area
	if err := b.prepareDrawBatch(&d); err != nil {
		return err
	}
	b.plan.Draws = append(b.plan.Draws, d)
	return nil
}
