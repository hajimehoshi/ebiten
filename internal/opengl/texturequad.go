/*
Copyright 2014 Hajime Hoshi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package opengl

import (
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/internal/opengl/internal/shader"
)

func NextPowerOf2(x uint64) uint64 {
	x -= 1
	x |= (x >> 1)
	x |= (x >> 2)
	x |= (x >> 4)
	x |= (x >> 8)
	x |= (x >> 16)
	x |= (x >> 32)
	return x + 1
}

func adjustSizeForTexture(size int) int {
	return int(NextPowerOf2(uint64(size)))
}

func u(x int, width int) float32 {
	return float32(x) / float32(adjustSizeForTexture(width))
}

func v(y int, height int) float32 {
	return float32(y) / float32(adjustSizeForTexture(height))
}

func textureQuads(parts []ebiten.TexturePart, width, height int) []shader.TextureQuad {
	quads := make([]shader.TextureQuad, 0, len(parts))
	for _, part := range parts {
		x1 := float32(part.LocationX)
		x2 := float32(part.LocationX + part.Source.Width)
		y1 := float32(part.LocationY)
		y2 := float32(part.LocationY + part.Source.Height)
		u1 := u(part.Source.X, width)
		u2 := u(part.Source.X+part.Source.Width, width)
		v1 := v(part.Source.Y, height)
		v2 := v(part.Source.Y+part.Source.Height, height)
		quad := shader.TextureQuad{x1, x2, y1, y2, u1, u2, v1, v2}
		quads = append(quads, quad)
	}
	return quads
}
