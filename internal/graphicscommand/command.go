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
	"math"

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/graphics"
)

var theGraphicsDriver driver.Graphics

func SetGraphicsDriver(driver driver.Graphics) {
	theGraphicsDriver = driver
}

func NeedsRestoring() bool {
	if theGraphicsDriver == nil {
		// This happens on initialization.
		// Return true for fail-safe
		return true
	}
	return theGraphicsDriver.NeedsRestoring()
}

// command represents a drawing command.
//
// A command for drawing that is created when Image functions are called like DrawTriangles,
// or Fill.
// A command is not immediately executed after created. Instaed, it is queued after created,
// and executed only when necessary.
type command interface {
	fmt.Stringer

	Exec(indexOffset int) error
	NumVertices() int
	NumIndices() int
	AddNumVertices(n int)
	AddNumIndices(n int)
	CanMergeWithDrawTrianglesCommand(dst, src *Image, color *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address) bool
}

type size struct {
	width  float32
	height float32
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

	srcSizes []size

	indices  []uint16
	nindices int

	tmpNumIndices int
	nextIndex     int

	err error
}

// theCommandQueue is the command queue for the current process.
var theCommandQueue = &commandQueue{}

// appendVertices appends vertices to the queue.
func (q *commandQueue) appendVertices(vertices []float32, width, height float32) {
	if len(q.vertices) < q.nvertices+len(vertices) {
		n := q.nvertices + len(vertices) - len(q.vertices)
		q.vertices = append(q.vertices, make([]float32, n)...)
		q.srcSizes = append(q.srcSizes, make([]size, n/graphics.VertexFloatNum)...)
	}
	copy(q.vertices[q.nvertices:], vertices)

	n := len(vertices) / graphics.VertexFloatNum
	base := q.nvertices / graphics.VertexFloatNum
	for i := 0; i < n; i++ {
		idx := base + i
		q.srcSizes[idx].width = width
		q.srcSizes[idx].height = height
	}
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

// EnqueueDrawTrianglesCommand enqueues a drawing-image command.
func (q *commandQueue) EnqueueDrawTrianglesCommand(dst, src *Image, vertices []float32, indices []uint16, color *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address) {
	if len(indices) > graphics.IndicesNum {
		panic(fmt.Sprintf("graphicscommand: len(indices) must be <= graphics.IndicesNum but not at EnqueueDrawTrianglesCommand: len(indices): %d, graphics.IndicesNum: %d", len(indices), graphics.IndicesNum))
	}

	split := false
	if q.tmpNumIndices+len(indices) > graphics.IndicesNum {
		q.tmpNumIndices = 0
		q.nextIndex = 0
		split = true
	}

	n := len(vertices) / graphics.VertexFloatNum
	iw, ih := src.InternalSize()
	q.appendVertices(vertices, float32(iw), float32(ih))
	q.appendIndices(indices, uint16(q.nextIndex))
	q.nextIndex += n
	q.tmpNumIndices += len(indices)

	// TODO: If dst is the screen, reorder the command to be the last.
	if !split && 0 < len(q.commands) {
		if last := q.commands[len(q.commands)-1]; last.CanMergeWithDrawTrianglesCommand(dst, src, color, mode, filter, address) {
			last.AddNumVertices(len(vertices))
			last.AddNumIndices(len(indices))
			return
		}
	}
	c := &drawTrianglesCommand{
		dst:       dst,
		src:       src,
		nvertices: len(vertices),
		nindices:  len(indices),
		color:     color,
		mode:      mode,
		filter:    filter,
		address:   address,
	}
	q.commands = append(q.commands, c)
}

// Enqueue enqueues a drawing command other than a draw-triangles command.
//
// For a draw-triangles command, use EnqueueDrawTrianglesCommand.
func (q *commandQueue) Enqueue(command command) {
	// TODO: If dst is the screen, reorder the command to be the last.
	q.commands = append(q.commands, command)
}

func fract(x float32) float32 {
	return x - float32(math.Floor(float64(x)))
}

// Flush flushes the command queue.
func (q *commandQueue) Flush() error {
	es := q.indices
	vs := q.vertices
	if recordLog() {
		fmt.Println("--")
	}

	if theGraphicsDriver.HasHighPrecisionFloat() {
		const dstAdjustmentFactor = 1.0 / 256.0
		const texelAdjustmentFactor = 1.0 / 512.0

		n := q.nvertices / graphics.VertexFloatNum
		for i := 0; i < n; i++ {
			s := q.srcSizes[i]

			// Convert pixels to texels.
			vs[i*graphics.VertexFloatNum+2] /= s.width
			vs[i*graphics.VertexFloatNum+3] /= s.height
			vs[i*graphics.VertexFloatNum+4] /= s.width
			vs[i*graphics.VertexFloatNum+5] /= s.height
			vs[i*graphics.VertexFloatNum+6] /= s.width
			vs[i*graphics.VertexFloatNum+7] /= s.height

			// Adjust the destination position to avoid jaggy (#929).
			// This is not a perfect solution since texels on a texture can take a position on borders
			// which can cause jaggy. But adjusting only edges should work in most cases.
			// The ideal solution is to fix shaders, but this makes the applications slow by adding 'if'
			// branches.
			switch f := fract(vs[i*graphics.VertexFloatNum+0]); {
			case 0.5-dstAdjustmentFactor <= f && f < 0.5:
				vs[i*graphics.VertexFloatNum+0] -= f - (0.5 - dstAdjustmentFactor)
			case 0.5 <= f && f < 0.5+dstAdjustmentFactor:
				vs[i*graphics.VertexFloatNum+0] += (0.5 + dstAdjustmentFactor) - f
			}
			switch f := fract(vs[i*graphics.VertexFloatNum+1]); {
			case 0.5-dstAdjustmentFactor <= f && f < 0.5:
				vs[i*graphics.VertexFloatNum+1] -= f - (0.5 - dstAdjustmentFactor)
			case 0.5 <= f && f < 0.5+dstAdjustmentFactor:
				vs[i*graphics.VertexFloatNum+1] += (0.5 + dstAdjustmentFactor) - f
			}

			// Adjust regions not to violate neighborhoods (#317, #558, #724).
			vs[i*graphics.VertexFloatNum+6] -= 1.0 / s.width * texelAdjustmentFactor
			vs[i*graphics.VertexFloatNum+7] -= 1.0 / s.height * texelAdjustmentFactor
		}
	} else {
		n := q.nvertices / graphics.VertexFloatNum
		for i := 0; i < n; i++ {
			s := q.srcSizes[i]

			// Convert pixels to texels.
			vs[i*graphics.VertexFloatNum+2] /= s.width
			vs[i*graphics.VertexFloatNum+3] /= s.height
			vs[i*graphics.VertexFloatNum+4] /= s.width
			vs[i*graphics.VertexFloatNum+5] /= s.height
			vs[i*graphics.VertexFloatNum+6] /= s.width
			vs[i*graphics.VertexFloatNum+7] /= s.height
		}
	}

	theGraphicsDriver.Begin()
	cs := q.commands
	for len(cs) > 0 {
		nv := 0
		ne := 0
		nc := 0
		for _, c := range cs {
			if c.NumIndices() > graphics.IndicesNum {
				panic(fmt.Sprintf("graphicscommand: c.NumIndices() must be <= graphics.IndicesNum but not at Flush: c.NumIndices(): %d, graphics.IndicesNum: %d", c.NumIndices(), graphics.IndicesNum))
			}
			if ne+c.NumIndices() > graphics.IndicesNum {
				break
			}
			nv += c.NumVertices()
			ne += c.NumIndices()
			nc++
		}
		if 0 < ne {
			theGraphicsDriver.SetVertices(vs[:nv], es[:ne])
			es = es[ne:]
			vs = vs[nv:]
		}
		indexOffset := 0
		for _, c := range cs[:nc] {
			if err := c.Exec(indexOffset); err != nil {
				return err
			}
			if recordLog() {
				fmt.Printf("%s\n", c)
			}
			// TODO: indexOffset should be reset if the command type is different
			// from the previous one. This fix is needed when another drawing command is
			// introduced than drawTrianglesCommand.
			indexOffset += c.NumIndices()
		}
		cs = cs[nc:]
	}
	theGraphicsDriver.End()
	q.commands = q.commands[:0]
	q.nvertices = 0
	q.nindices = 0
	q.tmpNumIndices = 0
	q.nextIndex = 0
	return nil
}

// FlushCommands flushes the command queue.
func FlushCommands() error {
	return theCommandQueue.Flush()
}

// drawTrianglesCommand represents a drawing command to draw an image on another image.
type drawTrianglesCommand struct {
	dst       *Image
	src       *Image
	nvertices int
	nindices  int
	color     *affine.ColorM
	mode      driver.CompositeMode
	filter    driver.Filter
	address   driver.Address
}

func (c *drawTrianglesCommand) String() string {
	mode := ""
	switch c.mode {
	case driver.CompositeModeSourceOver:
		mode = "source-over"
	case driver.CompositeModeClear:
		mode = "clear"
	case driver.CompositeModeCopy:
		mode = "copy"
	case driver.CompositeModeDestination:
		mode = "destination"
	case driver.CompositeModeDestinationOver:
		mode = "destination-over"
	case driver.CompositeModeSourceIn:
		mode = "source-in"
	case driver.CompositeModeDestinationIn:
		mode = "destination-in"
	case driver.CompositeModeSourceOut:
		mode = "source-out"
	case driver.CompositeModeDestinationOut:
		mode = "destination-out"
	case driver.CompositeModeSourceAtop:
		mode = "source-atop"
	case driver.CompositeModeDestinationAtop:
		mode = "destination-atop"
	case driver.CompositeModeXor:
		mode = "xor"
	case driver.CompositeModeLighter:
		mode = "lighter"
	default:
		panic(fmt.Sprintf("graphicscommand: invalid composite mode: %d", c.mode))
	}

	filter := ""
	switch c.filter {
	case driver.FilterNearest:
		filter = "nearest"
	case driver.FilterLinear:
		filter = "linear"
	case driver.FilterScreen:
		filter = "screen"
	default:
		panic(fmt.Sprintf("graphicscommand: invalid filter: %d", c.filter))
	}

	address := ""
	switch c.address {
	case driver.AddressClampToZero:
		address = "clamp_to_zero"
	case driver.AddressRepeat:
		address = "repeat"
	default:
		panic(fmt.Sprintf("graphicscommand: invalid address: %d", c.address))
	}

	dst := fmt.Sprintf("%d", c.dst.id)
	if c.dst.screen {
		dst += " (screen)"
	}
	src := fmt.Sprintf("%d", c.src.id)
	if c.src.screen {
		src += " (screen)"
	}

	return fmt.Sprintf("draw-triangles: dst: %s <- src: %s, colorm: %v, mode %s, filter: %s, address: %s", dst, src, c.color, mode, filter, address)
}

// Exec executes the drawTrianglesCommand.
func (c *drawTrianglesCommand) Exec(indexOffset int) error {
	// TODO: Is it ok not to bind any framebuffer here?
	if c.nindices == 0 {
		return nil
	}

	c.dst.image.SetAsDestination()
	c.src.image.SetAsSource()
	if err := theGraphicsDriver.Draw(c.nindices, indexOffset, c.mode, c.color, c.filter, c.address); err != nil {
		return err
	}
	return nil
}

func (c *drawTrianglesCommand) NumVertices() int {
	return c.nvertices
}

func (c *drawTrianglesCommand) NumIndices() int {
	return c.nindices
}

func (c *drawTrianglesCommand) AddNumVertices(n int) {
	c.nvertices += n
}

func (c *drawTrianglesCommand) AddNumIndices(n int) {
	c.nindices += n
}

// CanMergeWithDrawTrianglesCommand returns a boolean value indicating whether the other drawTrianglesCommand can be merged
// with the drawTrianglesCommand c.
func (c *drawTrianglesCommand) CanMergeWithDrawTrianglesCommand(dst, src *Image, color *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address) bool {
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
	if c.address != address {
		return false
	}
	return true
}

// replacePixelsCommand represents a command to replace pixels of an image.
type replacePixelsCommand struct {
	dst  *Image
	args []*driver.ReplacePixelsArgs
}

func (c *replacePixelsCommand) String() string {
	return fmt.Sprintf("replace-pixels: dst: %d, len(args): %d", c.dst.id, len(c.args))
}

// Exec executes the replacePixelsCommand.
func (c *replacePixelsCommand) Exec(indexOffset int) error {
	c.dst.image.ReplacePixels(c.args)
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

func (c *replacePixelsCommand) CanMergeWithDrawTrianglesCommand(dst, src *Image, color *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address) bool {
	return false
}

type pixelsCommand struct {
	result []byte
	img    *Image
}

// Exec executes a pixelsCommand.
func (c *pixelsCommand) Exec(indexOffset int) error {
	p, err := c.img.image.Pixels()
	if err != nil {
		return err
	}
	c.result = p
	return nil
}

func (c *pixelsCommand) String() string {
	return fmt.Sprintf("pixels: image: %d", c.img.id)
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

func (c *pixelsCommand) CanMergeWithDrawTrianglesCommand(dst, src *Image, color *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address) bool {
	return false
}

// disposeCommand represents a command to dispose an image.
type disposeCommand struct {
	target *Image
}

func (c *disposeCommand) String() string {
	return fmt.Sprintf("dispose: target: %d", c.target.id)
}

// Exec executes the disposeCommand.
func (c *disposeCommand) Exec(indexOffset int) error {
	c.target.image.Dispose()
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

func (c *disposeCommand) CanMergeWithDrawTrianglesCommand(dst, src *Image, color *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address) bool {
	return false
}

// newImageCommand represents a command to create an empty image with given width and height.
type newImageCommand struct {
	result *Image
	width  int
	height int
}

func (c *newImageCommand) String() string {
	return fmt.Sprintf("new-image: result: %d, width: %d, height: %d", c.result.id, c.width, c.height)
}

// Exec executes a newImageCommand.
func (c *newImageCommand) Exec(indexOffset int) error {
	i, err := theGraphicsDriver.NewImage(c.width, c.height)
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

func (c *newImageCommand) CanMergeWithDrawTrianglesCommand(dst, src *Image, color *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address) bool {
	return false
}

// newScreenFramebufferImageCommand is a command to create a special image for the screen.
type newScreenFramebufferImageCommand struct {
	result *Image
	width  int
	height int
}

func (c *newScreenFramebufferImageCommand) String() string {
	return fmt.Sprintf("new-screen-framebuffer-image: result: %d, width: %d, height: %d", c.result.id, c.width, c.height)
}

// Exec executes a newScreenFramebufferImageCommand.
func (c *newScreenFramebufferImageCommand) Exec(indexOffset int) error {
	var err error
	c.result.image, err = theGraphicsDriver.NewScreenFramebufferImage(c.width, c.height)
	return err
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

func (c *newScreenFramebufferImageCommand) CanMergeWithDrawTrianglesCommand(dst, src *Image, color *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address) bool {
	return false
}

// ResetGraphicsDriverState resets or initializes the current graphics driver state.
func ResetGraphicsDriverState() error {
	return theGraphicsDriver.Reset()
}
