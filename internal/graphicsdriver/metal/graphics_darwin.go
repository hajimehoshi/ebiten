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
	"image"
	"math"
	"runtime"
	"sort"
	"unsafe"

	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/ca"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/mtl"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

type Graphics struct {
	view view

	colorSpace graphicsdriver.ColorSpace

	cq   mtl.CommandQueue
	cb   mtl.CommandBuffer
	rce  mtl.RenderCommandEncoder
	dsss map[stencilMode]mtl.DepthStencilState

	screenDrawable ca.MetalDrawable

	buffers       map[mtl.CommandBuffer][]mtl.Buffer
	unusedBuffers map[mtl.Buffer]struct{}

	lastDst      *Image
	lastFillRule graphicsdriver.FillRule

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
	noStencil stencilMode = iota
	incrementStencil
	invertStencil
	drawWithStencil
)

var (
	systemDefaultDevice    mtl.Device
	systemDefaultDeviceErr error
)

func init() {
	// mtl.CreateSystemDefaultDevice must be called on the main thread (#2147).
	d, err := mtl.CreateSystemDefaultDevice()
	if err != nil {
		systemDefaultDeviceErr = err
		return
	}
	systemDefaultDevice = d
}

// NewGraphics creates an implementation of graphicsdriver.Graphics for Metal.
// The returned graphics value is nil iff the error is not nil.
func NewGraphics(colorSpace graphicsdriver.ColorSpace) (graphicsdriver.Graphics, error) {
	// On old mac devices like iMac 2011, Metal is not supported (#779).
	// TODO: Is there a better way to check whether Metal is available or not?
	// It seems OK to call MTLCreateSystemDefaultDevice multiple times, so this should be fine.
	if systemDefaultDeviceErr != nil {
		return nil, fmt.Errorf("metal: mtl.CreateSystemDefaultDevice failed: %w", systemDefaultDeviceErr)
	}

	g := &Graphics{
		colorSpace: colorSpace,
	}

	if runtime.GOOS != "ios" {
		// Initializing a Metal device and a layer must be done in the main thread on macOS.
		// Note that this assumes NewGraphics is called on the main thread on desktops.
		if err := g.view.initialize(systemDefaultDevice, colorSpace); err != nil {
			return nil, err
		}
	}
	return g, nil
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
	if x > (math.MaxUint+1)/2 {
		return math.MaxUint
	}

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
		g.cb = g.cq.CommandBuffer()
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
		newBuf = g.view.getMTLDevice().NewBufferWithLength(pow2(length), resourceStorageMode)
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

func (g *Graphics) SetVertices(vertices []float32, indices []uint32) error {
	vbSize := unsafe.Sizeof(vertices[0]) * uintptr(len(vertices))
	ibSize := unsafe.Sizeof(indices[0]) * uintptr(len(indices))

	g.vb = g.availableBuffer(vbSize)
	g.vb.CopyToContents(unsafe.Pointer(&vertices[0]), vbSize)

	g.ib = g.availableBuffer(ibSize)
	g.ib.CopyToContents(unsafe.Pointer(&indices[0]), ibSize)

	return nil
}

func (g *Graphics) flushIfNeeded(present bool) {
	if g.cb == (mtl.CommandBuffer{}) && !present {
		return
	}

	g.flushRenderCommandEncoderIfNeeded()

	if present {
		// This check is necessary when skipping to render the screen (SetScreenClearedEveryFrame(false)).
		if g.screenDrawable == (ca.MetalDrawable{}) && g.cb != (mtl.CommandBuffer{}) {
			g.screenDrawable = g.view.nextDrawable()
		}
		if g.screenDrawable != (ca.MetalDrawable{}) {
			g.cb.PresentDrawable(g.screenDrawable)
		}
	}

	g.cb.Commit()

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
	t := g.view.getMTLDevice().NewTextureWithDescriptor(td)
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
	case graphicsdriver.BlendFactorSourceColor:
		return mtl.BlendFactorSourceColor
	case graphicsdriver.BlendFactorOneMinusSourceColor:
		return mtl.BlendFactorOneMinusSourceColor
	case graphicsdriver.BlendFactorSourceAlpha:
		return mtl.BlendFactorSourceAlpha
	case graphicsdriver.BlendFactorOneMinusSourceAlpha:
		return mtl.BlendFactorOneMinusSourceAlpha
	case graphicsdriver.BlendFactorDestinationColor:
		return mtl.BlendFactorDestinationColor
	case graphicsdriver.BlendFactorOneMinusDestinationColor:
		return mtl.BlendFactorOneMinusDestinationColor
	case graphicsdriver.BlendFactorDestinationAlpha:
		return mtl.BlendFactorDestinationAlpha
	case graphicsdriver.BlendFactorOneMinusDestinationAlpha:
		return mtl.BlendFactorOneMinusDestinationAlpha
	case graphicsdriver.BlendFactorSourceAlphaSaturated:
		return mtl.BlendFactorSourceAlphaSaturated
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
	case graphicsdriver.BlendOperationMin:
		return mtl.BlendOperationMin
	case graphicsdriver.BlendOperationMax:
		return mtl.BlendOperationMax
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

	if runtime.GOOS == "ios" {
		// Initializing a Metal device and a layer must be done in the render thread on iOS.
		if err := g.view.initialize(systemDefaultDevice, g.colorSpace); err != nil {
			return err
		}
	}
	if g.transparent {
		g.view.ml.SetOpaque(false)
	}

	// The stencil reference value is always 0 (default).
	g.dsss[noStencil] = g.view.getMTLDevice().NewDepthStencilStateWithDescriptor(mtl.DepthStencilDescriptor{
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
	g.dsss[incrementStencil] = g.view.getMTLDevice().NewDepthStencilStateWithDescriptor(mtl.DepthStencilDescriptor{
		BackFaceStencil: mtl.StencilDescriptor{
			StencilFailureOperation:   mtl.StencilOperationKeep,
			DepthFailureOperation:     mtl.StencilOperationKeep,
			DepthStencilPassOperation: mtl.StencilOperationDecrementWrap,
			StencilCompareFunction:    mtl.CompareFunctionAlways,
		},
		FrontFaceStencil: mtl.StencilDescriptor{
			StencilFailureOperation:   mtl.StencilOperationKeep,
			DepthFailureOperation:     mtl.StencilOperationKeep,
			DepthStencilPassOperation: mtl.StencilOperationIncrementWrap,
			StencilCompareFunction:    mtl.CompareFunctionAlways,
		},
	})
	g.dsss[invertStencil] = g.view.getMTLDevice().NewDepthStencilStateWithDescriptor(mtl.DepthStencilDescriptor{
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
	g.dsss[drawWithStencil] = g.view.getMTLDevice().NewDepthStencilStateWithDescriptor(mtl.DepthStencilDescriptor{
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

	g.cq = g.view.getMTLDevice().NewCommandQueue()
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

func (g *Graphics) draw(dst *Image, dstRegions []graphicsdriver.DstRegion, srcs [graphics.ShaderSrcImageCount]*Image, indexOffset int, shader *Shader, uniforms [][]uint32, blend graphicsdriver.Blend, fillRule graphicsdriver.FillRule) error {
	// When preparing a stencil buffer, flush the current render command encoder
	// to make sure the stencil buffer is cleared when loading.
	// TODO: What about clearing the stencil buffer by vertices?
	if g.lastDst != dst || g.lastFillRule != fillRule || fillRule != graphicsdriver.FillRuleFillAll {
		g.flushRenderCommandEncoderIfNeeded()
	}
	g.lastDst = dst
	g.lastFillRule = fillRule

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

		if fillRule != graphicsdriver.FillRuleFillAll {
			dst.ensureStencil()
			rpd.StencilAttachment.LoadAction = mtl.LoadActionClear
			rpd.StencilAttachment.StoreAction = mtl.StoreActionDontCare
			rpd.StencilAttachment.Texture = dst.stencil
		}

		if g.cb == (mtl.CommandBuffer{}) {
			g.cb = g.cq.CommandBuffer()
		}
		g.rce = g.cb.RenderCommandEncoderWithDescriptor(rpd)
	}

	w, h := dst.internalSize()
	g.rce.SetViewport(mtl.Viewport{
		OriginX: 0,
		OriginY: 0,
		Width:   float64(w),
		Height:  float64(h),
		ZNear:   -1,
		ZFar:    1,
	})
	g.rce.SetVertexBuffer(g.vb, 0, 0)

	for i, u := range uniforms {
		if u == nil {
			continue
		}
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

	var (
		noStencilRpss        mtl.RenderPipelineState
		incrementStencilRpss mtl.RenderPipelineState
		invertStencilRpss    mtl.RenderPipelineState
		drawWithStencilRpss  mtl.RenderPipelineState
	)
	switch fillRule {
	case graphicsdriver.FillRuleFillAll:
		s, err := shader.RenderPipelineState(&g.view, blend, noStencil, dst.screen)
		if err != nil {
			return err
		}
		noStencilRpss = s
	case graphicsdriver.FillRuleNonZero:
		s, err := shader.RenderPipelineState(&g.view, blend, incrementStencil, dst.screen)
		if err != nil {
			return err
		}
		incrementStencilRpss = s
	case graphicsdriver.FillRuleEvenOdd:
		s, err := shader.RenderPipelineState(&g.view, blend, invertStencil, dst.screen)
		if err != nil {
			return err
		}
		invertStencilRpss = s
	}
	if fillRule != graphicsdriver.FillRuleFillAll {
		s, err := shader.RenderPipelineState(&g.view, blend, drawWithStencil, dst.screen)
		if err != nil {
			return err
		}
		drawWithStencilRpss = s
	}

	for _, dstRegion := range dstRegions {
		g.rce.SetScissorRect(mtl.ScissorRect{
			X:      dstRegion.Region.Min.X,
			Y:      dstRegion.Region.Min.Y,
			Width:  dstRegion.Region.Dx(),
			Height: dstRegion.Region.Dy(),
		})

		switch fillRule {
		case graphicsdriver.FillRuleFillAll:
			g.rce.SetDepthStencilState(g.dsss[noStencil])
			g.rce.SetRenderPipelineState(noStencilRpss)
			g.rce.DrawIndexedPrimitives(mtl.PrimitiveTypeTriangle, dstRegion.IndexCount, mtl.IndexTypeUInt32, g.ib, indexOffset*int(unsafe.Sizeof(uint32(0))))
		case graphicsdriver.FillRuleNonZero:
			g.rce.SetDepthStencilState(g.dsss[incrementStencil])
			g.rce.SetRenderPipelineState(incrementStencilRpss)
			g.rce.DrawIndexedPrimitives(mtl.PrimitiveTypeTriangle, dstRegion.IndexCount, mtl.IndexTypeUInt32, g.ib, indexOffset*int(unsafe.Sizeof(uint32(0))))
		case graphicsdriver.FillRuleEvenOdd:
			g.rce.SetDepthStencilState(g.dsss[invertStencil])
			g.rce.SetRenderPipelineState(invertStencilRpss)
			g.rce.DrawIndexedPrimitives(mtl.PrimitiveTypeTriangle, dstRegion.IndexCount, mtl.IndexTypeUInt32, g.ib, indexOffset*int(unsafe.Sizeof(uint32(0))))
		}
		if fillRule != graphicsdriver.FillRuleFillAll {
			g.rce.SetDepthStencilState(g.dsss[drawWithStencil])
			g.rce.SetRenderPipelineState(drawWithStencilRpss)
			g.rce.DrawIndexedPrimitives(mtl.PrimitiveTypeTriangle, dstRegion.IndexCount, mtl.IndexTypeUInt32, g.ib, indexOffset*int(unsafe.Sizeof(uint32(0))))
		}

		indexOffset += dstRegion.IndexCount
	}

	return nil
}

func (g *Graphics) DrawTriangles(dstID graphicsdriver.ImageID, srcIDs [graphics.ShaderSrcImageCount]graphicsdriver.ImageID, shaderID graphicsdriver.ShaderID, dstRegions []graphicsdriver.DstRegion, indexOffset int, blend graphicsdriver.Blend, uniforms []uint32, fillRule graphicsdriver.FillRule) error {
	if shaderID == graphicsdriver.InvalidShaderID {
		return fmt.Errorf("metal: shader ID is invalid")
	}

	dst := g.images[dstID]

	if dst.screen {
		g.view.update()
	}

	var srcs [graphics.ShaderSrcImageCount]*Image
	for i, srcID := range srcIDs {
		srcs[i] = g.images[srcID]
	}

	uniformVars := make([][]uint32, len(g.shaders[shaderID].ir.Uniforms))

	// Set the additional uniform variables.
	var idx int
	for i, t := range g.shaders[shaderID].ir.Uniforms {
		if i == graphics.ProjectionMatrixUniformVariableIndex {
			// In Metal, the NDC's Y direction (upward) and the framebuffer's Y direction (downward) don't
			// match. Then, the Y direction must be inverted.
			// Invert the sign bits as float32 values.
			uniforms[idx+1] ^= 1 << 31
			uniforms[idx+5] ^= 1 << 31
			uniforms[idx+9] ^= 1 << 31
			uniforms[idx+13] ^= 1 << 31
		}

		n := t.Uint32Count()

		switch t.Main {
		case shaderir.Vec3, shaderir.IVec3:
			// float3 requires 16-byte alignment (#2463).
			v1 := make([]uint32, 4)
			copy(v1[0:3], uniforms[idx:idx+3])
			uniformVars[i] = v1
		case shaderir.Mat3:
			// float3x3 requires 16-byte alignment (#2036).
			v1 := make([]uint32, 12)
			copy(v1[0:3], uniforms[idx:idx+3])
			copy(v1[4:7], uniforms[idx+3:idx+6])
			copy(v1[8:11], uniforms[idx+6:idx+9])
			uniformVars[i] = v1
		case shaderir.Array:
			switch t.Sub[0].Main {
			case shaderir.Vec3, shaderir.IVec3:
				v1 := make([]uint32, t.Length*4)
				for j := 0; j < t.Length; j++ {
					offset0 := j * 3
					offset1 := j * 4
					copy(v1[offset1:offset1+3], uniforms[idx+offset0:idx+offset0+3])
				}
				uniformVars[i] = v1
			case shaderir.Mat3:
				v1 := make([]uint32, t.Length*12)
				for j := 0; j < t.Length; j++ {
					offset0 := j * 9
					offset1 := j * 12
					copy(v1[offset1:offset1+3], uniforms[idx+offset0:idx+offset0+3])
					copy(v1[offset1+4:offset1+7], uniforms[idx+offset0+3:idx+offset0+6])
					copy(v1[offset1+8:offset1+11], uniforms[idx+offset0+6:idx+offset0+9])
				}
				uniformVars[i] = v1
			default:
				uniformVars[i] = uniforms[idx : idx+n]
			}
		default:
			uniformVars[i] = uniforms[idx : idx+n]
		}

		idx += n
	}

	if err := g.draw(dst, dstRegions, srcs, indexOffset, g.shaders[shaderID], uniformVars, blend, fillRule); err != nil {
		return err
	}

	return nil
}

func (g *Graphics) SetVsyncEnabled(enabled bool) {
	g.view.setDisplaySyncEnabled(enabled)
}

func (g *Graphics) NeedsClearingScreen() bool {
	return false
}

func (g *Graphics) MaxImageSize() int {
	if g.maxImageSize != 0 {
		return g.maxImageSize
	}

	d := g.view.getMTLDevice()

	// supportsFamily is available as of macOS 10.15+ and iOS 13.0+.
	// https://developer.apple.com/documentation/metal/mtldevice/3143473-supportsfamily
	if d.RespondsToSelector(objc.RegisterName("supportsFamily:")) {
		// https://developer.apple.com/metal/Metal-Feature-Set-Tables.pdf
		g.maxImageSize = 8192
		switch {
		case d.SupportsFamily(mtl.GPUFamilyApple3):
			g.maxImageSize = 16384
		case d.SupportsFamily(mtl.GPUFamilyMac2):
			g.maxImageSize = 16384
		}
		return g.maxImageSize
	}

	// supportsFeatureSet is deprecated but some old macOS/iOS versions support only this (#2553).
	switch {
	case d.SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily5_v1):
		g.maxImageSize = 16384
	case d.SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily4_v1):
		g.maxImageSize = 16384
	case d.SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily3_v1):
		g.maxImageSize = 16384
	case d.SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily2_v2):
		g.maxImageSize = 8192
	case d.SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily2_v1):
		g.maxImageSize = 4096
	case d.SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily1_v2):
		g.maxImageSize = 8192
	case d.SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily1_v1):
		g.maxImageSize = 4096
	case d.SupportsFeatureSet(mtl.FeatureSet_tvOS_GPUFamily2_v1):
		g.maxImageSize = 16384
	case d.SupportsFeatureSet(mtl.FeatureSet_tvOS_GPUFamily1_v1):
		g.maxImageSize = 8192
	case d.SupportsFeatureSet(mtl.FeatureSet_macOS_GPUFamily1_v1):
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

func (i *Image) syncTexture() {
	i.graphics.flushRenderCommandEncoderIfNeeded()

	// Calling SynchronizeTexture is ignored on iOS (see mtl.m), but it looks like committing BlitCommandEncoder
	// is necessary (#1337).
	if i.graphics.cb != (mtl.CommandBuffer{}) {
		panic("metal: command buffer must be empty at syncTexture: flushIfNeeded is not called yet?")
	}

	cb := i.graphics.cq.CommandBuffer()
	bce := cb.BlitCommandEncoder()
	bce.SynchronizeTexture(i.texture, 0, 0)
	bce.EndEncoding()

	cb.Commit()
	// TODO: Are fences available here?
	cb.WaitUntilCompleted()
}

func (i *Image) ReadPixels(args []graphicsdriver.PixelsArgs) error {
	i.graphics.flushIfNeeded(false)
	i.syncTexture()

	for _, arg := range args {
		if got, want := len(arg.Pixels), 4*arg.Region.Dx()*arg.Region.Dy(); got != want {
			return fmt.Errorf("metal: len(buf) must be %d but %d at ReadPixels", want, got)
		}
		i.texture.GetBytes(&arg.Pixels[0], uintptr(4*arg.Region.Dx()), mtl.Region{
			Origin: mtl.Origin{X: arg.Region.Min.X, Y: arg.Region.Min.Y},
			Size:   mtl.Size{Width: arg.Region.Dx(), Height: arg.Region.Dy(), Depth: 1},
		}, 0)
	}
	return nil
}

func (i *Image) WritePixels(args []graphicsdriver.PixelsArgs) error {
	g := i.graphics

	g.flushRenderCommandEncoderIfNeeded()

	// Calculate the smallest texture size to include all the values in args.
	var region image.Rectangle
	for _, a := range args {
		region = region.Union(a.Region)
	}

	// Use a temporary texture to send pixels asynchronously, whichever the memory is shared (e.g., iOS) or
	// managed (e.g., macOS). A temporary texture is needed since ReplaceRegion tries to sync the pixel
	// data between CPU and GPU, and doing it on the existing texture is inefficient (#1418).
	// The texture cannot be reused until sending the pixels finishes, then create new ones for each call.
	td := mtl.TextureDescriptor{
		TextureType: mtl.TextureType2D,
		PixelFormat: mtl.PixelFormatRGBA8UNorm,
		Width:       region.Dx(),
		Height:      region.Dy(),
		StorageMode: storageMode,
		Usage:       mtl.TextureUsageShaderRead | mtl.TextureUsageRenderTarget,
	}
	t := g.view.getMTLDevice().NewTextureWithDescriptor(td)
	g.tmpTextures = append(g.tmpTextures, t)

	for _, a := range args {
		t.ReplaceRegion(mtl.Region{
			Origin: mtl.Origin{X: a.Region.Min.X - region.Min.X, Y: a.Region.Min.Y - region.Min.Y, Z: 0},
			Size:   mtl.Size{Width: a.Region.Dx(), Height: a.Region.Dy(), Depth: 1},
		}, 0, unsafe.Pointer(&a.Pixels[0]), 4*a.Region.Dx())
	}

	if g.cb == (mtl.CommandBuffer{}) {
		g.cb = i.graphics.cq.CommandBuffer()
	}
	bce := g.cb.BlitCommandEncoder()
	for _, a := range args {
		so := mtl.Origin{X: a.Region.Min.X - region.Min.X, Y: a.Region.Min.Y - region.Min.Y, Z: 0}
		ss := mtl.Size{Width: a.Region.Dx(), Height: a.Region.Dy(), Depth: 1}
		do := mtl.Origin{X: a.Region.Min.X, Y: a.Region.Min.Y, Z: 0}
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
	i.stencil = i.graphics.view.getMTLDevice().NewTextureWithDescriptor(td)
}
