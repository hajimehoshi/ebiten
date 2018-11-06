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

package graphicscommand

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/opengl"
)

// command represents a drawing command.
//
// A command for drawing that is created when Image functions are called like DrawImage,
// or Fill.
// A command is not immediately executed after created. Instaed, it is queued after created,
// and executed only when necessary.
type command interface {
	fmt.Stringer

	Exec(indexOffsetInBytes int) error
	NumVertices() int
	NumIndices() int
	AddNumVertices(n int)
	AddNumIndices(n int)
	CanMerge(dst, src *Image, color *affine.ColorM, mode graphics.CompositeMode, filter graphics.Filter) bool
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
	//
	// TODO: This is a number of float32 values, not a number of vertices.
	// Rename or fix the program.
	nvertices int

	indices  []uint16
	nindices int

	tmpNumIndices int
	nextIndex     int

	err error
}

// theCommandQueue is the command queue for the current process.
var theCommandQueue = &commandQueue{}

// appendVertices appends vertices to the queue.
func (q *commandQueue) appendVertices(vertices []float32) {
	if len(q.vertices) < q.nvertices+len(vertices) {
		n := q.nvertices + len(vertices) - len(q.vertices)
		q.vertices = append(q.vertices, make([]float32, n)...)
	}
	copy(q.vertices[q.nvertices:], vertices)
	q.nvertices += len(vertices)
}

func (q *commandQueue) appendIndices(indices []uint16, offset uint16) {
	if len(q.indices) < q.nindices+len(indices) {
		n := q.nindices + len(indices) - len(q.indices)
		q.indices = append(q.indices, make([]uint16, n)...)
	}
	for i := range indices {
		q.indices[q.nindices+i] = indices[i] + offset
	}
	q.nindices += len(indices)
}

func (q *commandQueue) doEnqueueDrawImageCommand(dst, src *Image, nvertices, nindices int, color *affine.ColorM, mode graphics.CompositeMode, filter graphics.Filter, forceNewCommand bool) {
	if nindices > graphics.IndicesNum {
		panic("not implemented for too many indices")
	}
	if !forceNewCommand && 0 < len(q.commands) {
		if last := q.commands[len(q.commands)-1]; last.CanMerge(dst, src, color, mode, filter) {
			last.AddNumVertices(nvertices)
			last.AddNumIndices(nindices)
			return
		}
	}
	c := &drawImageCommand{
		dst:       dst,
		src:       src,
		nvertices: nvertices,
		nindices:  nindices,
		color:     color,
		mode:      mode,
		filter:    filter,
	}
	q.commands = append(q.commands, c)
}

// EnqueueDrawImageCommand enqueues a drawing-image command.
func (q *commandQueue) EnqueueDrawImageCommand(dst, src *Image, vertices []float32, indices []uint16, color *affine.ColorM, mode graphics.CompositeMode, filter graphics.Filter) {
	if len(indices) > graphics.IndicesNum {
		panic("not reached")
	}

	split := false
	if q.tmpNumIndices+len(indices) > graphics.IndicesNum {
		q.tmpNumIndices = 0
		q.nextIndex = 0
		split = true
	}

	q.appendVertices(vertices)
	q.appendIndices(indices, uint16(q.nextIndex))
	q.nextIndex += len(vertices) / graphics.VertexFloatNum
	q.tmpNumIndices += len(indices)

	q.doEnqueueDrawImageCommand(dst, src, len(vertices), len(indices), color, mode, filter, split)
}

// Enqueue enqueues a drawing command other than a draw-image command.
//
// For a draw-image command, use EnqueueDrawImageCommand.
func (q *commandQueue) Enqueue(command command) {
	q.commands = append(q.commands, command)
}

// Flush flushes the command queue.
func (q *commandQueue) Flush() {
	if q.err != nil {
		return
	}

	es := q.indices
	vs := q.vertices
	if recordLog() {
		fmt.Println("--")
	}
	for len(q.commands) > 0 {
		nv := 0
		ne := 0
		nc := 0
		for _, c := range q.commands {
			if c.NumIndices() > graphics.IndicesNum {
				panic("not reached")
			}
			if ne+c.NumIndices() > graphics.IndicesNum {
				break
			}
			nv += c.NumVertices()
			ne += c.NumIndices()
			nc++
		}
		if 0 < ne {
			// Note that the vertices passed to BufferSubData is not under GC management
			// in opengl package due to unsafe-way.
			// See BufferSubData in context_mobile.go.
			opengl.GetDriver().BufferSubData(vs[:nv], es[:ne])
			es = es[ne:]
			vs = vs[nv:]
		}
		indexOffsetInBytes := 0
		for _, c := range q.commands[:nc] {
			if err := c.Exec(indexOffsetInBytes); err != nil {
				q.err = err
				return
			}
			if recordLog() {
				fmt.Printf("%s\n", c)
			}
			// TODO: indexOffsetInBytes should be reset if the command type is different
			// from the previous one. This fix is needed when another drawing command is
			// introduced than drawImageCommand.
			indexOffsetInBytes += c.NumIndices() * 2 // 2 is uint16 size in bytes
		}
		if 0 < nc {
			// Call glFlush to prevent black flicking (especially on Android (#226) and iOS).
			opengl.GetDriver().Flush()
		}
		q.commands = q.commands[nc:]
	}
	q.commands = nil
	q.nvertices = 0
	q.nindices = 0
	q.tmpNumIndices = 0
	q.nextIndex = 0
}

// Error returns an OpenGL error for the last command.
func Error() error {
	return theCommandQueue.err
}

// FlushCommands flushes the command queue.
func FlushCommands() {
	theCommandQueue.Flush()
}

// drawImageCommand represents a drawing command to draw an image on another image.
type drawImageCommand struct {
	dst       *Image
	src       *Image
	nvertices int
	nindices  int
	color     *affine.ColorM
	mode      graphics.CompositeMode
	filter    graphics.Filter
}

func (c *drawImageCommand) String() string {
	return fmt.Sprintf("draw-image: dst: %p <- src: %p, colorm: %v, mode %d, filter: %d", c.dst, c.src, c.color, c.mode, c.filter)
}

// Exec executes the drawImageCommand.
func (c *drawImageCommand) Exec(indexOffsetInBytes int) error {
	// TODO: Is it ok not to bind any framebuffer here?
	if c.nindices == 0 {
		return nil
	}

	c.dst.image.SetAsDestination()
	c.src.image.SetAsSource()
	if err := opengl.GetDriver().UseProgram(c.mode, c.color, c.filter); err != nil {
		return err
	}
	opengl.GetDriver().DrawElements(c.nindices, indexOffsetInBytes)

	// glFlush() might be necessary at least on MacBook Pro (a smilar problem at #419),
	// but basically this pass the tests (esp. TestImageTooManyFill).
	// As glFlush() causes performance problems, this should be avoided as much as possible.
	// Let's wait and see, and file a new issue when this problem is newly found.
	return nil
}

func (c *drawImageCommand) NumVertices() int {
	return c.nvertices
}

func (c *drawImageCommand) NumIndices() int {
	return c.nindices
}

func (c *drawImageCommand) AddNumVertices(n int) {
	c.nvertices += n
}

func (c *drawImageCommand) AddNumIndices(n int) {
	c.nindices += n
}

// CanMerge returns a boolean value indicating whether the other drawImageCommand can be merged
// with the drawImageCommand c.
func (c *drawImageCommand) CanMerge(dst, src *Image, color *affine.ColorM, mode graphics.CompositeMode, filter graphics.Filter) bool {
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

// replacePixelsCommand represents a command to replace pixels of an image.
type replacePixelsCommand struct {
	dst    *Image
	pixels []byte
	x      int
	y      int
	width  int
	height int
}

func (c *replacePixelsCommand) String() string {
	return fmt.Sprintf("replace-pixels: dst: %p, x: %d, y: %d, width: %d, height: %d", c.dst, c.x, c.y, c.width, c.height)
}

// Exec executes the replacePixelsCommand.
func (c *replacePixelsCommand) Exec(indexOffsetInBytes int) error {
	// glFlush is necessary on Android.
	// glTexSubImage2D didn't work without this hack at least on Nexus 5x and NuAns NEO [Reloaded] (#211).
	opengl.GetDriver().Flush()
	c.dst.image.TexSubImage2D(c.pixels, c.x, c.y, c.width, c.height)
	return nil
}

func (c *replacePixelsCommand) NumVertices() int {
	return 0
}

func (c *replacePixelsCommand) NumIndices() int {
	return 0
}

func (c *replacePixelsCommand) AddNumVertices(n int) {
}

func (c *replacePixelsCommand) AddNumIndices(n int) {
}

func (c *replacePixelsCommand) CanMerge(dst, src *Image, color *affine.ColorM, mode graphics.CompositeMode, filter graphics.Filter) bool {
	return false
}

type pixelsCommand struct {
	result []byte
	img    *Image
}

// Exec executes a pixelsCommand.
func (c *pixelsCommand) Exec(indexOffsetInBytes int) error {
	p, err := c.img.image.Pixels()
	if err != nil {
		return err
	}
	c.result = p
	return nil
}

func (c *pixelsCommand) String() string {
	return fmt.Sprintf("pixels: img: %p", c.img)
}

func (c *pixelsCommand) NumVertices() int {
	return 0
}

func (c *pixelsCommand) NumIndices() int {
	return 0
}

func (c *pixelsCommand) AddNumVertices(n int) {
}

func (c *pixelsCommand) AddNumIndices(n int) {
}

func (c *pixelsCommand) CanMerge(dst, src *Image, color *affine.ColorM, mode graphics.CompositeMode, filter graphics.Filter) bool {
	return false
}

// disposeCommand represents a command to dispose an image.
type disposeCommand struct {
	target *Image
}

func (c *disposeCommand) String() string {
	return fmt.Sprintf("dispose: target: %p", c.target)
}

// Exec executes the disposeCommand.
func (c *disposeCommand) Exec(indexOffsetInBytes int) error {
	c.target.image.Delete()
	return nil
}

func (c *disposeCommand) NumVertices() int {
	return 0
}

func (c *disposeCommand) NumIndices() int {
	return 0
}

func (c *disposeCommand) AddNumVertices(n int) {
}

func (c *disposeCommand) AddNumIndices(n int) {
}

func (c *disposeCommand) CanMerge(dst, src *Image, color *affine.ColorM, mode graphics.CompositeMode, filter graphics.Filter) bool {
	return false
}

// newImageCommand represents a command to create an empty image with given width and height.
type newImageCommand struct {
	result *Image
	width  int
	height int
}

func (c *newImageCommand) String() string {
	return fmt.Sprintf("new-image: result: %p, width: %d, height: %d", c.result, c.width, c.height)
}

// Exec executes a newImageCommand.
func (c *newImageCommand) Exec(indexOffsetInBytes int) error {
	i, err := opengl.GetDriver().NewImage(c.width, c.height)
	if err != nil {
		return err
	}
	c.result.image = i
	return nil
}

func (c *newImageCommand) NumVertices() int {
	return 0
}

func (c *newImageCommand) NumIndices() int {
	return 0
}

func (c *newImageCommand) AddNumVertices(n int) {
}

func (c *newImageCommand) AddNumIndices(n int) {
}

func (c *newImageCommand) CanMerge(dst, src *Image, color *affine.ColorM, mode graphics.CompositeMode, filter graphics.Filter) bool {
	return false
}

// newScreenFramebufferImageCommand is a command to create a special image for the screen.
type newScreenFramebufferImageCommand struct {
	result *Image
	width  int
	height int
}

func (c *newScreenFramebufferImageCommand) String() string {
	return fmt.Sprintf("new-screen-framebuffer-image: result: %p, width: %d, height: %d", c.result, c.width, c.height)
}

// Exec executes a newScreenFramebufferImageCommand.
func (c *newScreenFramebufferImageCommand) Exec(indexOffsetInBytes int) error {
	c.result.image = opengl.GetDriver().NewScreenFramebufferImage(c.width, c.height)
	return nil
}

func (c *newScreenFramebufferImageCommand) NumVertices() int {
	return 0
}

func (c *newScreenFramebufferImageCommand) NumIndices() int {
	return 0
}

func (c *newScreenFramebufferImageCommand) AddNumVertices(n int) {
}

func (c *newScreenFramebufferImageCommand) AddNumIndices(n int) {
}

func (c *newScreenFramebufferImageCommand) CanMerge(dst, src *Image, color *affine.ColorM, mode graphics.CompositeMode, filter graphics.Filter) bool {
	return false
}

// ResetGraphicsDriverState resets or initializes the current graphics driver state.
func ResetGraphicsDriverState() error {
	return opengl.GetDriver().Reset()
}
