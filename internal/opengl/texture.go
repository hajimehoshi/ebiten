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
	"errors"
	"fmt"
	"github.com/go-gl/gl"
	"github.com/hajimehoshi/ebiten/internal"
	"image"
	"image/draw"
)

func adjustImageForTexture(img image.Image) *image.RGBA {
	width, height := img.Bounds().Size().X, img.Bounds().Size().Y
	adjustedImageBounds := image.Rectangle{
		image.ZP,
		image.Point{
			internal.NextPowerOf2Int(width),
			internal.NextPowerOf2Int(height),
		},
	}
	if nrgba, ok := img.(*image.RGBA); ok && img.Bounds() == adjustedImageBounds {
		return nrgba
	}

	adjustedImage := image.NewRGBA(adjustedImageBounds)
	dstBounds := image.Rectangle{
		image.ZP,
		img.Bounds().Size(),
	}
	draw.Draw(adjustedImage, dstBounds, img, image.ZP, draw.Src)
	return adjustedImage
}

type Texture struct {
	native gl.Texture
	width  int
	height int
}

func (t *Texture) Native() gl.Texture {
	return t.native
}

func (t *Texture) Size() (width, height int) {
	return t.width, t.height
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

func NewTexture(width, height int, filter int) (*Texture, error) {
	w := internal.NextPowerOf2Int(width)
	h := internal.NextPowerOf2Int(height)
	if w < 4 {
		return nil, errors.New("width must be equal or more than 4.")
	}
	if h < 4 {
		return nil, errors.New("height must be equal or more than 4.")
	}
	native := createNativeTexture(w, h, nil, filter)
	return &Texture{native, width, height}, nil
}

func NewTextureFromImage(img image.Image, filter int) (*Texture, error) {
	origSize := img.Bounds().Size()
	if origSize.X < 4 {
		return nil, errors.New("width must be equal or more than 4.")
	}
	if origSize.Y < 4 {
		return nil, errors.New("height must be equal or more than 4.")
	}
	adjustedImage := adjustImageForTexture(img)
	size := adjustedImage.Bounds().Size()
	native := createNativeTexture(size.X, size.Y, adjustedImage.Pix, filter)
	return &Texture{native, origSize.X, origSize.Y}, nil
}

func (t *Texture) Dispose() {
	t.native.Delete()
}

func (t *Texture) Pixels() ([]uint8, error) {
	w, h := internal.NextPowerOf2Int(t.width), internal.NextPowerOf2Int(t.height)
	pixels := make([]uint8, 4*w*h)
	t.native.Bind(gl.TEXTURE_2D)
	gl.GetTexImage(gl.TEXTURE_2D, 0, gl.RGBA, gl.UNSIGNED_BYTE, pixels)
	if e := gl.GetError(); e != gl.NO_ERROR {
		// TODO: Use glu.ErrorString
		return nil, errors.New(fmt.Sprintf("gl error: %d", e))
	}
	return pixels, nil
}
