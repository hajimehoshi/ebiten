package shader

// #cgo LDFLAGS: -framework OpenGL
//
// #include <OpenGL/gl.h>
// #include <stdlib.h>
import "C"
import (
	"fmt"
	"unsafe"
)

type shader struct {
	native     C.GLuint
	shaderType C.GLenum
	source     string
}

type shaderId int

const (
	shaderVertex shaderId = iota
	shaderFragment
	shaderColorMatrix
	shaderSolidColor
)

var shaders = map[shaderId]*shader{
	shaderVertex: &shader{
		shaderType: C.GL_VERTEX_SHADER,
		source: `
uniform mat4 projection_matrix;
uniform mat4 modelview_matrix;
attribute vec2 vertex;
attribute vec2 tex_coord;
varying vec2 vertex_out_tex_coord;

void main(void) {
  vertex_out_tex_coord = tex_coord;
  gl_Position = projection_matrix * modelview_matrix * vec4(vertex, 0, 1);
}
`,
	},
	shaderFragment: &shader{
		shaderType: C.GL_FRAGMENT_SHADER,
		source: `
uniform sampler2D texture;
varying vec2 vertex_out_tex_coord;

void main(void) {
  gl_FragColor = texture2D(texture, vertex_out_tex_coord);
}
`,
	},
	shaderColorMatrix: &shader{
		shaderType: C.GL_FRAGMENT_SHADER,
		source: `
uniform sampler2D texture;
uniform mat4 color_matrix;
uniform vec4 color_matrix_translation;
varying vec2 vertex_out_tex_coord;

void main(void) {
  vec4 color = texture2D(texture, vertex_out_tex_coord);
  gl_FragColor = (color_matrix * color) + color_matrix_translation;
}
`,
	},
	shaderSolidColor: &shader{
		shaderType: C.GL_FRAGMENT_SHADER,
		source: `
uniform vec4 color;

void main(void) {
  gl_FragColor = color;
}
`,
	},
}

func (s *shader) compile() {
	s.native = C.glCreateShader(s.shaderType)
	if s.native == 0 {
		panic("glCreateShader failed")
	}

	csource := (*C.GLchar)(C.CString(s.source))
	// TODO: defer?
	// defer C.free(unsafe.Pointer(csource))

	C.glShaderSource(s.native, 1, &csource, nil)
	C.glCompileShader(s.native)

	compiled := C.GLint(C.GL_FALSE)
	C.glGetShaderiv(s.native, C.GL_COMPILE_STATUS, &compiled)
	if compiled == C.GL_FALSE {
		s.showShaderLog()
		panic("shader compile failed")
	}
}

func (s *shader) showShaderLog() {
	logSize := C.GLint(0)
	C.glGetShaderiv(s.native, C.GL_INFO_LOG_LENGTH, &logSize)
	if logSize == 0 {
		return
	}
	length := C.GLsizei(0)
	buffer := make([]C.GLchar, logSize)
	C.glGetShaderInfoLog(s.native, C.GLsizei(logSize), &length, &buffer[0])

	message := string(C.GoBytes(unsafe.Pointer(&buffer[0]), C.int(length)))
	fmt.Printf("shader error: %s\n", message)
}

func (s *shader) delete() {
	C.glDeleteShader(s.native)
}
