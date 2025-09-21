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

//go:build microsoftgdk

package directx

// Some functions must be called with C++ directly for some reasons.
// Then, instead of calling them by LazyProc.Call, we have to defer this call to the C++ side.
// These functions are chosen based on the DirectX header file's implementation.
//
// These functions should be defined on the C++ side like this:
/*
extern "C" {

int32_t Ebitengine_D3D12_RESOURCE_STATE_PRESENT() {
	return static_cast<int32_t>(D3D12_RESOURCE_STATE_PRESENT);
}

void Ebitengine_ID3D12CommandQueue_ExecuteCommandLists(void* i, uint32_t numCommandLists, void* ppCommandLists) {
	static_cast<ID3D12CommandQueue*>(i)->ExecuteCommandLists(numCommandLists, static_cast<ID3D12CommandList**>(ppCommandLists));
}

uintptr_t Ebitengine_ID3D12CommandQueue_PresentX(void* i, uint32_t planeCount, void* pPlaneParameters, void* pPresentParameters) {
	auto r = static_cast<ID3D12CommandQueue*>(i)->PresentX(planeCount, static_cast<D3D12XBOX_PRESENT_PLANE_PARAMETERS*>(pPlaneParameters), static_cast<D3D12XBOX_PRESENT_PARAMETERS*>(pPresentParameters));
	return static_cast<uintptr_t>(r);
}

uint32_t Ebitengine_ID3D12CommandQueue_Release(void* i) {
	auto r = static_cast<ID3D12CommandQueue*>(i)->Release();
	return static_cast<uint32_t>(r);
}

uintptr_t Ebitengine_ID3D12CommandQueue_ResumeX(void* i) {
	auto r = static_cast<ID3D12CommandQueue*>(i)->ResumeX();
	return static_cast<uintptr_t>(r);
}

uintptr_t Ebitengine_ID3D12CommandQueue_Signal(void* i, void* pFence, uint64_t value) {
	auto r = static_cast<ID3D12CommandQueue*>(i)->Signal(static_cast<ID3D12Fence*>(pFence), value);
	return static_cast<uintptr_t>(r);
}

uintptr_t Ebitengine_ID3D12CommandQueue_SuspendX(void* i, uint32_t flags) {
	auto r = static_cast<ID3D12CommandQueue*>(i)->SuspendX(flags);
	return static_cast<uintptr_t>(r);
}

void Ebitengine_ID3D12GraphicsCommandList_ClearDepthStencilView(void* i, uintptr_t depthStencilView, int32_t clearFlags, float depth, uint8_t stencil, uint32_t numRects, void* pRects) {
    static_cast<ID3D12GraphicsCommandList*>(i)->ClearDepthStencilView(D3D12_CPU_DESCRIPTOR_HANDLE{ depthStencilView }, static_cast<D3D12_CLEAR_FLAGS>(clearFlags), depth, stencil, numRects, static_cast<D3D12_RECT*>(pRects));
}

void Ebitengine_ID3D12GraphicsCommandList_ClearRenderTargetView(void* i, uintptr_t pRenderTargetView, void* colorRGBA, uint32_t numRects, void* pRects) {
    static_cast<ID3D12GraphicsCommandList*>(i)->ClearRenderTargetView(D3D12_CPU_DESCRIPTOR_HANDLE{ pRenderTargetView }, static_cast<FLOAT*>(colorRGBA), numRects, static_cast<D3D12_RECT*>(pRects));
}

uintptr_t Ebitengine_ID3D12GraphicsCommandList_Close(void* i) {
    auto r = static_cast<ID3D12GraphicsCommandList*>(i)->Close();
    return uintptr_t(r);
}

void Ebitengine_ID3D12GraphicsCommandList_CopyTextureRegion(void* i, void* pDst, uint32_t dstX, uint32_t dstY, uint32_t dstZ, void* pSrc, void* pSrcBox) {
   static_cast<ID3D12GraphicsCommandList*>(i)->CopyTextureRegion(static_cast<D3D12_TEXTURE_COPY_LOCATION*>(pDst), dstX, dstY, dstZ, static_cast<D3D12_TEXTURE_COPY_LOCATION*>(pSrc), static_cast<D3D12_BOX*>(pSrcBox));
}

void Ebitengine_ID3D12GraphicsCommandList_DrawIndexedInstanced(void* i, uint32_t indexCountPerInstance, uint32_t instanceCount, uint32_t startIndexLocation, int32_t baseVertexLocation, uint32_t startInstanceLocation) {
    static_cast<ID3D12GraphicsCommandList*>(i)->DrawIndexedInstanced(indexCountPerInstance, instanceCount, startIndexLocation, baseVertexLocation, startInstanceLocation);
}

void Ebitengine_ID3D12GraphicsCommandList_IASetIndexBuffer(void* i, void* pView) {
    static_cast<ID3D12GraphicsCommandList*>(i)->IASetIndexBuffer(static_cast<D3D12_INDEX_BUFFER_VIEW*>(pView));
}

void Ebitengine_ID3D12GraphicsCommandList_IASetPrimitiveTopology(void* i, int32_t primitiveTopology) {
    static_cast<ID3D12GraphicsCommandList*>(i)->IASetPrimitiveTopology(static_cast<D3D12_PRIMITIVE_TOPOLOGY>(primitiveTopology));
}

void Ebitengine_ID3D12GraphicsCommandList_IASetVertexBuffers(void* i, uint32_t startSlot, uint32_t numViews, void* pViews) {
    static_cast<ID3D12GraphicsCommandList*>(i)->IASetVertexBuffers(startSlot, numViews, static_cast<D3D12_VERTEX_BUFFER_VIEW*>(pViews));
}

void Ebitengine_ID3D12GraphicsCommandList_OMSetRenderTargets(void* i, uint32_t numRenderTargetDescriptors, void* pRenderTargetDescriptors, int rtsSingleHandleToDescriptorRange, void* pDepthStencilDescriptor) {
    static_cast<ID3D12GraphicsCommandList*>(i)->OMSetRenderTargets(numRenderTargetDescriptors, static_cast<D3D12_CPU_DESCRIPTOR_HANDLE*>(pRenderTargetDescriptors), static_cast<BOOL>(rtsSingleHandleToDescriptorRange), static_cast<D3D12_CPU_DESCRIPTOR_HANDLE*>(pDepthStencilDescriptor));
}

void Ebitengine_ID3D12GraphicsCommandList_OMSetStencilRef(void* i, uint32_t stencilRef) {
    static_cast<ID3D12GraphicsCommandList*>(i)->OMSetStencilRef(stencilRef);
}

uint32_t Ebitengine_ID3D12GraphicsCommandList_Release(void* i) {
    return static_cast<uint32_t>(static_cast<ID3D12GraphicsCommandList*>(i)->Release());
}

uintptr_t Ebitengine_ID3D12GraphicsCommandList_Reset(void* i, void* pAllocator, void* pInitialState) {
    auto r = static_cast<ID3D12GraphicsCommandList*>(i)->Reset(static_cast<ID3D12CommandAllocator*>(pAllocator), static_cast<ID3D12PipelineState*>(pInitialState));
    return static_cast<uintptr_t>(r);
}

void Ebitengine_ID3D12GraphicsCommandList_ResourceBarrier(void* i, uint32_t numBarriers, void* pBarriers) {
    static_cast<ID3D12GraphicsCommandList*>(i)->ResourceBarrier(numBarriers, static_cast<D3D12_RESOURCE_BARRIER*>(pBarriers));
}

void Ebitengine_ID3D12GraphicsCommandList_RSSetViewports(void* i, uint32_t numViewports, void* pViewports) {
    static_cast<ID3D12GraphicsCommandList*>(i)->RSSetViewports(numViewports, static_cast<D3D12_VIEWPORT*>(pViewports));
}

void Ebitengine_ID3D12GraphicsCommandList_RSSetScissorRects(void* i, uint32_t numRects, void* pRects) {
    static_cast<ID3D12GraphicsCommandList*>(i)->RSSetScissorRects(numRects, static_cast<D3D12_RECT*>(pRects));
}

void Ebitengine_ID3D12GraphicsCommandList_SetDescriptorHeaps(void* i, uint32_t numDescriptorHeaps, void* ppDescriptorHeaps) {
   static_cast<ID3D12GraphicsCommandList*>(i)->SetDescriptorHeaps(numDescriptorHeaps, static_cast<ID3D12DescriptorHeap**>(ppDescriptorHeaps));
}

void Ebitengine_ID3D12GraphicsCommandList_SetGraphicsRootDescriptorTable(void* i, uint32_t rootParameterIndex, uint64_t baseDescriptorPtr) {
    static_cast<ID3D12GraphicsCommandList*>(i)->SetGraphicsRootDescriptorTable(rootParameterIndex, D3D12_GPU_DESCRIPTOR_HANDLE{ baseDescriptorPtr });
}

void Ebitengine_ID3D12GraphicsCommandList_SetGraphicsRootSignature(void* i, void* pRootSignature) {
    static_cast<ID3D12GraphicsCommandList*>(i)->SetGraphicsRootSignature(static_cast<ID3D12RootSignature*>(pRootSignature));
}

void Ebitengine_ID3D12GraphicsCommandList_SetPipelineState(void* i, void* pPipelineState) {
    static_cast<ID3D12GraphicsCommandList*>(i)->SetPipelineState(static_cast<ID3D12PipelineState*>(pPipelineState));
}

}
*/

// #include <stdint.h>
//
// #cgo noescape D3D12_RESOURCE_STATE_PRESENT
// #cgo nocallback D3D12_RESOURCE_STATE_PRESENT
// int32_t Ebitengine_D3D12_RESOURCE_STATE_PRESENT();
//
// #cgo noescape ID3D12CommandQueue_ExecuteCommandLists
// #cgo nocallback ID3D12CommandQueue_ExecuteCommandLists
// void Ebitengine_ID3D12CommandQueue_ExecuteCommandLists(void* i, uint32_t numCommandLists, void* ppCommandLists);
//
// #cgo noescape ID3D12CommandQueue_PresentX
// #cgo nocallback ID3D12CommandQueue_PresentX
// uintptr_t Ebitengine_ID3D12CommandQueue_PresentX(void* i, uint32_t planeCount, void* pPlaneParameters, void* pPresentParameters);
//
// #cgo noescape ID3D12CommandQueue_Release
// #cgo nocallback ID3D12CommandQueue_Release
// uint32_t Ebitengine_ID3D12CommandQueue_Release(void* i);
//
// #cgo noescape ID3D12CommandQueue_ResumeX
// #cgo nocallback ID3D12CommandQueue_ResumeX
// uintptr_t Ebitengine_ID3D12CommandQueue_ResumeX(void* i);
//
// #cgo noescape ID3D12CommandQueue_Signal
// #cgo nocallback ID3D12CommandQueue_Signal
// uintptr_t Ebitengine_ID3D12CommandQueue_Signal(void* i, void* pFence, uint64_t value);
//
// #cgo noescape ID3D12CommandQueue_SuspendX
// #cgo nocallback ID3D12CommandQueue_SuspendX
// uintptr_t Ebitengine_ID3D12CommandQueue_SuspendX(void* i, uint32_t flags);
//
// #cgo noescape ID3D12GraphicsCommandList_ClearDepthStencilView
// #cgo nocallback ID3D12GraphicsCommandList_ClearDepthStencilView
// void Ebitengine_ID3D12GraphicsCommandList_ClearDepthStencilView(void* i, uintptr_t depthStencilView, int32_t clearFlags, float depth, uint8_t stencil, uint32_t numRects, void* pRects);
//
// #cgo noescape ID3D12GraphicsCommandList_ClearRenderTargetView
// #cgo nocallback ID3D12GraphicsCommandList_ClearRenderTargetView
// void Ebitengine_ID3D12GraphicsCommandList_ClearRenderTargetView(void* i, uintptr_t pRenderTargetView, void* colorRGBA, uint32_t numRects, void* pRects);
//
// #cgo noescape ID3D12GraphicsCommandList_Close
// #cgo nocallback ID3D12GraphicsCommandList_Close
// uintptr_t Ebitengine_ID3D12GraphicsCommandList_Close(void* i);
//
// #cgo noescape ID3D12GraphicsCommandList_CopyTextureRegion
// #cgo nocallback ID3D12GraphicsCommandList_CopyTextureRegion
// void Ebitengine_ID3D12GraphicsCommandList_CopyTextureRegion(void* i, void* pDst, uint32_t dstX, uint32_t dstY, uint32_t dstZ, void* pSrc, void* pSrcBox);
//
// #cgo noescape ID3D12GraphicsCommandList_DrawIndexedInstanced
// #cgo nocallback ID3D12GraphicsCommandList_DrawIndexedInstanced
// void Ebitengine_ID3D12GraphicsCommandList_DrawIndexedInstanced(void* i, uint32_t indexCountPerInstance, uint32_t instanceCount, uint32_t startIndexLocation, int32_t baseVertexLocation, uint32_t startInstanceLocation);
//
// #cgo noescape ID3D12GraphicsCommandList_IASetIndexBuffer
// #cgo nocallback ID3D12GraphicsCommandList_IASetIndexBuffer
// void Ebitengine_ID3D12GraphicsCommandList_IASetIndexBuffer(void* i, void* pView);
//
// #cgo noescape ID3D12GraphicsCommandList_IASetPrimitiveTopology
// #cgo nocallback ID3D12GraphicsCommandList_IASetPrimitiveTopology
// void Ebitengine_ID3D12GraphicsCommandList_IASetPrimitiveTopology(void* i, int32_t primitiveTopology);
//
// #cgo noescape ID3D12GraphicsCommandList_IASetVertexBuffers
// #cgo nocallback ID3D12GraphicsCommandList_IASetVertexBuffers
// void Ebitengine_ID3D12GraphicsCommandList_IASetVertexBuffers(void* i, uint32_t startSlot, uint32_t numViews, void* pViews);
//
// #cgo noescape ID3D12GraphicsCommandList_OMSetRenderTargets
// #cgo nocallback ID3D12GraphicsCommandList_OMSetRenderTargets
// void Ebitengine_ID3D12GraphicsCommandList_OMSetRenderTargets(void* i, uint32_t numRenderTargetDescriptors, void* pRenderTargetDescriptors, int rtsSingleHandleToDescriptorRange, void* pDepthStencilDescriptor);
//
// #cgo noescape ID3D12GraphicsCommandList_OMSetStencilRef
// #cgo nocallback ID3D12GraphicsCommandList_OMSetStencilRef
// void Ebitengine_ID3D12GraphicsCommandList_OMSetStencilRef(void* i, uint32_t stencilRef);
//
// #cgo noescape ID3D12GraphicsCommandList_Release
// #cgo nocallback ID3D12GraphicsCommandList_Release
// uint32_t Ebitengine_ID3D12GraphicsCommandList_Release(void* i);
//
// #cgo noescape ID3D12GraphicsCommandList_Reset
// #cgo nocallback ID3D12GraphicsCommandList_Reset
// uintptr_t Ebitengine_ID3D12GraphicsCommandList_Reset(void* i, void* pAllocator, void* pInitialState);
//
// #cgo noescape ID3D12GraphicsCommandList_ResourceBarrier
// #cgo nocallback ID3D12GraphicsCommandList_ResourceBarrier
// void Ebitengine_ID3D12GraphicsCommandList_ResourceBarrier(void* i, uint32_t numBarriers, void* pBarriers);
//
// #cgo noescape ID3D12GraphicsCommandList_RSSetViewports
// #cgo nocallback ID3D12GraphicsCommandList_RSSetViewports
// void Ebitengine_ID3D12GraphicsCommandList_RSSetViewports(void* i, uint32_t numViewports, void* pViewports);
//
// #cgo noescape ID3D12GraphicsCommandList_RSSetScissorRects
// #cgo nocallback ID3D12GraphicsCommandList_RSSetScissorRects
// void Ebitengine_ID3D12GraphicsCommandList_RSSetScissorRects(void* i, uint32_t numRects, void* pRects);
//
// #cgo noescape ID3D12GraphicsCommandList_SetDescriptorHeaps
// #cgo nocallback ID3D12GraphicsCommandList_SetDescriptorHeaps
// void Ebitengine_ID3D12GraphicsCommandList_SetDescriptorHeaps(void* i, uint32_t numDescriptorHeaps, void* ppDescriptorHeaps);
//
// #cgo noescape ID3D12GraphicsCommandList_SetGraphicsRootDescriptorTable
// #cgo nocallback ID3D12GraphicsCommandList_SetGraphicsRootDescriptorTable
// void Ebitengine_ID3D12GraphicsCommandList_SetGraphicsRootDescriptorTable(void* i, uint32_t rootParameterIndex, uint64_t baseDescriptorPtr);
//
// #cgo noescape ID3D12GraphicsCommandList_SetGraphicsRootSignature
// #cgo nocallback ID3D12GraphicsCommandList_SetGraphicsRootSignature
// void Ebitengine_ID3D12GraphicsCommandList_SetGraphicsRootSignature(void* i, void* pRootSignature);
//
// #cgo noescape ID3D12GraphicsCommandList_SetPipelineState
// #cgo nocallback ID3D12GraphicsCommandList_SetPipelineState
// void Ebitengine_ID3D12GraphicsCommandList_SetPipelineState(void* i, void* pPipelineState);
import "C"

import (
	"unsafe"
)

func _D3D12_RESOURCE_STATE_PRESENT() _D3D12_RESOURCE_STATES {
	// This value depends on the environment.
	return _D3D12_RESOURCE_STATES(C.Ebitengine_D3D12_RESOURCE_STATE_PRESENT())
}

func _ID3D12CommandQueue_ExecuteCommandLists(i *_ID3D12CommandQueue, ppCommandLists []*_ID3D12GraphicsCommandList) {
	var ppCommandListsPtr **_ID3D12GraphicsCommandList
	if len(ppCommandLists) > 0 {
		ppCommandListsPtr = &ppCommandLists[0]
	}
	C.Ebitengine_ID3D12CommandQueue_ExecuteCommandLists(unsafe.Pointer(i), C.uint32_t(len(ppCommandLists)), unsafe.Pointer(ppCommandListsPtr))
}

func _ID3D12CommandQueue_PresentX(i *_ID3D12CommandQueue, planeCount uint32, pPlaneParameters *_D3D12XBOX_PRESENT_PLANE_PARAMETERS, pPresentParameters *_D3D12XBOX_PRESENT_PARAMETERS) uintptr {
	r := C.Ebitengine_ID3D12CommandQueue_PresentX(unsafe.Pointer(i), C.uint32_t(planeCount), unsafe.Pointer(pPlaneParameters), unsafe.Pointer(pPresentParameters))
	return uintptr(r)
}

func _ID3D12CommandQueue_Release(i *_ID3D12CommandQueue) uint32 {
	r := C.Ebitengine_ID3D12CommandQueue_Release(unsafe.Pointer(i))
	return uint32(r)
}

func _ID3D12CommandQueue_ResumeX(i *_ID3D12CommandQueue) uintptr {
	r := C.Ebitengine_ID3D12CommandQueue_ResumeX(unsafe.Pointer(i))
	return uintptr(r)
}

func _ID3D12CommandQueue_Signal(i *_ID3D12CommandQueue, pFence *_ID3D12Fence, value uint64) uintptr {
	r := C.Ebitengine_ID3D12CommandQueue_Signal(unsafe.Pointer(i), unsafe.Pointer(pFence), C.uint64_t(value))
	return uintptr(r)
}

func _ID3D12CommandQueue_SuspendX(i *_ID3D12CommandQueue, flags uint32) uintptr {
	r := C.Ebitengine_ID3D12CommandQueue_SuspendX(unsafe.Pointer(i), C.uint32_t(flags))
	return uintptr(r)
}

func _ID3D12GraphicsCommandList_ClearDepthStencilView(i *_ID3D12GraphicsCommandList, depthStencilView _D3D12_CPU_DESCRIPTOR_HANDLE, clearFlags _D3D12_CLEAR_FLAGS, depth float32, stencil uint8, rects []_D3D12_RECT) {
	var pRects *_D3D12_RECT
	if len(rects) > 0 {
		pRects = &rects[0]
	}
	C.Ebitengine_ID3D12GraphicsCommandList_ClearDepthStencilView(unsafe.Pointer(i), C.uintptr_t(depthStencilView.ptr), C.int32_t(clearFlags), C.float(depth), C.uint8_t(stencil), C.uint32_t(len(rects)), unsafe.Pointer(pRects))
}

func _ID3D12GraphicsCommandList_ClearRenderTargetView(i *_ID3D12GraphicsCommandList, pRenderTargetView _D3D12_CPU_DESCRIPTOR_HANDLE, colorRGBA [4]float32, rects []_D3D12_RECT) {
	var pRects *_D3D12_RECT
	if len(rects) > 0 {
		pRects = &rects[0]
	}
	C.Ebitengine_ID3D12GraphicsCommandList_ClearRenderTargetView(unsafe.Pointer(i), C.uintptr_t(pRenderTargetView.ptr), unsafe.Pointer(&colorRGBA[0]), C.uint32_t(len(rects)), unsafe.Pointer(pRects))
}

func _ID3D12GraphicsCommandList_Close(i *_ID3D12GraphicsCommandList) uintptr {
	r := C.Ebitengine_ID3D12GraphicsCommandList_Close(unsafe.Pointer(i))
	return uintptr(r)
}

func _ID3D12GraphicsCommandList_CopyTextureRegion(i *_ID3D12GraphicsCommandList, pDst unsafe.Pointer, dstX uint32, dstY uint32, dstZ uint32, pSrc unsafe.Pointer, pSrcBox *_D3D12_BOX) {
	C.Ebitengine_ID3D12GraphicsCommandList_CopyTextureRegion(unsafe.Pointer(i), pDst, C.uint32_t(dstX), C.uint32_t(dstY), C.uint32_t(dstZ), pSrc, unsafe.Pointer(pSrcBox))
}

func _ID3D12GraphicsCommandList_DrawIndexedInstanced(i *_ID3D12GraphicsCommandList, indexCountPerInstance uint32, instanceCount uint32, startIndexLocation uint32, baseVertexLocation int32, startInstanceLocation uint32) {
	C.Ebitengine_ID3D12GraphicsCommandList_DrawIndexedInstanced(unsafe.Pointer(i),
		C.uint32_t(indexCountPerInstance), C.uint32_t(instanceCount), C.uint32_t(startIndexLocation),
		C.int32_t(baseVertexLocation), C.uint32_t(startInstanceLocation))
}

func _ID3D12GraphicsCommandList_IASetIndexBuffer(i *_ID3D12GraphicsCommandList, pView *_D3D12_INDEX_BUFFER_VIEW) {
	C.Ebitengine_ID3D12GraphicsCommandList_IASetIndexBuffer(unsafe.Pointer(i), unsafe.Pointer(pView))
}

func _ID3D12GraphicsCommandList_IASetPrimitiveTopology(i *_ID3D12GraphicsCommandList, primitiveTopology _D3D_PRIMITIVE_TOPOLOGY) {
	C.Ebitengine_ID3D12GraphicsCommandList_IASetPrimitiveTopology(unsafe.Pointer(i), C.int32_t(primitiveTopology))
}

func _ID3D12GraphicsCommandList_IASetVertexBuffers(i *_ID3D12GraphicsCommandList, startSlot uint32, views []_D3D12_VERTEX_BUFFER_VIEW) {
	var pViews *_D3D12_VERTEX_BUFFER_VIEW
	if len(views) > 0 {
		pViews = &views[0]
	}
	C.Ebitengine_ID3D12GraphicsCommandList_IASetVertexBuffers(unsafe.Pointer(i), C.uint32_t(startSlot), C.uint32_t(len(views)), unsafe.Pointer(pViews))
}

func _ID3D12GraphicsCommandList_OMSetRenderTargets(i *_ID3D12GraphicsCommandList, renderTargetDescriptors []_D3D12_CPU_DESCRIPTOR_HANDLE, rtsSingleHandleToDescriptorRange bool, pDepthStencilDescriptor *_D3D12_CPU_DESCRIPTOR_HANDLE) {
	var pRenderTargetDescriptors *_D3D12_CPU_DESCRIPTOR_HANDLE
	if len(renderTargetDescriptors) > 0 {
		pRenderTargetDescriptors = &renderTargetDescriptors[0]
	}
	v := 0
	if rtsSingleHandleToDescriptorRange {
		v = 1
	}
	C.Ebitengine_ID3D12GraphicsCommandList_OMSetRenderTargets(unsafe.Pointer(i), C.uint32_t(len(renderTargetDescriptors)), unsafe.Pointer(pRenderTargetDescriptors), C.int(v), unsafe.Pointer(pDepthStencilDescriptor))
}

func _ID3D12GraphicsCommandList_OMSetStencilRef(i *_ID3D12GraphicsCommandList, stencilRef uint32) {
	C.Ebitengine_ID3D12GraphicsCommandList_OMSetStencilRef(unsafe.Pointer(i), C.uint32_t(stencilRef))
}

func _ID3D12GraphicsCommandList_Release(i *_ID3D12GraphicsCommandList) uint32 {
	return uint32(C.Ebitengine_ID3D12GraphicsCommandList_Release(unsafe.Pointer(i)))
}

func _ID3D12GraphicsCommandList_Reset(i *_ID3D12GraphicsCommandList, pAllocator *_ID3D12CommandAllocator, pInitialState *_ID3D12PipelineState) uintptr {
	r := C.Ebitengine_ID3D12GraphicsCommandList_Reset(unsafe.Pointer(i), unsafe.Pointer(pAllocator), unsafe.Pointer(pInitialState))
	return uintptr(r)
}

func _ID3D12GraphicsCommandList_ResourceBarrier(i *_ID3D12GraphicsCommandList, barriers []_D3D12_RESOURCE_BARRIER_Transition) {
	var pBarriers *_D3D12_RESOURCE_BARRIER_Transition
	if len(barriers) > 0 {
		pBarriers = &barriers[0]
	}
	C.Ebitengine_ID3D12GraphicsCommandList_ResourceBarrier(unsafe.Pointer(i), C.uint32_t(len(barriers)), unsafe.Pointer(pBarriers))
}

func _ID3D12GraphicsCommandList_RSSetViewports(i *_ID3D12GraphicsCommandList, viewports []_D3D12_VIEWPORT) {
	var pViewports *_D3D12_VIEWPORT
	if len(viewports) > 0 {
		pViewports = &viewports[0]
	}
	C.Ebitengine_ID3D12GraphicsCommandList_RSSetViewports(unsafe.Pointer(i), C.uint32_t(len(viewports)), unsafe.Pointer(pViewports))
}

func _ID3D12GraphicsCommandList_RSSetScissorRects(i *_ID3D12GraphicsCommandList, rects []_D3D12_RECT) {
	var pRects *_D3D12_RECT
	if len(rects) > 0 {
		pRects = &rects[0]
	}
	C.Ebitengine_ID3D12GraphicsCommandList_RSSetScissorRects(unsafe.Pointer(i), C.uint32_t(len(rects)), unsafe.Pointer(pRects))
}

func _ID3D12GraphicsCommandList_SetDescriptorHeaps(i *_ID3D12GraphicsCommandList, descriptorHeaps []*_ID3D12DescriptorHeap) {
	var ppDescriptorHeaps **_ID3D12DescriptorHeap
	if len(descriptorHeaps) > 0 {
		ppDescriptorHeaps = &descriptorHeaps[0]
	}
	C.Ebitengine_ID3D12GraphicsCommandList_SetDescriptorHeaps(unsafe.Pointer(i), C.uint32_t(len(descriptorHeaps)), unsafe.Pointer(ppDescriptorHeaps))
}

func _ID3D12GraphicsCommandList_SetGraphicsRootDescriptorTable(i *_ID3D12GraphicsCommandList, rootParameterIndex uint32, baseDescriptor _D3D12_GPU_DESCRIPTOR_HANDLE) {
	C.Ebitengine_ID3D12GraphicsCommandList_SetGraphicsRootDescriptorTable(unsafe.Pointer(i), C.uint32_t(rootParameterIndex), C.uint64_t(baseDescriptor.ptr))
}

func _ID3D12GraphicsCommandList_SetGraphicsRootSignature(i *_ID3D12GraphicsCommandList, pRootSignature *_ID3D12RootSignature) {
	C.Ebitengine_ID3D12GraphicsCommandList_SetGraphicsRootSignature(unsafe.Pointer(i), unsafe.Pointer(pRootSignature))
}

func _ID3D12GraphicsCommandList_SetPipelineState(i *_ID3D12GraphicsCommandList, pPipelineState *_ID3D12PipelineState) {
	C.Ebitengine_ID3D12GraphicsCommandList_SetPipelineState(unsafe.Pointer(i), unsafe.Pointer(pPipelineState))
}
