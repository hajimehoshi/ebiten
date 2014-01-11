package opengl

// #cgo LDFLAGS: -framework OpenGL
//
// #include <stdlib.h>
// #include <OpenGL/gl.h>
import "C"

func enableAlphaBlending() {
	C.glEnable(C.GL_TEXTURE_2D)
	C.glEnable(C.GL_BLEND)
}

func flush() {
	C.glFlush()
}
