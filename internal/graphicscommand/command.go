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
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/affine"
	"github.com/hajimehoshi/ebiten/v2/internal/debug"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

// command represents a drawing command.
//
// A command for drawing that is created when Image functions are called like DrawTriangles,
// or Fill.
// A command is not immediately executed after created. Instaed, it is queued after created,
// and executed only when necessary.
type command interface {
	fmt.Stringer

	Exec(graphicsDriver graphicsdriver.Graphics, indexOffset int) error
}

type size struct {
	width  float32
	height float32
}

type drawTrianglesCommandPool struct {
	pool []*drawTrianglesCommand
}

func (p *drawTrianglesCommandPool) get() *drawTrianglesCommand {
	if len(p.pool) == 0 {
		return &drawTrianglesCommand{}
	}
	v := p.pool[len(p.pool)-1]
	p.pool = p.pool[:len(p.pool)-1]
	return v
}

func (p *drawTrianglesCommandPool) put(v *drawTrianglesCommand) {
	if len(p.pool) >= 1024 {
		return
	}
	p.pool = append(p.pool, v)
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

	tmpNumVertexFloats int
	tmpNumIndices      int

	drawTrianglesCommandPool drawTrianglesCommandPool
}

// theCommandQueue is the command queue for the current process.
var theCommandQueue = &commandQueue{}

// appendVertices appends vertices to the queue.
func (q *commandQueue) appendVertices(vertices []float32, src *Image) {
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

// mustUseDifferentVertexBuffer reports whether a differnt vertex buffer must be used.
func mustUseDifferentVertexBuffer(nextNumVertexFloats, nextNumIndices int) bool {
	return nextNumVertexFloats > graphics.IndicesNum*graphics.VertexFloatNum || nextNumIndices > graphics.IndicesNum
}

// EnqueueDrawTrianglesCommand enqueues a drawing-image command.
func (q *commandQueue) EnqueueDrawTrianglesCommand(dst *Image, srcs [graphics.ShaderImageNum]*Image, offsets [graphics.ShaderImageNum - 1][2]float32, vertices []float32, indices []uint16, color affine.ColorM, mode graphicsdriver.CompositeMode, filter graphicsdriver.Filter, address graphicsdriver.Address, dstRegion, srcRegion graphicsdriver.Region, shader *Shader, uniforms [][]float32, evenOdd bool) {
	if len(indices) > graphics.IndicesNum {
		panic(fmt.Sprintf("graphicscommand: len(indices) must be <= graphics.IndicesNum but not at EnqueueDrawTrianglesCommand: len(indices): %d, graphics.IndicesNum: %d", len(indices), graphics.IndicesNum))
	}

	split := false
	if mustUseDifferentVertexBuffer(q.tmpNumVertexFloats+len(vertices), q.tmpNumIndices+len(indices)) {
		q.tmpNumVertexFloats = 0
		q.tmpNumIndices = 0
		split = true
	}

	// Assume that all the image sizes are same.
	// Assume that the images are packed from the front in the slice srcs.
	q.appendVertices(vertices, srcs[0])
	q.appendIndices(indices, uint16(q.tmpNumVertexFloats/graphics.VertexFloatNum))
	q.tmpNumVertexFloats += len(vertices)
	q.tmpNumIndices += len(indices)

	if srcs[0] != nil {
		w, h := srcs[0].InternalSize()
		srcRegion.X /= float32(w)
		srcRegion.Y /= float32(h)
		srcRegion.Width /= float32(w)
		srcRegion.Height /= float32(h)
		for i := range offsets {
			offsets[i][0] /= float32(w)
			offsets[i][1] /= float32(h)
		}
	}

	// TODO: If dst is the screen, reorder the command to be the last.
	if !split && 0 < len(q.commands) {
		if last, ok := q.commands[len(q.commands)-1].(*drawTrianglesCommand); ok {
			if last.CanMergeWithDrawTrianglesCommand(dst, srcs, vertices, color, mode, filter, address, dstRegion, srcRegion, shader, uniforms, evenOdd) {
				last.setVertices(q.lastVertices(len(vertices) + last.numVertices()))
				last.addNumIndices(len(indices))
				return
			}
		}
	}

	c := q.drawTrianglesCommandPool.get()
	c.dst = dst
	c.srcs = srcs
	c.offsets = offsets
	c.vertices = q.lastVertices(len(vertices))
	c.nindices = len(indices)
	c.color = color
	c.mode = mode
	c.filter = filter
	c.address = address
	c.dstRegion = dstRegion
	c.srcRegion = srcRegion
	c.shader = shader
	c.uniforms = uniforms
	c.evenOdd = evenOdd
	q.commands = append(q.commands, c)
}

func (q *commandQueue) lastVertices(n int) []float32 {
	return q.vertices[q.nvertices-n : q.nvertices]
}

// Enqueue enqueues a drawing command other than a draw-triangles command.
//
// For a draw-triangles command, use EnqueueDrawTrianglesCommand.
func (q *commandQueue) Enqueue(command command) {
	// TODO: If dst is the screen, reorder the command to be the last.
	q.commands = append(q.commands, command)
}

// Flush flushes the command queue.
func (q *commandQueue) Flush(graphicsDriver graphicsdriver.Graphics) (err error) {
	runOnRenderingThread(func() {
		err = q.flush(graphicsDriver)
	})
	return
}

// flush must be called the main thread.
func (q *commandQueue) flush(graphicsDriver graphicsdriver.Graphics) error {
	if len(q.commands) == 0 {
		return nil
	}

	es := q.indices
	vs := q.vertices
	debug.Logf("Graphics commands:\n")

	if err := graphicsDriver.Begin(); err != nil {
		return err
	}
	var present bool
	cs := q.commands
	for len(cs) > 0 {
		nv := 0
		ne := 0
		nc := 0
		for _, c := range cs {
			if dtc, ok := c.(*drawTrianglesCommand); ok {
				if dtc.numIndices() > graphics.IndicesNum {
					panic(fmt.Sprintf("graphicscommand: dtc.NumIndices() must be <= graphics.IndicesNum but not at Flush: dtc.NumIndices(): %d, graphics.IndicesNum: %d", dtc.numIndices(), graphics.IndicesNum))
				}
				if nc > 0 && mustUseDifferentVertexBuffer(nv+dtc.numVertices(), ne+dtc.numIndices()) {
					break
				}
				nv += dtc.numVertices()
				ne += dtc.numIndices()
				if dtc.dst.screen {
					present = true
				}
			}
			nc++
		}
		if 0 < ne {
			if err := graphicsDriver.SetVertices(vs[:nv], es[:ne]); err != nil {
				return err
			}
			es = es[ne:]
			vs = vs[nv:]
		}
		indexOffset := 0
		for _, c := range cs[:nc] {
			if err := c.Exec(graphicsDriver, indexOffset); err != nil {
				return err
			}
			debug.Logf("  %s\n", c)
			// TODO: indexOffset should be reset if the command type is different
			// from the previous one. This fix is needed when another drawing command is
			// introduced than drawTrianglesCommand.
			if dtc, ok := c.(*drawTrianglesCommand); ok {
				indexOffset += dtc.numIndices()
			}
		}
		cs = cs[nc:]
	}
	if err := graphicsDriver.End(present); err != nil {
		return err
	}

	// Release the commands explicitly (#1803).
	// Apparently, the part of a slice between len and cap-1 still holds references.
	// Then, resetting the length by [:0] doesn't release the references.
	for i, c := range q.commands {
		if c, ok := c.(*drawTrianglesCommand); ok {
			q.drawTrianglesCommandPool.put(c)
		}
		q.commands[i] = nil
	}
	q.commands = q.commands[:0]
	q.nvertices = 0
	q.nindices = 0
	q.tmpNumVertexFloats = 0
	q.tmpNumIndices = 0
	return nil
}

// FlushCommands flushes the command queue.
func FlushCommands(graphicsDriver graphicsdriver.Graphics) error {
	return theCommandQueue.Flush(graphicsDriver)
}

// drawTrianglesCommand represents a drawing command to draw an image on another image.
type drawTrianglesCommand struct {
	dst       *Image
	srcs      [graphics.ShaderImageNum]*Image
	offsets   [graphics.ShaderImageNum - 1][2]float32
	vertices  []float32
	nindices  int
	color     affine.ColorM
	mode      graphicsdriver.CompositeMode
	filter    graphicsdriver.Filter
	address   graphicsdriver.Address
	dstRegion graphicsdriver.Region
	srcRegion graphicsdriver.Region
	shader    *Shader
	uniforms  [][]float32
	evenOdd   bool
}

func (c *drawTrianglesCommand) String() string {
	mode := ""
	switch c.mode {
	case graphicsdriver.CompositeModeSourceOver:
		mode = "source-over"
	case graphicsdriver.CompositeModeClear:
		mode = "clear"
	case graphicsdriver.CompositeModeCopy:
		mode = "copy"
	case graphicsdriver.CompositeModeDestination:
		mode = "destination"
	case graphicsdriver.CompositeModeDestinationOver:
		mode = "destination-over"
	case graphicsdriver.CompositeModeSourceIn:
		mode = "source-in"
	case graphicsdriver.CompositeModeDestinationIn:
		mode = "destination-in"
	case graphicsdriver.CompositeModeSourceOut:
		mode = "source-out"
	case graphicsdriver.CompositeModeDestinationOut:
		mode = "destination-out"
	case graphicsdriver.CompositeModeSourceAtop:
		mode = "source-atop"
	case graphicsdriver.CompositeModeDestinationAtop:
		mode = "destination-atop"
	case graphicsdriver.CompositeModeXor:
		mode = "xor"
	case graphicsdriver.CompositeModeLighter:
		mode = "lighter"
	case graphicsdriver.CompositeModeMultiply:
		mode = "multiply"
	default:
		panic(fmt.Sprintf("graphicscommand: invalid composite mode: %d", c.mode))
	}

	dst := fmt.Sprintf("%d", c.dst.id)
	if c.dst.screen {
		dst += " (screen)"
	}

	if c.shader != nil {
		return fmt.Sprintf("draw-triangles: dst: %s, shader, num of indices: %d, mode %s", dst, c.nindices, mode)
	}

	filter := ""
	switch c.filter {
	case graphicsdriver.FilterNearest:
		filter = "nearest"
	case graphicsdriver.FilterLinear:
		filter = "linear"
	case graphicsdriver.FilterScreen:
		filter = "screen"
	default:
		panic(fmt.Sprintf("graphicscommand: invalid filter: %d", c.filter))
	}

	address := ""
	switch c.address {
	case graphicsdriver.AddressClampToZero:
		address = "clamp_to_zero"
	case graphicsdriver.AddressRepeat:
		address = "repeat"
	case graphicsdriver.AddressUnsafe:
		address = "unsafe"
	default:
		panic(fmt.Sprintf("graphicscommand: invalid address: %d", c.address))
	}

	var srcstrs [graphics.ShaderImageNum]string
	for i, src := range c.srcs {
		if src == nil {
			srcstrs[i] = "(nil)"
			continue
		}
		srcstrs[i] = fmt.Sprintf("%d", src.id)
		if src.screen {
			srcstrs[i] += " (screen)"
		}
	}

	r := fmt.Sprintf("(x:%d, y:%d, width:%d, height:%d)",
		int(c.dstRegion.X), int(c.dstRegion.Y), int(c.dstRegion.Width), int(c.dstRegion.Height))
	return fmt.Sprintf("draw-triangles: dst: %s <- src: [%s], dst region: %s, num of indices: %d, colorm: %v, mode: %s, filter: %s, address: %s, even-odd: %t", dst, strings.Join(srcstrs[:], ", "), r, c.nindices, c.color, mode, filter, address, c.evenOdd)
}

// Exec executes the drawTrianglesCommand.
func (c *drawTrianglesCommand) Exec(graphicsDriver graphicsdriver.Graphics, indexOffset int) error {
	// TODO: Is it ok not to bind any framebuffer here?
	if c.nindices == 0 {
		return nil
	}

	var shaderID graphicsdriver.ShaderID = graphicsdriver.InvalidShaderID
	var imgs [graphics.ShaderImageNum]graphicsdriver.ImageID
	if c.shader != nil {
		shaderID = c.shader.shader.ID()
		for i, src := range c.srcs {
			if src == nil {
				imgs[i] = graphicsdriver.InvalidImageID
				continue
			}
			imgs[i] = src.image.ID()
		}
	} else {
		imgs[0] = c.srcs[0].image.ID()
	}

	return graphicsDriver.DrawTriangles(c.dst.image.ID(), imgs, c.offsets, shaderID, c.nindices, indexOffset, c.mode, c.color, c.filter, c.address, c.dstRegion, c.srcRegion, c.uniforms, c.evenOdd)
}

func (c *drawTrianglesCommand) numVertices() int {
	return len(c.vertices)
}

func (c *drawTrianglesCommand) numIndices() int {
	return c.nindices
}

func (c *drawTrianglesCommand) setVertices(vertices []float32) {
	c.vertices = vertices
}

func (c *drawTrianglesCommand) addNumIndices(n int) {
	c.nindices += n
}

// CanMergeWithDrawTrianglesCommand returns a boolean value indicating whether the other drawTrianglesCommand can be merged
// with the drawTrianglesCommand c.
func (c *drawTrianglesCommand) CanMergeWithDrawTrianglesCommand(dst *Image, srcs [graphics.ShaderImageNum]*Image, vertices []float32, color affine.ColorM, mode graphicsdriver.CompositeMode, filter graphicsdriver.Filter, address graphicsdriver.Address, dstRegion, srcRegion graphicsdriver.Region, shader *Shader, uniforms [][]float32, evenOdd bool) bool {
	if c.shader != shader {
		return false
	}
	if c.shader != nil {
		if len(c.uniforms) != len(uniforms) {
			return false
		}
		for i := range c.uniforms {
			if len(c.uniforms[i]) != len(uniforms[i]) {
				return false
			}
			for j := range c.uniforms[i] {
				if c.uniforms[i][j] != uniforms[i][j] {
					return false
				}
			}
		}
	}
	if c.dst != dst {
		return false
	}
	if c.srcs != srcs {
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
	if c.dstRegion != dstRegion {
		return false
	}
	if c.srcRegion != srcRegion {
		return false
	}
	if c.evenOdd || evenOdd {
		if c.evenOdd && evenOdd {
			return !mightOverlapDstRegions(c.vertices, vertices)
		}
		return false
	}
	return true
}

var (
	posInf32 = float32(math.Inf(1))
	negInf32 = float32(math.Inf(-1))
)

func dstRegionFromVertices(vertices []float32) (minX, minY, maxX, maxY float32) {
	minX = posInf32
	minY = posInf32
	maxX = negInf32
	maxY = negInf32

	for i := 0; i < len(vertices)/graphics.VertexFloatNum; i++ {
		x := vertices[graphics.VertexFloatNum*i]
		y := vertices[graphics.VertexFloatNum*i+1]
		if x < minX {
			minX = x
		}
		if y < minY {
			minY = y
		}
		if maxX < x {
			maxX = x
		}
		if maxY < y {
			maxY = y
		}
	}
	return
}

func mightOverlapDstRegions(vertices1, vertices2 []float32) bool {
	minX1, minY1, maxX1, maxY1 := dstRegionFromVertices(vertices1)
	minX2, minY2, maxX2, maxY2 := dstRegionFromVertices(vertices2)
	const mergin = 1
	return minX1 < maxX2+mergin && minX2 < maxX1+mergin && minY1 < maxY2+mergin && minY2 < maxY1+mergin
}

// replacePixelsCommand represents a command to replace pixels of an image.
type replacePixelsCommand struct {
	dst  *Image
	args []*graphicsdriver.ReplacePixelsArgs
}

func (c *replacePixelsCommand) String() string {
	return fmt.Sprintf("replace-pixels: dst: %d, len(args): %d", c.dst.id, len(c.args))
}

// Exec executes the replacePixelsCommand.
func (c *replacePixelsCommand) Exec(graphicsDriver graphicsdriver.Graphics, indexOffset int) error {
	var lastArgIdx int
	for i, a := range c.args {
		if a.Mask == nil {
			continue
		}
		if len(c.args[lastArgIdx:i]) > 0 {
			c.dst.image.ReplacePixels(c.args[lastArgIdx:i])
			lastArgIdx = i
		}

		orig := make([]byte, 4*c.dst.width*c.dst.height)
		if err := c.dst.image.ReadPixels(orig); err != nil {
			return err
		}
		for j := 0; j < a.Height; j++ {
			for i := 0; i < a.Width; i++ {
				idx := j*a.Width + i
				if a.Mask[idx/8]>>(idx%8)&1 == 0 {
					srcIdx := (a.Y+j)*c.dst.width + a.X + i
					copy(a.Pixels[4*idx:4*(idx+1)], orig[4*srcIdx:4*(srcIdx+1)])
				}
			}
		}

		a.Mask = nil
	}

	if len(c.args[lastArgIdx:]) > 0 {
		if err := c.dst.image.ReplacePixels(c.args[lastArgIdx:]); err != nil {
			return err
		}
	}
	return nil
}

type pixelsCommand struct {
	result []byte
	img    *Image
}

// Exec executes a pixelsCommand.
func (c *pixelsCommand) Exec(graphicsDriver graphicsdriver.Graphics, indexOffset int) error {
	if err := c.img.image.ReadPixels(c.result); err != nil {
		return err
	}
	return nil
}

func (c *pixelsCommand) String() string {
	return fmt.Sprintf("pixels: image: %d", c.img.id)
}

// disposeImageCommand represents a command to dispose an image.
type disposeImageCommand struct {
	target *Image
}

func (c *disposeImageCommand) String() string {
	return fmt.Sprintf("dispose-image: target: %d", c.target.id)
}

// Exec executes the disposeImageCommand.
func (c *disposeImageCommand) Exec(graphicsDriver graphicsdriver.Graphics, indexOffset int) error {
	c.target.image.Dispose()
	return nil
}

// disposeShaderCommand represents a command to dispose a shader.
type disposeShaderCommand struct {
	target *Shader
}

func (c *disposeShaderCommand) String() string {
	return fmt.Sprintf("dispose-shader: target")
}

// Exec executes the disposeShaderCommand.
func (c *disposeShaderCommand) Exec(graphicsDriver graphicsdriver.Graphics, indexOffset int) error {
	c.target.shader.Dispose()
	return nil
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
func (c *newImageCommand) Exec(graphicsDriver graphicsdriver.Graphics, indexOffset int) error {
	i, err := graphicsDriver.NewImage(c.width, c.height)
	if err != nil {
		return err
	}
	c.result.image = i
	return nil
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
func (c *newScreenFramebufferImageCommand) Exec(graphicsDriver graphicsdriver.Graphics, indexOffset int) error {
	var err error
	c.result.image, err = graphicsDriver.NewScreenFramebufferImage(c.width, c.height)
	return err
}

// newShaderCommand is a command to create a shader.
type newShaderCommand struct {
	result *Shader
	ir     *shaderir.Program
}

func (c *newShaderCommand) String() string {
	return fmt.Sprintf("new-shader")
}

// Exec executes a newShaderCommand.
func (c *newShaderCommand) Exec(graphicsDriver graphicsdriver.Graphics, indexOffset int) error {
	s, err := graphicsDriver.NewShader(c.ir)
	if err != nil {
		return err
	}
	c.result.shader = s
	return nil
}

// InitializeGraphicsDriverState initialize the current graphics driver state.
func InitializeGraphicsDriverState(graphicsDriver graphicsdriver.Graphics) (err error) {
	runOnRenderingThread(func() {
		err = graphicsDriver.Initialize()
	})
	return
}

// ResetGraphicsDriverState resets the current graphics driver state.
// If the graphics driver doesn't have an API to reset, ResetGraphicsDriverState does nothing.
func ResetGraphicsDriverState(graphicsDriver graphicsdriver.Graphics) (err error) {
	if r, ok := graphicsDriver.(interface{ Reset() error }); ok {
		runOnRenderingThread(func() {
			err = r.Reset()
		})
	}
	return nil
}

// MaxImageSize returns the maximum size of an image.
func MaxImageSize(graphicsDriver graphicsdriver.Graphics) int {
	var size int
	runOnRenderingThread(func() {
		size = graphicsDriver.MaxImageSize()
	})
	return size
}
