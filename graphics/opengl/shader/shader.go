package shader

// #cgo LDFLAGS: -framework OpenGL
//
// #include <OpenGL/gl.h>
// #include <stdlib.h>
import "C"
import (
	"unsafe"
)

type shader struct {
	id     C.GLuint
	name   string
	source string
}

var (
	vertexShader = &shader{
		id:   0,
		name: "vertex_shader",
		source: `
attribute /*highp*/ vec2 vertex;
attribute /*highp*/ vec2 texture;
uniform /*highp*/ mat4 projection_matrix;
uniform /*highp*/ mat4 modelview_matrix;
varying /*highp*/ vec2 tex_coord;

void main(void) {
  tex_coord = texture;
  gl_Position = projection_matrix * modelview_matrix * vec4(vertex, 0, 1);
}
`,
	}
	fragmentShader = &shader{
		id:   0,
		name: "fragment_shader",
		source: `
uniform /*lowp*/ sampler2D texture;
varying /*highp*/ vec2 tex_coord;

void main(void) {
  gl_FragColor = texture2D(texture, tex_coord);
}
`,
	}
	colorMatrixShader = &shader{
		id:   0,
		name: "color_matrix_shader",
		source: `
uniform /*highp*/ sampler2D texture;
uniform /*lowp*/ mat4 color_matrix;
uniform /*lowp*/ vec4 color_matrix_translation;
varying /*highp*/ vec2 tex_coord;

void main(void) {
  /*lowp*/ vec4 color = texture2D(texture, tex_coord);
  gl_FragColor = (color_matrix * color) + color_matrix_translation;
}
`,
	}
)

var (
	programRegular     = C.GLuint(0)
	programColorMatrix = C.GLuint(0)
)

func (s *shader) compile() {
	csource := (*C.GLchar)(C.CString(s.source))
	// TODO: defer?
	// defer C.free(unsafe.Pointer(csource))

	C.glShaderSource(s.id, 1, &csource, nil)
	C.glCompileShader(s.id)

	compiled := C.GLint(C.GL_FALSE)
	C.glGetShaderiv(s.id, C.GL_COMPILE_STATUS, &compiled)
	if compiled == C.GL_FALSE {
		s.showShaderLog()
		panic("shader compile failed: " + s.name)
	}
}

func (s *shader) showShaderLog() {
	logSize := C.GLint(0)
	C.glGetShaderiv(s.id, C.GL_INFO_LOG_LENGTH, &logSize)
	if logSize == 0 {
		return
	}
	length := C.GLsizei(0)
	buffer := make([]C.GLchar, logSize)
	C.glGetShaderInfoLog(s.id, C.GLsizei(logSize), &length, &buffer[0])

	message := string(C.GoBytes(unsafe.Pointer(&buffer[0]), C.int(length)))
	print("shader error (", s.name, "):\n", message)
}
