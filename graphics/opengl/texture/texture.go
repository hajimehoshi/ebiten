package texture

// #cgo LDFLAGS: -framework OpenGL
//
// #include <OpenGL/gl.h>
import "C"
import (
	"github.com/hajimehoshi/go-ebiten/graphics"
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/shader"
	"image"
	"unsafe"
)

type Framebuffer C.GLuint

type Texture struct {
	native shader.NativeTexture
	width  int
	height int
}

func createNativeTexture(textureWidth, textureHeight int, pixels []uint8,
	filter graphics.Filter) shader.NativeTexture {
	nativeTexture := C.GLuint(0)

	C.glGenTextures(1, (*C.GLuint)(&nativeTexture))
	if nativeTexture < 0 {
		panic("glGenTexture failed")
	}
	C.glPixelStorei(C.GL_UNPACK_ALIGNMENT, 4)
	C.glBindTexture(C.GL_TEXTURE_2D, C.GLuint(nativeTexture))
	defer C.glBindTexture(C.GL_TEXTURE_2D, 0)

	glFilter := C.GLint(0)
	switch filter {
	case graphics.FilterLinear:
		glFilter = C.GL_LINEAR
	case graphics.FilterNearest:
		glFilter = C.GL_NEAREST
	default:
		panic("not reached")
	}
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MAG_FILTER, glFilter)
	C.glTexParameteri(C.GL_TEXTURE_2D, C.GL_TEXTURE_MIN_FILTER, glFilter)

	ptr := unsafe.Pointer(nil)
	if pixels != nil {
		ptr = unsafe.Pointer(&pixels[0])
	}
	C.glTexImage2D(C.GL_TEXTURE_2D, 0, C.GL_RGBA,
		C.GLsizei(textureWidth), C.GLsizei(textureHeight),
		0, C.GL_RGBA, C.GL_UNSIGNED_BYTE, ptr)

	return shader.NativeTexture(nativeTexture)
}

func Create(width, height int, filter graphics.Filter) (*Texture, error) {
	native := createNativeTexture(
		graphics.AdjustSizeForTexture(width),
		graphics.AdjustSizeForTexture(height), nil, filter)
	return &Texture{native, width, height}, nil
}

func CreateFromImage(img image.Image, filter graphics.Filter) (*Texture, error) {
	adjustedImage := graphics.AdjustImageForTexture(img)
	size := adjustedImage.Bounds().Size()
	native := createNativeTexture(size.X, size.Y, adjustedImage.Pix, filter)
	return &Texture{native, size.X, size.Y}, nil
}

func createFramebuffer(nativeTexture C.GLuint) Framebuffer {
	framebuffer := C.GLuint(0)
	C.glGenFramebuffers(1, &framebuffer)

	origFramebuffer := C.GLint(0)
	C.glGetIntegerv(C.GL_FRAMEBUFFER_BINDING, &origFramebuffer)

	C.glBindFramebuffer(C.GL_FRAMEBUFFER, framebuffer)
	defer C.glBindFramebuffer(C.GL_FRAMEBUFFER, C.GLuint(origFramebuffer))

	C.glFramebufferTexture2D(C.GL_FRAMEBUFFER, C.GL_COLOR_ATTACHMENT0,
		C.GL_TEXTURE_2D, nativeTexture, 0)
	if C.glCheckFramebufferStatus(C.GL_FRAMEBUFFER) !=
		C.GL_FRAMEBUFFER_COMPLETE {
		panic("creating framebuffer failed")
	}

	// Set this framebuffer opaque because alpha values on a target might be
	// confusing.
	C.glClearColor(0, 0, 0, 1)
	C.glClear(C.GL_COLOR_BUFFER_BIT)

	return Framebuffer(framebuffer)
}

func (t *Texture) CreateFramebuffer() Framebuffer {
	return createFramebuffer(C.GLuint(t.native))
}

func (t *Texture) Draw(projectionMatrix [16]float32, geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	quad := graphics.TextureQuadForTexture(t.width, t.height)
	shader.DrawTexture(t.native,
		projectionMatrix, []graphics.TextureQuad{quad},
		geometryMatrix, colorMatrix)
}

func (t *Texture) DrawParts(parts []graphics.TexturePart, projectionMatrix [16]float32,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	quads := graphics.TextureQuadsForTextureParts(parts, t.width, t.height)
	shader.DrawTexture(t.native,
		projectionMatrix, quads,
		geometryMatrix, colorMatrix)
}
