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
	"image/draw"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
)

type command interface {
	Exec(context *opengl.Context) error
}

type commandQueue struct {
	commands           []command
	indexOffsetInBytes int
	m                  sync.Mutex
}

var theCommandQueue = &commandQueue{
	commands: []command{},
}

func (q *commandQueue) Enqueue(command command) {
	q.m.Lock()
	defer q.m.Unlock()
	q.commands = append(q.commands, command)
}

func (q *commandQueue) Flush(context *opengl.Context) error {
	q.m.Lock()
	defer q.m.Unlock()
	// glViewport must be called at least at every frame on iOS.
	context.ResetViewportSize()
	q.indexOffsetInBytes = 0
	vertices := []int16{}
	for _, c := range q.commands {
		switch c := c.(type) {
		case *drawImageCommand:
			vertices = append(vertices, c.vertices...)
		}
	}
	if 0 < len(vertices) {
		context.BufferSubData(opengl.ArrayBuffer, vertices)
	}
	// NOTE: WebGL doesn't seem to have Check gl.MAX_ELEMENTS_VERTICES or gl.MAX_ELEMENTS_INDICES so far.
	// Let's use them to compare to len(quads) in the future.
	if MaxQuads < len(vertices)/16 {
		return errors.New(fmt.Sprintf("len(quads) must be equal to or less than %d", MaxQuads))
	}
	numc := len(q.commands)
	for _, c := range q.commands {
		if err := c.Exec(context); err != nil {
			return err
		}
	}
	q.commands = []command{}
	if 0 < numc {
		// Call glFlush to prevent black flicking (especially on Android (#226) and iOS).
		context.Flush()
	}
	return nil
}

func FlushCommands(context *opengl.Context) error {
	return theCommandQueue.Flush(context)
}

type fillCommand struct {
	dst   *Image
	color color.Color
}

func (c *fillCommand) Exec(context *opengl.Context) error {
	if err := c.dst.framebuffer.setAsViewport(context); err != nil {
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
	dst      *Image
	src      *Image
	vertices []int16
	geo      Matrix
	color    Matrix
	mode     opengl.CompositeMode
}

func (c *drawImageCommand) Exec(context *opengl.Context) error {
	if err := c.dst.framebuffer.setAsViewport(context); err != nil {
		return err
	}
	context.BlendFunc(c.mode)

	n := len(c.vertices) / 16
	if n == 0 {
		return nil
	}
	_, h := c.dst.Size()
	proj := glMatrix(c.dst.framebuffer.projectionMatrix(h))
	p := programContext{
		state:            &theOpenGLState,
		program:          theOpenGLState.programTexture,
		context:          context,
		projectionMatrix: proj,
		texture:          c.src.texture.native,
		geoM:             c.geo,
		colorM:           c.color,
	}
	p.begin()
	defer p.end()
	// TODO: We should call glBindBuffer here?
	// The buffer is already bound at begin() but it is counterintuitive.
	context.DrawElements(opengl.Triangles, 6*n, theCommandQueue.indexOffsetInBytes)
	theCommandQueue.indexOffsetInBytes += 6 * n * 2
	return nil
}

type replacePixelsCommand struct {
	dst    *Image
	pixels []uint8
}

func (c *replacePixelsCommand) Exec(context *opengl.Context) error {
	if err := c.dst.framebuffer.setAsViewport(context); err != nil {
		return err
	}
	// Filling with non black or white color is required here for glTexSubImage2D.
	// Very mysterious but this actually works (Issue #186).
	// This is needed even after fixing a shader bug at f537378f2a6a8ef56e1acf1c03034967b77c7b51.
	if err := context.FillFramebuffer(0, 0, 0.5, 1); err != nil {
		return err
	}
	context.BindTexture(c.dst.texture.native)
	context.TexSubImage2D(c.pixels, c.dst.width, c.dst.height)
	return nil
}

type disposeCommand struct {
	target *Image
}

func (c *disposeCommand) Exec(context *opengl.Context) error {
	if c.target.framebuffer != nil {
		context.DeleteFramebuffer(c.target.framebuffer.native)
	}
	if c.target.texture != nil {
		context.DeleteTexture(c.target.texture.native)
	}
	return nil
}

type newImageFromImageCommand struct {
	result *Image
	img    *image.RGBA
	filter opengl.Filter
}

func adjustImageForTexture(img *image.RGBA) *image.RGBA {
	width, height := img.Bounds().Size().X, img.Bounds().Size().Y
	adjustedImageBounds := image.Rectangle{
		image.ZP,
		image.Point{
			int(NextPowerOf2Int32(int32(width))),
			int(NextPowerOf2Int32(int32(height))),
		},
	}
	if img.Bounds() == adjustedImageBounds {
		return img
	}

	adjustedImage := image.NewRGBA(adjustedImageBounds)
	dstBounds := image.Rectangle{
		image.ZP,
		img.Bounds().Size(),
	}
	draw.Draw(adjustedImage, dstBounds, img, img.Bounds().Min, draw.Src)
	return adjustedImage
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
	c.result.texture = &texture{
		native: native,
	}
	c.result.framebuffer, err = newFramebufferFromTexture(context, c.result.texture)
	if err != nil {
		return err
	}
	return nil
}

type newImageCommand struct {
	result *Image
	width  int
	height int
	filter opengl.Filter
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
	c.result.texture = &texture{
		native: native,
	}
	c.result.framebuffer, err = newFramebufferFromTexture(context, c.result.texture)
	if err != nil {
		return err
	}
	return nil
}

type newScreenFramebufferImageCommand struct {
	result *Image
	width  int
	height int
}

func (c *newScreenFramebufferImageCommand) Exec(context *opengl.Context) error {
	if c.width < 4 {
		return errors.New("graphics: width must be equal or more than 4.")
	}
	if c.height < 4 {
		return errors.New("graphics: height must be equal or more than 4.")
	}
	f := &framebuffer{
		native: context.ScreenFramebuffer(),
		flipY:  true,
	}
	c.result.framebuffer = f
	return nil
}
