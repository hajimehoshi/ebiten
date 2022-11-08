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

//go:build amd64 || arm64

package directx

import (
	"unsafe"
)

type _D3D12_DEPTH_STENCIL_VIEW_DESC struct {
	Format        _DXGI_FORMAT
	ViewDimension _D3D12_DSV_DIMENSION
	Flags         _D3D12_DSV_FLAGS
	_             [4]byte                                      // Padding
	Texture2D     _D3D12_TEX2D_DSV                             // Union
	_             [12 - unsafe.Sizeof(_D3D12_TEX2D_DSV{})]byte // Padding for union
}

type _D3D12_RESOURCE_DESC struct {
	Dimension        _D3D12_RESOURCE_DIMENSION
	Alignment        uint64
	Width            uint64
	Height           uint32
	DepthOrArraySize uint16
	MipLevels        uint16
	Format           _DXGI_FORMAT
	SampleDesc       _DXGI_SAMPLE_DESC
	Layout           _D3D12_TEXTURE_LAYOUT
	Flags            _D3D12_RESOURCE_FLAGS
}

type _D3D12_ROOT_PARAMETER struct {
	ParameterType    _D3D12_ROOT_PARAMETER_TYPE
	DescriptorTable  _D3D12_ROOT_DESCRIPTOR_TABLE // Union
	ShaderVisibility _D3D12_SHADER_VISIBILITY
}
