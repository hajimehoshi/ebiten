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
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
)

type command interface {
	Exec(context *opengl.Context) error
}

type commandQueue struct {
	commands []command
}

var theCommandQueue = &commandQueue{
	commands: []command{},
}

func (q *commandQueue) Enqueue(command command) {
	q.commands = append(q.commands, command)
}

func (q *commandQueue) Flush(context *opengl.Context) error {
	// TODO: Do optimizing before executing
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
	p := c.dst.projectionMatrix()
	return drawTexture(context, c.src.native, p, c.vertices, c.geo, c.color, c.mode)
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
