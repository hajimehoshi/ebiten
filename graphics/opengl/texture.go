// Copyright 2013 Hajime Hoshi
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package opengl

// #cgo LDFLAGS: -framework OpenGL
//
// #include <OpenGL/gl.h>
import "C"
import (
	"image"
	"unsafe"
)

func clp2(x uint64) uint64 {
	x -= 1
	x |= (x >> 1)
	x |= (x >> 2)
	x |= (x >> 4)
	x |= (x >> 8)
	x |= (x >> 16)
	x |= (x >> 32)
	return x + 1
}

type Texture struct {
	id            C.GLuint
	width         int
	height        int
	textureWidth  int
	textureHeight int
}

func createTexture(width, height int, pixels []uint8) *Texture {
	textureWidth := int(clp2(uint64(width)))
	textureHeight := int(clp2(uint64(height)))
	if pixels != nil {
		if width != textureWidth {
			panic("sorry, but width should be power of 2")
		}
		if height != textureHeight {
			panic("sorry, but height should be power of 2")
		}
	}
	texture := &Texture{
		id:            0,
		width:         width,
		height:        height,
		textureWidth:  textureWidth,
		textureHeight: textureHeight,
	}

	textureID := C.GLuint(0)
	C.glGenTextures(1, (*C.GLuint)(&textureID))
	if textureID == 0 {
		panic("glGenTexture failed")
	}
	C.glPixelStorei(C.GL_UNPACK_ALIGNMENT, 4)
	C.glBindTexture(C.GL_TEXTURE_2D, C.GLuint(textureID))

	ptr := unsafe.Pointer(nil)
	if pixels != nil {
		ptr = unsafe.Pointer(&pixels[0])
	}
	C.glTexImage2D(C.GL_TEXTURE_2D, 0, C.GL_RGBA,
		C.GLsizei(textureWidth), C.GLsizei(textureHeight),
		0, C.GL_RGBA, C.GL_UNSIGNED_BYTE, ptr)

	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MAG_FILTER, C.GL_LINEAR)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MIN_FILTER, C.GL_LINEAR)
	C.glBindTexture(C.GL_TEXTURE_2D, 0)

	texture.id = textureID

	return texture
}

type textureError string

func (err textureError) Error() string {
	return "Texture Error: " + string(err)
}

func newTexture(width, height int) *Texture {
	return createTexture(width, height, nil)
}

func newTextureFromImage(img image.Image) (*Texture, error) {
	var pix []uint8
	switch img.(type) {
	case *image.RGBA:
		pix = img.(*image.RGBA).Pix
	case *image.NRGBA:
		pix = img.(*image.NRGBA).Pix
	default:
		return nil, textureError("image format must be RGBA or NRGBA")
	}
	size := img.Bounds().Size()
	return createTexture(size.X, size.Y, pix), nil
}
