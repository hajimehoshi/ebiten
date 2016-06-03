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
	// TODO: Check if len(vertices) is too big
	if 0 < len(vertices) {
		context.BufferSubData(context.ArrayBuffer, vertices)
	}
	for _, c := range q.commands {
		if err := c.Exec(context); err != nil {
			return err
		}
	}
	q.commands = []command{}
	return nil
}

type fillCommand struct {
	dst   *Framebuffer
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
	dst      *Framebuffer
	src      *Texture
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

	// NOTE: WebGL doesn't seem to have Check gl.MAX_ELEMENTS_VERTICES or gl.MAX_ELEMENTS_INDICES so far.
	// Let's use them to compare to len(quads) in the future.
	n := len(c.vertices) / 16
	if n == 0 {
		return nil
	}
	if MaxQuads < n/16 {
		return errors.New(fmt.Sprintf("len(quads) must be equal to or less than %d", MaxQuads))
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
	dst     *Framebuffer
	texture *Texture
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
