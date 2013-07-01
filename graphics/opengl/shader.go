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
	regularShaderProgram     = C.GLuint(0)
	colorMatrixShaderProgram = C.GLuint(0)
)

func (s *shader) compile() {
	csource := (*C.GLchar)(C.CString(s.source))
	// TODO: defer?
	//defer C.free(unsafe.Pointer(csource))

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

func createProgram(shaders ...*shader) C.GLuint {
	program := C.glCreateProgram()
	for _, shader := range shaders {
		C.glAttachShader(program, shader.id)
	}
	C.glLinkProgram(program)
	linked := C.GLint(C.GL_FALSE)
	C.glGetProgramiv(program, C.GL_LINK_STATUS, &linked)
	if linked == C.GL_FALSE {
		panic("program error")
	}
	return program
}

func initializeShaders() {
	// TODO: when should this function be called?
	vertexShader.id = C.glCreateShader(C.GL_VERTEX_SHADER)
	if vertexShader.id == 0 {
		panic("creating shader failed")
	}
	fragmentShader.id = C.glCreateShader(C.GL_FRAGMENT_SHADER)
	if fragmentShader.id == 0 {
		panic("creating shader failed")
	}
	colorMatrixShader.id = C.glCreateShader(C.GL_FRAGMENT_SHADER)
	if colorMatrixShader.id == 0 {
		panic("creating shader failed")
	}

	vertexShader.compile()
	fragmentShader.compile()
	colorMatrixShader.compile()

	regularShaderProgram = createProgram(vertexShader, fragmentShader)
	colorMatrixShaderProgram = createProgram(vertexShader, colorMatrixShader)

	C.glDeleteShader(vertexShader.id)
	C.glDeleteShader(fragmentShader.id)
	C.glDeleteShader(colorMatrixShader.id)
}

const (
	qualifierVariableTypeAttribute = iota
	qualifierVariableTypeUniform
)

var (
	shaderLocationCache = map[int]map[string]C.GLint{
		qualifierVariableTypeAttribute: map[string]C.GLint{},
		qualifierVariableTypeUniform:   map[string]C.GLint{},
	}
)

func getLocation(program C.GLuint, name string, qualifierVariableType int) C.GLint {
	if location, ok := shaderLocationCache[qualifierVariableType][name]; ok {
		return location
	}

	locationName := C.CString(name)
	defer C.free(unsafe.Pointer(locationName))

	location := C.GLint(-1)
	switch qualifierVariableType {
	case qualifierVariableTypeAttribute:
		location = C.glGetAttribLocation(program, (*C.GLchar)(locationName))
	case qualifierVariableTypeUniform:
		location = C.glGetUniformLocation(program, (*C.GLchar)(locationName))
	default:
		panic("no reach")
	}
	if location == -1 {
		panic("glGetUniformLocation failed")
	}
	shaderLocationCache[qualifierVariableType][name] = location

	return location
}

func getAttributeLocation(program C.GLuint, name string) C.GLint {
	return getLocation(program, name, qualifierVariableTypeAttribute)
}

func getUniformLocation(program C.GLuint, name string) C.GLint {
	return getLocation(program, name, qualifierVariableTypeUniform)
}
