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
	for i := range diff / 4 {
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

const (
	// initialDescriptorTableCount is the number of descriptor tables in a frame's first shader-visible descriptor heap.
	initialDescriptorTableCount = 256

	// maxDescriptorTableCount is the number of descriptor tables in the largest shader-visible descriptor heap that
	// every device supports.
	maxDescriptorTableCount = _D3D12_MAX_SHADER_VISIBLE_DESCRIPTOR_HEAP_SIZE_TIER_1 / numConstantBufferAndSourceTextures

	// initialConstantBufferSize is the size in bytes of a frame's first upload buffer for constant buffers.
	initialConstantBufferSize = 64 * 1024

	// maxConstantBufferSize is the size in bytes of the largest upload buffer for constant buffers that every device
	// supports.
	maxConstantBufferSize = _D3D12_REQ_RESOURCE_SIZE_IN_MEGABYTES_EXPRESSION_A_TERM * 1024 * 1024
)

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

	shaderDescriptorSize uint32

	// descriptorTables and constantBuffers are indexed by the frame index.
	descriptorTables [frameCount]descriptorTableAllocator
	constantBuffers  [frameCount]constantBufferAllocator
}

const numConstantBufferAndSourceTextures = 1 + graphics.ShaderSrcImageCount

func (p *pipelineStates) initialize(device *_ID3D12Device) (err error) {
	defer func() {
		if err != nil {
			p.release()
		}
	}()

	p.shaderDescriptorSize = device.GetDescriptorHandleIncrementSize(_D3D12_DESCRIPTOR_HEAP_TYPE_CBV_SRV_UAV)

	// Create the first shader-visible descriptor heaps.
	for i := range p.descriptorTables {
		if err := p.descriptorTables[i].grow(device); err != nil {
			return err
		}
	}

	return nil
}

// release releases the objects created by initialize.
func (p *pipelineStates) release() {
	for i := range p.descriptorTables {
		p.descriptorTables[i].release()
	}
	p.shaderDescriptorSize = 0
}

func (p *pipelineStates) drawTriangles(device *_ID3D12Device, commandList *_ID3D12GraphicsCommandList, frameIndex int, screen bool, srcs [graphics.ShaderSrcImageCount]*image12, shader *shader12, dstRegions []graphicsdriver.DstRegion, uniforms []uint32, blend graphicsdriver.Blend, indexOffset int) error {
	heap, h, gh, err := p.descriptorTables[frameIndex].allocate(device, p.shaderDescriptorSize)
	if err != nil {
		return err
	}

	cbv, err := p.constantBuffers[frameIndex].allocate(device, uniforms)
	if err != nil {
		return err
	}
	device.CreateConstantBufferView(&cbv, h)

	for _, src := range srcs {
		h.Offset(1, p.shaderDescriptorSize)
		var resource *_ID3D12Resource
		if src != nil {
			resource = src.resource()
		}
		device.CreateShaderResourceView(resource, &_D3D12_SHADER_RESOURCE_VIEW_DESC{
			Format:                  _DXGI_FORMAT_R8G8B8A8_UNORM,
			ViewDimension:           _D3D12_SRV_DIMENSION_TEXTURE2D,
			Shader4ComponentMapping: _D3D12_DEFAULT_SHADER_4_COMPONENT_MAPPING,
			Texture2D: _D3D12_TEX2D_SRV{
				MipLevels: 1,
			},
		}, h)
	}

	rs, err := p.ensureRootSignature(device)
	if err != nil {
		return err
	}
	commandList.SetGraphicsRootSignature(rs)

	commandList.SetDescriptorHeaps([]*_ID3D12DescriptorHeap{
		heap,
	})

	// Match the indices with rootParams in graphicsPipelineState.
	commandList.SetGraphicsRootDescriptorTable(0, gh)
	commandList.SetGraphicsRootDescriptorTable(1, gh)

	s, err := shader.pipelineState(blend, screen)
	if err != nil {
		return err
	}
	commandList.SetPipelineState(s)

	for _, dstRegion := range dstRegions {
		commandList.RSSetScissorRects([]_D3D12_RECT{
			{
				left:   int32(dstRegion.Region.Min.X),
				top:    int32(dstRegion.Region.Min.Y),
				right:  int32(dstRegion.Region.Max.X),
				bottom: int32(dstRegion.Region.Max.Y),
			},
		})
		commandList.DrawIndexedInstanced(uint32(dstRegion.IndexCount), 1, uint32(indexOffset), 0, 0)
		indexOffset += dstRegion.IndexCount
	}

	return nil
}

// ensureRootSignature returns the root signature owned by p, which the caller must not release.
func (p *pipelineStates) ensureRootSignature(device *_ID3D12Device) (*_ID3D12RootSignature, error) {
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

	p.rootSignature = rs

	return p.rootSignature, nil
}

func (p *pipelineStates) newPipelineState(device *_ID3D12Device, vsh, psh *_ID3DBlob, blend graphicsdriver.Blend, screen bool) (*_ID3D12PipelineState, error) {
	rootSignature, err := p.ensureRootSignature(device)
	if err != nil {
		return nil, err
	}

	writeMask := uint8(_D3D12_COLOR_WRITE_ENABLE_ALL)

	rtvFormat := _DXGI_FORMAT_R8G8B8A8_UNORM
	if screen {
		rtvFormat = _DXGI_FORMAT_B8G8R8A8_UNORM
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
		InputLayout: _D3D12_INPUT_LAYOUT_DESC{
			pInputElementDescs: &inputElementDescsForDX12[0],
			NumElements:        uint32(len(inputElementDescsForDX12)),
		},
		PrimitiveTopologyType: _D3D12_PRIMITIVE_TOPOLOGY_TYPE_TRIANGLE,
		NumRenderTargets:      1,
		RTVFormats: [8]_DXGI_FORMAT{
			rtvFormat,
		},
		DSVFormat: _DXGI_FORMAT_UNKNOWN,
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

// resetFrame makes the descriptor tables and the constant buffers allocated for the frame reusable.
// The GPU must have finished the commands using them.
func (p *pipelineStates) resetFrame(frameIndex int) {
	p.descriptorTables[frameIndex].reset()
	p.constantBuffers[frameIndex].reset()
}

// descriptorTableAllocator allocates a frame's descriptor tables from a shader-visible descriptor heap.
// A descriptor table is a CBV followed by the SRVs of the source textures.
type descriptorTableAllocator struct {
	heap           *_ID3D12DescriptorHeap
	cpuHandleStart _D3D12_CPU_DESCRIPTOR_HANDLE
	gpuHandleStart _D3D12_GPU_DESCRIPTOR_HANDLE
	tableCount     int
	usedTableCount int

	// retiredHeaps are the heaps that ran out earlier in the frame. The GPU might still read them.
	retiredHeaps []*_ID3D12DescriptorHeap
}

// allocate returns the heap and the CPU and GPU handles of an unused descriptor table.
func (d *descriptorTableAllocator) allocate(device *_ID3D12Device, descriptorSize uint32) (*_ID3D12DescriptorHeap, _D3D12_CPU_DESCRIPTOR_HANDLE, _D3D12_GPU_DESCRIPTOR_HANDLE, error) {
	if d.usedTableCount == d.tableCount {
		if err := d.grow(device); err != nil {
			return nil, _D3D12_CPU_DESCRIPTOR_HANDLE{}, _D3D12_GPU_DESCRIPTOR_HANDLE{}, err
		}
	}

	offset := int32(d.usedTableCount * numConstantBufferAndSourceTextures)
	d.usedTableCount++

	cpuHandle := d.cpuHandleStart
	cpuHandle.Offset(offset, descriptorSize)
	gpuHandle := d.gpuHandleStart
	gpuHandle.Offset(offset, descriptorSize)
	return d.heap, cpuHandle, gpuHandle, nil
}

// grow replaces the heap with a larger one. The replaced heap stays alive until reset.
func (d *descriptorTableAllocator) grow(device *_ID3D12Device) (err error) {
	tableCount := min(max(2*d.tableCount, initialDescriptorTableCount), maxDescriptorTableCount)
	heap, err := device.CreateDescriptorHeap(&_D3D12_DESCRIPTOR_HEAP_DESC{
		Type:           _D3D12_DESCRIPTOR_HEAP_TYPE_CBV_SRV_UAV,
		NumDescriptors: uint32(tableCount * numConstantBufferAndSourceTextures),
		Flags:          _D3D12_DESCRIPTOR_HEAP_FLAG_SHADER_VISIBLE,
		NodeMask:       0,
	})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			heap.Release()
		}
	}()

	cpuHandle, err := heap.GetCPUDescriptorHandleForHeapStart()
	if err != nil {
		return err
	}
	gpuHandle, err := heap.GetGPUDescriptorHandleForHeapStart()
	if err != nil {
		return err
	}

	if d.heap != nil {
		d.retiredHeaps = append(d.retiredHeaps, d.heap)
	}
	d.heap = heap
	d.cpuHandleStart = cpuHandle
	d.gpuHandleStart = gpuHandle
	d.tableCount = tableCount
	d.usedTableCount = 0
	return nil
}

// reset makes all the descriptor tables of the heap unused, and releases the retired heaps.
// The GPU must have finished the commands using them.
func (d *descriptorTableAllocator) reset() {
	for i, h := range d.retiredHeaps {
		h.Release()
		d.retiredHeaps[i] = nil
	}
	d.retiredHeaps = d.retiredHeaps[:0]
	d.usedTableCount = 0
}

// release releases all the heaps.
func (d *descriptorTableAllocator) release() {
	d.reset()
	if d.heap != nil {
		d.heap.Release()
	}
	*d = descriptorTableAllocator{}
}

// constantBufferAllocator allocates a frame's constant buffers from an upload buffer.
type constantBufferAllocator struct {
	buffer *_ID3D12Resource

	// mapped is the memory of buffer, which stays mapped while buffer is alive.
	mapped []uint32

	gpuAddress _D3D12_GPU_VIRTUAL_ADDRESS
	size       uint64
	usedSize   uint64

	// retiredBuffers are the upload buffers that ran out earlier in the frame. The GPU might still read them.
	retiredBuffers []*_ID3D12Resource
}

// allocate copies uniforms to an unused constant buffer and returns its view.
func (c *constantBufferAllocator) allocate(device *_ID3D12Device, uniforms []uint32) (_D3D12_CONSTANT_BUFFER_VIEW_DESC, error) {
	size := uint64(unsafe.Sizeof(uniforms[0])) * uint64(len(uniforms))
	size = (size + _D3D12_CONSTANT_BUFFER_DATA_PLACEMENT_ALIGNMENT - 1) &^ (_D3D12_CONSTANT_BUFFER_DATA_PLACEMENT_ALIGNMENT - 1)

	if c.usedSize+size > c.size {
		if err := c.grow(device, size); err != nil {
			return _D3D12_CONSTANT_BUFFER_VIEW_DESC{}, err
		}
	}

	offset := c.usedSize
	c.usedSize += size

	copy(c.mapped[offset/uint64(unsafe.Sizeof(c.mapped[0])):], uniforms)
	return _D3D12_CONSTANT_BUFFER_VIEW_DESC{
		BufferLocation: c.gpuAddress + _D3D12_GPU_VIRTUAL_ADDRESS(offset),
		SizeInBytes:    uint32(size),
	}, nil
}

// grow replaces the upload buffer with a larger one of at least minSize bytes. The replaced buffer stays alive until reset.
func (c *constantBufferAllocator) grow(device *_ID3D12Device, minSize uint64) (err error) {
	size := min(max(2*c.size, initialConstantBufferSize), maxConstantBufferSize)
	size = max(size, minSize)
	buffer, err := createBuffer(device, size, _D3D12_HEAP_TYPE_UPLOAD)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			buffer.Release()
		}
	}()

	m, err := buffer.Map(0, &_D3D12_RANGE{Begin: 0, End: 0})
	if err != nil {
		return err
	}

	if c.buffer != nil {
		c.retiredBuffers = append(c.retiredBuffers, c.buffer)
	}
	c.buffer = buffer
	c.gpuAddress = buffer.GetGPUVirtualAddress()
	c.size = size
	c.usedSize = 0
	c.mapped = unsafe.Slice((*uint32)(unsafe.Pointer(m)), size/uint64(unsafe.Sizeof(c.mapped[0])))
	return nil
}

// reset makes the whole upload buffer unused, and releases the retired buffers.
// The GPU must have finished the commands using them.
func (c *constantBufferAllocator) reset() {
	for i, b := range c.retiredBuffers {
		b.Unmap(0, nil)
		b.Release()
		c.retiredBuffers[i] = nil
	}
	c.retiredBuffers = c.retiredBuffers[:0]
	c.usedSize = 0
}
