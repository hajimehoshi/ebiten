package shader

import (
	"github.com/go-gl/gl"
	"log"
)

type shader struct {
	native     gl.Shader
	shaderType gl.GLenum
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
	shaderVertex: {
		shaderType: gl.VERTEX_SHADER,
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
	shaderFragment: {
		shaderType: gl.FRAGMENT_SHADER,
		source: `
uniform sampler2D texture;
varying vec2 vertex_out_tex_coord;

void main(void) {
  gl_FragColor = texture2D(texture, vertex_out_tex_coord);
}
`,
	},
	shaderColorMatrix: {
		shaderType: gl.FRAGMENT_SHADER,
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
	shaderSolidColor: {
		shaderType: gl.FRAGMENT_SHADER,
		source: `
uniform vec4 color;

void main(void) {
  gl_FragColor = color;
}
`,
	},
}

func (s *shader) compile() {
	s.native = gl.CreateShader(s.shaderType)
	if s.native == 0 {
		panic("glCreateShader failed")
	}

	s.native.Source(s.source)
	s.native.Compile()

	if s.native.Get(gl.COMPILE_STATUS) == gl.FALSE {
		s.showShaderLog()
		panic("shader compile failed")
	}
}

func (s *shader) showShaderLog() {
	if s.native.Get(gl.INFO_LOG_LENGTH) == 0 {
		return
	}
	log.Fatalf("shader error: %s\n", s.native.GetInfoLog())
}

func (s *shader) delete() {
	s.native.Delete()
}
