// Copyright 2022 The Ebitengine Authors
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

//go:build ignore
// +build ignore

#include <stdint.h>

__declspec(dllexport) __cdecl uint32_t XSystemGetDeviceType(void) {
  return 0;
}

__declspec(dllexport) __cdecl void Ebitengine_ID3D12GraphicsCommandList_DrawIndexedInstanced(void* i, uint32_t indexCountPerInstance, uint32_t instanceCount, uint32_t startIndexLocation, int32_t baseVertexLocation, uint32_t startInstanceLocation) {
}

__declspec(dllexport) __cdecl void Ebitengine_ID3D12GraphicsCommandList_IASetIndexBuffer(void* i, void* pView) {
}

__declspec(dllexport) __cdecl void Ebitengine_ID3D12GraphicsCommandList_IASetPrimitiveTopology(void* i, int32_t primitiveTopology) {
}

__declspec(dllexport) __cdecl void Ebitengine_ID3D12GraphicsCommandList_IASetVertexBuffers(void* i, uint32_t startSlot, uint32_t numViews, void* pViews) {
}

__declspec(dllexport) __cdecl void Ebitengine_ID3D12GraphicsCommandList_OMSetStencilRef(void* i, uint32_t stencilRef) {
}

__declspec(dllexport) __cdecl void Ebitengine_ID3D12GraphicsCommandList_SetGraphicsRootDescriptorTable(void* i, uint32_t rootParameterIndex, uint64_t baseDescriptorPtr) {
}

__declspec(dllexport) __cdecl void Ebitengine_ID3D12GraphicsCommandList_SetPipelineState(void* i, void* pPipelineState) {
}
