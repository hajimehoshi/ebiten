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
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/debug"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

// command represents a drawing command.
//
// A command for drawing that is created when Image functions are called like DrawTriangles,
// or Fill.
// A command is not immediately executed after created. Instead, it is queued after created,
// and executed only when necessary.
type command interface {
	fmt.Stringer

	Exec(graphicsDriver graphicsdriver.Graphics, indexOffset int) error
	NeedsSync() bool
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
	indices  []uint16

	tmpNumVertexFloats int

	drawTrianglesCommandPool drawTrianglesCommandPool

	uint32sBuffer uint32sBuffer

	temporaryBytes temporaryBytes

	err atomic.Value
}

// theCommandQueues is the set of command queues for the current process.
var (
	theCommandQueues = [...]*commandQueue{
		{},
		{},
	}
	commandQueueIndex int
)

func currentCommandQueue() *commandQueue {
	return theCommandQueues[commandQueueIndex]
}

func switchCommandQueue() {
	commandQueueIndex++
	commandQueueIndex = commandQueueIndex % len(theCommandQueues)
}

func (q *commandQueue) appendIndices(indices []uint16, offset uint16) {
	n := len(q.indices)
	q.indices = append(q.indices, indices...)
	for i := n; i < len(q.indices); i++ {
		q.indices[i] += offset
	}
}

// mustUseDifferentVertexBuffer reports whether a different vertex buffer must be used.
func mustUseDifferentVertexBuffer(nextNumVertexFloats int) bool {
	return nextNumVertexFloats > graphics.MaxVertexFloatsCount
}

// EnqueueDrawTrianglesCommand enqueues a drawing-image command.
func (q *commandQueue) EnqueueDrawTrianglesCommand(dst *Image, srcs [graphics.ShaderImageCount]*Image, vertices []float32, indices []uint16, blend graphicsdriver.Blend, dstRegion graphicsdriver.Region, srcRegions [graphics.ShaderImageCount]graphicsdriver.Region, shader *Shader, uniforms []uint32, evenOdd bool) {
	if len(vertices) > graphics.MaxVertexFloatsCount {
		panic(fmt.Sprintf("graphicscommand: len(vertices) must equal to or less than %d but was %d", graphics.MaxVertexFloatsCount, len(vertices)))
	}

	split := false
	if mustUseDifferentVertexBuffer(q.tmpNumVertexFloats + len(vertices)) {
		q.tmpNumVertexFloats = 0
		split = true
	}

	// Assume that all the image sizes are same.
	// Assume that the images are packed from the front in the slice srcs.
	q.vertices = append(q.vertices, vertices...)
	q.appendIndices(indices, uint16(q.tmpNumVertexFloats/graphics.VertexFloatCount))
	q.tmpNumVertexFloats += len(vertices)

	// prependPreservedUniforms not only prepends values to the given slice but also creates a new slice.
	// Allocating a new slice is necessary to make EnqueueDrawTrianglesCommand safe so far.
	// TODO: This might cause a performance issue (#2601).
	uniforms = q.prependPreservedUniforms(uniforms, shader, dst, srcs, dstRegion, srcRegions)

	// Remove unused uniform variables so that more commands can be merged.
	shader.ir.FilterUniformVariables(uniforms)

	// TODO: If dst is the screen, reorder the command to be the last.
	if !split && 0 < len(q.commands) {
		if last, ok := q.commands[len(q.commands)-1].(*drawTrianglesCommand); ok {
			if last.CanMergeWithDrawTrianglesCommand(dst, srcs, vertices, blend, shader, uniforms, evenOdd) {
				last.setVertices(q.lastVertices(len(vertices) + last.numVertices()))
				if last.dstRegions[len(last.dstRegions)-1].Region == dstRegion {
					last.dstRegions[len(last.dstRegions)-1].IndexCount += len(indices)
				} else {
					last.dstRegions = append(last.dstRegions, graphicsdriver.DstRegion{
						Region:     dstRegion,
						IndexCount: len(indices),
					})
				}
				return
			}
		}
	}

	c := q.drawTrianglesCommandPool.get()
	c.dst = dst
	c.srcs = srcs
	c.vertices = q.lastVertices(len(vertices))
	c.blend = blend
	c.dstRegions = []graphicsdriver.DstRegion{
		{
			Region:     dstRegion,
			IndexCount: len(indices),
		},
	}
	c.shader = shader
	c.uniforms = uniforms
	c.evenOdd = evenOdd
	q.commands = append(q.commands, c)
}

func (q *commandQueue) lastVertices(n int) []float32 {
	return q.vertices[len(q.vertices)-n : len(q.vertices)]
}

// Enqueue enqueues a drawing command other than a draw-triangles command.
//
// For a draw-triangles command, use EnqueueDrawTrianglesCommand.
func (q *commandQueue) Enqueue(command command) {
	// TODO: If dst is the screen, reorder the command to be the last.
	q.commands = append(q.commands, command)
}

// Flush flushes the command queue.
func (q *commandQueue) Flush(graphicsDriver graphicsdriver.Graphics, endFrame bool, swapBuffersForGL func()) error {
	if err := q.err.Load(); err != nil {
		return err.(error)
	}

	var sync bool
	for _, c := range q.commands {
		if c.NeedsSync() {
			sync = true
			break
		}
	}

	logger := debug.SwitchLogger()

	var flushErr error
	runOnRenderThread(func() {
		defer logger.Flush()

		if err := q.flush(graphicsDriver, endFrame, logger); err != nil {
			if sync {
				flushErr = err
				return
			}
			q.err.Store(err)
			return
		}

		if endFrame && swapBuffersForGL != nil {
			swapBuffersForGL()
		}
	}, sync)

	if sync && flushErr != nil {
		return flushErr
	}

	return nil
}

// flush must be called the main thread.
func (q *commandQueue) flush(graphicsDriver graphicsdriver.Graphics, endFrame bool, logger debug.Logger) (err error) {
	// If endFrame is true, Begin/End should be called to ensure the framebuffer is swapped.
	if len(q.commands) == 0 && !endFrame {
		return nil
	}

	es := q.indices
	vs := q.vertices
	logger.Logf("Graphics commands:\n")

	if err := graphicsDriver.Begin(); err != nil {
		return err
	}

	defer func() {
		// Call End even if an error causes, or the graphics driver's state might be stale (#2388).
		if err1 := graphicsDriver.End(endFrame); err1 != nil && err == nil {
			err = err1
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
		q.vertices = q.vertices[:0]
		q.indices = q.indices[:0]
		q.tmpNumVertexFloats = 0

		if endFrame {
			q.uint32sBuffer.reset()
			q.temporaryBytes.reset()
		}
	}()

	cs := q.commands
	for len(cs) > 0 {
		nv := 0
		ne := 0
		nc := 0
		for _, c := range cs {
			if dtc, ok := c.(*drawTrianglesCommand); ok {
				if nc > 0 && mustUseDifferentVertexBuffer(nv+dtc.numVertices()) {
					break
				}
				nv += dtc.numVertices()
				ne += dtc.numIndices()
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
			logger.Logf("  %s\n", c)
			// TODO: indexOffset should be reset if the command type is different
			// from the previous one. This fix is needed when another drawing command is
			// introduced than drawTrianglesCommand.
			if dtc, ok := c.(*drawTrianglesCommand); ok {
				indexOffset += dtc.numIndices()
			}
		}
		cs = cs[nc:]
	}

	return nil
}

// FlushCommands flushes the command queue and present the screen if needed.
// If endFrame is true, the current screen might be used to present.
func FlushCommands(graphicsDriver graphicsdriver.Graphics, endFrame bool, swapBuffersForGL func()) error {
	flushImageBuffers()
	if err := currentCommandQueue().Flush(graphicsDriver, endFrame, swapBuffersForGL); err != nil {
		return err
	}
	switchCommandQueue()
	return nil
}

// drawTrianglesCommand represents a drawing command to draw an image on another image.
type drawTrianglesCommand struct {
	dst        *Image
	srcs       [graphics.ShaderImageCount]*Image
	vertices   []float32
	blend      graphicsdriver.Blend
	dstRegions []graphicsdriver.DstRegion
	shader     *Shader
	uniforms   []uint32
	evenOdd    bool
}

func (c *drawTrianglesCommand) String() string {
	// TODO: Improve readability
	blend := fmt.Sprintf("{src-color: %d, src-alpha: %d, dst-color: %d, dst-alpha: %d, op-color: %d, op-alpha: %d}",
		c.blend.BlendFactorSourceRGB,
		c.blend.BlendFactorSourceAlpha,
		c.blend.BlendFactorDestinationRGB,
		c.blend.BlendFactorDestinationAlpha,
		c.blend.BlendOperationRGB,
		c.blend.BlendOperationAlpha)

	dst := fmt.Sprintf("%d", c.dst.id)
	if c.dst.screen {
		dst += " (screen)"
	}

	var srcstrs [graphics.ShaderImageCount]string
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

	return fmt.Sprintf("draw-triangles: dst: %s <- src: [%s], num of dst regions: %d, num of indices: %d, blend: %s, even-odd: %t", dst, strings.Join(srcstrs[:], ", "), len(c.dstRegions), c.numIndices(), blend, c.evenOdd)
}

// Exec executes the drawTrianglesCommand.
func (c *drawTrianglesCommand) Exec(graphicsDriver graphicsdriver.Graphics, indexOffset int) error {
	// TODO: Is it ok not to bind any framebuffer here?
	if len(c.dstRegions) == 0 {
		return nil
	}

	var imgs [graphics.ShaderImageCount]graphicsdriver.ImageID
	for i, src := range c.srcs {
		if src == nil {
			imgs[i] = graphicsdriver.InvalidImageID
			continue
		}
		imgs[i] = src.image.ID()
	}

	return graphicsDriver.DrawTriangles(c.dst.image.ID(), imgs, c.shader.shader.ID(), c.dstRegions, indexOffset, c.blend, c.uniforms, c.evenOdd)
}

func (c *drawTrianglesCommand) NeedsSync() bool {
	return false
}

func (c *drawTrianglesCommand) numVertices() int {
	return len(c.vertices)
}

func (c *drawTrianglesCommand) numIndices() int {
	var nindices int
	for _, dstRegion := range c.dstRegions {
		nindices += dstRegion.IndexCount
	}
	return nindices
}

func (c *drawTrianglesCommand) setVertices(vertices []float32) {
	c.vertices = vertices
}

// CanMergeWithDrawTrianglesCommand returns a boolean value indicating whether the other drawTrianglesCommand can be merged
// with the drawTrianglesCommand c.
func (c *drawTrianglesCommand) CanMergeWithDrawTrianglesCommand(dst *Image, srcs [graphics.ShaderImageCount]*Image, vertices []float32, blend graphicsdriver.Blend, shader *Shader, uniforms []uint32, evenOdd bool) bool {
	if c.shader != shader {
		return false
	}
	if len(c.uniforms) != len(uniforms) {
		return false
	}
	for i := range c.uniforms {
		if c.uniforms[i] != uniforms[i] {
			return false
		}
	}
	if c.dst != dst {
		return false
	}
	if c.srcs != srcs {
		return false
	}
	if c.blend != blend {
		return false
	}
	if c.evenOdd != evenOdd {
		return false
	}
	if c.evenOdd && mightOverlapDstRegions(c.vertices, vertices) {
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

	for i := 0; i < len(vertices)/graphics.VertexFloatCount; i++ {
		x := vertices[graphics.VertexFloatCount*i]
		y := vertices[graphics.VertexFloatCount*i+1]
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

// writePixelsCommand represents a command to replace pixels of an image.
type writePixelsCommand struct {
	dst  *Image
	args []graphicsdriver.PixelsArgs
}

func (c *writePixelsCommand) String() string {
	return fmt.Sprintf("write-pixels: dst: %d, len(args): %d", c.dst.id, len(c.args))
}

// Exec executes the writePixelsCommand.
func (c *writePixelsCommand) Exec(graphicsDriver graphicsdriver.Graphics, indexOffset int) error {
	if len(c.args) == 0 {
		return nil
	}
	if err := c.dst.image.WritePixels(c.args); err != nil {
		return err
	}
	return nil
}

func (c *writePixelsCommand) NeedsSync() bool {
	return false
}

type readPixelsCommand struct {
	img  *Image
	args []graphicsdriver.PixelsArgs
}

// Exec executes a readPixelsCommand.
func (c *readPixelsCommand) Exec(graphicsDriver graphicsdriver.Graphics, indexOffset int) error {
	if err := c.img.image.ReadPixels(c.args); err != nil {
		return err
	}
	return nil
}

func (c *readPixelsCommand) NeedsSync() bool {
	return true
}

func (c *readPixelsCommand) String() string {
	return fmt.Sprintf("read-pixels: image: %d", c.img.id)
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

func (c *disposeImageCommand) NeedsSync() bool {
	return false
}

// disposeShaderCommand represents a command to dispose a shader.
type disposeShaderCommand struct {
	target *Shader
}

func (c *disposeShaderCommand) String() string {
	return "dispose-shader: target"
}

// Exec executes the disposeShaderCommand.
func (c *disposeShaderCommand) Exec(graphicsDriver graphicsdriver.Graphics, indexOffset int) error {
	c.target.shader.Dispose()
	return nil
}

func (c *disposeShaderCommand) NeedsSync() bool {
	return false
}

// newImageCommand represents a command to create an empty image with given width and height.
type newImageCommand struct {
	result *Image
	width  int
	height int
	screen bool
}

func (c *newImageCommand) String() string {
	return fmt.Sprintf("new-image: result: %d, width: %d, height: %d, screen: %t", c.result.id, c.width, c.height, c.screen)
}

// Exec executes a newImageCommand.
func (c *newImageCommand) Exec(graphicsDriver graphicsdriver.Graphics, indexOffset int) error {
	var err error
	if c.screen {
		c.result.image, err = graphicsDriver.NewScreenFramebufferImage(c.width, c.height)
	} else {
		c.result.image, err = graphicsDriver.NewImage(c.width, c.height)
	}
	return err
}

func (c *newImageCommand) NeedsSync() bool {
	return true
}

// newShaderCommand is a command to create a shader.
type newShaderCommand struct {
	result *Shader
	ir     *shaderir.Program
}

func (c *newShaderCommand) String() string {
	return "new-shader"
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

func (c *newShaderCommand) NeedsSync() bool {
	return true
}

type isInvalidatedCommand struct {
	result bool
	image  *Image
}

func (c *isInvalidatedCommand) String() string {
	return fmt.Sprintf("is-invalidated: image: %d", c.image.id)
}

func (c *isInvalidatedCommand) Exec(graphicsDriver graphicsdriver.Graphics, indexOffset int) error {
	c.result = c.image.image.IsInvalidated()
	return nil
}

func (c *isInvalidatedCommand) NeedsSync() bool {
	return true
}

// InitializeGraphicsDriverState initialize the current graphics driver state.
func InitializeGraphicsDriverState(graphicsDriver graphicsdriver.Graphics) (err error) {
	runOnRenderThread(func() {
		err = graphicsDriver.Initialize()
	}, true)
	return
}

// ResetGraphicsDriverState resets the current graphics driver state.
// If the graphics driver doesn't have an API to reset, ResetGraphicsDriverState does nothing.
func ResetGraphicsDriverState(graphicsDriver graphicsdriver.Graphics) (err error) {
	if r, ok := graphicsDriver.(graphicsdriver.Resetter); ok {
		runOnRenderThread(func() {
			err = r.Reset()
		}, true)
	}
	return nil
}

// MaxImageSize returns the maximum size of an image.
func MaxImageSize(graphicsDriver graphicsdriver.Graphics) int {
	var size int
	runOnRenderThread(func() {
		size = graphicsDriver.MaxImageSize()
	}, true)
	return size
}

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

func roundUpPower2(x int) int {
	p2 := 1
	for p2 < x {
		p2 *= 2
	}
	return p2
}

func (q *commandQueue) prependPreservedUniforms(uniforms []uint32, shader *Shader, dst *Image, srcs [graphics.ShaderImageCount]*Image, dstRegion graphicsdriver.Region, srcRegions [graphics.ShaderImageCount]graphicsdriver.Region) []uint32 {
	origUniforms := uniforms
	uniforms = q.uint32sBuffer.alloc(len(origUniforms) + graphics.PreservedUniformUint32Count)
	copy(uniforms[graphics.PreservedUniformUint32Count:], origUniforms)

	// Set the destination texture size.
	dw, dh := dst.InternalSize()
	uniforms[0] = math.Float32bits(float32(dw))
	uniforms[1] = math.Float32bits(float32(dh))

	// Set the source texture sizes.
	if srcs[0] != nil {
		w, h := srcs[0].InternalSize()
		uniforms[2] = math.Float32bits(float32(w))
		uniforms[3] = math.Float32bits(float32(h))
	} else {
		uniforms[2] = 0
		uniforms[3] = 0
	}
	if srcs[1] != nil {
		w, h := srcs[1].InternalSize()
		uniforms[4] = math.Float32bits(float32(w))
		uniforms[5] = math.Float32bits(float32(h))
	} else {
		uniforms[4] = 0
		uniforms[5] = 0
	}
	if srcs[2] != nil {
		w, h := srcs[2].InternalSize()
		uniforms[6] = math.Float32bits(float32(w))
		uniforms[7] = math.Float32bits(float32(h))
	} else {
		uniforms[6] = 0
		uniforms[7] = 0
	}
	if srcs[3] != nil {
		w, h := srcs[3].InternalSize()
		uniforms[8] = math.Float32bits(float32(w))
		uniforms[9] = math.Float32bits(float32(h))
	} else {
		uniforms[8] = 0
		uniforms[9] = 0
	}

	if shader.unit() == shaderir.Texels {
		dstRegion.X /= float32(dw)
		dstRegion.Y /= float32(dh)
		dstRegion.Width /= float32(dw)
		dstRegion.Height /= float32(dh)
	}

	// Set the destination region origin.
	uniforms[10] = math.Float32bits(dstRegion.X)
	uniforms[11] = math.Float32bits(dstRegion.Y)

	// Set the destination region size.
	uniforms[12] = math.Float32bits(dstRegion.Width)
	uniforms[13] = math.Float32bits(dstRegion.Height)

	if shader.unit() == shaderir.Texels {
		for i, src := range srcs {
			if src == nil {
				continue
			}
			w, h := src.InternalSize()
			srcRegions[i].X /= float32(w)
			srcRegions[i].Y /= float32(h)
			srcRegions[i].Width /= float32(w)
			srcRegions[i].Height /= float32(h)
		}
	}

	// Set the source region origins.
	uniforms[14] = math.Float32bits(srcRegions[0].X)
	uniforms[15] = math.Float32bits(srcRegions[0].Y)
	uniforms[16] = math.Float32bits(srcRegions[1].X)
	uniforms[17] = math.Float32bits(srcRegions[1].Y)
	uniforms[18] = math.Float32bits(srcRegions[2].X)
	uniforms[19] = math.Float32bits(srcRegions[2].Y)
	uniforms[20] = math.Float32bits(srcRegions[3].X)
	uniforms[21] = math.Float32bits(srcRegions[3].Y)

	// Set the source region sizes.
	uniforms[22] = math.Float32bits(srcRegions[0].Width)
	uniforms[23] = math.Float32bits(srcRegions[0].Height)
	uniforms[24] = math.Float32bits(srcRegions[1].Width)
	uniforms[25] = math.Float32bits(srcRegions[1].Height)
	uniforms[26] = math.Float32bits(srcRegions[2].Width)
	uniforms[27] = math.Float32bits(srcRegions[2].Height)
	uniforms[28] = math.Float32bits(srcRegions[3].Width)
	uniforms[29] = math.Float32bits(srcRegions[3].Height)

	// Set the projection matrix.
	uniforms[30] = math.Float32bits(2 / float32(dw))
	uniforms[31] = 0
	uniforms[32] = 0
	uniforms[33] = 0
	uniforms[34] = 0
	uniforms[35] = math.Float32bits(2 / float32(dh))
	uniforms[36] = 0
	uniforms[37] = 0
	uniforms[38] = 0
	uniforms[39] = 0
	uniforms[40] = math.Float32bits(1)
	uniforms[41] = 0
	uniforms[42] = math.Float32bits(-1)
	uniforms[43] = math.Float32bits(-1)
	uniforms[44] = 0
	uniforms[45] = math.Float32bits(1)

	return uniforms
}

// uint32sBuffer is a reusable buffer to allocate []uint32.
type uint32sBuffer struct {
	buf []uint32
}

func (b *uint32sBuffer) alloc(n int) []uint32 {
	buf := b.buf
	if len(buf)+n > cap(buf) {
		buf = make([]uint32, 0, max(roundUpPower2(len(buf)+n), 16))
	}
	s := buf[len(buf) : len(buf)+n]
	b.buf = buf[:len(buf)+n]
	return s
}

func (b *uint32sBuffer) reset() {
	b.buf = b.buf[:0]
}
