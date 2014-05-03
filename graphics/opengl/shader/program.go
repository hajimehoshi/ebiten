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

type program struct {
	native    C.GLuint
	shaderIds []shaderId
}

type programId int

const (
	programRegular programId = iota
	programColorMatrix
)

var programs = map[programId]*program{
	programRegular: &program{
		shaderIds: []shaderId{shaderVertex, shaderFragment},
	},
	programColorMatrix: &program{
		shaderIds: []shaderId{shaderVertex, shaderColorMatrix},
	},
}

func (p *program) create() {
	p.native = C.glCreateProgram()
	if p.native == 0 {
		panic("glCreateProgram failed")
	}

	for _, shaderId := range p.shaderIds {
		C.glAttachShader(p.native, shaders[shaderId].native)
	}
	C.glLinkProgram(p.native)
	linked := C.GLint(C.GL_FALSE)
	C.glGetProgramiv(p.native, C.GL_LINK_STATUS, &linked)
	if linked == C.GL_FALSE {
		panic("program error")
	}
}

func initialize() {
	for _, shader := range shaders {
		shader.compile()
	}
	defer func() {
		for _, shader := range shaders {
			shader.delete()
		}
	}()

	for _, program := range programs {
		program.create()
	}
}

type qualifierVariableType int

const (
	qualifierVariableTypeAttribute qualifierVariableType = iota
	qualifierVariableTypeUniform
)

var (
	shaderLocationCache = map[qualifierVariableType]map[string]C.GLint{
		qualifierVariableTypeAttribute: map[string]C.GLint{},
		qualifierVariableTypeUniform:   map[string]C.GLint{},
	}
)

func getLocation(program C.GLuint, name string, qvType qualifierVariableType) C.GLint {
	if location, ok := shaderLocationCache[qvType][name]; ok {
		return location
	}

	locationName := C.CString(name)
	defer C.free(unsafe.Pointer(locationName))

	const invalidLocation = -1
	location := C.GLint(invalidLocation)

	switch qvType {
	case qualifierVariableTypeAttribute:
		location = C.glGetAttribLocation(program, (*C.GLchar)(locationName))
	case qualifierVariableTypeUniform:
		location = C.glGetUniformLocation(program, (*C.GLchar)(locationName))
	default:
		panic("no reach")
	}
	if location == invalidLocation {
		panic("glGetUniformLocation failed")
	}
	shaderLocationCache[qvType][name] = location

	return location
}

func getAttributeLocation(program C.GLuint, name string) C.GLint {
	return getLocation(program, name, qualifierVariableTypeAttribute)
}

func getUniformLocation(program C.GLuint, name string) C.GLint {
	return getLocation(program, name, qualifierVariableTypeUniform)
}

func use(projectionMatrix [16]float32,
	geometryMatrix matrix.Geometry,
	colorMatrix matrix.Color) C.GLuint {
	programId := programRegular
	if !colorMatrix.IsIdentity() {
		programId = programColorMatrix
	}
	program := programs[programId]
	C.glUseProgram(program.native)

	C.glUniformMatrix4fv(C.GLint(getUniformLocation(program.native, "projection_matrix")),
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
	C.glUniformMatrix4fv(getUniformLocation(program.native, "modelview_matrix"),
		1, C.GL_FALSE,
		(*C.GLfloat)(&glModelviewMatrix[0]))

	C.glUniform1i(getUniformLocation(program.native, "texture"), 0)

	if programId != programColorMatrix {
		return program.native
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
	C.glUniformMatrix4fv(getUniformLocation(program.native, "color_matrix"),
		1, C.GL_FALSE, (*C.GLfloat)(&glColorMatrix[0]))

	glColorMatrixTranslation := [...]float32{
		e[0][4], e[1][4], e[2][4], e[3][4],
	}
	C.glUniform4fv(getUniformLocation(program.native, "color_matrix_translation"),
		1, (*C.GLfloat)(&glColorMatrixTranslation[0]))

	return program.native
}
