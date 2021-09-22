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
	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

var theGraphicsDriver driver.Graphics

func SetGraphicsDriver(driver driver.Graphics) {
	theGraphicsDriver = driver
}

func NeedsRestoring() bool {
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
func (q *commandQueue) appendVertices(vertices []float32, src *Image) {
	if len(q.vertices) < q.nvertices+len(vertices) {
		n := q.nvertices + len(vertices) - len(q.vertices)
		q.vertices = append(q.vertices, make([]float32, n)...)
		q.srcSizes = append(q.srcSizes, make([]size, n/graphics.VertexFloatNum)...)
	}
	copy(q.vertices[q.nvertices:], vertices)

	n := len(vertices) / graphics.VertexFloatNum
	base := q.nvertices / graphics.VertexFloatNum

	width := float32(1)
	height := float32(1)
	// src is nil when a shader is used and there are no specified images.
	if src != nil {
		w, h := src.InternalSize()
		width = float32(w)
		height = float32(h)
	}
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
func (q *commandQueue) EnqueueDrawTrianglesCommand(dst *Image, srcs [graphics.ShaderImageNum]*Image, offsets [graphics.ShaderImageNum - 1][2]float32, vertices []float32, indices []uint16, color affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address, dstRegion, srcRegion driver.Region, shader *Shader, uniforms []interface{}, evenOdd bool) {
	if len(indices) > graphics.IndicesNum {
		panic(fmt.Sprintf("graphicscommand: len(indices) must be <= graphics.IndicesNum but not at EnqueueDrawTrianglesCommand: len(indices): %d, graphics.IndicesNum: %d", len(indices), graphics.IndicesNum))
	}

	split := false
	if q.tmpNumIndices+len(indices) > graphics.IndicesNum {
		q.tmpNumIndices = 0
		q.nextIndex = 0
		split = true
	}

	// Assume that all the image sizes are same.
	// Assume that the images are packed from the front in the slice srcs.
	q.appendVertices(vertices, srcs[0])
	q.appendIndices(indices, uint16(q.nextIndex))
	q.nextIndex += len(vertices) / graphics.VertexFloatNum
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
		// TODO: Pass offsets and uniforms when merging considers the shader.
		if last, ok := q.commands[len(q.commands)-1].(*drawTrianglesCommand); ok {
			if last.CanMergeWithDrawTrianglesCommand(dst, srcs, vertices, color, mode, filter, address, dstRegion, srcRegion, shader, evenOdd) {
				last.setVertices(q.lastVertices(len(vertices) + last.numVertices()))
				last.addNumIndices(len(indices))
				return
			}
		}
	}

	c := &drawTrianglesCommand{
		dst:       dst,
		srcs:      srcs,
		offsets:   offsets,
		vertices:  q.lastVertices(len(vertices)),
		nindices:  len(indices),
		color:     color,
		mode:      mode,
		filter:    filter,
		address:   address,
		dstRegion: dstRegion,
		srcRegion: srcRegion,
		shader:    shader,
		uniforms:  uniforms,
		evenOdd:   evenOdd,
	}
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
func (q *commandQueue) Flush() error {
	return runOnMainThread(func() error {
		return q.flush()
	})
}

// flush must be called the main thread.
func (q *commandQueue) flush() error {
	if len(q.commands) == 0 {
		return nil
	}

	es := q.indices
	vs := q.vertices
	debug.Logf("Graphics commands:\n")

	if theGraphicsDriver.HasHighPrecisionFloat() {
		n := q.nvertices / graphics.VertexFloatNum
		for i := 0; i < n; i++ {
			s := q.srcSizes[i]

			idx := i * graphics.VertexFloatNum

			// Convert pixels to texels.
			vs[idx+2] /= s.width
			vs[idx+3] /= s.height

			// Avoid the center of the pixel, which is problematic (#929, #1171).
			// Instead, align the vertices with about 1/3 pixels.
			x := vs[idx]
			y := vs[idx+1]
			ix := float32(math.Floor(float64(x)))
			iy := float32(math.Floor(float64(y)))
			fracx := x - ix
			fracy := y - iy
			switch {
			case fracx < 3.0/16.0:
				vs[idx] = ix
			case fracx < 8.0/16.0:
				vs[idx] = ix + 5.0/16.0
			case fracx < 13.0/16.0:
				vs[idx] = ix + 11.0/16.0
			default:
				vs[idx] = ix + 16.0/16.0
			}
			switch {
			case fracy < 3.0/16.0:
				vs[idx+1] = iy
			case fracy < 8.0/16.0:
				vs[idx+1] = iy + 5.0/16.0
			case fracy < 13.0/16.0:
				vs[idx+1] = iy + 11.0/16.0
			default:
				vs[idx+1] = iy + 16.0/16.0
			}
		}
	} else {
		n := q.nvertices / graphics.VertexFloatNum
		for i := 0; i < n; i++ {
			s := q.srcSizes[i]

			// Convert pixels to texels.
			vs[i*graphics.VertexFloatNum+2] /= s.width
			vs[i*graphics.VertexFloatNum+3] /= s.height
		}
	}

	theGraphicsDriver.Begin()
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
				if ne+dtc.numIndices() > graphics.IndicesNum {
					break
				}
				nv += dtc.numVertices()
				ne += dtc.numIndices()
			}
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
	theGraphicsDriver.End()

	// Release the commands explicitly (#1803).
	// Apparently, the part of a slice between len and cap-1 still holds references.
	// Then, resetting the length by [:0] doesn't release the references.
	for i := range q.commands {
		q.commands[i] = nil
	}
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
	srcs      [graphics.ShaderImageNum]*Image
	offsets   [graphics.ShaderImageNum - 1][2]float32
	vertices  []float32
	nindices  int
	color     affine.ColorM
	mode      driver.CompositeMode
	filter    driver.Filter
	address   driver.Address
	dstRegion driver.Region
	srcRegion driver.Region
	shader    *Shader
	uniforms  []interface{}
	evenOdd   bool
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
	case driver.CompositeModeMultiply:
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
	case driver.AddressUnsafe:
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
func (c *drawTrianglesCommand) Exec(indexOffset int) error {
	// TODO: Is it ok not to bind any framebuffer here?
	if c.nindices == 0 {
		return nil
	}

	var shaderID driver.ShaderID = driver.InvalidShaderID
	var imgs [graphics.ShaderImageNum]driver.ImageID
	if c.shader != nil {
		shaderID = c.shader.shader.ID()
		for i, src := range c.srcs {
			if src == nil {
				imgs[i] = driver.InvalidImageID
				continue
			}
			imgs[i] = src.image.ID()
		}
	} else {
		imgs[0] = c.srcs[0].image.ID()
	}

	return theGraphicsDriver.DrawTriangles(c.dst.image.ID(), imgs, c.offsets, shaderID, c.nindices, indexOffset, c.mode, c.color, c.filter, c.address, c.dstRegion, c.srcRegion, c.uniforms, c.evenOdd)
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
func (c *drawTrianglesCommand) CanMergeWithDrawTrianglesCommand(dst *Image, srcs [graphics.ShaderImageNum]*Image, vertices []float32, color affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address, dstRegion, srcRegion driver.Region, shader *Shader, evenOdd bool) bool {
	// If a shader is used, commands are not merged.
	//
	// TODO: Merge shader commands considering uniform variables.
	if c.shader != nil || shader != nil {
		return false
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

// disposeImageCommand represents a command to dispose an image.
type disposeImageCommand struct {
	target *Image
}

func (c *disposeImageCommand) String() string {
	return fmt.Sprintf("dispose-image: target: %d", c.target.id)
}

// Exec executes the disposeImageCommand.
func (c *disposeImageCommand) Exec(indexOffset int) error {
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
func (c *disposeShaderCommand) Exec(indexOffset int) error {
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
func (c *newImageCommand) Exec(indexOffset int) error {
	i, err := theGraphicsDriver.NewImage(c.width, c.height)
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
func (c *newScreenFramebufferImageCommand) Exec(indexOffset int) error {
	var err error
	c.result.image, err = theGraphicsDriver.NewScreenFramebufferImage(c.width, c.height)
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
func (c *newShaderCommand) Exec(indexOffset int) error {
	var err error
	c.result.shader, err = theGraphicsDriver.NewShader(c.ir)
	return err
}

// InitializeGraphicsDriverState initialize the current graphics driver state.
func InitializeGraphicsDriverState() error {
	return runOnMainThread(func() error {
		return theGraphicsDriver.Initialize()
	})
}

// ResetGraphicsDriverState resets the current graphics driver state.
// If the graphics driver doesn't have an API to reset, ResetGraphicsDriverState does nothing.
func ResetGraphicsDriverState() error {
	if r, ok := theGraphicsDriver.(interface{ Reset() error }); ok {
		return runOnMainThread(func() error {
			return r.Reset()
		})
	}
	return nil
}

// MaxImageSize returns the maximum size of an image.
func MaxImageSize() int {
	var size int
	_ = runOnMainThread(func() error {
		size = theGraphicsDriver.MaxImageSize()
		return nil
	})
	return size
}
