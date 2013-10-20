package opengl

// #cgo LDFLAGS: -framework OpenGL
//
// #include <OpenGL/gl.h>
import "C"
import (
	"image"
	"image/draw"
	"unsafe"
)

func nextPowerOf2(x uint64) uint64 {
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
	native        C.GLuint
	width         int
	height        int
	textureWidth  int
	textureHeight int
}

type RenderTarget struct {
	texture     *Texture
	framebuffer C.GLuint
}

func createNativeTexture(textureWidth, textureHeight int, pixels unsafe.Pointer) C.GLuint {
	nativeTexture := C.GLuint(0)

	C.glGenTextures(1, (*C.GLuint)(&nativeTexture))
	if nativeTexture < 0 {
		panic("glGenTexture failed")
	}
	C.glPixelStorei(C.GL_UNPACK_ALIGNMENT, 4)
	C.glBindTexture(C.GL_TEXTURE_2D, C.GLuint(nativeTexture))

	C.glTexImage2D(C.GL_TEXTURE_2D, 0, C.GL_RGBA,
		C.GLsizei(textureWidth), C.GLsizei(textureHeight),
		0, C.GL_RGBA, C.GL_UNSIGNED_BYTE, pixels)

	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MAG_FILTER, C.GL_LINEAR)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MIN_FILTER, C.GL_LINEAR)
	C.glBindTexture(C.GL_TEXTURE_2D, 0)

	return nativeTexture
}

func createTexture(width, height int) *Texture {
	textureWidth := int(nextPowerOf2(uint64(width)))
	textureHeight := int(nextPowerOf2(uint64(height)))
	return &Texture{
		width:         width,
		height:        height,
		textureWidth:  textureWidth,
		textureHeight: textureHeight,
		native:        createNativeTexture(textureWidth, textureHeight, nil),
	}
}

func createTextureFromImage(img image.Image) *Texture {
	size := img.Bounds().Size()
	width, height := size.X, size.Y

	textureWidth := int(nextPowerOf2(uint64(width)))
	textureHeight := int(nextPowerOf2(uint64(height)))

	adjustedImageBound := image.Rectangle{
		image.ZP,
		image.Point{textureWidth, textureHeight},
	}
	adjustedImage := image.NewNRGBA(adjustedImageBound)
	dstBound := image.Rectangle{
		image.ZP,
		img.Bounds().Size(),
	}
	draw.Draw(adjustedImage, dstBound, img, image.ZP, draw.Src)
	pixelsPtr := unsafe.Pointer(&adjustedImage.Pix[0])
	nativeTexture := createNativeTexture(textureWidth, textureHeight, pixelsPtr)

	return &Texture{
		width:         width,
		height:        height,
		textureWidth:  textureWidth,
		textureHeight: textureHeight,
		native:        nativeTexture,
	}
}

func newRenderTarget(width, height int) *RenderTarget {
	texture := createTexture(width, height)
	framebuffer := createFramebuffer(texture.native)
	return &RenderTarget{
		texture:     texture,
		framebuffer: framebuffer,
	}
}

func newTextureFromImage(img image.Image) (*Texture, error) {
	return createTextureFromImage(img), nil
}

func newRenderTargetWithFramebuffer(width, height int,
	framebuffer C.GLuint) *RenderTarget {
	texture := &Texture{
		width:         width,
		height:        height,
		textureWidth:  int(nextPowerOf2(uint64(width))),
		textureHeight: int(nextPowerOf2(uint64(height))),
	}
	return &RenderTarget{
		texture:     texture,
		framebuffer: framebuffer,
	}
}

func createFramebuffer(nativeTexture C.GLuint) C.GLuint {
	framebuffer := C.GLuint(0)
	C.glGenFramebuffers(1, &framebuffer)

	origFramebuffer := C.GLint(0)
	C.glGetIntegerv(C.GL_FRAMEBUFFER_BINDING, &origFramebuffer)
	C.glBindFramebuffer(C.GL_FRAMEBUFFER, framebuffer)
	C.glFramebufferTexture2D(C.GL_FRAMEBUFFER, C.GL_COLOR_ATTACHMENT0,
		C.GL_TEXTURE_2D, nativeTexture, 0)
	C.glBindFramebuffer(C.GL_FRAMEBUFFER, C.GLuint(origFramebuffer))
	if C.glCheckFramebufferStatus(C.GL_FRAMEBUFFER) !=
		C.GL_FRAMEBUFFER_COMPLETE {
		panic("creating framebuffer failed")
	}

	return framebuffer
}
