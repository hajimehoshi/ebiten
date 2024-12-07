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

package directx

import (
	"fmt"
	"math"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir/hlsl"
)

var inputElementDescsForDX11 []_D3D11_INPUT_ELEMENT_DESC

func init() {
	inputElementDescsForDX11 = []_D3D11_INPUT_ELEMENT_DESC{
		{
			SemanticName:         &([]byte("POSITION\000"))[0],
			SemanticIndex:        0,
			Format:               _DXGI_FORMAT_R32G32_FLOAT,
			InputSlot:            0,
			AlignedByteOffset:    _D3D11_APPEND_ALIGNED_ELEMENT,
			InputSlotClass:       _D3D11_INPUT_PER_VERTEX_DATA,
			InstanceDataStepRate: 0,
		},
		{
			SemanticName:         &([]byte("TEXCOORD\000"))[0],
			SemanticIndex:        0,
			Format:               _DXGI_FORMAT_R32G32_FLOAT,
			InputSlot:            0,
			AlignedByteOffset:    _D3D11_APPEND_ALIGNED_ELEMENT,
			InputSlotClass:       _D3D11_INPUT_PER_VERTEX_DATA,
			InstanceDataStepRate: 0,
		},
		{
			SemanticName:         &([]byte("COLOR\000"))[0],
			SemanticIndex:        0,
			Format:               _DXGI_FORMAT_R32G32B32A32_FLOAT,
			InputSlot:            0,
			AlignedByteOffset:    _D3D11_APPEND_ALIGNED_ELEMENT,
			InputSlotClass:       _D3D11_INPUT_PER_VERTEX_DATA,
			InstanceDataStepRate: 0,
		},
	}
	diff := graphics.VertexFloatCount - 8
	if diff == 0 {
		return
	}
	if diff%4 != 0 {
		panic("directx: unexpected attribute layout")
	}
	for i := 0; i < diff/4; i++ {
		inputElementDescsForDX11 = append(inputElementDescsForDX11, _D3D11_INPUT_ELEMENT_DESC{
			SemanticName:         &([]byte("COLOR\000"))[0],
			SemanticIndex:        uint32(i) + 1,
			Format:               _DXGI_FORMAT_R32G32B32A32_FLOAT,
			InputSlot:            0,
			AlignedByteOffset:    _D3D11_APPEND_ALIGNED_ELEMENT,
			InputSlotClass:       _D3D11_INPUT_PER_VERTEX_DATA,
			InstanceDataStepRate: 0,
		})
	}
}

func blendFactorToBlend11(f graphicsdriver.BlendFactor, alpha bool) _D3D11_BLEND {
	switch f {
	case graphicsdriver.BlendFactorZero:
		return _D3D11_BLEND_ZERO
	case graphicsdriver.BlendFactorOne:
		return _D3D11_BLEND_ONE
	case graphicsdriver.BlendFactorSourceColor:
		if alpha {
			return _D3D11_BLEND_SRC_ALPHA
		}
		return _D3D11_BLEND_SRC_COLOR
	case graphicsdriver.BlendFactorOneMinusSourceColor:
		if alpha {
			return _D3D11_BLEND_INV_SRC_ALPHA
		}
		return _D3D11_BLEND_INV_SRC_COLOR
	case graphicsdriver.BlendFactorSourceAlpha:
		return _D3D11_BLEND_SRC_ALPHA
	case graphicsdriver.BlendFactorOneMinusSourceAlpha:
		return _D3D11_BLEND_INV_SRC_ALPHA
	case graphicsdriver.BlendFactorDestinationColor:
		if alpha {
			return _D3D11_BLEND_DEST_ALPHA
		}
		return _D3D11_BLEND_DEST_COLOR
	case graphicsdriver.BlendFactorOneMinusDestinationColor:
		if alpha {
			return _D3D11_BLEND_INV_DEST_ALPHA
		}
		return _D3D11_BLEND_INV_DEST_COLOR
	case graphicsdriver.BlendFactorDestinationAlpha:
		return _D3D11_BLEND_DEST_ALPHA
	case graphicsdriver.BlendFactorOneMinusDestinationAlpha:
		return _D3D11_BLEND_INV_DEST_ALPHA
	case graphicsdriver.BlendFactorSourceAlphaSaturated:
		return _D3D11_BLEND_SRC_ALPHA_SAT
	default:
		panic(fmt.Sprintf("directx: invalid blend factor: %d", f))
	}
}

func blendOperationToBlendOp11(o graphicsdriver.BlendOperation) _D3D11_BLEND_OP {
	switch o {
	case graphicsdriver.BlendOperationAdd:
		return _D3D11_BLEND_OP_ADD
	case graphicsdriver.BlendOperationSubtract:
		return _D3D11_BLEND_OP_SUBTRACT
	case graphicsdriver.BlendOperationReverseSubtract:
		return _D3D11_BLEND_OP_REV_SUBTRACT
	case graphicsdriver.BlendOperationMin:
		return _D3D11_BLEND_OP_MIN
	case graphicsdriver.BlendOperationMax:
		return _D3D11_BLEND_OP_MAX
	default:
		panic(fmt.Sprintf("directx: invalid blend operation: %d", o))
	}
}

type blendStateKey struct {
	blend     graphicsdriver.Blend
	writeMask uint8
}

type graphics11 struct {
	graphicsInfra *graphicsInfra

	featureLevel _D3D_FEATURE_LEVEL

	device        *_ID3D11Device
	deviceContext *_ID3D11DeviceContext

	images      map[graphicsdriver.ImageID]*image11
	screenImage *image11
	nextImageID graphicsdriver.ImageID

	shaders      map[graphicsdriver.ShaderID]*shader11
	nextShaderID graphicsdriver.ShaderID

	vertexBuffer            *_ID3D11Buffer
	vertexBufferSizeInBytes uint32

	indexBuffer            *_ID3D11Buffer
	indexBufferSizeInBytes uint32

	rasterizerState    *_ID3D11RasterizerState
	samplerState       *_ID3D11SamplerState
	blendStates        map[blendStateKey]*_ID3D11BlendState
	depthStencilStates map[stencilMode]*_ID3D11DepthStencilState

	vsyncEnabled bool
	window       windows.HWND

	newScreenWidth  int
	newScreenHeight int
}

func newGraphics11(useWARP bool, useDebugLayer bool) (gr11 *graphics11, ferr error) {
	g := &graphics11{
		vsyncEnabled: true,
	}

	driverType := _D3D_DRIVER_TYPE_HARDWARE
	if useWARP {
		driverType = _D3D_DRIVER_TYPE_WARP
	}

	var flags _D3D11_CREATE_DEVICE_FLAG
	if useDebugLayer {
		flags |= _D3D11_CREATE_DEVICE_DEBUG
	}

	// Avoid _D3D_FEATURE_LEVEL_11_1 as DirectX 11.0 doesn't recognize this.
	// Avoid _D3D_FEATURE_LEVEL_9_* for some shaders features (#1431).
	featureLevels := []_D3D_FEATURE_LEVEL{
		_D3D_FEATURE_LEVEL_11_0,
		_D3D_FEATURE_LEVEL_10_1,
		_D3D_FEATURE_LEVEL_10_0,
	}

	// Apparently, adapter must be nil if the driver type is not unknown. This is not documented explicitly.
	// https://learn.microsoft.com/en-us/windows/win32/api/d3d11/nf-d3d11-d3d11createdevice
	d, fl, ctx, err := _D3D11CreateDevice(nil, driverType, 0, uint32(flags), featureLevels, true, true)
	if err != nil {
		return nil, err
	}
	g.device = (*_ID3D11Device)(d)
	g.featureLevel = fl
	g.deviceContext = (*_ID3D11DeviceContext)(ctx)

	// Get IDXGIFactory from the current device and use it, instead of CreateDXGIFactory.
	// Or, MakeWindowAssociation doesn't work well (#2661).
	dd, err := g.device.QueryInterface(&_IID_IDXGIDevice)
	if err != nil {
		return nil, err
	}
	dxgiDevice := (*_IDXGIDevice)(dd)
	defer dxgiDevice.Release()

	dxgiAdapter, err := dxgiDevice.GetAdapter()
	if err != nil {
		return nil, err
	}
	defer dxgiAdapter.Release()

	df, err := dxgiAdapter.GetParent(&_IID_IDXGIFactory)
	if err != nil {
		return nil, err
	}
	dxgiFactory := (*_IDXGIFactory)(df)

	gi, err := newGraphicsInfra(dxgiFactory)
	if err != nil {
		return nil, err
	}
	g.graphicsInfra = gi
	defer func() {
		if ferr != nil {
			g.graphicsInfra.release()
			g.graphicsInfra = nil
		}
	}()

	g.deviceContext.IASetPrimitiveTopology(_D3D11_PRIMITIVE_TOPOLOGY_TRIANGLELIST)

	// Set the rasterizer state.
	if g.rasterizerState == nil {
		rs, err := g.device.CreateRasterizerState(&_D3D11_RASTERIZER_DESC{
			FillMode:              _D3D11_FILL_SOLID,
			CullMode:              _D3D11_CULL_NONE,
			FrontCounterClockwise: 0,
			DepthBias:             0,
			DepthBiasClamp:        0,
			SlopeScaledDepthBias:  0,
			DepthClipEnable:       0,
			ScissorEnable:         1,
			MultisampleEnable:     0,
			AntialiasedLineEnable: 0,
		})
		if err != nil {
			return nil, err
		}
		g.rasterizerState = rs
	}
	g.deviceContext.RSSetState(g.rasterizerState)

	// Set the sampler state.
	if g.samplerState == nil {
		s, err := g.device.CreateSamplerState(&_D3D11_SAMPLER_DESC{
			Filter:         _D3D11_FILTER_MIN_MAG_MIP_POINT,
			AddressU:       _D3D11_TEXTURE_ADDRESS_WRAP,
			AddressV:       _D3D11_TEXTURE_ADDRESS_WRAP,
			AddressW:       _D3D11_TEXTURE_ADDRESS_WRAP,
			ComparisonFunc: _D3D11_COMPARISON_NEVER,
			MinLOD:         -math.MaxFloat32,
			MaxLOD:         math.MaxFloat32,
		})
		if err != nil {
			return nil, err
		}
		g.samplerState = s
	}
	g.deviceContext.PSSetSamplers(0, []*_ID3D11SamplerState{g.samplerState})

	return g, nil
}

func (g *graphics11) Initialize() error {
	return nil
}

func (g *graphics11) Begin() error {
	return nil
}

func (g *graphics11) End(present bool) error {
	if !present {
		return nil
	}

	if err := g.graphicsInfra.present(g.vsyncEnabled); err != nil {
		return err
	}

	if g.newScreenWidth != 0 && g.newScreenHeight != 0 {
		if g.screenImage != nil {
			// ResizeBuffer requires all the related resources released,
			// so release the swapchain's buffer.
			// Do not dispose the screen image itself since the image's ID is still used.
			g.screenImage.disposeBuffers()
		}

		if err := g.graphicsInfra.resizeSwapChain(g.newScreenWidth, g.newScreenHeight); err != nil {
			return err
		}

		t, err := g.graphicsInfra.getBuffer(0, &_IID_ID3D11Texture2D)
		if err != nil {
			return err
		}
		g.screenImage.width = g.newScreenWidth
		g.screenImage.height = g.newScreenHeight
		g.screenImage.texture = (*_ID3D11Texture2D)(t)

		g.newScreenWidth = 0
		g.newScreenHeight = 0
	}

	return nil
}

func (g *graphics11) SetWindow(window uintptr) {
	g.window = windows.HWND(window)
	// TODO: need to update the swap chain?
}

func (g *graphics11) SetTransparent(transparent bool) {
	// TODO: Implement this?
}

func (g *graphics11) SetVertices(vertices []float32, indices []uint32) error {
	if size := pow2(uint32(len(vertices)) * uint32(unsafe.Sizeof(vertices[0]))); g.vertexBufferSizeInBytes < size {
		if g.vertexBuffer != nil {
			g.vertexBuffer.Release()
			g.vertexBuffer = nil
		}
		b, err := g.device.CreateBuffer(&_D3D11_BUFFER_DESC{
			ByteWidth:      size,
			Usage:          _D3D11_USAGE_DYNAMIC,
			BindFlags:      uint32(_D3D11_BIND_VERTEX_BUFFER),
			CPUAccessFlags: uint32(_D3D11_CPU_ACCESS_WRITE),
		}, nil)
		if err != nil {
			return err
		}
		g.vertexBuffer = b
		g.vertexBufferSizeInBytes = size
		g.deviceContext.IASetVertexBuffers(0, []*_ID3D11Buffer{g.vertexBuffer},
			[]uint32{graphics.VertexFloatCount * uint32(unsafe.Sizeof(vertices[0]))}, []uint32{0})
	}
	if size := pow2(uint32(len(indices)) * uint32(unsafe.Sizeof(indices[0]))); g.indexBufferSizeInBytes < size {
		if g.indexBuffer != nil {
			g.indexBuffer.Release()
			g.indexBuffer = nil
		}
		b, err := g.device.CreateBuffer(&_D3D11_BUFFER_DESC{
			ByteWidth:      size,
			Usage:          _D3D11_USAGE_DYNAMIC,
			BindFlags:      uint32(_D3D11_BIND_INDEX_BUFFER),
			CPUAccessFlags: uint32(_D3D11_CPU_ACCESS_WRITE),
		}, nil)
		if err != nil {
			return err
		}
		g.indexBuffer = b
		g.indexBufferSizeInBytes = size
		g.deviceContext.IASetIndexBuffer(g.indexBuffer, _DXGI_FORMAT_R32_UINT, 0)
	}

	// Copy the vertices data.
	{
		var mapped _D3D11_MAPPED_SUBRESOURCE
		if err := g.deviceContext.Map(unsafe.Pointer(g.vertexBuffer), 0, _D3D11_MAP_WRITE_DISCARD, 0, &mapped); err != nil {
			return err
		}
		copy(unsafe.Slice((*float32)(mapped.pData), len(vertices)), vertices)
		g.deviceContext.Unmap(unsafe.Pointer(g.vertexBuffer), 0)
	}

	// Copy the indices data.
	{
		var mapped _D3D11_MAPPED_SUBRESOURCE
		if err := g.deviceContext.Map(unsafe.Pointer(g.indexBuffer), 0, _D3D11_MAP_WRITE_DISCARD, 0, &mapped); err != nil {
			return err
		}
		copy(unsafe.Slice((*uint32)(mapped.pData), len(indices)), indices)
		g.deviceContext.Unmap(unsafe.Pointer(g.indexBuffer), 0)
	}

	return nil
}

func (g *graphics11) NewImage(width, height int) (graphicsdriver.Image, error) {
	t, err := g.device.CreateTexture2D(&_D3D11_TEXTURE2D_DESC{
		Width:     uint32(graphics.InternalImageSize(width)),
		Height:    uint32(graphics.InternalImageSize(height)),
		MipLevels: 1, // 0 doesn't work when shrinking the image.
		ArraySize: 1,
		Format:    _DXGI_FORMAT_R8G8B8A8_UNORM,
		SampleDesc: _DXGI_SAMPLE_DESC{
			Count:   1,
			Quality: 0,
		},
		Usage:          _D3D11_USAGE_DEFAULT,
		BindFlags:      uint32(_D3D11_BIND_SHADER_RESOURCE | _D3D11_BIND_RENDER_TARGET),
		CPUAccessFlags: 0,
		MiscFlags:      0,
	}, nil)
	if err != nil {
		return nil, err
	}

	i := &image11{
		graphics: g,
		id:       g.genNextImageID(),
		width:    width,
		height:   height,
		texture:  t,
	}
	g.addImage(i)
	return i, nil
}

func (g *graphics11) NewScreenFramebufferImage(width, height int) (graphicsdriver.Image, error) {
	imageWidth := width
	imageHeight := height
	if g.screenImage != nil {
		imageWidth = g.screenImage.width
		imageHeight = g.screenImage.height
		g.screenImage.Dispose()
		g.screenImage = nil
	}

	if g.graphicsInfra.isSwapChainInited() {
		g.newScreenWidth, g.newScreenHeight = width, height
	} else {
		if err := g.graphicsInfra.initSwapChain(width, height, unsafe.Pointer(g.device), g.window); err != nil {
			return nil, err
		}
	}

	t, err := g.graphicsInfra.getBuffer(0, &_IID_ID3D11Texture2D)
	if err != nil {
		return nil, err
	}

	i := &image11{
		graphics: g,
		id:       g.genNextImageID(),
		width:    imageWidth,
		height:   imageHeight,
		screen:   true,
		texture:  (*_ID3D11Texture2D)(t),
	}
	g.addImage(i)
	g.screenImage = i
	return i, nil
}

func (g *graphics11) addImage(img *image11) {
	if g.images == nil {
		g.images = map[graphicsdriver.ImageID]*image11{}
	}
	if _, ok := g.images[img.id]; ok {
		panic(fmt.Sprintf("directx: image ID %d was already registered", img.id))
	}
	g.images[img.id] = img
}

func (g *graphics11) removeImage(image *image11) {
	delete(g.images, image.id)
}

func (g *graphics11) SetVsyncEnabled(enabled bool) {
	g.vsyncEnabled = enabled
}

func (g *graphics11) NeedsClearingScreen() bool {
	// TODO: Confirm this is really true.
	return true
}

func (g *graphics11) MaxImageSize() int {
	switch g.featureLevel {
	case _D3D_FEATURE_LEVEL_10_0:
		return 8192
	case _D3D_FEATURE_LEVEL_10_1:
		return 8192
	case _D3D_FEATURE_LEVEL_11_0:
		return 16384
	default:
		panic(fmt.Sprintf("directx: invalid feature level: 0x%x", g.featureLevel))
	}
}

func (g *graphics11) NewShader(program *shaderir.Program) (graphicsdriver.Shader, error) {
	vsh, psh, err := compileShader(program)
	if err != nil {
		return nil, err
	}

	s := &shader11{
		graphics:         g,
		id:               g.genNextShaderID(),
		uniformTypes:     program.Uniforms,
		uniformOffsets:   hlsl.UniformVariableOffsetsInDwords(program),
		vertexShaderBlob: vsh,
		pixelShaderBlob:  psh,
	}
	g.addShader(s)
	return s, nil
}

func (g *graphics11) addShader(s *shader11) {
	if g.shaders == nil {
		g.shaders = map[graphicsdriver.ShaderID]*shader11{}
	}
	if _, ok := g.shaders[s.id]; ok {
		panic(fmt.Sprintf("directx: shader ID %d was already registered", s.id))
	}
	g.shaders[s.id] = s
}

func (g *graphics11) removeShader(s *shader11) {
	s.disposeImpl()
	delete(g.shaders, s.id)
}

func (g *graphics11) DrawTriangles(dstID graphicsdriver.ImageID, srcIDs [graphics.ShaderSrcImageCount]graphicsdriver.ImageID, shaderID graphicsdriver.ShaderID, dstRegions []graphicsdriver.DstRegion, indexOffset int, blend graphicsdriver.Blend, uniforms []uint32, fillRule graphicsdriver.FillRule) error {
	// Remove bound textures first. This is needed to avoid warnings on the debugger.
	g.deviceContext.OMSetRenderTargets([]*_ID3D11RenderTargetView{nil}, nil)
	srvs := [graphics.ShaderSrcImageCount]*_ID3D11ShaderResourceView{}
	g.deviceContext.PSSetShaderResources(0, srvs[:])

	dst := g.images[dstID]
	var srcs [graphics.ShaderSrcImageCount]*image11
	for i, id := range srcIDs {
		img := g.images[id]
		if img == nil {
			continue
		}
		srcs[i] = img
	}

	w, h := dst.internalSize()
	g.deviceContext.RSSetViewports([]_D3D11_VIEWPORT{
		{
			TopLeftX: 0,
			TopLeftY: 0,
			Width:    float32(w),
			Height:   float32(h),
			MinDepth: 0,
			MaxDepth: 1,
		},
	})

	if err := dst.setAsRenderTarget(fillRule != graphicsdriver.FillRuleFillAll); err != nil {
		return err
	}

	// Set the shader parameters.
	shader := g.shaders[shaderID]
	if err := shader.use(uniforms, srcs); err != nil {
		return err
	}

	if fillRule == graphicsdriver.FillRuleFillAll {
		bs, err := g.blendState(blend, noStencil)
		if err != nil {
			return err
		}
		g.deviceContext.OMSetBlendState(bs, nil, 0xffffffff)

		dss, err := g.depthStencilState(noStencil)
		if err != nil {
			return err
		}
		g.deviceContext.OMSetDepthStencilState(dss, 0)
	}

	for _, dstRegion := range dstRegions {
		g.deviceContext.RSSetScissorRects([]_D3D11_RECT{
			{
				left:   int32(dstRegion.Region.Min.X),
				top:    int32(dstRegion.Region.Min.Y),
				right:  int32(dstRegion.Region.Max.X),
				bottom: int32(dstRegion.Region.Max.Y),
			},
		})

		switch fillRule {
		case graphicsdriver.FillRuleFillAll:
			g.deviceContext.DrawIndexed(uint32(dstRegion.IndexCount), uint32(indexOffset), 0)
		case graphicsdriver.FillRuleNonZero:
			bs, err := g.blendState(blend, incrementStencil)
			if err != nil {
				return err
			}
			g.deviceContext.OMSetBlendState(bs, nil, 0xffffffff)
			dss, err := g.depthStencilState(incrementStencil)
			if err != nil {
				return err
			}
			g.deviceContext.OMSetDepthStencilState(dss, 0)
			g.deviceContext.DrawIndexed(uint32(dstRegion.IndexCount), uint32(indexOffset), 0)
		case graphicsdriver.FillRuleEvenOdd:
			bs, err := g.blendState(blend, invertStencil)
			if err != nil {
				return err
			}
			g.deviceContext.OMSetBlendState(bs, nil, 0xffffffff)
			dss, err := g.depthStencilState(invertStencil)
			if err != nil {
				return err
			}
			g.deviceContext.OMSetDepthStencilState(dss, 0)
			g.deviceContext.DrawIndexed(uint32(dstRegion.IndexCount), uint32(indexOffset), 0)
		}

		if fillRule != graphicsdriver.FillRuleFillAll {
			bs, err := g.blendState(blend, drawWithStencil)
			if err != nil {
				return err
			}
			g.deviceContext.OMSetBlendState(bs, nil, 0xffffffff)
			dss, err := g.depthStencilState(drawWithStencil)
			if err != nil {
				return err
			}
			g.deviceContext.OMSetDepthStencilState(dss, 0)
			g.deviceContext.DrawIndexed(uint32(dstRegion.IndexCount), uint32(indexOffset), 0)
		}

		indexOffset += dstRegion.IndexCount
	}

	return nil
}

func (g *graphics11) genNextImageID() graphicsdriver.ImageID {
	g.nextImageID++
	return g.nextImageID
}

func (g *graphics11) genNextShaderID() graphicsdriver.ShaderID {
	g.nextShaderID++
	return g.nextShaderID
}

func (g *graphics11) blendState(blend graphicsdriver.Blend, stencilMode stencilMode) (*_ID3D11BlendState, error) {
	var writeMask uint8
	if stencilMode == noStencil || stencilMode == drawWithStencil {
		writeMask = uint8(_D3D11_COLOR_WRITE_ENABLE_ALL)
	}

	key := blendStateKey{
		blend:     blend,
		writeMask: writeMask,
	}
	if bs, ok := g.blendStates[key]; ok {
		return bs, nil
	}

	bs, err := g.device.CreateBlendState(&_D3D11_BLEND_DESC{
		AlphaToCoverageEnable:  0,
		IndependentBlendEnable: 0,
		RenderTarget: [8]_D3D11_RENDER_TARGET_BLEND_DESC{
			{
				BlendEnable:           1,
				SrcBlend:              blendFactorToBlend11(blend.BlendFactorSourceRGB, false),
				DestBlend:             blendFactorToBlend11(blend.BlendFactorDestinationRGB, false),
				BlendOp:               blendOperationToBlendOp11(blend.BlendOperationRGB),
				SrcBlendAlpha:         blendFactorToBlend11(blend.BlendFactorSourceAlpha, true),
				DestBlendAlpha:        blendFactorToBlend11(blend.BlendFactorDestinationAlpha, true),
				BlendOpAlpha:          blendOperationToBlendOp11(blend.BlendOperationAlpha),
				RenderTargetWriteMask: writeMask,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if g.blendStates == nil {
		g.blendStates = map[blendStateKey]*_ID3D11BlendState{}
	}
	g.blendStates[key] = bs
	return bs, nil
}

func (g *graphics11) depthStencilState(mode stencilMode) (*_ID3D11DepthStencilState, error) {
	if s, ok := g.depthStencilStates[mode]; ok {
		return s, nil
	}

	desc := &_D3D11_DEPTH_STENCIL_DESC{
		DepthEnable:      0,
		DepthWriteMask:   _D3D11_DEPTH_WRITE_MASK_ALL,
		DepthFunc:        _D3D11_COMPARISON_LESS,
		StencilEnable:    0,
		StencilReadMask:  _D3D11_DEFAULT_STENCIL_READ_MASK,
		StencilWriteMask: _D3D11_DEFAULT_STENCIL_WRITE_MASK,
		FrontFace: _D3D11_DEPTH_STENCILOP_DESC{
			StencilFailOp:      _D3D11_STENCIL_OP_KEEP,
			StencilDepthFailOp: _D3D11_STENCIL_OP_KEEP,
			StencilPassOp:      _D3D11_STENCIL_OP_KEEP,
			StencilFunc:        _D3D11_COMPARISON_ALWAYS,
		},
		BackFace: _D3D11_DEPTH_STENCILOP_DESC{
			StencilFailOp:      _D3D11_STENCIL_OP_KEEP,
			StencilDepthFailOp: _D3D11_STENCIL_OP_KEEP,
			StencilPassOp:      _D3D11_STENCIL_OP_KEEP,
			StencilFunc:        _D3D11_COMPARISON_ALWAYS,
		},
	}
	switch mode {
	case incrementStencil:
		desc.StencilEnable = 1
		desc.FrontFace.StencilPassOp = _D3D11_STENCIL_OP_INCR
		desc.BackFace.StencilPassOp = _D3D11_STENCIL_OP_DECR
	case invertStencil:
		desc.StencilEnable = 1
		desc.FrontFace.StencilPassOp = _D3D11_STENCIL_OP_INVERT
		desc.BackFace.StencilPassOp = _D3D11_STENCIL_OP_INVERT
	case drawWithStencil:
		desc.StencilEnable = 1
		desc.FrontFace.StencilFunc = _D3D11_COMPARISON_NOT_EQUAL
		desc.BackFace.StencilFunc = _D3D11_COMPARISON_NOT_EQUAL
	}

	s, err := g.device.CreateDepthStencilState(desc)
	if err != nil {
		return nil, err
	}

	if g.depthStencilStates == nil {
		g.depthStencilStates = map[stencilMode]*_ID3D11DepthStencilState{}
	}
	g.depthStencilStates[mode] = s
	return s, nil
}
