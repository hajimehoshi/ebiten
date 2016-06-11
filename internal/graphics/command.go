// Copyright 2016 Hajime Hoshi
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

package graphics

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
)

type command interface {
	Exec(context *opengl.Context) error
}

type commandQueue struct {
	commands           []command
	indexOffsetInBytes int
}

var theCommandQueue = &commandQueue{
	commands: []command{},
}

func (q *commandQueue) Enqueue(command command) {
	q.commands = append(q.commands, command)
}

func (q *commandQueue) Flush(context *opengl.Context) error {
	q.indexOffsetInBytes = 0
	vertices := []int16{}
	for _, c := range q.commands {
		switch c := c.(type) {
		case *drawImageCommand:
			vertices = append(vertices, c.vertices...)
		}
	}
	if 0 < len(vertices) {
		context.BufferSubData(context.ArrayBuffer, vertices)
	}
	// NOTE: WebGL doesn't seem to have Check gl.MAX_ELEMENTS_VERTICES or gl.MAX_ELEMENTS_INDICES so far.
	// Let's use them to compare to len(quads) in the future.
	if MaxQuads < len(vertices)/16 {
		return errors.New(fmt.Sprintf("len(quads) must be equal to or less than %d", MaxQuads))
	}
	for _, c := range q.commands {
		if err := c.Exec(context); err != nil {
			return err
		}
	}
	q.commands = []command{}
	return nil
}

func FlushCommands(context *opengl.Context) error {
	return theCommandQueue.Flush(context)
}

type fillCommand struct {
	dst   *framebuffer
	color color.Color
}

func (c *fillCommand) Exec(context *opengl.Context) error {
	if err := c.dst.setAsViewport(context); err != nil {
		return err
	}
	cr, cg, cb, ca := c.color.RGBA()
	const max = math.MaxUint16
	r := float64(cr) / max
	g := float64(cg) / max
	b := float64(cb) / max
	a := float64(ca) / max
	return context.FillFramebuffer(r, g, b, a)
}

type drawImageCommand struct {
	dst      *framebuffer
	src      *texture
	vertices []int16
	geo      Matrix
	color    Matrix
	mode     opengl.CompositeMode
}

func (c *drawImageCommand) Exec(context *opengl.Context) error {
	if err := c.dst.setAsViewport(context); err != nil {
		return err
	}
	context.BlendFunc(c.mode)

	n := len(c.vertices) / 16
	if n == 0 {
		return nil
	}
	p := programContext{
		state:            &theOpenGLState,
		program:          theOpenGLState.programTexture,
		context:          context,
		projectionMatrix: glMatrix(c.dst.projectionMatrix()),
		texture:          c.src.native,
		geoM:             c.geo,
		colorM:           c.color,
	}
	p.begin()
	defer p.end()
	// TODO: We should call glBindBuffer here?
	// The buffer is already bound at begin() but it is counterintuitive.
	context.DrawElements(context.Triangles, 6*n, theCommandQueue.indexOffsetInBytes)
	theCommandQueue.indexOffsetInBytes += 6 * n * 2
	return nil
}

type replacePixelsCommand struct {
	dst     *framebuffer
	texture *texture
	pixels  []uint8
}

func (c *replacePixelsCommand) Exec(context *opengl.Context) error {
	// Filling with non black or white color is required here for glTexSubImage2D.
	// Very mysterious but this actually works (Issue #186)
	if err := c.dst.setAsViewport(context); err != nil {
		return err
	}
	if err := context.FillFramebuffer(0, 0, 0.5, 1); err != nil {
		return err
	}
	context.BindTexture(c.texture.native)
	context.TexSubImage2D(c.pixels, c.texture.width, c.texture.height)
	return nil
}

type disposeCommand struct {
	framebuffer *framebuffer
	texture     *texture
}

func (c *disposeCommand) Exec(context *opengl.Context) error {
	if c.framebuffer != nil && c.framebuffer.native != opengl.ZeroFramebuffer {
		context.DeleteFramebuffer(c.framebuffer.native)
	}
	if c.texture != nil {
		context.DeleteTexture(c.texture.native)
	}
	return nil
}

type newImageFromImageCommand struct {
	texture     *texture
	framebuffer *framebuffer
	img         *image.RGBA
	filter      opengl.Filter
}

func (c *newImageFromImageCommand) Exec(context *opengl.Context) error {
	origSize := c.img.Bounds().Size()
	if origSize.X < 4 {
		return errors.New("graphics: width must be equal or more than 4.")
	}
	if origSize.Y < 4 {
		return errors.New("graphics: height must be equal or more than 4.")
	}
	adjustedImage := adjustImageForTexture(c.img)
	size := adjustedImage.Bounds().Size()
	native, err := context.NewTexture(size.X, size.Y, adjustedImage.Pix, c.filter)
	if err != nil {
		return err
	}
	c.texture.native = native
	c.texture.width = origSize.X
	c.texture.height = origSize.Y
	if err := c.framebuffer.initFromTexture(context, c.texture); err != nil {
		return err
	}
	return nil
}

type newImageCommand struct {
	texture     *texture
	framebuffer *framebuffer
	width       int
	height      int
	filter      opengl.Filter
}

func (c *newImageCommand) Exec(context *opengl.Context) error {
	w := int(NextPowerOf2Int32(int32(c.width)))
	h := int(NextPowerOf2Int32(int32(c.height)))
	if w < 4 {
		return errors.New("graphics: width must be equal or more than 4.")
	}
	if h < 4 {
		return errors.New("graphics: height must be equal or more than 4.")
	}
	native, err := context.NewTexture(w, h, nil, c.filter)
	if err != nil {
		return err
	}
	c.texture.native = native
	c.texture.width = c.width
	c.texture.height = c.height
	if err := c.framebuffer.initFromTexture(context, c.texture); err != nil {
		return err
	}
	return nil
}
