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
	"sync"

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/opengl"
)

type command interface {
	Exec(context *opengl.Context, indexOffsetInBytes int) error
}

type commandQueue struct {
	commands    []command
	vertices    []float32
	verticesNum int
	m           sync.Mutex
}

var theCommandQueue = &commandQueue{
	commands: []command{},
	vertices: []float32{},
}

func (q *commandQueue) AppendVertices(vertices []float32) {
	q.m.Lock()
	defer q.m.Unlock()
	if len(q.vertices) < q.verticesNum+len(vertices) {
		n := q.verticesNum + len(vertices) - len(q.vertices)
		q.vertices = append(q.vertices, make([]float32, n)...)
	}
	copy(q.vertices[q.verticesNum:q.verticesNum+len(vertices)], vertices)
	q.verticesNum += len(vertices)
}

func (q *commandQueue) Enqueue(command command) {
	q.m.Lock()
	defer q.m.Unlock()
	if 0 < len(q.commands) {
		if c1, ok := q.commands[len(q.commands)-1].(*drawImageCommand); ok {
			if c2, ok := command.(*drawImageCommand); ok {
				if c1.isMergeable(c2) {
					c1.verticesNum += c2.verticesNum
					return
				}
			}
		}
	}
	q.commands = append(q.commands, command)
}

// commandGroups separates q.commands into some groups.
// The number of quads of drawImageCommand in one groups must be equal to or less than
// its limit (maxQuads).
func (q *commandQueue) commandGroups() [][]command {
	cs := q.commands
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
	n := 0
	lastN := 0
	for _, g := range q.commandGroups() {
		for _, c := range g {
			switch c := c.(type) {
			case *drawImageCommand:
				n += c.verticesNum
			}
		}
		if 0 < n-lastN {
			context.BufferSubData(opengl.ArrayBuffer, q.vertices[lastN:n])
		}
		// NOTE: WebGL doesn't seem to have Check gl.MAX_ELEMENTS_VERTICES or gl.MAX_ELEMENTS_INDICES so far.
		// Let's use them to compare to len(quads) in the future.
		if maxQuads < (n-lastN)*opengl.Float.SizeInBytes()/QuadVertexSizeInBytes() {
			return fmt.Errorf("len(quads) must be equal to or less than %d", maxQuads)
		}
		numc := len(g)
		indexOffsetInBytes := 0
		for _, c := range g {
			if err := c.Exec(context, indexOffsetInBytes); err != nil {
				return err
			}
			if c, ok := c.(*drawImageCommand); ok {
				n := c.verticesNum * opengl.Float.SizeInBytes() / QuadVertexSizeInBytes()
				indexOffsetInBytes += 6 * n * 2
			}
		}
		if 0 < numc {
			// Call glFlush to prevent black flicking (especially on Android (#226) and iOS).
			context.Flush()
		}
		lastN = n
	}
	q.commands = []command{}
	q.verticesNum = 0
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
	f, err := c.dst.createFramebufferIfNeeded(context)
	if err != nil {
		return err
	}
	if err := f.setAsViewport(context); err != nil {
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
	dst         *Image
	src         *Image
	verticesNum int
	color       affine.ColorM
	mode        opengl.CompositeMode
}

func QuadVertexSizeInBytes() int {
	return 4 * theArrayBufferLayout.totalBytes()
}

func (c *drawImageCommand) Exec(context *opengl.Context, indexOffsetInBytes int) error {
	f, err := c.dst.createFramebufferIfNeeded(context)
	if err != nil {
		return err
	}
	if err := f.setAsViewport(context); err != nil {
		return err
	}
	context.BlendFunc(c.mode)

	n := c.quadsNum()
	if n == 0 {
		return nil
	}
	_, h := c.dst.Size()
	proj := f.projectionMatrix(h)
	p := &programContext{
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
	// TODO: We should call glBindBuffer here?
	// The buffer is already bound at begin() but it is counterintuitive.
	context.DrawElements(opengl.Triangles, 6*n, indexOffsetInBytes)
	return nil
}

func (c *drawImageCommand) split(quadsNum int) [2]*drawImageCommand {
	c1 := *c
	c2 := *c
	s := opengl.Float.SizeInBytes()
	n := quadsNum * QuadVertexSizeInBytes() / s
	c1.verticesNum = n
	c2.verticesNum -= n
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

func (c *drawImageCommand) quadsNum() int {
	return c.verticesNum * opengl.Float.SizeInBytes() / QuadVertexSizeInBytes()
}

type replacePixelsCommand struct {
	dst    *Image
	pixels []uint8
}

func (c *replacePixelsCommand) Exec(context *opengl.Context, indexOffsetInBytes int) error {
	f, err := c.dst.createFramebufferIfNeeded(context)
	if err != nil {
		return err
	}
	if err := f.setAsViewport(context); err != nil {
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

func (c *newImageFromImageCommand) Exec(context *opengl.Context, indexOffsetInBytes int) error {
	origSize := c.img.Bounds().Size()
	if origSize.X < 1 {
		return errors.New("graphics: width must be equal or more than 1.")
	}
	if origSize.Y < 1 {
		return errors.New("graphics: height must be equal or more than 1.")
	}
	w, h := c.img.Bounds().Size().X, c.img.Bounds().Size().Y
	if c.img.Bounds() != image.Rect(0, 0, NextPowerOf2Int(w), NextPowerOf2Int(h)) {
		panic(fmt.Sprintf("graphics: invalid image bounds: %v", c.img.Bounds()))
	}
	native, err := context.NewTexture(w, h, c.img.Pix, c.filter)
	if err != nil {
		return err
	}
	c.result.texture = &texture{
		native: native,
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
