// Copyright 2018 The Ebiten Authors
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

package opengl

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/shaderir"
	"github.com/hajimehoshi/ebiten/internal/thread"
)

var theGraphics Graphics

func Get() *Graphics {
	return &theGraphics
}

type Graphics struct {
	state   openGLState
	context context

	nextImageID driver.ImageID
	images      map[driver.ImageID]*Image

	nextShaderID driver.ShaderID
	shaders      map[driver.ShaderID]*Shader

	// drawCalled is true just after Draw is called. This holds true until ReplacePixels is called.
	drawCalled bool
}

func (g *Graphics) SetThread(thread *thread.Thread) {
	g.context.t = thread
}

func (g *Graphics) Begin() {
	// Do nothing.
}

func (g *Graphics) End() {
	// Call glFlush to prevent black flicking (especially on Android (#226) and iOS).
	// TODO: examples/sprites worked without this. Is this really needed?
	g.context.flush()
}

func (g *Graphics) SetTransparent(transparent bool) {
	// Do nothings.
}

func (g *Graphics) checkSize(width, height int) {
	if width < 1 {
		panic(fmt.Sprintf("opengl: width (%d) must be equal or more than %d", width, 1))
	}
	if height < 1 {
		panic(fmt.Sprintf("opengl: height (%d) must be equal or more than %d", height, 1))
	}
	m := g.context.getMaxTextureSize()
	if width > m {
		panic(fmt.Sprintf("opengl: width (%d) must be less than or equal to %d", width, m))
	}
	if height > m {
		panic(fmt.Sprintf("opengl: height (%d) must be less than or equal to %d", height, m))
	}
}

func (g *Graphics) genNextImageID() driver.ImageID {
	id := g.nextImageID
	g.nextImageID++
	return id
}

func (g *Graphics) InvalidImageID() driver.ImageID {
	return -1
}

func (g *Graphics) genNextShaderID() driver.ShaderID {
	id := g.nextShaderID
	g.nextShaderID++
	return id
}

func (g *Graphics) NewImage(width, height int) (driver.Image, error) {
	i := &Image{
		id:       g.genNextImageID(),
		graphics: g,
		width:    width,
		height:   height,
	}
	w := graphics.InternalImageSize(width)
	h := graphics.InternalImageSize(height)
	g.checkSize(w, h)
	t, err := g.context.newTexture(w, h)
	if err != nil {
		return nil, err
	}
	i.textureNative = t
	g.addImage(i)
	return i, nil
}

func (g *Graphics) NewScreenFramebufferImage(width, height int) (driver.Image, error) {
	g.checkSize(width, height)
	i := &Image{
		id:       g.genNextImageID(),
		graphics: g,
		width:    width,
		height:   height,
		screen:   true,
	}
	g.addImage(i)
	return i, nil
}

func (g *Graphics) addImage(img *Image) {
	if g.images == nil {
		g.images = map[driver.ImageID]*Image{}
	}
	if _, ok := g.images[img.id]; ok {
		panic(fmt.Sprintf("opengl: image ID %d was already registered", img.id))
	}
	g.images[img.id] = img
}

func (g *Graphics) removeImage(img *Image) {
	delete(g.images, img.id)
}

// Reset resets or initializes the current OpenGL state.
func (g *Graphics) Reset() error {
	return g.state.reset(&g.context)
}

func (g *Graphics) SetVertices(vertices []float32, indices []uint16) {
	// Note that the vertices passed to BufferSubData is not under GC management
	// in opengl package due to unsafe-way.
	// See BufferSubData in context_mobile.go.
	g.context.arrayBufferSubData(vertices)
	g.context.elementArrayBufferSubData(indices)
}

func (g *Graphics) Draw(dst, src driver.ImageID, indexLen int, indexOffset int, mode driver.CompositeMode, colorM *affine.ColorM, filter driver.Filter, address driver.Address, sourceRegion driver.Region) error {
	destination := g.images[dst]
	source := g.images[src]

	g.drawCalled = true

	if err := destination.setViewport(); err != nil {
		return err
	}
	g.context.blendFunc(mode)

	program := g.state.programs[programKey{
		useColorM: colorM != nil,
		filter:    filter,
		address:   address,
	}]

	uniforms := []uniformVariable{}

	vw := destination.framebuffer.width
	vh := destination.framebuffer.height
	uniforms = append(uniforms, uniformVariable{
		name:  "viewport_size",
		value: []float32{float32(vw), float32(vh)},
		typ:   shaderir.Type{Main: shaderir.Vec2},
	}, uniformVariable{
		name: "source_region",
		value: []float32{
			sourceRegion.X,
			sourceRegion.Y,
			sourceRegion.X + sourceRegion.Width,
			sourceRegion.Y + sourceRegion.Height,
		},
		typ: shaderir.Type{Main: shaderir.Vec4},
	})

	if colorM != nil {
		// ColorM's elements are immutable. It's OK to hold the reference without copying.
		esBody, esTranslate := colorM.UnsafeElements()
		uniforms = append(uniforms, uniformVariable{
			name:  "color_matrix_body",
			value: esBody,
			typ:   shaderir.Type{Main: shaderir.Mat4},
		}, uniformVariable{
			name:  "color_matrix_translation",
			value: esTranslate,
			typ:   shaderir.Type{Main: shaderir.Vec4},
		})
	}

	if filter != driver.FilterNearest {
		sw, sh := source.framebufferSize()
		uniforms = append(uniforms, uniformVariable{
			name:  "source_size",
			value: []float32{float32(sw), float32(sh)},
			typ:   shaderir.Type{Main: shaderir.Vec2},
		})
	}

	if filter == driver.FilterScreen {
		scale := float32(destination.width) / float32(source.width)
		uniforms = append(uniforms, uniformVariable{
			name:  "scale",
			value: scale,
			typ:   shaderir.Type{Main: shaderir.Float},
		})
	}

	var imgs [graphics.ShaderImageNum]textureVariable
	for i := range imgs {
		if i == 0 {
			imgs[i].valid = true
			imgs[i].native = source.textureNative
		}
	}

	if err := g.useProgram(program, uniforms, imgs); err != nil {
		return err
	}

	g.context.drawElements(indexLen, indexOffset*2) // 2 is uint16 size in bytes

	// glFlush() might be necessary at least on MacBook Pro (a smilar problem at #419),
	// but basically this pass the tests (esp. TestImageTooManyFill).
	// As glFlush() causes performance problems, this should be avoided as much as possible.
	// Let's wait and see, and file a new issue when this problem is newly foung.
	return nil
}

func (g *Graphics) SetVsyncEnabled(enabled bool) {
	// Do nothing
}

func (g *Graphics) FramebufferYDirection() driver.YDirection {
	return driver.Upward
}

func (g *Graphics) NeedsRestoring() bool {
	return g.context.needsRestoring()
}

func (g *Graphics) IsGL() bool {
	return true
}

func (g *Graphics) HasHighPrecisionFloat() bool {
	return g.context.hasHighPrecisionFloat()
}

func (g *Graphics) MaxImageSize() int {
	return g.context.getMaxTextureSize()
}

func (g *Graphics) NewShader(program *shaderir.Program) (driver.Shader, error) {
	s, err := newShader(g.genNextShaderID(), g, program)
	if err != nil {
		return nil, err
	}
	g.addShader(s)
	return s, nil
}

func (g *Graphics) addShader(shader *Shader) {
	if g.shaders == nil {
		g.shaders = map[driver.ShaderID]*Shader{}
	}
	if _, ok := g.shaders[shader.id]; ok {
		panic(fmt.Sprintf("opengl: shader ID %d was already registered", shader.id))
	}
	g.shaders[shader.id] = shader
}

func (g *Graphics) removeShader(shader *Shader) {
	delete(g.shaders, shader.id)
}

func (g *Graphics) DrawShader(dst driver.ImageID, srcs [graphics.ShaderImageNum]driver.ImageID, offsets [graphics.ShaderImageNum - 1][2]float32, shader driver.ShaderID, indexLen int, indexOffset int, sourceRegion driver.Region, mode driver.CompositeMode, uniforms []interface{}) error {
	d := g.images[dst]
	s := g.shaders[shader]

	g.drawCalled = true

	if err := d.setViewport(); err != nil {
		return err
	}
	g.context.blendFunc(mode)

	us := make([]uniformVariable, graphics.PreservedUniformVariablesNum+len(uniforms))

	{
		const idx = graphics.DestinationTextureSizeUniformVariableIndex
		w, h := d.framebufferSize()
		us[idx].name = "U0"
		us[idx].value = []float32{float32(w), float32(h)}
		us[idx].typ = s.ir.Uniforms[0]
	}
	{
		sizes := make([]float32, 2*len(srcs))
		for i, src := range srcs {
			if img := g.images[src]; img != nil {
				w, h := img.framebufferSize()
				sizes[2*i] = float32(w)
				sizes[2*i+1] = float32(h)
			}

		}
		const idx = graphics.TextureSizesUniformVariableIndex
		us[idx].name = fmt.Sprintf("U%d", idx)
		us[idx].value = sizes
		us[idx].typ = s.ir.Uniforms[idx]
	}
	{
		voffsets := make([]float32, 2*len(offsets))
		for i, o := range offsets {
			voffsets[2*i] = o[0]
			voffsets[2*i+1] = o[1]
		}
		const idx = graphics.TextureSourceOffsetsUniformVariableIndex
		us[idx].name = fmt.Sprintf("U%d", idx)
		us[idx].value = voffsets
		us[idx].typ = s.ir.Uniforms[idx]
	}
	{
		origin := []float32{float32(sourceRegion.X), float32(sourceRegion.Y)}
		const idx = graphics.TextureSourceRegionOriginUniformVariableIndex
		us[idx].name = fmt.Sprintf("U%d", idx)
		us[idx].value = origin
		us[idx].typ = s.ir.Uniforms[idx]
	}
	{
		size := []float32{float32(sourceRegion.Width), float32(sourceRegion.Height)}
		const idx = graphics.TextureSourceRegionSizeUniformVariableIndex
		us[idx].name = fmt.Sprintf("U%d", idx)
		us[idx].value = size
		us[idx].typ = s.ir.Uniforms[idx]
	}

	for i, v := range uniforms {
		const offset = graphics.PreservedUniformVariablesNum
		us[i+offset].name = fmt.Sprintf("U%d", i+offset)
		us[i+offset].value = v
		us[i+offset].typ = s.ir.Uniforms[i+offset]
	}

	var ts [graphics.ShaderImageNum]textureVariable
	for i, src := range srcs {
		if src == g.InvalidImageID() {
			continue
		}
		ts[i].valid = true
		ts[i].native = g.images[src].textureNative
	}

	if err := g.useProgram(s.p, us, ts); err != nil {
		return err
	}
	g.context.drawElements(indexLen, indexOffset*2) // 2 is uint16 size in bytes

	return nil
}
