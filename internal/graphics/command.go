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
	"fmt"

	"github.com/hajimehoshi/ebiten/internal/affine"
	emath "github.com/hajimehoshi/ebiten/internal/math"
	"github.com/hajimehoshi/ebiten/internal/opengl"
)

// command represents a drawing command.
//
// A command for drawing that is created when Image functions are called like DrawImage,
// or Fill.
// A command is not immediately executed after created. Instaed, it is queued after created,
// and executed only when necessary.
type command interface {
	Exec(indexOffsetInBytes int) error
	NumVertices() int
	NumElements() int
	AddNumVertices(n int)
	AddNumElements(n int)
	CanMerge(dst, src *Image, color *affine.ColorM, mode opengl.CompositeMode, filter Filter) bool
}

// commandQueue is a command queue for drawing commands.
type commandQueue struct {
	// commands is a queue of drawing commands.
	commands []command

	// vertices represents a vertices data in OpenGL's array buffer.
	vertices []float32

	// nvertices represents the current length of vertices.
	// nvertices must <= len(vertices).
	// vertices is never shrunk since re-extending a vertices buffer is heavy.
	nvertices int

	elements  []uint16
	nelements int
	nextIndex int
}

// theCommandQueue is the command queue for the current process.
var theCommandQueue = &commandQueue{}

// appendVertices appends vertices to the queue.
func (q *commandQueue) appendVertices(vertices []float32) {
	if len(q.vertices) < q.nvertices+len(vertices) {
		n := q.nvertices + len(vertices) - len(q.vertices)
		q.vertices = append(q.vertices, make([]float32, n)...)
	}
	// for-loop might be faster than copy:
	// On GopherJS, copy might cause subarray calls.
	for i := 0; i < len(vertices); i++ {
		q.vertices[q.nvertices+i] = vertices[i]
	}
	q.nvertices += len(vertices)
}

func (q *commandQueue) appendElements(e0, e1, e2, e3, e4, e5 uint16) {
	// For GopherJS performance, take 6 arguments instead of an array.
	if len(q.elements) < q.nelements+6 {
		n := q.nelements + 6 - len(q.elements)
		q.elements = append(q.elements, make([]uint16, n)...)
	}
	q.elements[q.nelements+0] = e0
	q.elements[q.nelements+1] = e1
	q.elements[q.nelements+2] = e2
	q.elements[q.nelements+3] = e3
	q.elements[q.nelements+4] = e4
	q.elements[q.nelements+5] = e5
	q.nelements += 6
}

// EnqueueDrawImageCommand enqueues a drawing-image command.
func (q *commandQueue) EnqueueDrawImageCommand(dst, src *Image, vertices []float32, color *affine.ColorM, mode opengl.CompositeMode, filter Filter) {
	// Avoid defer for performance
	q.appendVertices(vertices)
	nelements := 6 * len(vertices) * opengl.Float.SizeInBytes() / QuadVertexSizeInBytes()
	for i := 0; i < nelements/6; i++ {
		if q.nelements%indicesNum > (q.nelements+6)%indicesNum {
			q.nextIndex = 0
		}
		q.appendElements(
			uint16(q.nextIndex+0),
			uint16(q.nextIndex+1),
			uint16(q.nextIndex+2),
			uint16(q.nextIndex+1),
			uint16(q.nextIndex+2),
			uint16(q.nextIndex+3),
		)
		q.nextIndex += 4
	}
	if 0 < len(q.commands) {
		last := q.commands[len(q.commands)-1]
		if last.CanMerge(dst, src, color, mode, filter) {
			last.AddNumVertices(len(vertices))
			last.AddNumElements(nelements)
			return
		}
	}
	c := &drawImageCommand{
		dst:       dst,
		src:       src,
		nvertices: len(vertices),
		nelements: nelements,
		color:     color,
		mode:      mode,
		filter:    filter,
	}
	q.commands = append(q.commands, c)
}

// Enqueue enqueues a drawing command other than a draw-image command.
//
// For a draw-image command, use EnqueueDrawImageCommand.
func (q *commandQueue) Enqueue(command command) {
	q.commands = append(q.commands, command)
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
			// NOTE: WebGL doesn't seem to have gl.MAX_ELEMENTS_VERTICES or
			// gl.MAX_ELEMENTS_INDICES so far.
			// Let's use them to compare to len(quads) in the future.
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
	// glViewport must be called at least at every frame on iOS.
	opengl.GetContext().ResetViewportSize()
	nv := 0
	lastNv := 0
	ne := 0
	lastNe := 0
	for _, g := range q.commandGroups() {
		for _, c := range g {
			nv += c.NumVertices()
			ne += c.NumElements()
		}
		if 0 < ne-lastNe {
			// Note that the vertices passed to BufferSubData is not under GC management
			// in opengl package due to unsafe-way.
			// See BufferSubData in context_mobile.go.
			opengl.GetContext().ElementArrayBufferSubData(q.elements[lastNe:ne])
			opengl.GetContext().ArrayBufferSubData(q.vertices[lastNv:nv])
		}
		numc := len(g)
		indexOffsetInBytes := 0
		for _, c := range g {
			if err := c.Exec(indexOffsetInBytes); err != nil {
				return err
			}
			// TODO: indexOffsetInBytes should be reset if the command type is different
			// from the previous one. This fix is needed when another drawing command is
			// introduced than drawImageCommand.
			indexOffsetInBytes += c.NumElements() * 2 // 2 is uint16 size in bytes
		}
		if 0 < numc {
			// Call glFlush to prevent black flicking (especially on Android (#226) and iOS).
			opengl.GetContext().Flush()
		}
		lastNv = nv
		lastNe = ne
	}
	q.commands = nil
	q.nvertices = 0
	q.nelements = 0
	q.nextIndex = 0
	return nil
}

// FlushCommands flushes the command queue.
func FlushCommands() error {
	return theCommandQueue.Flush()
}

// drawImageCommand represents a drawing command to draw an image on another image.
type drawImageCommand struct {
	dst       *Image
	src       *Image
	nvertices int
	nelements int
	color     *affine.ColorM
	mode      opengl.CompositeMode
	filter    Filter
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

	if c.nelements == 0 {
		return nil
	}
	proj := f.projectionMatrix()
	theOpenGLState.useProgram(proj, c.src.texture.native, c.dst, c.src, c.color, c.filter)
	opengl.GetContext().DrawElements(opengl.Triangles, c.nelements, indexOffsetInBytes)

	// glFlush() might be necessary at least on MacBook Pro (a smilar problem at #419),
	// but basically this pass the tests (esp. TestImageTooManyFill).
	// As glFlush() causes performance problems, this should be avoided as much as possible.
	// Let's wait and see, and file a new issue when this problem is newly found.
	return nil
}

func (c *drawImageCommand) NumVertices() int {
	return c.nvertices
}

func (c *drawImageCommand) NumElements() int {
	return c.nelements
}

func (c *drawImageCommand) AddNumVertices(n int) {
	c.nvertices += n
}

func (c *drawImageCommand) AddNumElements(n int) {
	c.nelements += n
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
	c1.nvertices = n
	c2.nvertices -= n
	c1.nelements = 6 * quadsNum
	c2.nelements -= 6 * quadsNum
	return [2]*drawImageCommand{&c1, &c2}
}

// CanMerge returns a boolean value indicating whether the other drawImageCommand can be merged
// with the drawImageCommand c.
func (c *drawImageCommand) CanMerge(dst, src *Image, color *affine.ColorM, mode opengl.CompositeMode, filter Filter) bool {
	if c.dst != dst {
		return false
	}
	if c.src != src {
		return false
	}
	if !c.color.Equals(color) {
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
	return c.nvertices * opengl.Float.SizeInBytes() / QuadVertexSizeInBytes()
}

// replacePixelsCommand represents a command to replace pixels of an image.
type replacePixelsCommand struct {
	dst    *Image
	pixels []byte
	x      int
	y      int
	width  int
	height int
}

// Exec executes the replacePixelsCommand.
func (c *replacePixelsCommand) Exec(indexOffsetInBytes int) error {
	// glFlush is necessary on Android.
	// glTexSubImage2D didn't work without this hack at least on Nexus 5x and NuAns NEO [Reloaded] (#211).
	opengl.GetContext().Flush()
	opengl.GetContext().BindTexture(c.dst.texture.native)
	opengl.GetContext().TexSubImage2D(c.pixels, c.x, c.y, c.width, c.height)
	return nil
}

func (c *replacePixelsCommand) NumVertices() int {
	return 0
}

func (c *replacePixelsCommand) NumElements() int {
	return 0
}

func (c *replacePixelsCommand) AddNumVertices(n int) {
}

func (c *replacePixelsCommand) AddNumElements(n int) {
}

func (c *replacePixelsCommand) CanMerge(dst, src *Image, color *affine.ColorM, mode opengl.CompositeMode, filter Filter) bool {
	return false
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

func (c *disposeCommand) NumVertices() int {
	return 0
}

func (c *disposeCommand) NumElements() int {
	return 0
}

func (c *disposeCommand) AddNumVertices(n int) {
}

func (c *disposeCommand) AddNumElements(n int) {
}

func (c *disposeCommand) CanMerge(dst, src *Image, color *affine.ColorM, mode opengl.CompositeMode, filter Filter) bool {
	return false
}

// newImageCommand represents a command to create an empty image with given width and height.
type newImageCommand struct {
	result *Image
	width  int
	height int
}

func checkSize(width, height int) {
	if width < 1 {
		panic(fmt.Sprintf("graphics: width (%d) must be equal or more than 1.", width))
	}
	if height < 1 {
		panic(fmt.Sprintf("graphics: height (%d) must be equal or more than 1.", height))
	}
	m := MaxImageSize()
	if width > m {
		panic(fmt.Sprintf("graphics: width (%d) must be less than or equal to %d", width, m))
	}
	if height > m {
		panic(fmt.Sprintf("graphics: height (%d) must be less than or equal to %d", height, m))
	}
}

// Exec executes a newImageCommand.
func (c *newImageCommand) Exec(indexOffsetInBytes int) error {
	w := emath.NextPowerOf2Int(c.width)
	h := emath.NextPowerOf2Int(c.height)
	checkSize(w, h)
	native, err := opengl.GetContext().NewTexture(w, h)
	if err != nil {
		return err
	}
	c.result.texture = &texture{
		native: native,
	}
	return nil
}

func (c *newImageCommand) NumVertices() int {
	return 0
}

func (c *newImageCommand) NumElements() int {
	return 0
}

func (c *newImageCommand) AddNumVertices(n int) {
}

func (c *newImageCommand) AddNumElements(n int) {
}

func (c *newImageCommand) CanMerge(dst, src *Image, color *affine.ColorM, mode opengl.CompositeMode, filter Filter) bool {
	return false
}

// newScreenFramebufferImageCommand is a command to create a special image for the screen.
type newScreenFramebufferImageCommand struct {
	result *Image
	width  int
	height int
}

// Exec executes a newScreenFramebufferImageCommand.
func (c *newScreenFramebufferImageCommand) Exec(indexOffsetInBytes int) error {
	checkSize(c.width, c.height)
	// The (default) framebuffer size can't be converted to a power of 2.
	// On browsers, c.width and c.height are used as viewport size and
	// Edge can't treat a bigger viewport than the drawing area (#71).
	c.result.framebuffer = newScreenFramebuffer(c.width, c.height)
	return nil
}

func (c *newScreenFramebufferImageCommand) NumVertices() int {
	return 0
}

func (c *newScreenFramebufferImageCommand) NumElements() int {
	return 0
}

func (c *newScreenFramebufferImageCommand) AddNumVertices(n int) {
}

func (c *newScreenFramebufferImageCommand) AddNumElements(n int) {
}

func (c *newScreenFramebufferImageCommand) CanMerge(dst, src *Image, color *affine.ColorM, mode opengl.CompositeMode, filter Filter) bool {
	return false
}
