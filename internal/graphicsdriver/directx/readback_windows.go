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

package directx

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

func readbackRegion(args []graphicsdriver.PixelsArgs) (image.Rectangle, error) {
	var region image.Rectangle
	for _, a := range args {
		if n := 4 * a.Region.Dx() * a.Region.Dy(); len(a.Pixels) != n {
			return image.Rectangle{}, fmt.Errorf("directx: len(Pixels) must be %d but %d at ReadPixelsAsync", n, len(a.Pixels))
		}
		region = region.Union(a.Region)
	}
	return region, nil
}

func copyReadbackPixels(args []graphicsdriver.PixelsArgs, region image.Rectangle, src []byte, stride int) {
	for _, a := range args {
		w := 4 * a.Region.Dx()
		offset := 4*(a.Region.Min.X-region.Min.X) + stride*(a.Region.Min.Y-region.Min.Y)
		for y := 0; y < a.Region.Dy(); y++ {
			copy(a.Pixels[y*w:(y+1)*w], src[offset+y*stride:offset+y*stride+w])
		}
	}
}
