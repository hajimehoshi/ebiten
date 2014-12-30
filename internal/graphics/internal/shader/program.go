// Copyright 2014 Hajime Hoshi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package shader

import (
	"errors"
	"github.com/go-gl/gl"
)

type program struct {
	native    gl.Program
	shaderIds []shaderId
}

var programColorMatrix = program{
	shaderIds: []shaderId{shaderVertex, shaderColorMatrix},
}

func (p *program) create() error {
	p.native = gl.CreateProgram()
	if p.native == 0 {
		return errors.New("glCreateProgram failed")
	}

	for _, shaderId := range p.shaderIds {
		p.native.AttachShader(shaders[shaderId].native)
	}
	p.native.Link()
	if p.native.Get(gl.LINK_STATUS) == gl.FALSE {
		return errors.New("program error")
	}
	return nil
}

func initialize() error {
	for _, shader := range shaders {
		if err := shader.compile(); err != nil {
			return err
		}
	}
	defer func() {
		for _, shader := range shaders {
			shader.delete()
		}
	}()

	return programColorMatrix.create()
}

func getAttributeLocation(program gl.Program, name string) gl.AttribLocation {
	return program.GetAttribLocation(name)
}

func getUniformLocation(program gl.Program, name string) gl.UniformLocation {
	return program.GetUniformLocation(name)
}

var lastProgram gl.Program = 0

func useProgramColorMatrix(projectionMatrix [16]float32, geo Matrix, color Matrix) gl.Program {
	if lastProgram != programColorMatrix.native {
		programColorMatrix.native.Use()
		lastProgram = programColorMatrix.native
	}
	// TODO: Check the performance.
	program := programColorMatrix

	getUniformLocation(program.native, "projection_matrix").UniformMatrix4fv(false, projectionMatrix)

	a := float32(geo.Element(0, 0))
	b := float32(geo.Element(0, 1))
	c := float32(geo.Element(1, 0))
	d := float32(geo.Element(1, 1))
	tx := float32(geo.Element(0, 2))
	ty := float32(geo.Element(1, 2))
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
			e[i][j] = float32(color.Element(i, j))
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
