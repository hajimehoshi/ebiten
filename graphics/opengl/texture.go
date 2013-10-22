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
	native interface{}
	width  int
	height int
}

func (texture *Texture) Native() interface{} {
	return texture.native
}

func (texture *Texture) textureWidth() int {
	return int(nextPowerOf2(uint64(texture.width)))
}

func (texture *Texture) textureHeight() int {
	return int(nextPowerOf2(uint64(texture.height)))
}

func (texture *Texture) U(x int) float64 {
	return float64(x) / float64(texture.textureWidth())
}

func (texture *Texture) V(y int) float64 {
	return float64(y) / float64(texture.textureHeight())
}

type RenderTarget struct {
	texture     *Texture
	framebuffer C.GLuint
}

func (renderTarget *RenderTarget) SetAsViewport(setter interface{
	SetViewport(x, y, width, height int)
}) {
	texture := renderTarget.texture
	x, y, width, height := 0, 0, texture.textureWidth(), texture.textureHeight()
	setter.SetViewport(x, y, width, height)
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
	texture := &Texture{
		width:  width,
		height: height,
	}
	texture.native = createNativeTexture(texture.textureWidth(), texture.textureHeight(), nil)
	return texture
}

func newRenderTarget(width, height int) (*RenderTarget, error) {
	texture := createTexture(width, height)
	framebuffer := createFramebuffer(texture.Native().(C.GLuint))
	return &RenderTarget{
		texture:     texture,
		framebuffer: framebuffer,
	}, nil
}

func newTextureFromImage(img image.Image) (*Texture, error) {
	size := img.Bounds().Size()
	width, height := size.X, size.Y

	texture := &Texture{
		width:         width,
		height:        height,
	}

	adjustedImageBound := image.Rectangle{
		image.ZP,
		image.Point{texture.textureWidth(), texture.textureHeight()},
	}
	adjustedImage := image.NewNRGBA(adjustedImageBound)
	dstBound := image.Rectangle{
		image.ZP,
		img.Bounds().Size(),
	}
	draw.Draw(adjustedImage, dstBound, img, image.ZP, draw.Src)
	pixelsPtr := unsafe.Pointer(&adjustedImage.Pix[0])
	texture.native = createNativeTexture(
		texture.textureWidth(), texture.textureHeight(), pixelsPtr)
	return texture, nil
}

func newRenderTargetWithFramebuffer(width, height int,
	framebuffer C.GLuint) *RenderTarget {
	texture := &Texture{
		width:  width,
		height: height,
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
