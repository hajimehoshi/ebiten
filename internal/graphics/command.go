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

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/graphics/opengl"
)

type command interface {
	Exec(context *opengl.Context, indexOffsetInBytes int) error
}

type commandQueue struct {
	commands []command
	m        sync.Mutex
}

var theCommandQueue = &commandQueue{
	commands: []command{},
}

func (q *commandQueue) Enqueue(command command) {
	q.m.Lock()
	defer q.m.Unlock()
	q.commands = append(q.commands, command)
}

func mergeCommands(commands []command) []command {
	// TODO: This logic is relatively complicated. Add tests.
	cs := make([]command, 0, len(commands))
	var prev *drawImageCommand
	for _, c := range commands {
		switch c := c.(type) {
		case *drawImageCommand:
			if prev == nil {
				prev = c
				continue
			}
			if prev.isMergeable(c) {
				prev = prev.merge(c)
				continue
			}
			cs = append(cs, prev)
			prev = c
			continue
		}
		if prev != nil {
			cs = append(cs, prev)
			prev = nil
		}
		cs = append(cs, c)
	}
	if prev != nil {
		cs = append(cs, prev)
	}
	return cs
}

// commandGroups separates q.commands into some groups.
// The number of quads of drawImageCommand in one groups must be equal to or less than
// its limit (maxQuads).
func (q *commandQueue) commandGroups() [][]command {
	cs := mergeCommands(q.commands)
	gs := [][]command{}
	quads := 0
	for 0 < len(cs) {
		if len(gs) == 0 {
			gs = append(gs, []command{})
		}
		c := cs[0]
		switch c := c.(type) {
		case *drawImageCommand:
			if maxQuads >= quads+c.quadsNum() {
				quads += c.quadsNum()
				break
			}
			cc := c.split(maxQuads - quads)
			gs[len(gs)-1] = append(gs[len(gs)-1], cc[0])
			cs[0] = cc[1]
			quads = 0
			gs = append(gs, []command{})
			continue
		}
		gs[len(gs)-1] = append(gs[len(gs)-1], c)
		cs = cs[1:]
	}
	return gs
}

func (q *commandQueue) Flush(context *opengl.Context) error {
	q.m.Lock()
	defer q.m.Unlock()
	// glViewport must be called at least at every frame on iOS.
	context.ResetViewportSize()
	for _, g := range q.commandGroups() {
		n := 0
		for _, c := range g {
			switch c := c.(type) {
			case *drawImageCommand:
				n += len(c.vertices)
			}
		}
		vertices := make([]float32, 0, n)
		for _, c := range g {
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
		if maxQuads < len(vertices)*opengl.Float.SizeInBytes()/QuadVertexSizeInBytes() {
			return fmt.Errorf("len(quads) must be equal to or less than %d", maxQuads)
		}
		numc := len(g)
		indexOffsetInBytes := 0
		for _, c := range g {
			if err := c.Exec(context, indexOffsetInBytes); err != nil {
				return err
			}
			if c, ok := c.(*drawImageCommand); ok {
				n := len(c.vertices) * opengl.Float.SizeInBytes() / QuadVertexSizeInBytes()
				indexOffsetInBytes += 6 * n * 2
			}
		}
		if 0 < numc {
			// Call glFlush to prevent black flicking (especially on Android (#226) and iOS).
			context.Flush()
		}
	}
	q.commands = []command{}
	return nil
}

func FlushCommands(context *opengl.Context) error {
	return theCommandQueue.Flush(context)
}

type fillCommand struct {
	dst   *Image
	color color.RGBA
}

func (c *fillCommand) Exec(context *opengl.Context, indexOffsetInBytes int) error {
	if err := c.dst.framebuffer.setAsViewport(context); err != nil {
		return err
	}
	cr, cg, cb, ca := c.color.R, c.color.G, c.color.B, c.color.A
	const max = math.MaxUint8
	r := float64(cr) / max
	g := float64(cg) / max
	b := float64(cb) / max
	a := float64(ca) / max
	return context.FillFramebuffer(r, g, b, a)
}

type drawImageCommand struct {
	dst      *Image
	src      *Image
	vertices []float32
	color    affine.ColorM
	mode     opengl.CompositeMode
}

func QuadVertexSizeInBytes() int {
	return 4 * theArrayBufferLayout.totalBytes()
}

func (c *drawImageCommand) Exec(context *opengl.Context, indexOffsetInBytes int) error {
	if err := c.dst.framebuffer.setAsViewport(context); err != nil {
		return err
	}
	context.BlendFunc(c.mode)

	n := c.quadsNum()
	if n == 0 {
		return nil
	}
	_, h := c.dst.Size()
	proj := c.dst.framebuffer.projectionMatrix(h)
	p := programContext{
		state:            &theOpenGLState,
		program:          theOpenGLState.programTexture,
		context:          context,
		projectionMatrix: proj,
		texture:          c.src.texture.native,
		colorM:           c.color,
	}
	if err := p.begin(); err != nil {
		return err
	}
	defer p.end()
	// TODO: We should call glBindBuffer here?
	// The buffer is already bound at begin() but it is counterintuitive.
	context.DrawElements(opengl.Triangles, 6*n, indexOffsetInBytes)
	return nil
}

func (c *drawImageCommand) split(quadsNum int) [2]*drawImageCommand {
	c1 := *c
	c2 := *c
	s := opengl.Float.SizeInBytes()
	c1.vertices = c.vertices[:quadsNum*QuadVertexSizeInBytes()/s]
	c2.vertices = c.vertices[quadsNum*QuadVertexSizeInBytes()/s:]
	return [2]*drawImageCommand{&c1, &c2}
}

func (c *drawImageCommand) isMergeable(other *drawImageCommand) bool {
	if c.dst != other.dst {
		return false
	}
	if c.src != other.src {
		return false
	}
	if !c.color.Equals(&other.color) {
		return false
	}
	if c.mode != other.mode {
		return false
	}
	return true
}

func (c *drawImageCommand) merge(other *drawImageCommand) *drawImageCommand {
	newC := *c
	newC.vertices = make([]float32, 0, len(c.vertices)+len(other.vertices))
	newC.vertices = append(newC.vertices, c.vertices...)
	newC.vertices = append(newC.vertices, other.vertices...)
	return &newC
}

func (c *drawImageCommand) quadsNum() int {
	return len(c.vertices) * opengl.Float.SizeInBytes() / QuadVertexSizeInBytes()
}

type replacePixelsCommand struct {
	dst    *Image
	pixels []uint8
}

func (c *replacePixelsCommand) Exec(context *opengl.Context, indexOffsetInBytes int) error {
	if err := c.dst.framebuffer.setAsViewport(context); err != nil {
		return err
	}
	// Filling with non black or white color is required here for glTexSubImage2D.
	// Very mysterious but this actually works (Issue #186).
	// This is needed even after fixing a shader bug at f537378f2a6a8ef56e1acf1c03034967b77c7b51.
	if err := context.FillFramebuffer(0, 0, 0.5, 1); err != nil {
		return err
	}
	// This is necessary on Android. We can't call glClear just before glTexSubImage2D without
	// glFlush. glTexSubImage2D didn't work without this hack at least on Nexus 5x (#211).
	// This also happens when a fillCommand precedes a replacePixelsCommand.
	// TODO: Can we have a better way like optimizing commands?
	context.Flush()
	if err := context.BindTexture(c.dst.texture.native); err != nil {
		return err
	}
	context.TexSubImage2D(c.pixels, c.dst.width, c.dst.height)
	return nil
}

type disposeCommand struct {
	target *Image
}

func (c *disposeCommand) Exec(context *opengl.Context, indexOffsetInBytes int) error {
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
			NextPowerOf2Int(width),
			NextPowerOf2Int(height),
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

func (c *newImageFromImageCommand) Exec(context *opengl.Context, indexOffsetInBytes int) error {
	origSize := c.img.Bounds().Size()
	if origSize.X < 1 {
		return errors.New("graphics: width must be equal or more than 1.")
	}
	if origSize.Y < 1 {
		return errors.New("graphics: height must be equal or more than 1.")
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

func (c *newImageCommand) Exec(context *opengl.Context, indexOffsetInBytes int) error {
	w := NextPowerOf2Int(c.width)
	h := NextPowerOf2Int(c.height)
	if w < 1 {
		return errors.New("graphics: width must be equal or more than 1.")
	}
	if h < 1 {
		return errors.New("graphics: height must be equal or more than 1.")
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

func (c *newScreenFramebufferImageCommand) Exec(context *opengl.Context, indexOffsetInBytes int) error {
	if c.width < 1 {
		return errors.New("graphics: width must be equal or more than 1.")
	}
	if c.height < 1 {
		return errors.New("graphics: height must be equal or more than 1.")
	}
	f := &framebuffer{
		native: context.ScreenFramebuffer(),
		flipY:  true,
	}
	c.result.framebuffer = f
	return nil
}
