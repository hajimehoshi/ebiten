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
// #include <stdlib.h>
// #include <OpenGL/gl.h>
import "C"
import (
	"fmt"
	"github.com/hajimehoshi/go.ebiten/graphics"
	"github.com/hajimehoshi/go.ebiten/graphics/matrix"
	"image"
	"image/color"
	"math"
	"unsafe"
)

type Context struct {
	screen                 graphics.Texture
	screenScale            int
	textures               map[graphics.TextureID]*Texture
	currentOffscreenWidth  int
	currentOffscreenHeight int
	projectionMatrix       [16]float32
	currentShaderProgram   C.GLuint
	mainFramebuffer        C.GLuint
	mainFramebufferTexture *Texture
	framebuffers           map[C.GLuint]C.GLuint
}

// This method should be called on the UI thread.
func newContext(screenWidth, screenHeight, screenScale int) *Context {
	context := &Context{
		screenScale:     screenScale,
		textures:        map[graphics.TextureID]*Texture{},
		mainFramebuffer: 0,
		framebuffers:    map[C.GLuint]C.GLuint{},
	}
	// main framebuffer should be created sooner than any other framebuffers!
	mainFramebuffer := C.GLint(0)
	C.glGetIntegerv(C.GL_FRAMEBUFFER_BINDING, &mainFramebuffer)
	context.mainFramebuffer = C.GLuint(mainFramebuffer)

	context.mainFramebufferTexture = newVirtualTexture(
		screenWidth * screenScale,
		screenHeight * screenScale)

	initializeShaders()

	context.screen = context.NewTexture(screenWidth, screenHeight)

	return context
}

func (context *Context) Screen() graphics.Texture {
	return context.screen
}

func (context *Context) Clear() {
	C.glClearColor(0, 0, 0, 0)
	C.glClear(C.GL_COLOR_BUFFER_BIT)
}

func (context *Context) Fill(clr color.Color) {
	r, g, b, a := clr.RGBA()
	max := float64(math.MaxUint16)
	C.glClearColor(
		C.GLclampf(float64(r)/max),
		C.GLclampf(float64(g)/max),
		C.GLclampf(float64(b)/max),
		C.GLclampf(float64(a)/max))
	C.glClear(C.GL_COLOR_BUFFER_BIT)
}

func (context *Context) DrawRect(rect graphics.Rect, clr color.Color) {
	width := float32(context.currentOffscreenWidth)
	height := float32(context.currentOffscreenHeight)
	textureWidth := float32(clp2(uint64(width)))
	textureHeight := float32(clp2(uint64(height)))

	// Normalize the coord between -1.0 and 1.0.
	x1 := float32(rect.X)/textureWidth*2.0 - 1.0
	x2 := float32(rect.X+rect.Width)/textureHeight*2.0 - 1.0
	y1 := float32(rect.Y)/textureHeight*2.0 - 1.0
	y2 := float32(rect.Y+rect.Height)/textureHeight*2.0 - 1.0
	vertex := [...]float32{
		x1, y1,
		x2, y1,
		x1, y2,
		x2, y2,
	}

	origR, origG, origB, origA := clr.RGBA()
	max := float32(math.MaxUint16)
	r := float32(origR) / max
	g := float32(origG) / max
	b := float32(origB) / max
	a := float32(origA) / max
	color := [...]float32{
		r, g, b, a,
		r, g, b, a,
		r, g, b, a,
		r, g, b, a,
	}

	C.glUseProgram(0)
	context.currentShaderProgram = 0
	C.glDisable(C.GL_TEXTURE_2D)
	C.glEnableClientState(C.GL_VERTEX_ARRAY)
	C.glEnableClientState(C.GL_COLOR_ARRAY)
	C.glVertexPointer(2, C.GL_FLOAT, C.GL_FALSE, unsafe.Pointer(&vertex[0]))
	C.glColorPointer(4, C.GL_FLOAT, C.GL_FALSE, unsafe.Pointer(&color[0]))
	C.glDrawArrays(C.GL_TRIANGLE_STRIP, 0, 4)
	C.glDisableClientState(C.GL_COLOR_ARRAY)
	C.glDisableClientState(C.GL_VERTEX_ARRAY)
	C.glEnable(C.GL_TEXTURE_2D)

	if glError := C.glGetError(); glError != C.GL_NO_ERROR {
		panic("OpenGL error")
	}
}

func (context *Context) DrawTexture(
	textureID graphics.TextureID,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	texture := context.textures[textureID]

	source := graphics.Rect{0, 0, texture.width, texture.height}
	locations := []graphics.TexturePart{{0, 0, source}}
	context.DrawTextureParts(textureID, locations,
		geometryMatrix, colorMatrix)
}

func (context *Context) DrawTextureParts(
	textureID graphics.TextureID, parts []graphics.TexturePart,
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	texture := context.textures[textureID]
	if texture == nil {
		panic("invalid texture ID")
	}

	context.setShaderProgram(geometryMatrix, colorMatrix)
	C.glBindTexture(C.GL_TEXTURE_2D, texture.id)

	vertexAttrLocation := getAttributeLocation(context.currentShaderProgram, "vertex")
	textureAttrLocation := getAttributeLocation(context.currentShaderProgram, "texture")

	C.glEnableClientState(C.GL_VERTEX_ARRAY)
	C.glEnableClientState(C.GL_TEXTURE_COORD_ARRAY)
	C.glEnableVertexAttribArray(C.GLuint(vertexAttrLocation))
	C.glEnableVertexAttribArray(C.GLuint(textureAttrLocation))
	// TODO: Optimization
	for _, part := range parts {
		x1 := float32(part.LocationX)
		x2 := float32(part.LocationX + part.Source.Width)
		y1 := float32(part.LocationY)
		y2 := float32(part.LocationY + part.Source.Height)
		vertex := [...]float32{
			x1, y1,
			x2, y1,
			x1, y2,
			x2, y2,
		}

		src := part.Source
		tu1 := float32(src.X) / float32(texture.textureWidth)
		tu2 := float32(src.X+src.Width) / float32(texture.textureWidth)
		tv1 := float32(src.Y) / float32(texture.textureHeight)
		tv2 := float32(src.Y+src.Height) / float32(texture.textureHeight)
		texCoord := [...]float32{
			tu1, tv1,
			tu2, tv1,
			tu1, tv2,
			tu2, tv2,
		}
		C.glVertexAttribPointer(C.GLuint(vertexAttrLocation), 2, C.GL_FLOAT, C.GL_FALSE,
			0, unsafe.Pointer(&vertex[0]))
		C.glVertexAttribPointer(C.GLuint(textureAttrLocation), 2, C.GL_FLOAT, C.GL_FALSE,
			0, unsafe.Pointer(&texCoord[0]))
		C.glDrawArrays(C.GL_TRIANGLE_STRIP, 0, 4)
	}
	C.glDisableVertexAttribArray(C.GLuint(textureAttrLocation))
	C.glDisableVertexAttribArray(C.GLuint(vertexAttrLocation))
	C.glDisableClientState(C.GL_TEXTURE_COORD_ARRAY)
	C.glDisableClientState(C.GL_VERTEX_ARRAY)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (context *Context) SetOffscreen(textureID graphics.TextureID) {
	texture := context.textures[textureID]
	context.currentOffscreenWidth = texture.width
	context.currentOffscreenHeight = texture.height

	framebuffer := context.getFramebuffer(texture.id)
	if framebuffer == context.mainFramebuffer {
		panic("invalid framebuffer")
	}
	context.setOffscreenFramebuffer(framebuffer,
		texture.textureWidth, texture.textureHeight)
}

func (context *Context) setOffscreenFramebuffer(framebuffer C.GLuint,
	textureWidth, textureHeight int) {
	if framebuffer == context.mainFramebuffer {
		textureWidth = context.mainFramebufferTexture.textureWidth
		textureHeight = context.mainFramebufferTexture.textureHeight
	}

	C.glFlush()

	C.glBindFramebuffer(C.GL_FRAMEBUFFER, framebuffer)
	if err := C.glCheckFramebufferStatus(C.GL_FRAMEBUFFER); err != C.GL_FRAMEBUFFER_COMPLETE {
		panic(fmt.Sprintf("glBindFramebuffer failed: %d", err))
	}
	C.glEnable(C.GL_BLEND)
	C.glBlendFunc(C.GL_SRC_ALPHA, C.GL_ONE_MINUS_SRC_ALPHA)

	C.glViewport(0, 0, C.GLsizei(abs(textureWidth)), C.GLsizei(abs(textureHeight)))

	var e11, e22, e41, e42 float32
	if framebuffer != context.mainFramebuffer {
		e11 = float32(2) / float32(textureWidth)
		e22 = float32(2) / float32(textureWidth)
		e41 = -1
		e42 = -1
	} else {
		height := float32(context.mainFramebufferTexture.Height())
		e11 = float32(2) / float32(textureWidth)
		e22 = -1 * float32(2) / float32(textureHeight)
		e41 = -1
		e42 = -1 + height/float32(textureHeight)*2
	}

	context.projectionMatrix = [...]float32{
		e11, 0, 0, 0,
		0, e22, 0, 0,
		0, 0, 1, 0,
		e41, e42, 0, 1,
	}
}

func (context *Context) resetOffscreen() {
	context.setOffscreenFramebuffer(context.mainFramebuffer, 0, 0)
	context.currentOffscreenWidth = context.mainFramebufferTexture.Width()
	context.currentOffscreenHeight = context.mainFramebufferTexture.Height()
}

// This method should be called on the UI thread.
func (context *Context) flush() {
	C.glFlush()
}

// This method should be called on the UI thread.
func (context *Context) setShaderProgram(
	geometryMatrix matrix.Geometry, colorMatrix matrix.Color) {
	program := C.GLuint(0)
	if colorMatrix.IsIdentity() {
		program = regularShaderProgram
	} else {
		program = colorMatrixShaderProgram
	}
	// TODO: cache and skip?
	C.glUseProgram(program)
	context.currentShaderProgram = program

	C.glUniformMatrix4fv(getUniformLocation(program, "projection_matrix"),
		1, C.GL_FALSE,
		(*C.GLfloat)(&context.projectionMatrix[0]))

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
	C.glUniformMatrix4fv(getUniformLocation(program, "modelview_matrix"),
		1, C.GL_FALSE,
		(*C.GLfloat)(&glModelviewMatrix[0]))

	C.glUniform1i(getUniformLocation(program, "texture"), 0)

	if program != colorMatrixShaderProgram {
		return
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
	C.glUniformMatrix4fv(getUniformLocation(program, "color_matrix"),
		1, C.GL_FALSE, (*C.GLfloat)(&glColorMatrix[0]))

	glColorMatrixTranslation := [...]float32{
		e[0][4], e[1][4], e[2][4], e[3][4],
	}
	C.glUniform4fv(getUniformLocation(program, "color_matrix_translation"),
		1, (*C.GLfloat)(&glColorMatrixTranslation[0]))
}

func (context *Context) getFramebuffer(textureID C.GLuint) C.GLuint {
	framebuffer, ok := context.framebuffers[textureID]
	if ok {
		return framebuffer
	}

	newFramebuffer := C.GLuint(0)
	C.glGenFramebuffers(1, &newFramebuffer)

	origFramebuffer := C.GLint(0)
	C.glGetIntegerv(C.GL_FRAMEBUFFER_BINDING, &origFramebuffer)
	C.glBindFramebuffer(C.GL_FRAMEBUFFER, newFramebuffer)
	C.glFramebufferTexture2D(C.GL_FRAMEBUFFER, C.GL_COLOR_ATTACHMENT0,
		C.GL_TEXTURE_2D, textureID, 0)
	C.glBindFramebuffer(C.GL_FRAMEBUFFER, C.GLuint(origFramebuffer))
	if C.glCheckFramebufferStatus(C.GL_FRAMEBUFFER) != C.GL_FRAMEBUFFER_COMPLETE {
		panic("creating framebuffer failed")
	}

	context.framebuffers[textureID] = newFramebuffer
	return newFramebuffer
}

func (context *Context) deleteFramebuffer(textureID C.GLuint) {
	framebuffer, ok := context.framebuffers[textureID]
	if !ok {
		// TODO: panic?
		return
	}
	C.glDeleteFramebuffers(1, &framebuffer)
	delete(context.framebuffers, textureID)
}

func (context *Context) NewTexture(width, height int) graphics.Texture {
	texture := newTexture(width, height)
	id := graphics.TextureID(texture.id)
	context.textures[id] = texture

	context.SetOffscreen(texture.ID())
	context.Clear()
	context.resetOffscreen()

	return texture
}

func (context *Context) NewTextureFromImage(img image.Image) (graphics.Texture, error) {
	texture, err := newTextureFromImage(img)
	if err != nil {
		return nil, err
	}
	id := graphics.TextureID(texture.id)
	context.textures[id] = texture
	return texture, nil
}
