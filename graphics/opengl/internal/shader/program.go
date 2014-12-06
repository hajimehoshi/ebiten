package shader

import (
	"github.com/go-gl/gl"
	"github.com/hajimehoshi/ebiten/graphics/matrix"
)

type program struct {
	native    gl.Program
	shaderIds []shaderId
}

type programId int

const (
	programRegular programId = iota
	programColorMatrix
)

var programs = map[programId]*program{
	// TODO: programRegular is not used for now. Remove this.
	programRegular: &program{
		shaderIds: []shaderId{shaderVertex, shaderFragment},
	},
	programColorMatrix: &program{
		shaderIds: []shaderId{shaderVertex, shaderColorMatrix},
	},
}

func (p *program) create() {
	p.native = gl.CreateProgram()
	if p.native == 0 {
		panic("glCreateProgram failed")
	}

	for _, shaderId := range p.shaderIds {
		p.native.AttachShader(shaders[shaderId].native)
	}
	p.native.Link()
	if p.native.Get(gl.LINK_STATUS) == gl.FALSE {
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
	shaderLocationCache = map[qualifierVariableType]map[string]gl.AttribLocation{
		qualifierVariableTypeAttribute: map[string]gl.AttribLocation{},
		qualifierVariableTypeUniform:   map[string]gl.AttribLocation{},
	}
)

func getLocation(program gl.Program, name string, qvType qualifierVariableType) gl.AttribLocation {
	if location, ok := shaderLocationCache[qvType][name]; ok {
		return location
	}

	location := gl.AttribLocation(-1)
	switch qvType {
	case qualifierVariableTypeAttribute:
		location = program.GetAttribLocation(name)
	case qualifierVariableTypeUniform:
		location = gl.AttribLocation(program.GetUniformLocation(name))
	default:
		panic("no reach")
	}
	if location == -1 {
		panic("GetAttribLocation failed")
	}
	shaderLocationCache[qvType][name] = location

	return location
}

func getAttributeLocation(program gl.Program, name string) gl.AttribLocation {
	return getLocation(program, name, qualifierVariableTypeAttribute)
}

func getUniformLocation(program gl.Program, name string) gl.UniformLocation {
	return gl.UniformLocation(getLocation(program, name, qualifierVariableTypeUniform))
}

func use(projectionMatrix [16]float32, geometryMatrix matrix.Geometry, colorMatrix matrix.Color) gl.Program {
	programId :=  programColorMatrix
	program := programs[programId]
	// TODO: Check the performance.
	program.native.Use()

	getUniformLocation(program.native, "projection_matrix").UniformMatrix4fv(false, projectionMatrix)

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
	getUniformLocation(program.native, "modelview_matrix").UniformMatrix4fv(false, glModelviewMatrix)

	getUniformLocation(program.native, "texture").Uniform1i(0)

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
	getUniformLocation(program.native, "color_matrix").UniformMatrix4fv(false, glColorMatrix)
	glColorMatrixTranslation := [...]float32{
		e[0][4], e[1][4], e[2][4], e[3][4],
	}
	getUniformLocation(program.native, "color_matrix_translation").Uniform4fv(1, glColorMatrixTranslation[:])

	return program.native
}
