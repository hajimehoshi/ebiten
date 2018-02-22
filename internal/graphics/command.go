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
	emath "github.com/hajimehoshi/ebiten/internal/math"
	"github.com/hajimehoshi/ebiten/internal/opengl"
	"github.com/hajimehoshi/ebiten/internal/sync"
)

// command represents a drawing command.
//
// A command for drawing that is created when Image functions are called like DrawImage,
// or Fill.
// A command is not immediately executed after created. Instaed, it is queued after created,
// and executed only when necessary.
type command interface {
	Exec(indexOffsetInBytes int) error
}

// commandQueue is a command queue for drawing commands.
type commandQueue struct {
	// commands is a queue of drawing commands.
	commands []command

	// vertices represents a vertices data in OpenGL's array buffer.
	vertices []float32

	// verticesNum represents the current length of vertices.
	// verticesNum must <= len(vertices).
	// vertices is never shrunk since re-extending a vertices buffer is heavy.
	verticesNum int

	m sync.Mutex
}

// theCommandQueue is the command queue for the current process.
var theCommandQueue = &commandQueue{}

// appendVertices appends vertices to the queue.
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

// EnqueueDrawImageCommand enqueues a drawing-image command.
func (q *commandQueue) EnqueueDrawImageCommand(dst, src *Image, vertices []float32, clr *affine.ColorM, mode opengl.CompositeMode, filter Filter) {
	// Avoid defer for performance
	q.m.Lock()
	q.appendVertices(vertices)
	if 0 < len(q.commands) {
		if c, ok := q.commands[len(q.commands)-1].(*drawImageCommand); ok {
			if c.canMerge(dst, src, clr, mode, filter) {
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
		filter:      filter,
	}
	q.commands = append(q.commands, c)
	q.m.Unlock()
}

// Enqueue enqueues a drawing command other than a draw-image command.
//
// For a draw-image command, use EnqueueDrawImageCommand.
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

// Flush flushes the command queue.
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

// FlushCommands flushes the command queue.
func FlushCommands() error {
	return theCommandQueue.Flush()
}

// fillCommand represents a drawing command to fill an image with a solid color.
type fillCommand struct {
	dst   *Image
	color color.RGBA
}

// Exec executes the fillCommand.
func (c *fillCommand) Exec(indexOffsetInBytes int) error {
	f, err := c.dst.createFramebufferIfNeeded()
	if err != nil {
		return err
	}
	f.setAsViewport()

	cr, cg, cb, ca := c.color.R, c.color.G, c.color.B, c.color.A
	const max = math.MaxUint8
	r := float32(cr) / max
	g := float32(cg) / max
	b := float32(cb) / max
	a := float32(ca) / max
	if err := opengl.GetContext().FillFramebuffer(r, g, b, a); err != nil {
		return err
	}

	// Flush is needed after filling (#419)
	opengl.GetContext().Flush()
	// Mysterious, but binding texture is needed after filling
	// on some mechines like Photon 2 (#492).
	opengl.GetContext().BindTexture(opengl.InvalidTexture)
	return nil
}

// drawImageCommand represents a drawing command to draw an image on another image.
type drawImageCommand struct {
	dst         *Image
	src         *Image
	verticesNum int
	color       affine.ColorM
	mode        opengl.CompositeMode
	filter      Filter
}

// QuadVertexSizeInBytes returns the size in bytes of vertices for a quadrangle.
func QuadVertexSizeInBytes() int {
	return 4 * theArrayBufferLayout.totalBytes()
}

// Exec executes the drawImageCommand.
func (c *drawImageCommand) Exec(indexOffsetInBytes int) error {
	f, err := c.dst.createFramebufferIfNeeded()
	if err != nil {
		return err
	}
	f.setAsViewport()

	opengl.GetContext().BlendFunc(c.mode)

	n := c.quadsNum()
	if n == 0 {
		return nil
	}
	_, dh := c.dst.Size()
	proj := f.projectionMatrix(dh)
	theOpenGLState.useProgram(proj, c.src.texture.native, c.dst, c.src, c.color, c.filter)
	// TODO: We should call glBindBuffer here?
	// The buffer is already bound at begin() but it is counterintuitive.
	opengl.GetContext().DrawElements(opengl.Triangles, 6*n, indexOffsetInBytes)

	// glFlush() might be necessary at least on MacBook Pro (a smilar problem at #419),
	// but basically this pass the tests (esp. TestImageTooManyFill).
	// As glFlush() causes performance problems, this should be avoided as much as possible.
	// Let's wait and see, and file a new issue when this problem is newly found.
	return nil
}

// split splits the drawImageCommand c into two drawImageCommands.
//
// split is called when the number of vertices reaches of the maximum and
// a command is needed to be executed as another draw call.
func (c *drawImageCommand) split(quadsNum int) [2]*drawImageCommand {
	c1 := *c
	c2 := *c
	s := opengl.Float.SizeInBytes()
	n := quadsNum * QuadVertexSizeInBytes() / s
	c1.verticesNum = n
	c2.verticesNum -= n
	return [2]*drawImageCommand{&c1, &c2}
}

// canMerge returns a boolean value indicating whether the other drawImageCommand can be merged
// with the drawImageCommand c.
func (c *drawImageCommand) canMerge(dst, src *Image, clr *affine.ColorM, mode opengl.CompositeMode, filter Filter) bool {
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
	if c.filter != filter {
		return false
	}
	return true
}

// quadsNum returns the number of quadrangles.
func (c *drawImageCommand) quadsNum() int {
	return c.verticesNum * opengl.Float.SizeInBytes() / QuadVertexSizeInBytes()
}

// replacePixelsCommand represents a command to replace pixels of an image.
type replacePixelsCommand struct {
	dst    *Image
	pixels []byte
}

// Exec executes the replacePixelsCommand.
func (c *replacePixelsCommand) Exec(indexOffsetInBytes int) error {
	f, err := c.dst.createFramebufferIfNeeded()
	if err != nil {
		return err
	}
	f.setAsViewport()

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
	opengl.GetContext().BindTexture(c.dst.texture.native)
	opengl.GetContext().TexSubImage2D(c.pixels, emath.NextPowerOf2Int(c.dst.width), emath.NextPowerOf2Int(c.dst.height))
	return nil
}

// disposeCommand represents a command to dispose an image.
type disposeCommand struct {
	target *Image
}

// Exec executes the disposeCommand.
func (c *disposeCommand) Exec(indexOffsetInBytes int) error {
	if c.target.framebuffer != nil &&
		c.target.framebuffer.native != opengl.GetContext().ScreenFramebuffer() {
		opengl.GetContext().DeleteFramebuffer(c.target.framebuffer.native)
	}
	if c.target.texture != nil {
		opengl.GetContext().DeleteTexture(c.target.texture.native)
	}
	return nil
}

// newImageFromImageCommand represents a command to create an image from an image.RGBA.
type newImageFromImageCommand struct {
	result *Image
	img    *image.RGBA
}

// Exec executes the newImageFromImageCommand.
func (c *newImageFromImageCommand) Exec(indexOffsetInBytes int) error {
	origSize := c.img.Bounds().Size()
	if origSize.X < 1 {
		return errors.New("graphics: width must be equal or more than 1.")
	}
	if origSize.Y < 1 {
		return errors.New("graphics: height must be equal or more than 1.")
	}
	w, h := c.img.Bounds().Size().X, c.img.Bounds().Size().Y
	if c.img.Bounds() != image.Rect(0, 0, emath.NextPowerOf2Int(w), emath.NextPowerOf2Int(h)) {
		panic(fmt.Sprintf("graphics: invalid image bounds: %v", c.img.Bounds()))
	}
	native, err := opengl.GetContext().NewTexture(w, h, c.img.Pix)
	if err != nil {
		return err
	}
	c.result.texture = &texture{
		native: native,
	}
	return nil
}

// newImageCommand represents a command to create an empty image with given width and height.
type newImageCommand struct {
	result *Image
	width  int
	height int
}

// Exec executes a newImageCommand.
func (c *newImageCommand) Exec(indexOffsetInBytes int) error {
	w := emath.NextPowerOf2Int(c.width)
	h := emath.NextPowerOf2Int(c.height)
	if w < 1 {
		return errors.New("graphics: width must be equal or more than 1.")
	}
	if h < 1 {
		return errors.New("graphics: height must be equal or more than 1.")
	}
	native, err := opengl.GetContext().NewTexture(w, h, nil)
	if err != nil {
		return err
	}
	c.result.texture = &texture{
		native: native,
	}
	return nil
}

// newScreenFramebufferImageCommand is a command to create a special image for the screen.
type newScreenFramebufferImageCommand struct {
	result  *Image
	width   int
	height  int
	offsetX float64
	offsetY float64
}

// Exec executes a newScreenFramebufferImageCommand.
func (c *newScreenFramebufferImageCommand) Exec(indexOffsetInBytes int) error {
	if c.width < 1 {
		return errors.New("graphics: width must be equal or more than 1.")
	}
	if c.height < 1 {
		return errors.New("graphics: height must be equal or more than 1.")
	}
	// The (default) framebuffer size can't be converted to a power of 2.
	// On browsers, c.width and c.height are used as viewport size and
	// Edge can't treat a bigger viewport than the drawing area (#71).
	c.result.framebuffer = newScreenFramebuffer(c.width, c.height, c.offsetX, c.offsetY)
	return nil
}
