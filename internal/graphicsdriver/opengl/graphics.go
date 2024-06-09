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

//go:build !playstation5

package opengl

import (
	"fmt"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl/gl"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

type activatedTexture struct {
	textureNative textureNative
	index         int
}

type Graphics struct {
	state   openGLState
	context context
	vsync   bool

	nextImageID graphicsdriver.ImageID
	images      map[graphicsdriver.ImageID]*Image

	nextShaderID graphicsdriver.ShaderID
	shaders      map[graphicsdriver.ShaderID]*Shader

	// drawCalled is true just after Draw is called. This holds true until WritePixels is called.
	drawCalled bool

	uniformVariableNameCache map[int]string
	textureVariableNameCache map[int]string

	uniformVars []uniformVariable

	// activatedTextures is a set of activated textures.
	// textureNative cannot be a map key unfortunately.
	activatedTextures []activatedTexture

	graphicsPlatform
}

func newGraphics(ctx gl.Context) *Graphics {
	g := &Graphics{
		vsync: true,
	}
	if isDebug {
		g.context.ctx = &gl.DebugContext{Context: ctx}
	} else {
		g.context.ctx = ctx
	}
	return g
}

func (g *Graphics) Begin() error {
	// Do nothing.
	return nil
}

func (g *Graphics) End(present bool) error {
	// Call glFlush to prevent black flicking (especially on Android (#226) and iOS).
	// TODO: examples/sprites worked without this. Is this really needed?
	g.context.ctx.Flush()

	// The last uniforms must be reset before swapping the buffer (#2517).
	if present {
		g.state.resetLastUniforms()
		if err := g.swapBuffers(); err != nil {
			return err
		}
	}

	return nil
}

func (g *Graphics) SetTransparent(transparent bool) {
	// Do nothing.
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

func (g *Graphics) genNextImageID() graphicsdriver.ImageID {
	g.nextImageID++
	return g.nextImageID
}

func (g *Graphics) genNextShaderID() graphicsdriver.ShaderID {
	g.nextShaderID++
	return g.nextShaderID
}

func (g *Graphics) NewImage(width, height int) (graphicsdriver.Image, error) {
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
	i.texture = t
	g.addImage(i)
	return i, nil
}

func (g *Graphics) NewScreenFramebufferImage(width, height int) (graphicsdriver.Image, error) {
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
		g.images = map[graphicsdriver.ImageID]*Image{}
	}
	if _, ok := g.images[img.id]; ok {
		panic(fmt.Sprintf("opengl: image ID %d was already registered", img.id))
	}
	g.images[img.id] = img
}

func (g *Graphics) removeImage(img *Image) {
	delete(g.images, img.id)
}

func (g *Graphics) Initialize() error {
	if err := g.makeContextCurrent(); err != nil {
		return err
	}
	if err := g.state.reset(&g.context); err != nil {
		return err
	}
	return nil
}

// Reset resets or initializes the current OpenGL state.
func (g *Graphics) Reset() error {
	return g.state.reset(&g.context)
}

func (g *Graphics) SetVertices(vertices []float32, indices []uint32) error {
	g.state.setVertices(&g.context, vertices, indices)
	return nil
}

func (g *Graphics) uniformVariableName(idx int) string {
	if v, ok := g.uniformVariableNameCache[idx]; ok {
		return v
	}
	if g.uniformVariableNameCache == nil {
		g.uniformVariableNameCache = map[int]string{}
	}
	name := fmt.Sprintf("U%d", idx)
	g.uniformVariableNameCache[idx] = name
	return name
}

func (g *Graphics) DrawTriangles(dstIDs [graphics.ShaderDstImageCount]graphicsdriver.ImageID, srcIDs [graphics.ShaderSrcImageCount]graphicsdriver.ImageID, shaderID graphicsdriver.ShaderID, dstRegions []graphicsdriver.DstRegion, indexOffset int, blend graphicsdriver.Blend, uniforms []uint32, fillRule graphicsdriver.FillRule) error {
	if shaderID == graphicsdriver.InvalidShaderID {
		return fmt.Errorf("opengl: shader ID is invalid")
	}

	g.drawCalled = true
	targetCount := 0
	firstTarget := -1
	var dsts [graphics.ShaderDstImageCount]*Image
	for i, dstID := range dstIDs {
		if dstID == graphicsdriver.InvalidImageID {
			continue
		}
		dst := g.images[dstIDs[i]]
		if dst == nil {
			continue
		}
		if firstTarget == -1 {
			firstTarget = i
		}
		if err := dst.ensureFramebuffer(); err != nil {
			return err
		}
		dsts[i] = dst
		targetCount++
	}

	f := uint32(dsts[firstTarget].framebuffer.native)
	// If the number of targets is more than one, or if the only target is the first one, then
	// it is safe to assume that MRT is used.
	// Also, it only matters in order to specify empty targets/viewports when not all slots are
	// being filled.
	usesMRT := firstTarget > 0 || targetCount > 1
	if usesMRT {
		f = uint32(g.context.mrtFramebuffer)
		// Create the initial MRT framebuffer
		if f == 0 {
			f = g.context.ctx.CreateFramebuffer()
			if f <= 0 {
				return fmt.Errorf("opengl: creating framebuffer failed: the returned value is not positive but %d", f)
			}
			g.context.mrtFramebuffer = framebufferNative(f)
		}

		g.context.bindFramebuffer(framebufferNative(f))

		// Reset color attachments
		if s := g.context.ctx.CheckFramebufferStatus(gl.FRAMEBUFFER); s == gl.FRAMEBUFFER_COMPLETE {
			g.context.ctx.Clear(gl.COLOR_BUFFER_BIT | gl.STENCIL_BUFFER_BIT)
		}
		for i, dst := range dsts {
			if dst == nil {
				continue
			}
			g.context.ctx.FramebufferTexture2D(gl.FRAMEBUFFER, gl.COLOR_ATTACHMENT0+uint32(i), gl.TEXTURE_2D, uint32(dst.texture), 0)
		}
		if s := g.context.ctx.CheckFramebufferStatus(gl.FRAMEBUFFER); s != gl.FRAMEBUFFER_COMPLETE {
			if s != 0 {
				return fmt.Errorf("opengl: creating framebuffer failed: %v", s)
			}
			if e := g.context.ctx.GetError(); e != gl.NO_ERROR {
				return fmt.Errorf("opengl: creating framebuffer failed: (glGetError) %d", e)
			}
			return fmt.Errorf("opengl: creating framebuffer failed: unknown error")
		}
		// Color attachments
		var attached []uint32
		for i, dst := range dsts {
			if dst == nil {
				attached = append(attached, gl.NONE)
				continue
			}
			attached = append(attached, uint32(gl.COLOR_ATTACHMENT0+i))
		}
		g.context.ctx.DrawBuffers(attached)
	} else {
		g.context.bindFramebuffer(framebufferNative(f))
	}

	w, h := dsts[firstTarget].viewportSize() //.framebuffer.viewportWidth, dsts[firstTarget].framebuffer.viewportHeight
	g.context.setViewport(w, h, dsts[firstTarget].screen)

	g.context.blend(blend)

	shader := g.shaders[shaderID]
	program := shader.p

	ulen := len(shader.ir.Uniforms)
	if cap(g.uniformVars) < ulen {
		g.uniformVars = make([]uniformVariable, ulen)
	} else {
		g.uniformVars = g.uniformVars[:ulen]
	}

	var idx int
	for i, typ := range shader.ir.Uniforms {
		n := typ.Uint32Count()
		g.uniformVars[i].name = g.uniformVariableName(i)
		g.uniformVars[i].value = uniforms[idx : idx+n]
		g.uniformVars[i].typ = typ
		idx += n
	}

	// In OpenGL, the NDC's Y direction is upward, so flip the Y direction for the final framebuffer.
	if !usesMRT && dsts[firstTarget].screen {
		const idx = graphics.ProjectionMatrixUniformVariableIndex
		// Invert the sign bits as float32 values.
		g.uniformVars[idx].value[1] ^= 1 << 31
		g.uniformVars[idx].value[5] ^= 1 << 31
		g.uniformVars[idx].value[9] ^= 1 << 31
		g.uniformVars[idx].value[13] ^= 1 << 31
	}

	var imgs [graphics.ShaderSrcImageCount]textureVariable
	for i, srcID := range srcIDs {
		if srcID == graphicsdriver.InvalidImageID {
			continue
		}
		imgs[i].valid = true
		imgs[i].native = g.images[srcID].texture
	}

	if err := g.useProgram(program, g.uniformVars, imgs); err != nil {
		return err
	}

	for i := range g.uniformVars {
		g.uniformVars[i] = uniformVariable{}
	}
	g.uniformVars = g.uniformVars[:0]

	if fillRule != graphicsdriver.FillRuleFillAll {
		if err := dsts[firstTarget].ensureStencilBuffer(framebufferNative(f)); err != nil {
			return err
		}
		g.context.ctx.Enable(gl.STENCIL_TEST)
	}

	for _, dstRegion := range dstRegions {
		g.context.ctx.Scissor(
			int32(dstRegion.Region.Min.X),
			int32(dstRegion.Region.Min.Y),
			int32(dstRegion.Region.Dx()),
			int32(dstRegion.Region.Dy()),
		)

		switch fillRule {
		case graphicsdriver.FillRuleNonZero:
			g.context.ctx.Clear(gl.STENCIL_BUFFER_BIT)
			g.context.ctx.StencilFunc(gl.ALWAYS, 0x00, 0xff)
			g.context.ctx.StencilOpSeparate(gl.FRONT, gl.KEEP, gl.KEEP, gl.INCR_WRAP)
			g.context.ctx.StencilOpSeparate(gl.BACK, gl.KEEP, gl.KEEP, gl.DECR_WRAP)
			g.context.ctx.ColorMask(false, false, false, false)

			g.context.ctx.DrawElements(gl.TRIANGLES, int32(dstRegion.IndexCount), gl.UNSIGNED_INT, indexOffset*int(unsafe.Sizeof(uint32(0))))
		case graphicsdriver.FillRuleEvenOdd:
			g.context.ctx.Clear(gl.STENCIL_BUFFER_BIT)
			g.context.ctx.StencilFunc(gl.ALWAYS, 0x00, 0xff)
			g.context.ctx.StencilOpSeparate(gl.FRONT_AND_BACK, gl.KEEP, gl.KEEP, gl.INVERT)
			g.context.ctx.ColorMask(false, false, false, false)

			g.context.ctx.DrawElements(gl.TRIANGLES, int32(dstRegion.IndexCount), gl.UNSIGNED_INT, indexOffset*int(unsafe.Sizeof(uint32(0))))
		}
		if fillRule != graphicsdriver.FillRuleFillAll {
			g.context.ctx.StencilFunc(gl.NOTEQUAL, 0x00, 0xff)
			g.context.ctx.StencilOpSeparate(gl.FRONT_AND_BACK, gl.KEEP, gl.KEEP, gl.KEEP)
			g.context.ctx.ColorMask(true, true, true, true)
		}
		g.context.ctx.DrawElements(gl.TRIANGLES, int32(dstRegion.IndexCount), gl.UNSIGNED_INT, indexOffset*int(unsafe.Sizeof(uint32(0))))
		indexOffset += dstRegion.IndexCount
	}

	if fillRule != graphicsdriver.FillRuleFillAll {
		g.context.ctx.Disable(gl.STENCIL_TEST)
	}

	// Detach existing color attachments
	//g.context.bindFramebuffer(fb)
	//TODO:

	return nil
}

func (g *Graphics) SetVsyncEnabled(enabled bool) {
	g.vsync = enabled
}

func (g *Graphics) NeedsClearingScreen() bool {
	return true
}

func (g *Graphics) MaxImageSize() int {
	return g.context.getMaxTextureSize()
}

func (g *Graphics) NewShader(program *shaderir.Program) (graphicsdriver.Shader, error) {
	s, err := newShader(g.genNextShaderID(), g, program)
	if err != nil {
		return nil, err
	}
	g.addShader(s)
	return s, nil
}

func (g *Graphics) addShader(shader *Shader) {
	if g.shaders == nil {
		g.shaders = map[graphicsdriver.ShaderID]*Shader{}
	}
	if _, ok := g.shaders[shader.id]; ok {
		panic(fmt.Sprintf("opengl: shader ID %d was already registered", shader.id))
	}
	g.shaders[shader.id] = shader
}

func (g *Graphics) removeShader(shader *Shader) {
	delete(g.shaders, shader.id)
}
