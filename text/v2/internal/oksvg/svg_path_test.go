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

package oksvg_test

import (
	"image"
	"strings"
	"testing"

	"github.com/srwiley/rasterx"

	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/oksvg"
)

// TestDrawTransformedUserSpaceGradient checks that a gradient with
// gradientUnits="userSpaceOnUse" is evaluated with the drawing transform
// applied (#2649).
func TestDrawTransformedUserSpaceGradient(t *testing.T) {
	const svg = `<svg xmlns="http://www.w3.org/2000/svg">
<defs><radialGradient id="g" cx="500" cy="500" r="500" gradientUnits="userSpaceOnUse">
<stop offset="0" stop-color="#FFFFFF"/><stop offset="1" stop-color="#000000"/>
</radialGradient></defs>
<rect x="0" y="0" width="1000" height="1000" fill="url(#g)"/>
</svg>`

	icon, err := oksvg.ReadIconStream(strings.NewReader(svg))
	if err != nil {
		t.Fatal(err)
	}

	const w, h = 100, 100
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	scanner := rasterx.NewScannerGV(w, h, img, img.Bounds())
	dasher := rasterx.NewDasher(w, h, scanner)
	icon.Transform = rasterx.Matrix2D{A: 0.1, D: 0.1}
	icon.Draw(dasher, 1)

	// The gradient center (500, 500) in user space maps to (50, 50).
	if r, _, _, _ := img.At(50, 50).RGBA(); r>>8 < 0xc0 {
		t.Errorf("center pixel is not near-white: got R=0x%02x, want >= 0xc0", r>>8)
	}
	// (950, 500) in user space maps to (95, 50), near the gradient edge.
	if r, _, _, _ := img.At(95, 50).RGBA(); r>>8 >= 0x40 {
		t.Errorf("edge pixel is not near-black: got R=0x%02x, want < 0x40", r>>8)
	}
}
