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
	"image"
	"math"
	"strings"

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

	Exec(commandQueue *commandQueue, graphicsDriver graphicsdriver.Graphics, indexOffset int) error
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
	p.pool[len(p.pool)-1] = nil
	p.pool = p.pool[:len(p.pool)-1]
	return v
}

func (p *drawTrianglesCommandPool) put(v *drawTrianglesCommand) {
	if len(p.pool) >= 1024 {
		return
	}
	p.pool = append(p.pool, v)
}

// drawTrianglesCommand represents a drawing command to draw an image on another image.
type drawTrianglesCommand struct {
	dst        *Image
	srcs       [graphics.ShaderSrcImageCount]*Image
	vertices   []float32
	blend      graphicsdriver.Blend
	dstRegions []graphicsdriver.DstRegion
	shader     *Shader
	uniforms   []uint32
	fillRule   graphicsdriver.FillRule
}

func (c *drawTrianglesCommand) String() string {
	var blend string
	switch c.blend {
	case graphicsdriver.BlendSourceOver:
		blend = "(source-over)"
	case graphicsdriver.BlendClear:
		blend = "(clear)"
	case graphicsdriver.BlendCopy:
		blend = "(copy)"
	default:
		blend = fmt.Sprintf("{src-rgb: %d, src-alpha: %d, dst-rgb: %d, dst-alpha: %d, op-rgb: %d, op-alpha: %d}",
			c.blend.BlendFactorSourceRGB,
			c.blend.BlendFactorSourceAlpha,
			c.blend.BlendFactorDestinationRGB,
			c.blend.BlendFactorDestinationAlpha,
			c.blend.BlendOperationRGB,
			c.blend.BlendOperationAlpha)
	}

	dst := fmt.Sprintf("%d", c.dst.id)
	if c.dst.screen {
		dst += " (screen)"
	} else if c.dst.attribute != "" {
		dst += " (" + c.dst.attribute + ")"
	}

	var srcstrs [graphics.ShaderSrcImageCount]string
	for i, src := range c.srcs {
		if src == nil {
			srcstrs[i] = "(nil)"
			continue
		}
		srcstrs[i] = fmt.Sprintf("%d", src.id)
		if src.screen {
			srcstrs[i] += " (screen)"
		} else if src.attribute != "" {
			srcstrs[i] += " (" + src.attribute + ")"
		}
	}

	shader := fmt.Sprintf("%d", c.shader.id)
	if c.shader.name != "" {
		shader += " (" + c.shader.name + ")"
	}

	return fmt.Sprintf("draw-triangles: dst: %s <- src: [%s], num of dst regions: %d, num of indices: %d, blend: %s, fill rule: %s, shader: %s", dst, strings.Join(srcstrs[:], ", "), len(c.dstRegions), c.numIndices(), blend, c.fillRule, shader)
}

// Exec executes the drawTrianglesCommand.
func (c *drawTrianglesCommand) Exec(commandQueue *commandQueue, graphicsDriver graphicsdriver.Graphics, indexOffset int) error {
	// TODO: Is it ok not to bind any framebuffer here?
	if len(c.dstRegions) == 0 {
		return nil
	}

	var imgs [graphics.ShaderSrcImageCount]graphicsdriver.ImageID
	for i, src := range c.srcs {
		if src == nil {
			imgs[i] = graphicsdriver.InvalidImageID
			continue
		}
		imgs[i] = src.image.ID()
	}

	return graphicsDriver.DrawTriangles(c.dst.image.ID(), imgs, c.shader.shader.ID(), c.dstRegions, indexOffset, c.blend, c.uniforms, c.fillRule)
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
func (c *drawTrianglesCommand) CanMergeWithDrawTrianglesCommand(dst *Image, srcs [graphics.ShaderSrcImageCount]*Image, vertices []float32, blend graphicsdriver.Blend, shader *Shader, uniforms []uint32, fillRule graphicsdriver.FillRule) bool {
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
	if c.fillRule != fillRule {
		return false
	}
	if c.fillRule != graphicsdriver.FillRuleFillAll && mightOverlapDstRegions(c.vertices, vertices) {
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

	for i := 0; i < len(vertices); i += graphics.VertexFloatCount {
		x := vertices[i]
		y := vertices[i+1]
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
	const margin = 1
	return minX1 < maxX2+margin && minX2 < maxX1+margin && minY1 < maxY2+margin && minY2 < maxY1+margin
}

// writePixelsCommand represents a command to replace pixels of an image.
type writePixelsCommand struct {
	dst  *Image
	args []writePixelsCommandArgs
}

type writePixelsCommandArgs struct {
	pixels *graphics.ManagedBytes
	region image.Rectangle
}

func (c *writePixelsCommand) String() string {
	var args []string
	for _, a := range c.args {
		args = append(args, fmt.Sprintf("region: %s", a.region.String()))
	}
	return fmt.Sprintf("write-pixels: dst: %d, args: %s", c.dst.id, strings.Join(args, ", "))
}

// Exec executes the writePixelsCommand.
func (c *writePixelsCommand) Exec(commandQueue *commandQueue, graphicsDriver graphicsdriver.Graphics, indexOffset int) error {
	if len(c.args) == 0 {
		return nil
	}
	args := make([]graphicsdriver.PixelsArgs, 0, len(c.args))
	for _, a := range c.args {
		pix, f := a.pixels.GetAndRelease()
		// A finalizer is executed when flushing the queue at the end of the frame.
		// At the end of the frame, the last command is rendering triangles onto the screen,
		// so the bytes are already sent to GPU and synced.
		// TODO: This might be fragile. When is the better time to call finalizers by a command queue?
		commandQueue.addFinalizer(f)
		args = append(args, graphicsdriver.PixelsArgs{
			Pixels: pix,
			Region: a.region,
		})
	}
	if err := c.dst.image.WritePixels(args); err != nil {
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
func (c *readPixelsCommand) Exec(commandQueue *commandQueue, graphicsDriver graphicsdriver.Graphics, indexOffset int) error {
	if err := c.img.image.ReadPixels(c.args); err != nil {
		return err
	}
	return nil
}

func (c *readPixelsCommand) NeedsSync() bool {
	return true
}

func (c *readPixelsCommand) String() string {
	var args []string
	for _, a := range c.args {
		args = append(args, fmt.Sprintf("region: %s", a.Region.String()))
	}
	return fmt.Sprintf("read-pixels: image: %d, args: %v", c.img.id, strings.Join(args, ", "))
}

// disposeImageCommand represents a command to dispose an image.
type disposeImageCommand struct {
	target *Image
}

func (c *disposeImageCommand) String() string {
	return fmt.Sprintf("dispose-image: target: %d", c.target.id)
}

// Exec executes the disposeImageCommand.
func (c *disposeImageCommand) Exec(commandQueue *commandQueue, graphicsDriver graphicsdriver.Graphics, indexOffset int) error {
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
func (c *disposeShaderCommand) Exec(commandQueue *commandQueue, graphicsDriver graphicsdriver.Graphics, indexOffset int) error {
	c.target.shader.Dispose()
	return nil
}

func (c *disposeShaderCommand) NeedsSync() bool {
	return false
}

// newImageCommand represents a command to create an empty image with given width and height.
type newImageCommand struct {
	result    *Image
	width     int
	height    int
	screen    bool
	attribute string
}

func (c *newImageCommand) String() string {
	str := fmt.Sprintf("new-image: result: %d, width: %d, height: %d, screen: %t", c.result.id, c.width, c.height, c.screen)
	if c.attribute != "" {
		str += ", attribute: " + c.attribute
	}
	return str
}

// Exec executes a newImageCommand.
func (c *newImageCommand) Exec(commandQueue *commandQueue, graphicsDriver graphicsdriver.Graphics, indexOffset int) error {
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
func (c *newShaderCommand) Exec(commandQueue *commandQueue, graphicsDriver graphicsdriver.Graphics, indexOffset int) error {
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
