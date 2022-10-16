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

package metal

import (
	"fmt"
	"math"
	"sort"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/ca"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/mtl"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

type Graphics struct {
	view view

	cq   mtl.CommandQueue
	cb   mtl.CommandBuffer
	rce  mtl.RenderCommandEncoder
	dsss map[stencilMode]mtl.DepthStencilState

	screenDrawable ca.MetalDrawable

	buffers       map[mtl.CommandBuffer][]mtl.Buffer
	unusedBuffers map[mtl.Buffer]struct{}

	lastDst         *Image
	lastStencilMode stencilMode

	vb mtl.Buffer
	ib mtl.Buffer

	images      map[graphicsdriver.ImageID]*Image
	nextImageID graphicsdriver.ImageID

	shaders      map[graphicsdriver.ShaderID]*Shader
	nextShaderID graphicsdriver.ShaderID

	transparent  bool
	maxImageSize int
	tmpTextures  []mtl.Texture

	pool cocoa.NSAutoreleasePool
}

type stencilMode int

const (
	prepareStencil stencilMode = iota
	drawWithStencil
	noStencil
)

var creatingSystemDefaultDeviceSucceeded bool

func init() {
	// mtl.CreateSystemDefaultDevice must be called on the main thread (#2147).
	_, ok := mtl.CreateSystemDefaultDevice()
	creatingSystemDefaultDeviceSucceeded = ok
}

// NewGraphics creates an implementation of graphicsdriver.Graphics for Metal.
// The returned graphics value is nil iff the error is not nil.
func NewGraphics() (graphicsdriver.Graphics, error) {
	// On old mac devices like iMac 2011, Metal is not supported (#779).
	// TODO: Is there a better way to check whether Metal is available or not?
	// It seems OK to call MTLCreateSystemDefaultDevice multiple times, so this should be fine.
	if !creatingSystemDefaultDeviceSucceeded {
		return nil, fmt.Errorf("metal: mtl.CreateSystemDefaultDevice failed")
	}

	return &Graphics{}, nil
}

func (g *Graphics) Begin() error {
	// NSAutoreleasePool is required to release drawable correctly (#847).
	// https://developer.apple.com/library/archive/documentation/3DDrawing/Conceptual/MTLBestPracticesGuide/Drawables.html
	g.pool = cocoa.NSAutoreleasePool_new()
	return nil
}

func (g *Graphics) End(present bool) error {
	g.flushIfNeeded(present)
	g.screenDrawable = ca.MetalDrawable{}
	g.pool.Release()
	g.pool.ID = 0
	return nil
}

func (g *Graphics) SetWindow(window uintptr) {
	// Note that [NSApp mainWindow] returns nil when the window is borderless.
	// Then the window is needed to be given explicitly.
	g.view.setWindow(window)
}

func (g *Graphics) SetUIView(uiview uintptr) {
	// TODO: Should this be called on the main thread?
	g.view.setUIView(uiview)
}

func pow2(x uintptr) uintptr {
	var p2 uintptr = 1
	for p2 < x {
		p2 *= 2
	}
	return p2
}

func (g *Graphics) gcBuffers() {
	for cb, bs := range g.buffers {
		// If the command buffer still lives, the buffer must not be updated.
		// TODO: Handle an error?
		if cb.Status() != mtl.CommandBufferStatusCompleted {
			continue
		}

		for _, b := range bs {
			if g.unusedBuffers == nil {
				g.unusedBuffers = map[mtl.Buffer]struct{}{}
			}
			g.unusedBuffers[b] = struct{}{}
		}
		delete(g.buffers, cb)
		cb.Release()
	}

	const maxUnusedBuffers = 10
	if len(g.unusedBuffers) > maxUnusedBuffers {
		bufs := make([]mtl.Buffer, 0, len(g.unusedBuffers))
		for b := range g.unusedBuffers {
			bufs = append(bufs, b)
		}
		sort.Slice(bufs, func(a, b int) bool {
			return bufs[a].Length() > bufs[b].Length()
		})
		for _, b := range bufs[maxUnusedBuffers:] {
			delete(g.unusedBuffers, b)
			b.Release()
		}
	}
}

func (g *Graphics) availableBuffer(length uintptr) mtl.Buffer {
	if g.cb == (mtl.CommandBuffer{}) {
		g.cb = g.cq.MakeCommandBuffer()
	}

	var newBuf mtl.Buffer
	for b := range g.unusedBuffers {
		if b.Length() >= length {
			newBuf = b
			delete(g.unusedBuffers, b)
			break
		}
	}

	if newBuf == (mtl.Buffer{}) {
		newBuf = g.view.getMTLDevice().MakeBufferWithLength(pow2(length), resourceStorageMode)
	}

	if g.buffers == nil {
		g.buffers = map[mtl.CommandBuffer][]mtl.Buffer{}
	}
	if _, ok := g.buffers[g.cb]; !ok {
		g.cb.Retain()
	}
	g.buffers[g.cb] = append(g.buffers[g.cb], newBuf)
	return newBuf
}

func (g *Graphics) SetVertices(vertices []float32, indices []uint16) error {
	vbSize := unsafe.Sizeof(vertices[0]) * uintptr(len(vertices))
	ibSize := unsafe.Sizeof(indices[0]) * uintptr(len(indices))

	g.vb = g.availableBuffer(vbSize)
	g.vb.CopyToContents(unsafe.Pointer(&vertices[0]), vbSize)

	g.ib = g.availableBuffer(ibSize)
	g.ib.CopyToContents(unsafe.Pointer(&indices[0]), ibSize)

	return nil
}

func (g *Graphics) flushIfNeeded(present bool) {
	if g.cb == (mtl.CommandBuffer{}) {
		return
	}
	g.flushRenderCommandEncoderIfNeeded()

	if !g.view.presentsWithTransaction() && present && g.screenDrawable != (ca.MetalDrawable{}) {
		g.cb.PresentDrawable(g.screenDrawable)
	}
	g.cb.Commit()
	if g.view.presentsWithTransaction() && present && g.screenDrawable != (ca.MetalDrawable{}) {
		g.cb.WaitUntilScheduled()
		g.screenDrawable.Present()
	}

	for _, t := range g.tmpTextures {
		t.Release()
	}
	g.tmpTextures = g.tmpTextures[:0]

	g.cb = mtl.CommandBuffer{}
}

func (g *Graphics) checkSize(width, height int) {
	if width < 1 {
		panic(fmt.Sprintf("metal: width (%d) must be equal or more than %d", width, 1))
	}
	if height < 1 {
		panic(fmt.Sprintf("metal: height (%d) must be equal or more than %d", height, 1))
	}
	m := g.MaxImageSize()
	if width > m {
		panic(fmt.Sprintf("metal: width (%d) must be less than or equal to %d", width, m))
	}
	if height > m {
		panic(fmt.Sprintf("metal: height (%d) must be less than or equal to %d", height, m))
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
	g.checkSize(width, height)
	td := mtl.TextureDescriptor{
		TextureType: mtl.TextureType2D,
		PixelFormat: mtl.PixelFormatRGBA8UNorm,
		Width:       graphics.InternalImageSize(width),
		Height:      graphics.InternalImageSize(height),
		StorageMode: storageMode,
		Usage:       mtl.TextureUsageShaderRead | mtl.TextureUsageRenderTarget,
	}
	t := g.view.getMTLDevice().MakeTexture(td)
	i := &Image{
		id:       g.genNextImageID(),
		graphics: g,
		width:    width,
		height:   height,
		texture:  t,
	}
	g.addImage(i)
	return i, nil
}

func (g *Graphics) NewScreenFramebufferImage(width, height int) (graphicsdriver.Image, error) {
	g.view.setDrawableSize(width, height)
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
		panic(fmt.Sprintf("metal: image ID %d was already registered", img.id))
	}
	g.images[img.id] = img
}

func (g *Graphics) removeImage(img *Image) {
	delete(g.images, img.id)
}

func (g *Graphics) SetTransparent(transparent bool) {
	g.transparent = transparent
}

func blendFactorToMetalBlendFactor(c graphicsdriver.BlendFactor) mtl.BlendFactor {
	switch c {
	case graphicsdriver.BlendFactorZero:
		return mtl.BlendFactorZero
	case graphicsdriver.BlendFactorOne:
		return mtl.BlendFactorOne
	case graphicsdriver.BlendFactorSourceAlpha:
		return mtl.BlendFactorSourceAlpha
	case graphicsdriver.BlendFactorDestinationAlpha:
		return mtl.BlendFactorDestinationAlpha
	case graphicsdriver.BlendFactorOneMinusSourceAlpha:
		return mtl.BlendFactorOneMinusSourceAlpha
	case graphicsdriver.BlendFactorOneMinusDestinationAlpha:
		return mtl.BlendFactorOneMinusDestinationAlpha
	case graphicsdriver.BlendFactorDestinationColor:
		return mtl.BlendFactorDestinationColor
	default:
		panic(fmt.Sprintf("metal: invalid blend factor: %d", c))
	}
}

func blendOperationToMetalBlendOperation(o graphicsdriver.BlendOperation) mtl.BlendOperation {
	switch o {
	case graphicsdriver.BlendOperationAdd:
		return mtl.BlendOperationAdd
	case graphicsdriver.BlendOperationSubtract:
		return mtl.BlendOperationSubtract
	case graphicsdriver.BlendOperationReverseSubtract:
		return mtl.BlendOperationReverseSubtract
	default:
		panic(fmt.Sprintf("metal: invalid blend operation: %d", o))
	}
}

func (g *Graphics) Initialize() error {
	// Creating *State objects are expensive and reuse them whenever possible.
	// See https://developer.apple.com/library/archive/documentation/Miscellaneous/Conceptual/MetalProgrammingGuide/Cmd-Submiss/Cmd-Submiss.html

	for _, dss := range g.dsss {
		dss.Release()
	}
	if g.dsss == nil {
		g.dsss = map[stencilMode]mtl.DepthStencilState{}
	}

	if err := g.view.initialize(); err != nil {
		return err
	}
	if g.transparent {
		g.view.ml.SetOpaque(false)
	}

	// The stencil reference value is always 0 (default).
	g.dsss[prepareStencil] = g.view.getMTLDevice().MakeDepthStencilState(mtl.DepthStencilDescriptor{
		BackFaceStencil: mtl.StencilDescriptor{
			StencilFailureOperation:   mtl.StencilOperationKeep,
			DepthFailureOperation:     mtl.StencilOperationKeep,
			DepthStencilPassOperation: mtl.StencilOperationInvert,
			StencilCompareFunction:    mtl.CompareFunctionAlways,
		},
		FrontFaceStencil: mtl.StencilDescriptor{
			StencilFailureOperation:   mtl.StencilOperationKeep,
			DepthFailureOperation:     mtl.StencilOperationKeep,
			DepthStencilPassOperation: mtl.StencilOperationInvert,
			StencilCompareFunction:    mtl.CompareFunctionAlways,
		},
	})
	g.dsss[drawWithStencil] = g.view.getMTLDevice().MakeDepthStencilState(mtl.DepthStencilDescriptor{
		BackFaceStencil: mtl.StencilDescriptor{
			StencilFailureOperation:   mtl.StencilOperationKeep,
			DepthFailureOperation:     mtl.StencilOperationKeep,
			DepthStencilPassOperation: mtl.StencilOperationKeep,
			StencilCompareFunction:    mtl.CompareFunctionNotEqual,
		},
		FrontFaceStencil: mtl.StencilDescriptor{
			StencilFailureOperation:   mtl.StencilOperationKeep,
			DepthFailureOperation:     mtl.StencilOperationKeep,
			DepthStencilPassOperation: mtl.StencilOperationKeep,
			StencilCompareFunction:    mtl.CompareFunctionNotEqual,
		},
	})
	g.dsss[noStencil] = g.view.getMTLDevice().MakeDepthStencilState(mtl.DepthStencilDescriptor{
		BackFaceStencil: mtl.StencilDescriptor{
			StencilFailureOperation:   mtl.StencilOperationKeep,
			DepthFailureOperation:     mtl.StencilOperationKeep,
			DepthStencilPassOperation: mtl.StencilOperationKeep,
			StencilCompareFunction:    mtl.CompareFunctionAlways,
		},
		FrontFaceStencil: mtl.StencilDescriptor{
			StencilFailureOperation:   mtl.StencilOperationKeep,
			DepthFailureOperation:     mtl.StencilOperationKeep,
			DepthStencilPassOperation: mtl.StencilOperationKeep,
			StencilCompareFunction:    mtl.CompareFunctionAlways,
		},
	})

	g.cq = g.view.getMTLDevice().MakeCommandQueue()
	return nil
}

func (g *Graphics) flushRenderCommandEncoderIfNeeded() {
	if g.rce == (mtl.RenderCommandEncoder{}) {
		return
	}
	g.rce.EndEncoding()
	g.rce = mtl.RenderCommandEncoder{}
	g.lastDst = nil
}

func (g *Graphics) draw(rps mtl.RenderPipelineState, dst *Image, dstRegion graphicsdriver.Region, srcs [graphics.ShaderImageCount]*Image, indexLen int, indexOffset int, uniforms [][]float32, stencilMode stencilMode) error {
	// When prepareing a stencil buffer, flush the current render command encoder
	// to make sure the stencil buffer is cleared when loading.
	// TODO: What about clearing the stencil buffer by vertices?
	if g.lastDst != dst || (g.lastStencilMode == noStencil) != (stencilMode == noStencil) || stencilMode == prepareStencil {
		g.flushRenderCommandEncoderIfNeeded()
	}
	g.lastDst = dst
	g.lastStencilMode = stencilMode

	if g.rce == (mtl.RenderCommandEncoder{}) {
		rpd := mtl.RenderPassDescriptor{}
		// Even though the destination pixels are not used, mtl.LoadActionDontCare might cause glitches
		// (#1019). Always using mtl.LoadActionLoad is safe.
		if dst.screen {
			rpd.ColorAttachments[0].LoadAction = mtl.LoadActionClear
		} else {
			rpd.ColorAttachments[0].LoadAction = mtl.LoadActionLoad
		}

		// The store action should always be 'store' even for the screen (#1700).
		rpd.ColorAttachments[0].StoreAction = mtl.StoreActionStore

		t := dst.mtlTexture()
		if t == (mtl.Texture{}) {
			return nil
		}
		rpd.ColorAttachments[0].Texture = t
		rpd.ColorAttachments[0].ClearColor = mtl.ClearColor{}

		if stencilMode == prepareStencil {
			dst.ensureStencil()
			rpd.StencilAttachment.LoadAction = mtl.LoadActionClear
			rpd.StencilAttachment.StoreAction = mtl.StoreActionDontCare
			rpd.StencilAttachment.Texture = dst.stencil
		}

		if g.cb == (mtl.CommandBuffer{}) {
			g.cb = g.cq.MakeCommandBuffer()
		}
		g.rce = g.cb.MakeRenderCommandEncoder(rpd)
	}

	g.rce.SetRenderPipelineState(rps)

	w, h := dst.internalSize()
	g.rce.SetViewport(mtl.Viewport{
		OriginX: 0,
		OriginY: 0,
		Width:   float64(w),
		Height:  float64(h),
		ZNear:   -1,
		ZFar:    1,
	})
	g.rce.SetScissorRect(mtl.ScissorRect{
		X:      int(dstRegion.X),
		Y:      int(dstRegion.Y),
		Width:  int(dstRegion.Width),
		Height: int(dstRegion.Height),
	})
	g.rce.SetVertexBuffer(g.vb, 0, 0)

	for i, u := range uniforms {
		g.rce.SetVertexBytes(unsafe.Pointer(&u[0]), unsafe.Sizeof(u[0])*uintptr(len(u)), i+1)
		g.rce.SetFragmentBytes(unsafe.Pointer(&u[0]), unsafe.Sizeof(u[0])*uintptr(len(u)), i+1)
	}

	for i, src := range srcs {
		if src != nil {
			g.rce.SetFragmentTexture(src.texture, i)
		} else {
			g.rce.SetFragmentTexture(mtl.Texture{}, i)
		}
	}

	g.rce.SetDepthStencilState(g.dsss[stencilMode])

	g.rce.DrawIndexedPrimitives(mtl.PrimitiveTypeTriangle, indexLen, mtl.IndexTypeUInt16, g.ib, indexOffset*2)

	return nil
}

func (g *Graphics) DrawTriangles(dstID graphicsdriver.ImageID, srcIDs [graphics.ShaderImageCount]graphicsdriver.ImageID, offsets [graphics.ShaderImageCount - 1][2]float32, shaderID graphicsdriver.ShaderID, indexLen int, indexOffset int, blend graphicsdriver.Blend, dstRegion, srcRegion graphicsdriver.Region, uniforms [][]float32, evenOdd bool) error {
	if shaderID == graphicsdriver.InvalidShaderID {
		return fmt.Errorf("metal: shader ID is invalid")
	}

	dst := g.images[dstID]

	if dst.screen {
		g.view.update()
	}

	var srcs [graphics.ShaderImageCount]*Image
	for i, srcID := range srcIDs {
		srcs[i] = g.images[srcID]
	}

	uniformVars := make([][]float32, graphics.PreservedUniformVariablesCount+len(uniforms))

	// Set the destination texture size.
	dw, dh := dst.internalSize()
	uniformVars[graphics.TextureDestinationSizeUniformVariableIndex] = []float32{float32(dw), float32(dh)}

	// Set the source texture sizes.
	usizes := make([]float32, 2*len(srcs))
	for i, src := range srcs {
		if src != nil {
			w, h := src.internalSize()
			usizes[2*i] = float32(w)
			usizes[2*i+1] = float32(h)
		}
	}
	uniformVars[graphics.TextureSourceSizesUniformVariableIndex] = usizes

	// Set the destination region's origin.
	udorigin := []float32{float32(dstRegion.X) / float32(dw), float32(dstRegion.Y) / float32(dh)}
	uniformVars[graphics.TextureDestinationRegionOriginUniformVariableIndex] = udorigin

	// Set the destination region's size.
	udsize := []float32{float32(dstRegion.Width) / float32(dw), float32(dstRegion.Height) / float32(dh)}
	uniformVars[graphics.TextureDestinationRegionSizeUniformVariableIndex] = udsize

	// Set the source offsets.
	uoffsets := make([]float32, 2*len(offsets))
	for i, offset := range offsets {
		uoffsets[2*i] = offset[0]
		uoffsets[2*i+1] = offset[1]
	}
	uniformVars[graphics.TextureSourceOffsetsUniformVariableIndex] = uoffsets

	// Set the source region's origin of texture0.
	usorigin := []float32{float32(srcRegion.X), float32(srcRegion.Y)}
	uniformVars[graphics.TextureSourceRegionOriginUniformVariableIndex] = usorigin

	// Set the source region's size of texture0.
	ussize := []float32{float32(srcRegion.Width), float32(srcRegion.Height)}
	uniformVars[graphics.TextureSourceRegionSizeUniformVariableIndex] = ussize

	uniformVars[graphics.ProjectionMatrixUniformVariableIndex] = []float32{
		2 / float32(dw), 0, 0, 0,
		0, -2 / float32(dh), 0, 0,
		0, 0, 1, 0,
		-1, 1, 0, 1,
	}

	// Set the additional uniform variables.
	for i, v := range uniforms {
		const offset = graphics.PreservedUniformVariablesCount
		t := g.shaders[shaderID].ir.Uniforms[offset+i]
		switch t.Main {
		case shaderir.Mat3:
			// float3x3 requires 16-byte alignment (#2036).
			v1 := make([]float32, 12)
			copy(v1[0:3], v[0:3])
			copy(v1[4:7], v[3:6])
			copy(v1[8:11], v[6:9])
			uniformVars[offset+i] = v1
		case shaderir.Array:
			switch t.Sub[0].Main {
			case shaderir.Mat3:
				v1 := make([]float32, t.Length*12)
				for j := 0; j < t.Length; j++ {
					offset0 := j * 9
					offset1 := j * 12
					copy(v1[offset1:offset1+3], v[offset0:offset0+3])
					copy(v1[offset1+4:offset1+7], v[offset0+3:offset0+6])
					copy(v1[offset1+8:offset1+11], v[offset0+6:offset0+9])
				}
				uniformVars[offset+i] = v1
			default:
				uniformVars[offset+i] = v
			}
		default:
			uniformVars[offset+i] = v
		}
	}

	if evenOdd {
		prepareStencilRpss, err := g.shaders[shaderID].RenderPipelineState(&g.view, blend, prepareStencil, dst.screen)
		if err != nil {
			return err
		}
		if err := g.draw(prepareStencilRpss, dst, dstRegion, srcs, indexLen, indexOffset, uniformVars, prepareStencil); err != nil {
			return err
		}
		drawWithStencilRpss, err := g.shaders[shaderID].RenderPipelineState(&g.view, blend, drawWithStencil, dst.screen)
		if err != nil {
			return err
		}
		if err := g.draw(drawWithStencilRpss, dst, dstRegion, srcs, indexLen, indexOffset, uniformVars, drawWithStencil); err != nil {
			return err
		}
	} else {
		rpss, err := g.shaders[shaderID].RenderPipelineState(&g.view, blend, noStencil, dst.screen)
		if err != nil {
			return err
		}
		if err := g.draw(rpss, dst, dstRegion, srcs, indexLen, indexOffset, uniformVars, noStencil); err != nil {
			return err
		}
	}

	return nil
}

func (g *Graphics) SetVsyncEnabled(enabled bool) {
	g.view.setDisplaySyncEnabled(enabled)
}

func (g *Graphics) SetFullscreen(fullscreen bool) {
	g.view.setFullscreen(fullscreen)
}

func (g *Graphics) NeedsRestoring() bool {
	return false
}

func (g *Graphics) NeedsClearingScreen() bool {
	return false
}

func (g *Graphics) IsGL() bool {
	return false
}

func (g *Graphics) IsDirectX() bool {
	return false
}

func (g *Graphics) MaxImageSize() int {
	if g.maxImageSize != 0 {
		return g.maxImageSize
	}

	g.maxImageSize = 4096
	// https://developer.apple.com/metal/Metal-Feature-Set-Tables.pdf
	switch {
	case g.view.getMTLDevice().SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily5_v1):
		g.maxImageSize = 16384
	case g.view.getMTLDevice().SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily4_v1):
		g.maxImageSize = 16384
	case g.view.getMTLDevice().SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily3_v1):
		g.maxImageSize = 16384
	case g.view.getMTLDevice().SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily2_v2):
		g.maxImageSize = 8192
	case g.view.getMTLDevice().SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily2_v1):
		g.maxImageSize = 4096
	case g.view.getMTLDevice().SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily1_v2):
		g.maxImageSize = 8192
	case g.view.getMTLDevice().SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily1_v1):
		g.maxImageSize = 4096
	case g.view.getMTLDevice().SupportsFeatureSet(mtl.FeatureSet_tvOS_GPUFamily2_v1):
		g.maxImageSize = 16384
	case g.view.getMTLDevice().SupportsFeatureSet(mtl.FeatureSet_tvOS_GPUFamily1_v1):
		g.maxImageSize = 8192
	case g.view.getMTLDevice().SupportsFeatureSet(mtl.FeatureSet_macOS_GPUFamily1_v1):
		g.maxImageSize = 16384
	default:
		panic("metal: there is no supported feature set")
	}
	return g.maxImageSize
}

func (g *Graphics) NewShader(program *shaderir.Program) (graphicsdriver.Shader, error) {
	s, err := newShader(g.view.getMTLDevice(), g.genNextShaderID(), program)
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
		panic(fmt.Sprintf("metal: shader ID %d was already registered", shader.id))
	}
	g.shaders[shader.id] = shader
}

func (g *Graphics) removeShader(shader *Shader) {
	delete(g.shaders, shader.id)
}

type Image struct {
	id       graphicsdriver.ImageID
	graphics *Graphics
	width    int
	height   int
	screen   bool
	texture  mtl.Texture
	stencil  mtl.Texture
}

func (i *Image) ID() graphicsdriver.ImageID {
	return i.id
}

func (i *Image) internalSize() (int, int) {
	if i.screen {
		return i.width, i.height
	}
	return graphics.InternalImageSize(i.width), graphics.InternalImageSize(i.height)
}

func (i *Image) Dispose() {
	if i.stencil != (mtl.Texture{}) {
		i.stencil.Release()
		i.stencil = mtl.Texture{}
	}
	if i.texture != (mtl.Texture{}) {
		i.texture.Release()
		i.texture = mtl.Texture{}
	}
	i.graphics.removeImage(i)
}

func (i *Image) IsInvalidated() bool {
	// TODO: Does Metal cause context lost?
	// https://developer.apple.com/documentation/metal/mtlresource/1515898-setpurgeablestate
	// https://developer.apple.com/documentation/metal/mtldevicenotificationhandler
	return false
}

func (i *Image) syncTexture() {
	i.graphics.flushRenderCommandEncoderIfNeeded()

	// Calling SynchronizeTexture is ignored on iOS (see mtl.m), but it looks like committing BlitCommandEncoder
	// is necessary (#1337).
	if i.graphics.cb != (mtl.CommandBuffer{}) {
		panic("metal: command buffer must be empty at syncTexture: flushIfNeeded is not called yet?")
	}

	cb := i.graphics.cq.MakeCommandBuffer()
	bce := cb.MakeBlitCommandEncoder()
	bce.SynchronizeTexture(i.texture, 0, 0)
	bce.EndEncoding()

	cb.Commit()
	// TODO: Are fences available here?
	cb.WaitUntilCompleted()
}

func (i *Image) ReadPixels(buf []byte, x, y, width, height int) error {
	if got, want := len(buf), 4*width*height; got != want {
		return fmt.Errorf("metal: len(buf) must be %d but %d at ReadPixels", want, got)
	}

	i.graphics.flushIfNeeded(false)
	i.syncTexture()

	i.texture.GetBytes(&buf[0], uintptr(4*width), mtl.Region{
		Origin: mtl.Origin{X: x, Y: y},
		Size:   mtl.Size{Width: width, Height: height, Depth: 1},
	}, 0)
	return nil
}

func (i *Image) WritePixels(args []*graphicsdriver.WritePixelsArgs) error {
	g := i.graphics

	g.flushRenderCommandEncoderIfNeeded()

	// Calculate the smallest texture size to include all the values in args.
	minX := math.MaxInt32
	minY := math.MaxInt32
	maxX := 0
	maxY := 0
	for _, a := range args {
		if minX > a.X {
			minX = a.X
		}
		if maxX < a.X+a.Width {
			maxX = a.X + a.Width
		}
		if minY > a.Y {
			minY = a.Y
		}
		if maxY < a.Y+a.Height {
			maxY = a.Y + a.Height
		}
	}
	w := maxX - minX
	h := maxY - minY

	// Use a temporary texture to send pixels asynchrounsly, whichever the memory is shared (e.g., iOS) or
	// managed (e.g., macOS). A temporary texture is needed since ReplaceRegion tries to sync the pixel
	// data between CPU and GPU, and doing it on the existing texture is inefficient (#1418).
	// The texture cannot be reused until sending the pixels finishes, then create new ones for each call.
	td := mtl.TextureDescriptor{
		TextureType: mtl.TextureType2D,
		PixelFormat: mtl.PixelFormatRGBA8UNorm,
		Width:       w,
		Height:      h,
		StorageMode: storageMode,
		Usage:       mtl.TextureUsageShaderRead | mtl.TextureUsageRenderTarget,
	}
	t := g.view.getMTLDevice().MakeTexture(td)
	g.tmpTextures = append(g.tmpTextures, t)

	for _, a := range args {
		t.ReplaceRegion(mtl.Region{
			Origin: mtl.Origin{X: a.X - minX, Y: a.Y - minY, Z: 0},
			Size:   mtl.Size{Width: a.Width, Height: a.Height, Depth: 1},
		}, 0, unsafe.Pointer(&a.Pixels[0]), 4*a.Width)
	}

	if g.cb == (mtl.CommandBuffer{}) {
		g.cb = i.graphics.cq.MakeCommandBuffer()
	}
	bce := g.cb.MakeBlitCommandEncoder()
	for _, a := range args {
		so := mtl.Origin{X: a.X - minX, Y: a.Y - minY, Z: 0}
		ss := mtl.Size{Width: a.Width, Height: a.Height, Depth: 1}
		do := mtl.Origin{X: a.X, Y: a.Y, Z: 0}
		bce.CopyFromTexture(t, 0, 0, so, ss, i.texture, 0, 0, do)
	}
	bce.EndEncoding()

	return nil
}

func (i *Image) mtlTexture() mtl.Texture {
	if i.screen {
		g := i.graphics
		if g.screenDrawable == (ca.MetalDrawable{}) {
			drawable := g.view.nextDrawable()
			if drawable == (ca.MetalDrawable{}) {
				return mtl.Texture{}
			}
			g.screenDrawable = drawable
			// After nextDrawable, it is expected some command buffers are completed.
			g.gcBuffers()
		}
		return g.screenDrawable.Texture()
	}
	return i.texture
}

func (i *Image) ensureStencil() {
	if i.stencil != (mtl.Texture{}) {
		return
	}

	td := mtl.TextureDescriptor{
		TextureType: mtl.TextureType2D,
		PixelFormat: mtl.PixelFormatStencil8,
		Width:       graphics.InternalImageSize(i.width),
		Height:      graphics.InternalImageSize(i.height),
		StorageMode: mtl.StorageModePrivate,
		Usage:       mtl.TextureUsageRenderTarget,
	}
	i.stencil = i.graphics.view.getMTLDevice().MakeTexture(td)
}
