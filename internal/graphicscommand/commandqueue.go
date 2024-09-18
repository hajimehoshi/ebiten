// Copyright 2023 The Ebitengine Authors
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
	"image"
	"math"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/debug"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

const (
	is32bit = 1 >> (^uint(0) >> 63)
	is64bit = 1 - is32bit

	// MaxVertexCount is the maximum number of vertices for one draw call.
	//
	// On 64bit architectures, this value is 2^32-1, as the index type is uint32.
	// This value cannot be exactly 2^32 especially with WebGL 2, as 2^32th vertex is not rendered correctly.
	// See https://registry.khronos.org/webgl/specs/latest/2.0/#5.18 .
	//
	// On 32bit architectures, this value is an adjusted number so that maxVertexFloatCount doesn't overflow int.
	MaxVertexCount = is64bit*math.MaxUint32 + is32bit*(math.MaxInt32/graphics.VertexFloatCount)

	maxVertexFloatCount = MaxVertexCount * graphics.VertexFloatCount
)

var vsyncEnabled atomic.Bool

func init() {
	vsyncEnabled.Store(true)
}

func SetVsyncEnabled(enabled bool, graphicsDriver graphicsdriver.Graphics) {
	vsyncEnabled.Store(enabled)

	runOnRenderThread(func() {
		graphicsDriver.SetVsyncEnabled(enabled)
	}, true)
}

// FlushCommands flushes the command queue and present the screen if needed.
// If endFrame is true, the current screen might be used to present.
func FlushCommands(graphicsDriver graphicsdriver.Graphics, endFrame bool) error {
	if err := theCommandQueueManager.flush(graphicsDriver, endFrame); err != nil {
		return err
	}
	return nil
}

// commandQueue is a command queue for drawing commands.
type commandQueue struct {
	// commands is a queue of drawing commands.
	commands []command

	// vertices represents a vertices data in OpenGL's array buffer.
	vertices []float32
	indices  []uint32

	tmpNumVertexFloats int

	drawTrianglesCommandPool drawTrianglesCommandPool

	uint32sBuffer uint32sBuffer
	finalizers    []func()

	err atomic.Value
}

// addFinalizer adds a finalizer function to this queue.
// A finalizer is executed when the command queue is flushed at the end of the frame.
func (q *commandQueue) addFinalizer(f func()) {
	q.finalizers = append(q.finalizers, f)
}

func (q *commandQueue) appendIndices(indices []uint32, offset uint32) {
	n := len(q.indices)
	q.indices = append(q.indices, indices...)
	for i := n; i < len(q.indices); i++ {
		q.indices[i] += offset
	}
}

// mustUseDifferentVertexBuffer reports whether a different vertex buffer must be used.
func mustUseDifferentVertexBuffer(nextNumVertexFloats int) bool {
	return nextNumVertexFloats > maxVertexFloatCount
}

// EnqueueDrawTrianglesCommand enqueues a drawing-image command.
func (q *commandQueue) EnqueueDrawTrianglesCommand(dst *Image, srcs [graphics.ShaderSrcImageCount]*Image, vertices []float32, indices []uint32, blend graphicsdriver.Blend, dstRegion image.Rectangle, srcRegions [graphics.ShaderSrcImageCount]image.Rectangle, shader *Shader, uniforms []uint32, fillRule graphicsdriver.FillRule) {
	if len(vertices) > maxVertexFloatCount {
		panic(fmt.Sprintf("graphicscommand: len(vertices) must equal to or less than %d but was %d", maxVertexFloatCount, len(vertices)))
	}

	split := false
	if mustUseDifferentVertexBuffer(q.tmpNumVertexFloats + len(vertices)) {
		q.tmpNumVertexFloats = 0
		split = true
	}

	// Assume that all the image sizes are same.
	// Assume that the images are packed from the front in the slice srcs.
	q.vertices = append(q.vertices, vertices...)
	q.appendIndices(indices, uint32(q.tmpNumVertexFloats/graphics.VertexFloatCount))
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
			if last.CanMergeWithDrawTrianglesCommand(dst, srcs, vertices, blend, shader, uniforms, fillRule) {
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
	c.fillRule = fillRule
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
func (q *commandQueue) Flush(graphicsDriver graphicsdriver.Graphics, endFrame bool) error {
	if err := q.err.Load(); err != nil {
		return err.(error)
	}

	var sync bool
	// Disable asynchronous rendering when vsync is on, as this causes a rendering delay (#2822).
	if endFrame && vsyncEnabled.Load() {
		sync = true
	}
	if !sync {
		for _, c := range q.commands {
			if c.NeedsSync() {
				sync = true
				break
			}
		}
	}

	logger := debug.SwitchFrameLogger()

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

		theCommandQueueManager.putCommandQueue(q)
	}, sync)

	if sync && flushErr != nil {
		return flushErr
	}

	return nil
}

// flush must be called the render thread.
func (q *commandQueue) flush(graphicsDriver graphicsdriver.Graphics, endFrame bool, logger debug.FrameLogger) (err error) {
	// If endFrame is true, Begin/End should be called to ensure the framebuffer is swapped.
	if len(q.commands) == 0 && !endFrame {
		return nil
	}

	es := q.indices
	vs := q.vertices
	logger.FrameLogf("Graphics commands:\n")

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
			for i, f := range q.finalizers {
				f()
				q.finalizers[i] = nil
			}
			q.finalizers = q.finalizers[:0]
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
			if err := c.Exec(q, graphicsDriver, indexOffset); err != nil {
				return err
			}
			logger.FrameLogf("  %s\n", c)
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

type rectangleF32 struct {
	x      float32
	y      float32
	width  float32
	height float32
}

func imageRectangleToRectangleF32(r image.Rectangle) rectangleF32 {
	return rectangleF32{
		x:      float32(r.Min.X),
		y:      float32(r.Min.Y),
		width:  float32(r.Dx()),
		height: float32(r.Dy()),
	}
}

func (q *commandQueue) prependPreservedUniforms(uniforms []uint32, shader *Shader, dst *Image, srcs [graphics.ShaderSrcImageCount]*Image, dstRegion image.Rectangle, srcRegions [graphics.ShaderSrcImageCount]image.Rectangle) []uint32 {
	origUniforms := uniforms
	uniforms = q.uint32sBuffer.alloc(len(origUniforms) + graphics.PreservedUniformUint32Count)
	copy(uniforms[graphics.PreservedUniformUint32Count:], origUniforms)

	// Set the destination texture size.
	dw, dh := dst.InternalSize()
	uniforms[0] = math.Float32bits(float32(dw))
	uniforms[1] = math.Float32bits(float32(dh))
	uniformIndex := 2

	for i := 0; i < graphics.ShaderSrcImageCount; i++ {
		var floatW, floatH uint32
		if srcs[i] != nil {
			w, h := srcs[i].InternalSize()
			floatW = math.Float32bits(float32(w))
			floatH = math.Float32bits(float32(h))
		}

		uniforms[uniformIndex+i*2] = floatW
		uniforms[uniformIndex+1+i*2] = floatH
	}
	uniformIndex += graphics.ShaderSrcImageCount * 2

	dr := imageRectangleToRectangleF32(dstRegion)
	if shader.unit() == shaderir.Texels {
		dr.x /= float32(dw)
		dr.y /= float32(dh)
		dr.width /= float32(dw)
		dr.height /= float32(dh)
	}

	// Set the destination region origin.
	uniforms[uniformIndex] = math.Float32bits(dr.x)
	uniforms[uniformIndex+1] = math.Float32bits(dr.y)
	uniformIndex += 2

	// Set the destination region size.
	uniforms[uniformIndex] = math.Float32bits(dr.width)
	uniforms[uniformIndex+1] = math.Float32bits(dr.height)
	uniformIndex += 2

	var srs [graphics.ShaderSrcImageCount]rectangleF32
	for i, r := range srcRegions {
		srs[i] = imageRectangleToRectangleF32(r)
	}
	if shader.unit() == shaderir.Texels {
		for i, src := range srcs {
			if src == nil {
				continue
			}
			w, h := src.InternalSize()
			srs[i].x /= float32(w)
			srs[i].y /= float32(h)
			srs[i].width /= float32(w)
			srs[i].height /= float32(h)
		}
	}

	// Set the source region origins.
	for i := 0; i < graphics.ShaderSrcImageCount; i++ {
		uniforms[uniformIndex+i*2] = math.Float32bits(srs[i].x)
		uniforms[uniformIndex+1+i*2] = math.Float32bits(srs[i].y)
	}
	uniformIndex += graphics.ShaderSrcImageCount * 2

	// Set the source region sizes.
	for i := 0; i < graphics.ShaderSrcImageCount; i++ {
		uniforms[uniformIndex+i*2] = math.Float32bits(srs[i].width)
		uniforms[uniformIndex+1+i*2] = math.Float32bits(srs[i].height)
	}
	uniformIndex += graphics.ShaderSrcImageCount * 2

	// Set the projection matrix.
	uniforms[uniformIndex] = math.Float32bits(2 / float32(dw))
	uniforms[uniformIndex+1] = 0
	uniforms[uniformIndex+2] = 0
	uniforms[uniformIndex+3] = 0
	uniforms[uniformIndex+4] = 0
	uniforms[uniformIndex+5] = math.Float32bits(2 / float32(dh))
	uniforms[uniformIndex+6] = 0
	uniforms[uniformIndex+7] = 0
	uniforms[uniformIndex+8] = 0
	uniforms[uniformIndex+9] = 0
	uniforms[uniformIndex+10] = math.Float32bits(1)
	uniforms[uniformIndex+11] = 0
	uniforms[uniformIndex+12] = math.Float32bits(-1)
	uniforms[uniformIndex+13] = math.Float32bits(-1)
	uniforms[uniformIndex+14] = 0
	uniforms[uniformIndex+15] = math.Float32bits(1)

	return uniforms
}

type commandQueuePool struct {
	cache []*commandQueue
	m     sync.Mutex
}

func (c *commandQueuePool) get() (*commandQueue, error) {
	c.m.Lock()
	defer c.m.Unlock()

	if len(c.cache) == 0 {
		return &commandQueue{}, nil
	}

	for _, q := range c.cache {
		if err := q.err.Load(); err != nil {
			return nil, err.(error)
		}
	}

	q := c.cache[len(c.cache)-1]
	c.cache[len(c.cache)-1] = nil
	c.cache = c.cache[:len(c.cache)-1]
	return q, nil
}

func (c *commandQueuePool) put(queue *commandQueue) {
	c.m.Lock()
	defer c.m.Unlock()

	c.cache = append(c.cache, queue)
}

type commandQueueManager struct {
	pool    commandQueuePool
	current *commandQueue
}

var theCommandQueueManager commandQueueManager

func (c *commandQueueManager) enqueueCommand(command command) {
	if c.current == nil {
		c.current, _ = c.pool.get()
	}
	c.current.Enqueue(command)
}

// put can be called from any goroutines.
func (c *commandQueueManager) putCommandQueue(commandQueue *commandQueue) {
	c.pool.put(commandQueue)
}

func (c *commandQueueManager) enqueueDrawTrianglesCommand(dst *Image, srcs [graphics.ShaderSrcImageCount]*Image, vertices []float32, indices []uint32, blend graphicsdriver.Blend, dstRegion image.Rectangle, srcRegions [graphics.ShaderSrcImageCount]image.Rectangle, shader *Shader, uniforms []uint32, fillRule graphicsdriver.FillRule) {
	if c.current == nil {
		c.current, _ = c.pool.get()
	}
	c.current.EnqueueDrawTrianglesCommand(dst, srcs, vertices, indices, blend, dstRegion, srcRegions, shader, uniforms, fillRule)
}

func (c *commandQueueManager) flush(graphicsDriver graphicsdriver.Graphics, endFrame bool) error {
	// Switch the command queue.
	prev := c.current
	q, err := c.pool.get()
	if err != nil {
		return err
	}
	c.current = q

	if prev == nil {
		return nil
	}
	if err := prev.Flush(graphicsDriver, endFrame); err != nil {
		return err
	}
	return nil
}

// uint32sBuffer is a reusable buffer to allocate []uint32.
type uint32sBuffer struct {
	buf []uint32
}

func roundUpPower2(x int) int {
	p2 := 1
	for p2 < x {
		p2 *= 2
	}
	return p2
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
