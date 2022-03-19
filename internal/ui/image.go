// Copyright 2022 The Ebiten Authors
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

package ui

import (
	"github.com/hajimehoshi/ebiten/v2/internal/affine"
	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/mipmap"
)

type Image struct {
	mipmap *mipmap.Mipmap
}

func NewImage(width, height int) *Image {
	return &Image{
		mipmap: mipmap.New(width, height),
	}
}

func NewScreenFramebufferImage(width, height int) *Image {
	return &Image{
		mipmap: mipmap.NewScreenFramebufferMipmap(width, height),
	}
}

func (i *Image) MarkDisposed() {
	i.mipmap.MarkDisposed()
	i.mipmap = nil
}

func (i *Image) DrawTriangles(srcs [graphics.ShaderImageNum]*Image, vertices []float32, indices []uint16, colorm affine.ColorM, mode graphicsdriver.CompositeMode, filter graphicsdriver.Filter, address graphicsdriver.Address, dstRegion, srcRegion graphicsdriver.Region, subimageOffsets [graphics.ShaderImageNum - 1][2]float32, shader *Shader, uniforms [][]float32, evenOdd bool, canSkipMipmap bool) {
	var srcMipmaps [graphics.ShaderImageNum]*mipmap.Mipmap
	for i, src := range srcs {
		if src == nil {
			continue
		}
		srcMipmaps[i] = src.mipmap
	}

	var s *mipmap.Shader
	if shader != nil {
		s = shader.shader
	}

	i.mipmap.DrawTriangles(srcMipmaps, vertices, indices, colorm, mode, filter, address, dstRegion, srcRegion, subimageOffsets, s, uniforms, evenOdd, canSkipMipmap)
}

func (i *Image) ReplacePixels(pix []byte, x, y, width, height int) error {
	return i.mipmap.ReplacePixels(graphicsDriver(), pix, x, y, width, height)
}

func (i *Image) Pixels(x, y, width, height int) ([]byte, error) {
	return i.mipmap.Pixels(graphicsDriver(), x, y, width, height)
}

func (i *Image) DumpScreenshot(name string, blackbg bool) error {
	return i.mipmap.DumpScreenshot(graphicsDriver(), name, blackbg)
}

func (i *Image) SetIndependent(independent bool) {
	i.mipmap.SetIndependent(independent)
}

func (i *Image) SetVolatile(volatile bool) {
	i.mipmap.SetVolatile(volatile)
}

func DumpImages(dir string) error {
	return atlas.DumpImages(graphicsDriver(), dir)
}
