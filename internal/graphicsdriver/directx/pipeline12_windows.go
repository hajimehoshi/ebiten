// Copyright 2022 The Ebiten Authors
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

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

var inputElementDescsForDX12 []_D3D12_INPUT_ELEMENT_DESC

func init() {
	inputElementDescsForDX12 = []_D3D12_INPUT_ELEMENT_DESC{
		{
			SemanticName:         &([]byte("POSITION\000"))[0],
			SemanticIndex:        0,
			Format:               _DXGI_FORMAT_R32G32_FLOAT,
			InputSlot:            0,
			AlignedByteOffset:    _D3D12_APPEND_ALIGNED_ELEMENT,
			InputSlotClass:       _D3D12_INPUT_CLASSIFICATION_PER_VERTEX_DATA,
			InstanceDataStepRate: 0,
		},
		{
			SemanticName:         &([]byte("TEXCOORD\000"))[0],
			SemanticIndex:        0,
			Format:               _DXGI_FORMAT_R32G32_FLOAT,
			InputSlot:            0,
			AlignedByteOffset:    _D3D12_APPEND_ALIGNED_ELEMENT,
			InputSlotClass:       _D3D12_INPUT_CLASSIFICATION_PER_VERTEX_DATA,
			InstanceDataStepRate: 0,
		},
		{
			SemanticName:         &([]byte("COLOR\000"))[0],
			SemanticIndex:        0,
			Format:               _DXGI_FORMAT_R32G32B32A32_FLOAT,
			InputSlot:            0,
			AlignedByteOffset:    _D3D12_APPEND_ALIGNED_ELEMENT,
			InputSlotClass:       _D3D12_INPUT_CLASSIFICATION_PER_VERTEX_DATA,
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
		inputElementDescsForDX12 = append(inputElementDescsForDX12, _D3D12_INPUT_ELEMENT_DESC{
			SemanticName:         &([]byte("COLOR\000"))[0],
			SemanticIndex:        uint32(i) + 1,
			Format:               _DXGI_FORMAT_R32G32B32A32_FLOAT,
			InputSlot:            0,
			AlignedByteOffset:    _D3D12_APPEND_ALIGNED_ELEMENT,
			InputSlotClass:       _D3D12_INPUT_CLASSIFICATION_PER_VERTEX_DATA,
			InstanceDataStepRate: 0,
		})
	}
}

const numDescriptorsPerFrame = 32

func blendFactorToBlend12(f graphicsdriver.BlendFactor, alpha bool) _D3D12_BLEND {
	// D3D12_RENDER_TARGET_BLEND_DESC's *BlendAlpha members don't allow *_COLOR values.
	// See https://learn.microsoft.com/en-us/windows/win32/api/d3d12/ns-d3d12-d3d12_render_target_blend_desc.

	switch f {
	case graphicsdriver.BlendFactorZero:
		return _D3D12_BLEND_ZERO
	case graphicsdriver.BlendFactorOne:
		return _D3D12_BLEND_ONE
	case graphicsdriver.BlendFactorSourceColor:
		if alpha {
			return _D3D12_BLEND_SRC_ALPHA
		}
		return _D3D12_BLEND_SRC_COLOR
	case graphicsdriver.BlendFactorOneMinusSourceColor:
		if alpha {
			return _D3D12_BLEND_INV_SRC_ALPHA
		}
		return _D3D12_BLEND_INV_SRC_COLOR
	case graphicsdriver.BlendFactorSourceAlpha:
		return _D3D12_BLEND_SRC_ALPHA
	case graphicsdriver.BlendFactorOneMinusSourceAlpha:
		return _D3D12_BLEND_INV_SRC_ALPHA
	case graphicsdriver.BlendFactorDestinationColor:
		if alpha {
			return _D3D12_BLEND_DEST_ALPHA
		}
		return _D3D12_BLEND_DEST_COLOR
	case graphicsdriver.BlendFactorOneMinusDestinationColor:
		if alpha {
			return _D3D12_BLEND_INV_DEST_ALPHA
		}
		return _D3D12_BLEND_INV_DEST_COLOR
	case graphicsdriver.BlendFactorDestinationAlpha:
		return _D3D12_BLEND_DEST_ALPHA
	case graphicsdriver.BlendFactorOneMinusDestinationAlpha:
		return _D3D12_BLEND_INV_DEST_ALPHA
	case graphicsdriver.BlendFactorSourceAlphaSaturated:
		return _D3D12_BLEND_SRC_ALPHA_SAT
	default:
		panic(fmt.Sprintf("directx: invalid blend factor: %d", f))
	}
}

func blendOperationToBlendOp12(o graphicsdriver.BlendOperation) _D3D12_BLEND_OP {
	switch o {
	case graphicsdriver.BlendOperationAdd:
		return _D3D12_BLEND_OP_ADD
	case graphicsdriver.BlendOperationSubtract:
		return _D3D12_BLEND_OP_SUBTRACT
	case graphicsdriver.BlendOperationReverseSubtract:
		return _D3D12_BLEND_OP_REV_SUBTRACT
	case graphicsdriver.BlendOperationMin:
		return _D3D12_BLEND_OP_MIN
	case graphicsdriver.BlendOperationMax:
		return _D3D12_BLEND_OP_MAX
	default:
		panic(fmt.Sprintf("directx: invalid blend operation: %d", o))
	}
}

type pipelineStates struct {
	rootSignature *_ID3D12RootSignature

	shaderDescriptorHeap *_ID3D12DescriptorHeap
	shaderDescriptorSize uint32

	samplerDescriptorHeap *_ID3D12DescriptorHeap

	constantBuffers    [frameCount][]*_ID3D12Resource
	constantBufferMaps [frameCount][]uintptr
}

const numConstantBufferAndSourceTextures = 1 + graphics.ShaderSrcImageCount

func (p *pipelineStates) initialize(device *_ID3D12Device) (ferr error) {
	// Create a CBV/SRV/UAV descriptor heap.
	//   5n+0:        constants
	//   5n+m (1<=4): textures
	shaderH, err := device.CreateDescriptorHeap(&_D3D12_DESCRIPTOR_HEAP_DESC{
		Type:           _D3D12_DESCRIPTOR_HEAP_TYPE_CBV_SRV_UAV,
		NumDescriptors: frameCount * numDescriptorsPerFrame * numConstantBufferAndSourceTextures,
		Flags:          _D3D12_DESCRIPTOR_HEAP_FLAG_SHADER_VISIBLE,
		NodeMask:       0,
	})
	if err != nil {
		return err
	}
	p.shaderDescriptorHeap = shaderH
	defer func() {
		if ferr != nil {
			p.shaderDescriptorHeap.Release()
			p.shaderDescriptorHeap = nil
		}
	}()
	p.shaderDescriptorSize = device.GetDescriptorHandleIncrementSize(_D3D12_DESCRIPTOR_HEAP_TYPE_CBV_SRV_UAV)

	samplerH, err := device.CreateDescriptorHeap(&_D3D12_DESCRIPTOR_HEAP_DESC{
		Type:           _D3D12_DESCRIPTOR_HEAP_TYPE_SAMPLER,
		NumDescriptors: 1,
		Flags:          _D3D12_DESCRIPTOR_HEAP_FLAG_SHADER_VISIBLE,
		NodeMask:       0,
	})
	if err != nil {
		return err
	}
	p.samplerDescriptorHeap = samplerH

	h, err := p.samplerDescriptorHeap.GetCPUDescriptorHandleForHeapStart()
	if err != nil {
		return err
	}
	device.CreateSampler(&_D3D12_SAMPLER_DESC{
		Filter:         _D3D12_FILTER_MIN_MAG_MIP_POINT,
		AddressU:       _D3D12_TEXTURE_ADDRESS_MODE_WRAP,
		AddressV:       _D3D12_TEXTURE_ADDRESS_MODE_WRAP,
		AddressW:       _D3D12_TEXTURE_ADDRESS_MODE_WRAP,
		ComparisonFunc: _D3D12_COMPARISON_FUNC_NEVER,
		MinLOD:         -math.MaxFloat32,
		MaxLOD:         math.MaxFloat32,
	}, h)

	return nil
}

func (p *pipelineStates) drawTriangles(device *_ID3D12Device, commandList *_ID3D12GraphicsCommandList, frameIndex int, screen bool, srcs [graphics.ShaderSrcImageCount]*image12, shader *shader12, dstRegions []graphicsdriver.DstRegion, uniforms []uint32, blend graphicsdriver.Blend, indexOffset int, fillRule graphicsdriver.FillRule) error {
	idx := len(p.constantBuffers[frameIndex])
	if idx >= numDescriptorsPerFrame {
		return fmt.Errorf("directx: too many constant buffers")
	}

	if cap(p.constantBuffers[frameIndex]) > idx {
		p.constantBuffers[frameIndex] = p.constantBuffers[frameIndex][:idx+1]
		p.constantBufferMaps[frameIndex] = p.constantBufferMaps[frameIndex][:idx+1]
	} else {
		p.constantBuffers[frameIndex] = append(p.constantBuffers[frameIndex], nil)
		p.constantBufferMaps[frameIndex] = append(p.constantBufferMaps[frameIndex], 0)
	}

	const bufferSizeAlignment = 256
	bufferSize := uint32(unsafe.Sizeof(uint32(0))) * uint32(len(uniforms))
	if bufferSize > 0 {
		bufferSize = ((bufferSize-1)/bufferSizeAlignment + 1) * bufferSizeAlignment
	}

	cb := p.constantBuffers[frameIndex][idx]
	m := p.constantBufferMaps[frameIndex][idx]
	if cb != nil {
		if uint32(cb.GetDesc().Width) < bufferSize {
			p.constantBuffers[frameIndex][idx].Unmap(0, nil)
			p.constantBuffers[frameIndex][idx].Release()
			p.constantBuffers[frameIndex][idx] = nil
			p.constantBufferMaps[frameIndex][idx] = 0
			cb = nil
		}
	}
	if cb == nil {
		var err error
		cb, err = createBuffer(device, uint64(bufferSize), _D3D12_HEAP_TYPE_UPLOAD)
		if err != nil {
			return err
		}
		p.constantBuffers[frameIndex][idx] = cb

		h, err := p.shaderDescriptorHeap.GetCPUDescriptorHandleForHeapStart()
		if err != nil {
			return err
		}
		offset := int32(numConstantBufferAndSourceTextures * (frameIndex*numDescriptorsPerFrame + idx))
		h.Offset(offset, p.shaderDescriptorSize)
		device.CreateConstantBufferView(&_D3D12_CONSTANT_BUFFER_VIEW_DESC{
			BufferLocation: cb.GetGPUVirtualAddress(),
			SizeInBytes:    bufferSize,
		}, h)

		m, err = cb.Map(0, &_D3D12_RANGE{0, 0})
		if err != nil {
			return err
		}
		p.constantBufferMaps[frameIndex][idx] = m
	}
	if m == 0 {
		return fmt.Errorf("directx: ID3D12Resource::Map failed")
	}

	h, err := p.shaderDescriptorHeap.GetCPUDescriptorHandleForHeapStart()
	if err != nil {
		return err
	}
	offset := int32(numConstantBufferAndSourceTextures * (frameIndex*numDescriptorsPerFrame + idx))
	h.Offset(offset, p.shaderDescriptorSize)
	for _, src := range srcs {
		h.Offset(1, p.shaderDescriptorSize)
		if src == nil {
			continue
		}
		device.CreateShaderResourceView(src.resource(), &_D3D12_SHADER_RESOURCE_VIEW_DESC{
			Format:                  _DXGI_FORMAT_R8G8B8A8_UNORM,
			ViewDimension:           _D3D12_SRV_DIMENSION_TEXTURE2D,
			Shader4ComponentMapping: _D3D12_DEFAULT_SHADER_4_COMPONENT_MAPPING,
			Texture2D: _D3D12_TEX2D_SRV{
				MipLevels: 1, // TODO: Can this be 0?
			},
		}, h)
	}

	// Update the constant buffer.
	copy(unsafe.Slice((*uint32)(unsafe.Pointer(m)), len(uniforms)), uniforms)

	rs, err := p.ensureRootSignature(device)
	if err != nil {
		return err
	}
	commandList.SetGraphicsRootSignature(rs)

	commandList.SetDescriptorHeaps([]*_ID3D12DescriptorHeap{
		p.shaderDescriptorHeap,
		p.samplerDescriptorHeap,
	})

	// Match the indices with rootParams in graphicsPipelineState.
	gh, err := p.shaderDescriptorHeap.GetGPUDescriptorHandleForHeapStart()
	if err != nil {
		return err
	}
	gh.Offset(offset, p.shaderDescriptorSize)
	commandList.SetGraphicsRootDescriptorTable(0, gh)
	commandList.SetGraphicsRootDescriptorTable(1, gh)
	sh, err := p.samplerDescriptorHeap.GetGPUDescriptorHandleForHeapStart()
	if err != nil {
		return err
	}
	commandList.SetGraphicsRootDescriptorTable(2, sh)

	if fillRule == graphicsdriver.FillRuleFillAll {
		s, err := shader.pipelineState(blend, noStencil, screen)
		if err != nil {
			return err
		}
		commandList.SetPipelineState(s)
	}

	for _, dstRegion := range dstRegions {
		commandList.RSSetScissorRects([]_D3D12_RECT{
			{
				left:   int32(dstRegion.Region.Min.X),
				top:    int32(dstRegion.Region.Min.Y),
				right:  int32(dstRegion.Region.Max.X),
				bottom: int32(dstRegion.Region.Max.Y),
			},
		})
		switch fillRule {
		case graphicsdriver.FillRuleFillAll:
			commandList.DrawIndexedInstanced(uint32(dstRegion.IndexCount), 1, uint32(indexOffset), 0, 0)
		case graphicsdriver.FillRuleNonZero:
			s, err := shader.pipelineState(blend, incrementStencil, screen)
			if err != nil {
				return err
			}
			commandList.SetPipelineState(s)
			commandList.DrawIndexedInstanced(uint32(dstRegion.IndexCount), 1, uint32(indexOffset), 0, 0)
		case graphicsdriver.FillRuleEvenOdd:
			s, err := shader.pipelineState(blend, invertStencil, screen)
			if err != nil {
				return err
			}
			commandList.SetPipelineState(s)
			commandList.DrawIndexedInstanced(uint32(dstRegion.IndexCount), 1, uint32(indexOffset), 0, 0)
		}

		if fillRule != graphicsdriver.FillRuleFillAll {
			s, err := shader.pipelineState(blend, drawWithStencil, screen)
			if err != nil {
				return err
			}
			commandList.SetPipelineState(s)
			commandList.DrawIndexedInstanced(uint32(dstRegion.IndexCount), 1, uint32(indexOffset), 0, 0)
		}

		indexOffset += dstRegion.IndexCount
	}

	return nil
}

func (p *pipelineStates) ensureRootSignature(device *_ID3D12Device) (rootSignature *_ID3D12RootSignature, ferr error) {
	if p.rootSignature != nil {
		return p.rootSignature, nil
	}

	cbv := _D3D12_DESCRIPTOR_RANGE{
		RangeType:                         _D3D12_DESCRIPTOR_RANGE_TYPE_CBV, // b0
		NumDescriptors:                    1,
		BaseShaderRegister:                0,
		RegisterSpace:                     0,
		OffsetInDescriptorsFromTableStart: 0,
	}
	srv := _D3D12_DESCRIPTOR_RANGE{
		RangeType:                         _D3D12_DESCRIPTOR_RANGE_TYPE_SRV, // t0
		NumDescriptors:                    graphics.ShaderSrcImageCount,
		BaseShaderRegister:                0,
		RegisterSpace:                     0,
		OffsetInDescriptorsFromTableStart: 1,
	}
	sampler := _D3D12_DESCRIPTOR_RANGE{
		RangeType:                         _D3D12_DESCRIPTOR_RANGE_TYPE_SAMPLER, // s0
		NumDescriptors:                    1,
		BaseShaderRegister:                0,
		RegisterSpace:                     0,
		OffsetInDescriptorsFromTableStart: 0,
	}

	rootParams := [...]_D3D12_ROOT_PARAMETER{
		{
			ParameterType: _D3D12_ROOT_PARAMETER_TYPE_DESCRIPTOR_TABLE,
			DescriptorTable: _D3D12_ROOT_DESCRIPTOR_TABLE{
				NumDescriptorRanges: 1,
				pDescriptorRanges:   &cbv,
			},
			ShaderVisibility: _D3D12_SHADER_VISIBILITY_ALL,
		},
		{
			ParameterType: _D3D12_ROOT_PARAMETER_TYPE_DESCRIPTOR_TABLE,
			DescriptorTable: _D3D12_ROOT_DESCRIPTOR_TABLE{
				NumDescriptorRanges: 1,
				pDescriptorRanges:   &srv,
			},
			ShaderVisibility: _D3D12_SHADER_VISIBILITY_PIXEL,
		},
		{
			ParameterType: _D3D12_ROOT_PARAMETER_TYPE_DESCRIPTOR_TABLE,
			DescriptorTable: _D3D12_ROOT_DESCRIPTOR_TABLE{
				NumDescriptorRanges: 1,
				pDescriptorRanges:   &sampler,
			},
			ShaderVisibility: _D3D12_SHADER_VISIBILITY_PIXEL,
		},
	}

	// Create a root signature.
	sig, err := _D3D12SerializeRootSignature(&_D3D12_ROOT_SIGNATURE_DESC{
		NumParameters:     uint32(len(rootParams)),
		pParameters:       &rootParams[0],
		NumStaticSamplers: 0,
		pStaticSamplers:   nil,
		Flags:             _D3D12_ROOT_SIGNATURE_FLAG_ALLOW_INPUT_ASSEMBLER_INPUT_LAYOUT,
	}, _D3D_ROOT_SIGNATURE_VERSION_1_0)
	if err != nil {
		return nil, err
	}
	defer sig.Release()

	rs, err := device.CreateRootSignature(0, sig.GetBufferPointer(), sig.GetBufferSize())
	if err != nil {
		return nil, err
	}
	defer func() {
		if ferr != nil {
			rootSignature.Release()
		}
	}()

	p.rootSignature = rs

	return p.rootSignature, nil
}

func (p *pipelineStates) newPipelineState(device *_ID3D12Device, vsh, psh *_ID3DBlob, blend graphicsdriver.Blend, stencilMode stencilMode, screen bool) (state *_ID3D12PipelineState, ferr error) {
	rootSignature, err := p.ensureRootSignature(device)
	if err != nil {
		return nil, err
	}
	defer func() {
		if ferr != nil {
			rootSignature.Release()
		}
	}()

	depthStencilDesc := _D3D12_DEPTH_STENCIL_DESC{
		DepthEnable:      0,
		DepthWriteMask:   _D3D12_DEPTH_WRITE_MASK_ALL,
		DepthFunc:        _D3D12_COMPARISON_FUNC_LESS,
		StencilEnable:    0,
		StencilReadMask:  _D3D12_DEFAULT_STENCIL_READ_MASK,
		StencilWriteMask: _D3D12_DEFAULT_STENCIL_WRITE_MASK,
		FrontFace: _D3D12_DEPTH_STENCILOP_DESC{
			StencilFailOp:      _D3D12_STENCIL_OP_KEEP,
			StencilDepthFailOp: _D3D12_STENCIL_OP_KEEP,
			StencilPassOp:      _D3D12_STENCIL_OP_KEEP,
			StencilFunc:        _D3D12_COMPARISON_FUNC_ALWAYS,
		},
		BackFace: _D3D12_DEPTH_STENCILOP_DESC{
			StencilFailOp:      _D3D12_STENCIL_OP_KEEP,
			StencilDepthFailOp: _D3D12_STENCIL_OP_KEEP,
			StencilPassOp:      _D3D12_STENCIL_OP_KEEP,
			StencilFunc:        _D3D12_COMPARISON_FUNC_ALWAYS,
		},
	}

	var writeMask uint8
	if stencilMode == noStencil || stencilMode == drawWithStencil {
		writeMask = uint8(_D3D12_COLOR_WRITE_ENABLE_ALL)
	}

	switch stencilMode {
	case incrementStencil:
		depthStencilDesc.StencilEnable = 1
		depthStencilDesc.FrontFace.StencilPassOp = _D3D12_STENCIL_OP_INCR
		depthStencilDesc.BackFace.StencilPassOp = _D3D12_STENCIL_OP_DECR
	case invertStencil:
		depthStencilDesc.StencilEnable = 1
		depthStencilDesc.FrontFace.StencilPassOp = _D3D12_STENCIL_OP_INVERT
		depthStencilDesc.BackFace.StencilPassOp = _D3D12_STENCIL_OP_INVERT
	case drawWithStencil:
		depthStencilDesc.StencilEnable = 1
		depthStencilDesc.FrontFace.StencilFunc = _D3D12_COMPARISON_FUNC_NOT_EQUAL
		depthStencilDesc.BackFace.StencilFunc = _D3D12_COMPARISON_FUNC_NOT_EQUAL
	}

	rtvFormat := _DXGI_FORMAT_R8G8B8A8_UNORM
	if screen {
		rtvFormat = _DXGI_FORMAT_B8G8R8A8_UNORM
	}
	dsvFormat := _DXGI_FORMAT_UNKNOWN
	if stencilMode != noStencil {
		dsvFormat = _DXGI_FORMAT_D24_UNORM_S8_UINT
	}

	// Create a pipeline state.
	psoDesc := _D3D12_GRAPHICS_PIPELINE_STATE_DESC{
		pRootSignature: rootSignature,
		VS: _D3D12_SHADER_BYTECODE{
			pShaderBytecode: vsh.GetBufferPointer(),
			BytecodeLength:  vsh.GetBufferSize(),
		},
		PS: _D3D12_SHADER_BYTECODE{
			pShaderBytecode: psh.GetBufferPointer(),
			BytecodeLength:  psh.GetBufferSize(),
		},
		BlendState: _D3D12_BLEND_DESC{
			AlphaToCoverageEnable:  0,
			IndependentBlendEnable: 0,
			RenderTarget: [8]_D3D12_RENDER_TARGET_BLEND_DESC{
				{
					BlendEnable:           1,
					LogicOpEnable:         0,
					SrcBlend:              blendFactorToBlend12(blend.BlendFactorSourceRGB, false),
					DestBlend:             blendFactorToBlend12(blend.BlendFactorDestinationRGB, false),
					BlendOp:               blendOperationToBlendOp12(blend.BlendOperationRGB),
					SrcBlendAlpha:         blendFactorToBlend12(blend.BlendFactorSourceAlpha, true),
					DestBlendAlpha:        blendFactorToBlend12(blend.BlendFactorDestinationAlpha, true),
					BlendOpAlpha:          blendOperationToBlendOp12(blend.BlendOperationAlpha),
					LogicOp:               _D3D12_LOGIC_OP_NOOP,
					RenderTargetWriteMask: writeMask,
				},
			},
		},
		SampleMask: math.MaxUint32,
		RasterizerState: _D3D12_RASTERIZER_DESC{
			FillMode:              _D3D12_FILL_MODE_SOLID,
			CullMode:              _D3D12_CULL_MODE_NONE,
			FrontCounterClockwise: 0,
			DepthBias:             _D3D12_DEFAULT_DEPTH_BIAS,
			DepthBiasClamp:        _D3D12_DEFAULT_DEPTH_BIAS_CLAMP,
			SlopeScaledDepthBias:  _D3D12_DEFAULT_SLOPE_SCALED_DEPTH_BIAS,
			DepthClipEnable:       0,
			MultisampleEnable:     0,
			AntialiasedLineEnable: 0,
			ForcedSampleCount:     0,
			ConservativeRaster:    _D3D12_CONSERVATIVE_RASTERIZATION_MODE_OFF,
		},
		DepthStencilState: depthStencilDesc,
		InputLayout: _D3D12_INPUT_LAYOUT_DESC{
			pInputElementDescs: &inputElementDescsForDX12[0],
			NumElements:        uint32(len(inputElementDescsForDX12)),
		},
		PrimitiveTopologyType: _D3D12_PRIMITIVE_TOPOLOGY_TYPE_TRIANGLE,
		NumRenderTargets:      1,
		RTVFormats: [8]_DXGI_FORMAT{
			rtvFormat,
		},
		DSVFormat: dsvFormat,
		SampleDesc: _DXGI_SAMPLE_DESC{
			Count:   1,
			Quality: 0,
		},
	}

	s, err := device.CreateGraphicsPipelineState(&psoDesc)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (p *pipelineStates) releaseConstantBuffers(frameIndex int) {
	for i := range p.constantBuffers[frameIndex] {
		p.constantBuffers[frameIndex][i].Unmap(0, nil)
		p.constantBuffers[frameIndex][i].Release()
		p.constantBuffers[frameIndex][i] = nil
		p.constantBufferMaps[frameIndex][i] = 0
	}
	p.constantBuffers[frameIndex] = p.constantBuffers[frameIndex][:0]
	p.constantBufferMaps[frameIndex] = p.constantBufferMaps[frameIndex][:0]
}

func (p *pipelineStates) resetConstantBuffers(frameIndex int) {
	p.constantBuffers[frameIndex] = p.constantBuffers[frameIndex][:0]
	p.constantBufferMaps[frameIndex] = p.constantBufferMaps[frameIndex][:0]
}
