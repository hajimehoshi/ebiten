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

//go:build !microsoftgdk

package directx

import (
	"unsafe"
)

func _D3D12_RESOURCE_STATE_PRESENT() _D3D12_RESOURCE_STATES {
	return 0
}

func _ID3D12CommandQueue_ExecuteCommandLists(i *_ID3D12CommandQueue, ppCommandLists []*_ID3D12GraphicsCommandList) {
	panic("directx: not implemented")
}

func _ID3D12CommandQueue_PresentX(i *_ID3D12CommandQueue, planeCount uint32, pPlaneParameters *_D3D12XBOX_PRESENT_PLANE_PARAMETERS, pPresentParameters *_D3D12XBOX_PRESENT_PARAMETERS) uintptr {
	panic("directx: not implemented")
}

func _ID3D12CommandQueue_Release(i *_ID3D12CommandQueue) uint32 {
	panic("directx: not implemented")
}

func _ID3D12CommandQueue_ResumeX(i *_ID3D12CommandQueue) uintptr {
	panic("directx: not implemented")
}

func _ID3D12CommandQueue_Signal(i *_ID3D12CommandQueue, pFence *_ID3D12Fence, value uint64) uintptr {
	panic("directx: not implemented")
}

func _ID3D12CommandQueue_SuspendX(i *_ID3D12CommandQueue, flags uint32) uintptr {
	panic("directx: not implemented")
}

func _ID3D12GraphicsCommandList_ClearDepthStencilView(i *_ID3D12GraphicsCommandList, depthStencilView _D3D12_CPU_DESCRIPTOR_HANDLE, clearFlags _D3D12_CLEAR_FLAGS, depth float32, stencil uint8, rects []_D3D12_RECT) {
	panic("directx: not implemented")
}

func _ID3D12GraphicsCommandList_ClearRenderTargetView(i *_ID3D12GraphicsCommandList, pRenderTargetView _D3D12_CPU_DESCRIPTOR_HANDLE, colorRGBA [4]float32, rects []_D3D12_RECT) {
	panic("directx: not implemented")
}

func _ID3D12GraphicsCommandList_Close(i *_ID3D12GraphicsCommandList) uintptr {
	panic("directx: not implemented")
}

func _ID3D12GraphicsCommandList_CopyTextureRegion(i *_ID3D12GraphicsCommandList, pDst unsafe.Pointer, dstX uint32, dstY uint32, dstZ uint32, pSrc unsafe.Pointer, pSrcBox *_D3D12_BOX) {
	panic("directx: not implemented")
}

func _ID3D12GraphicsCommandList_DrawIndexedInstanced(i *_ID3D12GraphicsCommandList, indexCountPerInstance uint32, instanceCount uint32, startIndexLocation uint32, baseVertexLocation int32, startInstanceLocation uint32) {
	panic("directx: not implemented")
}

func _ID3D12GraphicsCommandList_IASetIndexBuffer(i *_ID3D12GraphicsCommandList, pView *_D3D12_INDEX_BUFFER_VIEW) {
	panic("directx: not implemented")
}

func _ID3D12GraphicsCommandList_IASetPrimitiveTopology(i *_ID3D12GraphicsCommandList, primitiveTopology _D3D_PRIMITIVE_TOPOLOGY) {
	panic("directx: not implemented")
}

func _ID3D12GraphicsCommandList_IASetVertexBuffers(i *_ID3D12GraphicsCommandList, startSlot uint32, pViews []_D3D12_VERTEX_BUFFER_VIEW) {
	panic("directx: not implemented")
}

func _ID3D12GraphicsCommandList_OMSetRenderTargets(i *_ID3D12GraphicsCommandList, renderTargetDescriptors []_D3D12_CPU_DESCRIPTOR_HANDLE, rtsSingleHandleToDescriptorRange bool, pDepthStencilDescriptor *_D3D12_CPU_DESCRIPTOR_HANDLE) {
	panic("directx: not implemented")
}

func _ID3D12GraphicsCommandList_OMSetStencilRef(i *_ID3D12GraphicsCommandList, stencilRef uint32) {
	panic("directx: not implemented")
}

func _ID3D12GraphicsCommandList_Release(i *_ID3D12GraphicsCommandList) uint32 {
	panic("directx: not implemented")
}

func _ID3D12GraphicsCommandList_Reset(i *_ID3D12GraphicsCommandList, pAllocator *_ID3D12CommandAllocator, pInitialState *_ID3D12PipelineState) uintptr {
	panic("directx: not implemented")
}

func _ID3D12GraphicsCommandList_ResourceBarrier(i *_ID3D12GraphicsCommandList, barriers []_D3D12_RESOURCE_BARRIER_Transition) {
	panic("directx: not implemented")
}

func _ID3D12GraphicsCommandList_RSSetViewports(i *_ID3D12GraphicsCommandList, viewports []_D3D12_VIEWPORT) {
	panic("directx: not implemented")
}

func _ID3D12GraphicsCommandList_RSSetScissorRects(i *_ID3D12GraphicsCommandList, rects []_D3D12_RECT) {
	panic("directx: not implemented")
}

func _ID3D12GraphicsCommandList_SetDescriptorHeaps(i *_ID3D12GraphicsCommandList, descriptorHeaps []*_ID3D12DescriptorHeap) {
	panic("directx: not implemented")
}

func _ID3D12GraphicsCommandList_SetGraphicsRootDescriptorTable(i *_ID3D12GraphicsCommandList, rootParameterIndex uint32, baseDescriptor _D3D12_GPU_DESCRIPTOR_HANDLE) {
	panic("directx: not implemented")
}

func _ID3D12GraphicsCommandList_SetGraphicsRootSignature(i *_ID3D12GraphicsCommandList, pRootSignature *_ID3D12RootSignature) {
	panic("directx: not implemented")
}

func _ID3D12GraphicsCommandList_SetPipelineState(i *_ID3D12GraphicsCommandList, pPipelineState *_ID3D12PipelineState) {
	panic("directx: not implemented")
}
