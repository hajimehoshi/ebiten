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
	"github.com/go-gl/gl"
	"image"
	"image/draw"
)

func adjustImageForTexture(img image.Image) *image.NRGBA {
	width, height := img.Bounds().Size().X, img.Bounds().Size().Y
	adjustedImageBounds := image.Rectangle{
		image.ZP,
		image.Point{
			adjustSizeForTexture(width),
			adjustSizeForTexture(height),
		},
	}
	if nrgba, ok := img.(*image.NRGBA); ok && img.Bounds() == adjustedImageBounds {
		return nrgba
	}

	adjustedImage := image.NewNRGBA(adjustedImageBounds)
	dstBounds := image.Rectangle{
		image.ZP,
		img.Bounds().Size(),
	}
	draw.Draw(adjustedImage, dstBounds, img, image.ZP, draw.Src)
	return adjustedImage
}

type texture struct {
	native gl.Texture
	width  int
	height int
}

func createNativeTexture(textureWidth, textureHeight int, pixels []uint8, filter int) gl.Texture {
	nativeTexture := gl.GenTexture()
	if nativeTexture < 0 {
		panic("glGenTexture failed")
	}
	gl.PixelStorei(gl.UNPACK_ALIGNMENT, 4)
	nativeTexture.Bind(gl.TEXTURE_2D)
	defer gl.Texture(0).Bind(gl.TEXTURE_2D)

	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, filter)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, filter)

	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, textureWidth, textureHeight, 0, gl.RGBA, gl.UNSIGNED_BYTE, pixels)

	return nativeTexture
}

func createTexture(width, height int, filter int) (*texture, error) {
	w := adjustSizeForTexture(width)
	h := adjustSizeForTexture(height)
	native := createNativeTexture(w, h, nil, filter)
	return &texture{native, width, height}, nil
}

func createTextureFromImage(img image.Image, filter int) (*texture, error) {
	adjustedImage := adjustImageForTexture(img)
	size := adjustedImage.Bounds().Size()
	native := createNativeTexture(size.X, size.Y, adjustedImage.Pix, filter)
	return &texture{native, size.X, size.Y}, nil
}

func (t *texture) dispose() {
	t.native.Delete()
}
