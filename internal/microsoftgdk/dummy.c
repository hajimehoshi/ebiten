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

__declspec(dllexport) __cdecl void Ebitengine_ID3D12GraphicsCommandList_ClearDepthStencilView(void* i, uintptr_t depthStencilView, int32_t clearFlags, float depth, uint8_t stencil, uint32_t numRects, void* pRects) {
}

__declspec(dllexport) __cdecl void Ebitengine_ID3D12GraphicsCommandList_ClearRenderTargetView(void* i, uintptr_t pRenderTargetView, void* colorRGBA, uint32_t numRects, void* pRects) {
}

__declspec(dllexport) __cdecl uintptr_t Ebitengine_ID3D12GraphicsCommandList_Close(void* i) {
}

__declspec(dllexport) __cdecl void Ebitengine_ID3D12GraphicsCommandList_CopyTextureRegion(void* i, void* pDst, uint32_t dstX, uint32_t dstY, uint32_t dstZ, void* pSrc, void* pSrcBox) {
}

__declspec(dllexport) __cdecl void Ebitengine_ID3D12GraphicsCommandList_DrawIndexedInstanced(void* i, uint32_t indexCountPerInstance, uint32_t instanceCount, uint32_t startIndexLocation, int32_t baseVertexLocation, uint32_t startInstanceLocation) {
}

__declspec(dllexport) __cdecl void Ebitengine_ID3D12GraphicsCommandList_IASetIndexBuffer(void* i, void* pView) {
}

__declspec(dllexport) __cdecl void Ebitengine_ID3D12GraphicsCommandList_IASetPrimitiveTopology(void* i, int32_t primitiveTopology) {
}

__declspec(dllexport) __cdecl void Ebitengine_ID3D12GraphicsCommandList_IASetVertexBuffers(void* i, uint32_t startSlot, uint32_t numViews, void* pViews) {
}

__declspec(dllexport) __cdecl void Ebitengine_ID3D12GraphicsCommandList_OMSetRenderTargets(void* i, uint32_t numRenderTargetDescriptors, void* pRenderTargetDescriptors, int rtsSingleHandleToDescriptorRange, void* pDepthStencilDescriptor) {
}

__declspec(dllexport) __cdecl void Ebitengine_ID3D12GraphicsCommandList_OMSetStencilRef(void* i, uint32_t stencilRef) {
}

__declspec(dllexport) __cdecl void Ebitengine_ID3D12GraphicsCommandList_Release(void* i) {
}

__declspec(dllexport) __cdecl uintptr_t Ebitengine_ID3D12GraphicsCommandList_Reset(void* i, void* pAllocator, void* pInitialStatexo) {
  return 0;
}

__declspec(dllexport) __cdecl void Ebitengine_ID3D12GraphicsCommandList_ResourceBarrier(void* i, uint32_t numBarriers, void* pBarriers) {
}

__declspec(dllexport) __cdecl void Ebitengine_ID3D12GraphicsCommandList_RSSetViewports(void* i, uint32_t numViewports, void* pViewports) {
}

__declspec(dllexport) __cdecl void Ebitengine_ID3D12GraphicsCommandList_RSSetScissorRects(void* i, uint32_t numRects, void* pRects) {
}

__declspec(dllexport) __cdecl void Ebitengine_ID3D12GraphicsCommandList_SetDescriptorHeaps(void* i, uint32_t numDescriptorHeaps, void* ppDescriptorHeaps) {
}

__declspec(dllexport) __cdecl void Ebitengine_ID3D12GraphicsCommandList_SetGraphicsRootDescriptorTable(void* i, uint32_t rootParameterIndex, uint64_t baseDescriptorPtr) {
}

__declspec(dllexport) __cdecl void Ebitengine_ID3D12GraphicsCommandList_SetGraphicsRootSignature(void* i, void* pRootSignature) {
}

__declspec(dllexport) __cdecl void Ebitengine_ID3D12GraphicsCommandList_SetPipelineState(void* i, void* pPipelineState) {
}
