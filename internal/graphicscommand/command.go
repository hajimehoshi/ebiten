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

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/shaderir"
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
	CanMergeWithDrawTrianglesCommand(dst *Image, src [graphics.ShaderImageNum]*Image, color *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address, sourceRegion driver.Region, shader *Shader) bool
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
func (q *commandQueue) EnqueueDrawTrianglesCommand(dst *Image, srcs [graphics.ShaderImageNum]*Image, offsets [graphics.ShaderImageNum - 1][2]float32, vertices []float32, indices []uint16, color *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address, sourceRegion driver.Region, shader *Shader, uniforms []interface{}) {
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
		sourceRegion.X /= float32(w)
		sourceRegion.Y /= float32(h)
		sourceRegion.Width /= float32(w)
		sourceRegion.Height /= float32(h)
		for i := range offsets {
			offsets[i][0] /= float32(w)
			offsets[i][1] /= float32(h)
		}
	}

	// TODO: If dst is the screen, reorder the command to be the last.
	if !split && 0 < len(q.commands) {
		// TODO: Pass offsets and uniforms when merging considers the shader.
		if last := q.commands[len(q.commands)-1]; last.CanMergeWithDrawTrianglesCommand(dst, srcs, color, mode, filter, address, sourceRegion, shader) {
			last.AddNumVertices(len(vertices))
			last.AddNumIndices(len(indices))
			return
		}
	}

	c := &drawTrianglesCommand{
		dst:          dst,
		srcs:         srcs,
		offsets:      offsets,
		nvertices:    len(vertices),
		nindices:     len(indices),
		color:        color,
		mode:         mode,
		filter:       filter,
		address:      address,
		sourceRegion: sourceRegion,
		shader:       shader,
		uniforms:     uniforms,
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

// Flush flushes the command queue.
func (q *commandQueue) Flush() error {
	if len(q.commands) == 0 {
		return nil
	}

	es := q.indices
	vs := q.vertices
	if recordLog() {
		fmt.Println("--")
	}

	if theGraphicsDriver.HasHighPrecisionFloat() {
		n := q.nvertices / graphics.VertexFloatNum
		for i := 0; i < n; i++ {
			s := q.srcSizes[i]

			// Convert pixels to texels.
			vs[i*graphics.VertexFloatNum+2] /= s.width
			vs[i*graphics.VertexFloatNum+3] /= s.height

			// Avoid the center of the pixel, which is problematic (#929, #1171).
			// Instead, align the vertices with about 1/3 pixels.
			for idx := 0; idx < 2; idx++ {
				x := vs[i*graphics.VertexFloatNum+idx]
				int := float32(math.Floor(float64(x)))
				frac := x - int
				switch {
				case frac < 3.0/16.0:
					vs[i*graphics.VertexFloatNum+idx] = int
				case frac < 8.0/16.0:
					vs[i*graphics.VertexFloatNum+idx] = int + 5.0/16.0
				case frac < 13.0/16.0:
					vs[i*graphics.VertexFloatNum+idx] = int + 11.0/16.0
				default:
					vs[i*graphics.VertexFloatNum+idx] = int + 16.0/16.0
				}
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
	dst          *Image
	srcs         [graphics.ShaderImageNum]*Image
	offsets      [graphics.ShaderImageNum - 1][2]float32
	nvertices    int
	nindices     int
	color        *affine.ColorM
	mode         driver.CompositeMode
	filter       driver.Filter
	address      driver.Address
	sourceRegion driver.Region
	shader       *Shader
	uniforms     []interface{}
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

	return fmt.Sprintf("draw-triangles: dst: %s <- src: [%s], num of indices: %d, colorm: %v, mode %s, filter: %s, address: %s", dst, strings.Join(srcstrs[:], ", "), c.nindices, c.color, mode, filter, address)
}

// Exec executes the drawTrianglesCommand.
func (c *drawTrianglesCommand) Exec(indexOffset int) error {
	// TODO: Is it ok not to bind any framebuffer here?
	if c.nindices == 0 {
		return nil
	}

	if c.shader != nil {
		var imgs [graphics.ShaderImageNum]driver.ImageID
		for i, src := range c.srcs {
			if src == nil {
				imgs[i] = theGraphicsDriver.InvalidImageID()
				continue
			}
			imgs[i] = src.image.ID()
		}

		return theGraphicsDriver.DrawShader(c.dst.image.ID(), imgs, c.offsets, c.shader.shader.ID(), c.nindices, indexOffset, c.sourceRegion, c.mode, c.uniforms)
	}
	return theGraphicsDriver.Draw(c.dst.image.ID(), c.srcs[0].image.ID(), c.nindices, indexOffset, c.mode, c.color, c.filter, c.address, c.sourceRegion)
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
func (c *drawTrianglesCommand) CanMergeWithDrawTrianglesCommand(dst *Image, srcs [graphics.ShaderImageNum]*Image, color *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address, sourceRegion driver.Region, shader *Shader) bool {
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
	if c.sourceRegion != sourceRegion {
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

func (c *replacePixelsCommand) CanMergeWithDrawTrianglesCommand(dst *Image, src [graphics.ShaderImageNum]*Image, color *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address, sourceRegion driver.Region, shader *Shader) bool {
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

func (c *pixelsCommand) CanMergeWithDrawTrianglesCommand(dst *Image, src [graphics.ShaderImageNum]*Image, color *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address, sourceRegion driver.Region, shader *Shader) bool {
	return false
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

func (c *disposeImageCommand) NumVertices() int {
	return 0
}

func (c *disposeImageCommand) NumIndices() int {
	return 0
}

func (c *disposeImageCommand) AddNumVertices(n int) {
}

func (c *disposeImageCommand) AddNumIndices(n int) {
}

func (c *disposeImageCommand) CanMergeWithDrawTrianglesCommand(dst *Image, src [graphics.ShaderImageNum]*Image, color *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address, sourceRegion driver.Region, shader *Shader) bool {
	return false
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

func (c *disposeShaderCommand) NumVertices() int {
	return 0
}

func (c *disposeShaderCommand) NumIndices() int {
	return 0
}

func (c *disposeShaderCommand) AddNumVertices(n int) {
}

func (c *disposeShaderCommand) AddNumIndices(n int) {
}

func (c *disposeShaderCommand) CanMergeWithDrawTrianglesCommand(dst *Image, src [graphics.ShaderImageNum]*Image, color *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address, sourceRegion driver.Region, shader *Shader) bool {
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

func (c *newImageCommand) CanMergeWithDrawTrianglesCommand(dst *Image, src [graphics.ShaderImageNum]*Image, color *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address, sourceRegion driver.Region, shader *Shader) bool {
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

func (c *newScreenFramebufferImageCommand) CanMergeWithDrawTrianglesCommand(dst *Image, src [graphics.ShaderImageNum]*Image, color *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address, sourceRegion driver.Region, shader *Shader) bool {
	return false
}

// newShaderCommand is a command to create a shader.
type newShaderCommand struct {
	result *Shader
	ir     *shaderir.Program
}

func (c *newShaderCommand) String() string {
	return fmt.Sprintf("new-shader-image")
}

// Exec executes a newShaderCommand.
func (c *newShaderCommand) Exec(indexOffset int) error {
	var err error
	c.result.shader, err = theGraphicsDriver.NewShader(c.ir)
	return err
}

func (c *newShaderCommand) NumVertices() int {
	return 0
}

func (c *newShaderCommand) NumIndices() int {
	return 0
}

func (c *newShaderCommand) AddNumVertices(n int) {
}

func (c *newShaderCommand) AddNumIndices(n int) {
}

func (c *newShaderCommand) CanMergeWithDrawTrianglesCommand(dst *Image, src [graphics.ShaderImageNum]*Image, color *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address, sourceRegion driver.Region, shader *Shader) bool {
	return false
}

// ResetGraphicsDriverState resets or initializes the current graphics driver state.
func ResetGraphicsDriverState() error {
	return theGraphicsDriver.Reset()
}
