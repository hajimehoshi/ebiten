package rendertarget

// #cgo LDFLAGS: -framework OpenGL
//
// #include <OpenGL/gl.h>
import "C"
import (
	"github.com/hajimehoshi/go-ebiten/graphics/opengl/texture"
	gtexture "github.com/hajimehoshi/go-ebiten/graphics/texture"
)

type Framebuffer C.GLuint

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

// TODO: Rename them
func New(width, height int, filter texture.Filter) (
	*gtexture.RenderTarget, *gtexture.Texture, error) {
	tex, err := texture.New(width, height, filter)
	if err != nil {
		return nil, nil, err
	}
	f := func(native interface{}) interface{} {
		return createFramebuffer(C.GLuint(native.(texture.Native)))
	}
	return gtexture.NewRenderTarget(tex, f), tex, nil
}

func NewWithFramebuffer(width, height int, framebuffer Framebuffer) (
	*gtexture.RenderTarget, error) {
	tex, err := texture.NewEmpty(width, height)
	if err != nil {
		return nil, err
	}
	f := func(native interface{}) interface{} {
		return framebuffer
	}
	return gtexture.NewRenderTarget(tex, f), nil
}
