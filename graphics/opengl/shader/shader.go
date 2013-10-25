package shader

// #cgo LDFLAGS: -framework OpenGL
//
// #include <OpenGL/gl.h>
// #include <stdlib.h>
import "C"
import (
	"github.com/hajimehoshi/go-ebiten/graphics/matrix"
	"unsafe"
)

type Program int

const (
	ProgramRegular Program = iota
	ProgramColorMatrix
)

type Location int

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

func Initialize() {
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

func toInnerProgram(program Program) C.GLuint {
	switch program {
	case ProgramRegular:
		return regularShaderProgram
	case ProgramColorMatrix:
		return colorMatrixShaderProgram
	default:
		panic("no reach")
	}
	return C.GLuint(0)
}

func getLocation(program Program, name string, qualifierVariableType int) int {
	if location, ok := shaderLocationCache[qualifierVariableType][name]; ok {
		return int(location)
	}

	locationName := C.CString(name)
	defer C.free(unsafe.Pointer(locationName))

	location := C.GLint(-1)
	innerProgram := toInnerProgram(program)
	
	switch qualifierVariableType {
	case qualifierVariableTypeAttribute:
		location = C.glGetAttribLocation(innerProgram, (*C.GLchar)(locationName))
	case qualifierVariableTypeUniform:
		location = C.glGetUniformLocation(innerProgram, (*C.GLchar)(locationName))
	default:
		panic("no reach")
	}
	if location == -1 {
		panic("glGetUniformLocation failed")
	}
	shaderLocationCache[qualifierVariableType][name] = location

	return int(location)
}

func GetAttributeLocation(program Program, name string) int {
	return getLocation(program, name, qualifierVariableTypeAttribute)
}

func getUniformLocation(program Program, name string) int {
	return getLocation(program, name, qualifierVariableTypeUniform)
}

func Use(projectionMatrix [16]float32, geometryMatrix matrix.Geometry, colorMatrix matrix.Color) Program {
	program := ProgramRegular
	if !colorMatrix.IsIdentity() {
		program = ProgramColorMatrix
	}
	C.glUseProgram(toInnerProgram(program))
	
	C.glUniformMatrix4fv(C.GLint(getUniformLocation(program, "projection_matrix")),
		1, C.GL_FALSE, (*C.GLfloat)(&projectionMatrix[0]))

	a := float32(geometryMatrix.Elements[0][0])
	b := float32(geometryMatrix.Elements[0][1])
	c := float32(geometryMatrix.Elements[1][0])
	d := float32(geometryMatrix.Elements[1][1])
	tx := float32(geometryMatrix.Elements[0][2])
	ty := float32(geometryMatrix.Elements[1][2])
	glModelviewMatrix := [...]float32{
		a, c, 0, 0,
		b, d, 0, 0,
		0, 0, 1, 0,
		tx, ty, 0, 1,
	}
	C.glUniformMatrix4fv(C.GLint(getUniformLocation(program, "modelview_matrix")),
		1, C.GL_FALSE,
		(*C.GLfloat)(&glModelviewMatrix[0]))

	C.glUniform1i(C.GLint(getUniformLocation(program, "texture")), 0)

	if program != ProgramColorMatrix {
		return program
	}

	e := [4][5]float32{}
	for i := 0; i < 4; i++ {
		for j := 0; j < 5; j++ {
			e[i][j] = float32(colorMatrix.Elements[i][j])
		}
	}

	glColorMatrix := [...]float32{
		e[0][0], e[1][0], e[2][0], e[3][0],
		e[0][1], e[1][1], e[2][1], e[3][1],
		e[0][2], e[1][2], e[2][2], e[3][2],
		e[0][3], e[1][3], e[2][3], e[3][3],
	}
	C.glUniformMatrix4fv(C.GLint(getUniformLocation(program, "color_matrix")),
		1, C.GL_FALSE, (*C.GLfloat)(&glColorMatrix[0]))

	glColorMatrixTranslation := [...]float32{
		e[0][4], e[1][4], e[2][4], e[3][4],
	}
	C.glUniform4fv(C.GLint(getUniformLocation(program, "color_matrix_translation")),
		1, (*C.GLfloat)(&glColorMatrixTranslation[0]))

	return program
}
