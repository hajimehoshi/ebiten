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

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/opengl"
	"github.com/hajimehoshi/ebiten/internal/sync"
)

type command interface {
	Exec(indexOffsetInBytes int) error
}

type commandQueue struct {
	commands    []command
	vertices    []float32
	verticesNum int
	m           sync.Mutex
}

var theCommandQueue = &commandQueue{}

func (q *commandQueue) appendVertices(vertices []float32) {
	if len(q.vertices) < q.verticesNum+len(vertices) {
		n := q.verticesNum + len(vertices) - len(q.vertices)
		q.vertices = append(q.vertices, make([]float32, n)...)
	}
	// for-loop might be faster than copy:
	// On GopherJS, copy might cause subarray calls.
	for i := 0; i < len(vertices); i++ {
		q.vertices[q.verticesNum+i] = vertices[i]
	}
	q.verticesNum += len(vertices)
}

func (q *commandQueue) EnqueueDrawImageCommand(dst, src *Image, vertices []float32, clr *affine.ColorM, mode opengl.CompositeMode) {
	// Avoid defer for performance
	q.m.Lock()
	q.appendVertices(vertices)
	if 0 < len(q.commands) {
		if c, ok := q.commands[len(q.commands)-1].(*drawImageCommand); ok {
			if c.isMergeable(dst, src, clr, mode) {
				c.verticesNum += len(vertices)
				q.m.Unlock()
				return
			}
		}
	}
	c := &drawImageCommand{
		dst:         dst,
		src:         src,
		verticesNum: len(vertices),
		color:       *clr,
		mode:        mode,
	}
	q.commands = append(q.commands, c)
	q.m.Unlock()
}

func (q *commandQueue) Enqueue(command command) {
	q.m.Lock()
	q.commands = append(q.commands, command)
	q.m.Unlock()
}

// commandGroups separates q.commands into some groups.
// The number of quads of drawImageCommand in one groups must be equal to or less than
// its limit (maxQuads).
func (q *commandQueue) commandGroups() [][]command {
	cs := q.commands
	var gs [][]command
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

func (q *commandQueue) Flush() error {
	q.m.Lock()
	defer q.m.Unlock()
	// glViewport must be called at least at every frame on iOS.
	opengl.GetContext().ResetViewportSize()
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
			opengl.GetContext().BufferSubData(opengl.ArrayBuffer, q.vertices[lastN:n])
		}
		// NOTE: WebGL doesn't seem to have Check gl.MAX_ELEMENTS_VERTICES or gl.MAX_ELEMENTS_INDICES so far.
		// Let's use them to compare to len(quads) in the future.
		if maxQuads < (n-lastN)*opengl.Float.SizeInBytes()/QuadVertexSizeInBytes() {
			return fmt.Errorf("len(quads) must be equal to or less than %d", maxQuads)
		}
		numc := len(g)
		indexOffsetInBytes := 0
		for _, c := range g {
			if err := c.Exec(indexOffsetInBytes); err != nil {
				return err
			}
			if c, ok := c.(*drawImageCommand); ok {
				n := c.verticesNum * opengl.Float.SizeInBytes() / QuadVertexSizeInBytes()
				indexOffsetInBytes += 6 * n * 2
			}
		}
		if 0 < numc {
			// Call glFlush to prevent black flicking (especially on Android (#226) and iOS).
			opengl.GetContext().Flush()
		}
		lastN = n
	}
	q.commands = nil
	q.verticesNum = 0
	return nil
}

func FlushCommands() error {
	return theCommandQueue.Flush()
}

type fillCommand struct {
	dst   *Image
	color color.RGBA
}

func (c *fillCommand) Exec(indexOffsetInBytes int) error {
	f, err := c.dst.createFramebufferIfNeeded()
	if err != nil {
		return err
	}
	if err := f.setAsViewport(); err != nil {
		return err
	}
	cr, cg, cb, ca := c.color.R, c.color.G, c.color.B, c.color.A
	const max = math.MaxUint8
	r := float64(cr) / max
	g := float64(cg) / max
	b := float64(cb) / max
	a := float64(ca) / max
	return opengl.GetContext().FillFramebuffer(r, g, b, a)
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

func (c *drawImageCommand) Exec(indexOffsetInBytes int) error {
	f, err := c.dst.createFramebufferIfNeeded()
	if err != nil {
		return err
	}
	if err := f.setAsViewport(); err != nil {
		return err
	}
	opengl.GetContext().BlendFunc(c.mode)

	n := c.quadsNum()
	if n == 0 {
		return nil
	}
	_, h := c.dst.Size()
	proj := f.projectionMatrix(h)
	p := &programContext{
		state:            &theOpenGLState,
		program:          theOpenGLState.programTexture,
		projectionMatrix: proj,
		texture:          c.src.texture.native,
		colorM:           c.color,
	}
	if err := p.begin(); err != nil {
		return err
	}
	// TODO: We should call glBindBuffer here?
	// The buffer is already bound at begin() but it is counterintuitive.
	opengl.GetContext().DrawElements(opengl.Triangles, 6*n, indexOffsetInBytes)
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

func (c *drawImageCommand) isMergeable(dst, src *Image, clr *affine.ColorM, mode opengl.CompositeMode) bool {
	if c.dst != dst {
		return false
	}
	if c.src != src {
		return false
	}
	if !c.color.Equals(clr) {
		return false
	}
	if c.mode != mode {
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

func (c *replacePixelsCommand) Exec(indexOffsetInBytes int) error {
	f, err := c.dst.createFramebufferIfNeeded()
	if err != nil {
		return err
	}
	if err := f.setAsViewport(); err != nil {
		return err
	}
	// Filling with non black or white color is required here for glTexSubImage2D.
	// Very mysterious but this actually works (Issue #186).
	// This is needed even after fixing a shader bug at f537378f2a6a8ef56e1acf1c03034967b77c7b51.
	if err := opengl.GetContext().FillFramebuffer(0, 0, 0.5, 1); err != nil {
		return err
	}
	// This is necessary on Android. We can't call glClear just before glTexSubImage2D without
	// glFlush. glTexSubImage2D didn't work without this hack at least on Nexus 5x (#211).
	// This also happens when a fillCommand precedes a replacePixelsCommand.
	// TODO: Can we have a better way like optimizing commands?
	opengl.GetContext().Flush()
	if err := opengl.GetContext().BindTexture(c.dst.texture.native); err != nil {
		return err
	}
	opengl.GetContext().TexSubImage2D(c.pixels, NextPowerOf2Int(c.dst.width), NextPowerOf2Int(c.dst.height))
	return nil
}

type disposeCommand struct {
	target *Image
}

func (c *disposeCommand) Exec(indexOffsetInBytes int) error {
	if c.target.framebuffer != nil {
		opengl.GetContext().DeleteFramebuffer(c.target.framebuffer.native)
	}
	if c.target.texture != nil {
		opengl.GetContext().DeleteTexture(c.target.texture.native)
	}
	return nil
}

type newImageFromImageCommand struct {
	result *Image
	img    *image.RGBA
	filter opengl.Filter
}

func (c *newImageFromImageCommand) Exec(indexOffsetInBytes int) error {
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
	native, err := opengl.GetContext().NewTexture(w, h, c.img.Pix, c.filter)
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

func (c *newImageCommand) Exec(indexOffsetInBytes int) error {
	w := NextPowerOf2Int(c.width)
	h := NextPowerOf2Int(c.height)
	if w < 1 {
		return errors.New("graphics: width must be equal or more than 1.")
	}
	if h < 1 {
		return errors.New("graphics: height must be equal or more than 1.")
	}
	native, err := opengl.GetContext().NewTexture(w, h, nil, c.filter)
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

func (c *newScreenFramebufferImageCommand) Exec(indexOffsetInBytes int) error {
	if c.width < 1 {
		return errors.New("graphics: width must be equal or more than 1.")
	}
	if c.height < 1 {
		return errors.New("graphics: height must be equal or more than 1.")
	}
	f := &framebuffer{
		native: opengl.GetContext().ScreenFramebuffer(),
		flipY:  true,
	}
	c.result.framebuffer = f
	return nil
}
