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

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/microsoftgdk"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

// NewGraphics creates an implementation of graphicsdriver.Graphics for OpenGL.
// The returned graphics value is nil iff the error is not nil.
func NewGraphics() (graphicsdriver.Graphics, error) {
	if microsoftgdk.IsXbox() {
		return nil, fmt.Errorf("opengl: OpenGL is not supported on Xbox")
	}
	g := &Graphics{}
	g.init()
	return g, nil
}

type activatedTexture struct {
	textureNative textureNative
	index         int
}

type Graphics struct {
	state   openGLState
	context context

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
}

func (g *Graphics) Begin() error {
	// Do nothing.
	return nil
}

func (g *Graphics) End(present bool) error {
	// Call glFlush to prevent black flicking (especially on Android (#226) and iOS).
	// TODO: examples/sprites worked without this. Is this really needed?
	g.context.flush()
	return nil
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
	return g.state.reset(&g.context)
}

// Reset resets or initializes the current OpenGL state.
func (g *Graphics) Reset() error {
	return g.state.reset(&g.context)
}

func (g *Graphics) SetVertices(vertices []float32, indices []uint16) error {
	// Note that the vertices passed to BufferSubData is not under GC management
	// in opengl package due to unsafe-way.
	// See BufferSubData in context_mobile.go.
	g.context.arrayBufferSubData(vertices)
	g.context.elementArrayBufferSubData(indices)
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

func (g *Graphics) DrawTriangles(dstID graphicsdriver.ImageID, srcIDs [graphics.ShaderImageCount]graphicsdriver.ImageID, offsets [graphics.ShaderImageCount - 1][2]float32, shaderID graphicsdriver.ShaderID, indexLen int, indexOffset int, mode graphicsdriver.CompositeMode, colorM graphicsdriver.ColorM, filter graphicsdriver.Filter, address graphicsdriver.Address, dstRegion, srcRegion graphicsdriver.Region, uniforms [][]float32, evenOdd bool) error {
	destination := g.images[dstID]

	g.drawCalled = true

	if err := destination.setViewport(); err != nil {
		return err
	}
	g.context.scissor(
		int(dstRegion.X),
		int(dstRegion.Y),
		int(dstRegion.Width),
		int(dstRegion.Height),
	)
	g.context.blendFunc(mode)

	var program program
	if shaderID == graphicsdriver.InvalidShaderID {
		program = g.state.programs[programKey{
			useColorM: !colorM.IsIdentity(),
			filter:    filter,
			address:   address,
		}]

		dw, dh := destination.framebufferSize()
		g.uniformVars = append(g.uniformVars, uniformVariable{
			name:  "viewport_size",
			value: []float32{float32(dw), float32(dh)},
			typ:   shaderir.Type{Main: shaderir.Vec2},
		}, uniformVariable{
			name: "source_region",
			value: []float32{
				srcRegion.X,
				srcRegion.Y,
				srcRegion.X + srcRegion.Width,
				srcRegion.Y + srcRegion.Height,
			},
			typ: shaderir.Type{Main: shaderir.Vec4},
		})

		if !colorM.IsIdentity() {
			// ColorM's elements are immutable. It's OK to hold the reference without copying.
			var esBody [16]float32
			var esTranslate [4]float32
			colorM.Elements(&esBody, &esTranslate)
			g.uniformVars = append(g.uniformVars, uniformVariable{
				name:  "color_matrix_body",
				value: esBody[:],
				typ:   shaderir.Type{Main: shaderir.Mat4},
			}, uniformVariable{
				name:  "color_matrix_translation",
				value: esTranslate[:],
				typ:   shaderir.Type{Main: shaderir.Vec4},
			})
		}

		if filter != graphicsdriver.FilterNearest {
			sw, sh := g.images[srcIDs[0]].framebufferSize()
			g.uniformVars = append(g.uniformVars, uniformVariable{
				name:  "source_size",
				value: []float32{float32(sw), float32(sh)},
				typ:   shaderir.Type{Main: shaderir.Vec2},
			})
		}

		if filter == graphicsdriver.FilterScreen {
			scale := float32(destination.width) / float32(g.images[srcIDs[0]].width)
			g.uniformVars = append(g.uniformVars, uniformVariable{
				name:  "scale",
				value: []float32{scale},
				typ:   shaderir.Type{Main: shaderir.Float},
			})
		}
	} else {
		shader := g.shaders[shaderID]
		program = shader.p

		ulen := graphics.PreservedUniformVariablesCount + len(uniforms)
		if cap(g.uniformVars) < ulen {
			g.uniformVars = make([]uniformVariable, ulen)
		} else {
			g.uniformVars = g.uniformVars[:ulen]
		}

		{
			const idx = graphics.TextureDestinationSizeUniformVariableIndex
			w, h := destination.framebufferSize()
			g.uniformVars[idx].name = g.uniformVariableName(idx)
			g.uniformVars[idx].value = []float32{float32(w), float32(h)}
			g.uniformVars[idx].typ = shader.ir.Uniforms[idx]
		}
		{
			sizes := make([]float32, 2*len(srcIDs))
			for i, srcID := range srcIDs {
				if img := g.images[srcID]; img != nil {
					w, h := img.framebufferSize()
					sizes[2*i] = float32(w)
					sizes[2*i+1] = float32(h)
				}

			}
			const idx = graphics.TextureSourceSizesUniformVariableIndex
			g.uniformVars[idx].name = g.uniformVariableName(idx)
			g.uniformVars[idx].value = sizes
			g.uniformVars[idx].typ = shader.ir.Uniforms[idx]
		}
		dw, dh := destination.framebufferSize()
		{
			origin := []float32{float32(dstRegion.X) / float32(dw), float32(dstRegion.Y) / float32(dh)}
			const idx = graphics.TextureDestinationRegionOriginUniformVariableIndex
			g.uniformVars[idx].name = g.uniformVariableName(idx)
			g.uniformVars[idx].value = origin
			g.uniformVars[idx].typ = shader.ir.Uniforms[idx]
		}
		{
			size := []float32{float32(dstRegion.Width) / float32(dw), float32(dstRegion.Height) / float32(dh)}
			const idx = graphics.TextureDestinationRegionSizeUniformVariableIndex
			g.uniformVars[idx].name = g.uniformVariableName(idx)
			g.uniformVars[idx].value = size
			g.uniformVars[idx].typ = shader.ir.Uniforms[idx]
		}
		{
			voffsets := make([]float32, 2*len(offsets))
			for i, o := range offsets {
				voffsets[2*i] = o[0]
				voffsets[2*i+1] = o[1]
			}
			const idx = graphics.TextureSourceOffsetsUniformVariableIndex
			g.uniformVars[idx].name = g.uniformVariableName(idx)
			g.uniformVars[idx].value = voffsets
			g.uniformVars[idx].typ = shader.ir.Uniforms[idx]
		}
		{
			origin := []float32{float32(srcRegion.X), float32(srcRegion.Y)}
			const idx = graphics.TextureSourceRegionOriginUniformVariableIndex
			g.uniformVars[idx].name = g.uniformVariableName(idx)
			g.uniformVars[idx].value = origin
			g.uniformVars[idx].typ = shader.ir.Uniforms[idx]
		}
		{
			size := []float32{float32(srcRegion.Width), float32(srcRegion.Height)}
			const idx = graphics.TextureSourceRegionSizeUniformVariableIndex
			g.uniformVars[idx].name = g.uniformVariableName(idx)
			g.uniformVars[idx].value = size
			g.uniformVars[idx].typ = shader.ir.Uniforms[idx]
		}
		{
			const idx = graphics.ProjectionMatrixUniformVariableIndex
			g.uniformVars[idx].name = g.uniformVariableName(idx)
			g.uniformVars[idx].value = []float32{
				2 / float32(dw), 0, 0, 0,
				0, 2 / float32(dh), 0, 0,
				0, 0, 1, 0,
				-1, -1, 0, 1,
			}
			g.uniformVars[idx].typ = shader.ir.Uniforms[idx]
		}

		for i, v := range uniforms {
			const offset = graphics.PreservedUniformVariablesCount
			g.uniformVars[i+offset].name = g.uniformVariableName(i + offset)
			g.uniformVars[i+offset].value = v
			g.uniformVars[i+offset].typ = shader.ir.Uniforms[i+offset]
		}
	}

	var imgs [graphics.ShaderImageCount]textureVariable
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

	if evenOdd {
		if err := destination.ensureStencilBuffer(); err != nil {
			return err
		}
		g.context.enableStencilTest()
		g.context.beginStencilWithEvenOddRule()
		g.context.drawElements(indexLen, indexOffset*2)
		g.context.endStencilWithEvenOddRule()
	}
	g.context.drawElements(indexLen, indexOffset*2) // 2 is uint16 size in bytes
	if evenOdd {
		g.context.disableStencilTest()
	}

	return nil
}

func (g *Graphics) SetVsyncEnabled(enabled bool) {
	// Do nothing
}

func (g *Graphics) SetFullscreen(fullscreen bool) {
	// Do nothing
}

func (g *Graphics) FramebufferYDirection() graphicsdriver.YDirection {
	return graphicsdriver.Upward
}

func (g *Graphics) NeedsRestoring() bool {
	return g.context.needsRestoring()
}

func (g *Graphics) NeedsClearingScreen() bool {
	return true
}

func (g *Graphics) IsGL() bool {
	return true
}

func (g *Graphics) IsDirectX() bool {
	return false
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
