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
// +build microsoftgdk

package directx

// Some functions of ID3D12GraphicsCommandList has additional logics besides the original COM function call.
// Then, instead of calling them by LazyProc.Call, we have to defer this call to the C++ side.
// These functions are chosen based on the DirectX header file's implementation.
//
// These functions should be defined on the C++ side like this:
//
// extern "C" {
//     void Ebitengine_ID3D12GraphicsCommandList_DrawIndexedInstanced(void* i, uint32_t indexCountPerInstance, uint32_t instanceCount, uint32_t startIndexLocation, int32_t baseVertexLocation, uint32_t startInstanceLocation) {
//         static_cast<ID3D12GraphicsCommandList*>(i)->DrawIndexedInstanced(indexCountPerInstance, instanceCount, startIndexLocation, baseVertexLocation, startInstanceLocation);
//     }
//     void Ebitengine_ID3D12GraphicsCommandList_IASetIndexBuffer(void* i, void* pView) {
//         static_cast<ID3D12GraphicsCommandList*>(i)->IASetIndexBuffer(static_cast<D3D12_INDEX_BUFFER_VIEW*>(pView));
//     }
//     void Ebitengine_ID3D12GraphicsCommandList_IASetPrimitiveTopology(void* i, int32_t primitiveTopology) {
//         static_cast<ID3D12GraphicsCommandList*>(i)->IASetPrimitiveTopology(static_cast<D3D12_PRIMITIVE_TOPOLOGY>(primitiveTopology));
//     }
//     void Ebitengine_ID3D12GraphicsCommandList_IASetVertexBuffers(void* i, uint32_t startSlot, uint32_t numViews, void* pViews) {
//         static_cast<ID3D12GraphicsCommandList*>(i)->IASetVertexBuffers(startSlot, numViews, static_cast<D3D12_VERTEX_BUFFER_VIEW*>(pViews));
//     }
//     void Ebitengine_ID3D12GraphicsCommandList_OMSetStencilRef(void* i, uint32_t stencilRef) {
//         static_cast<ID3D12GraphicsCommandList*>(i)->OMSetStencilRef(stencilRef);
//     }
//     void Ebitengine_ID3D12GraphicsCommandList_SetGraphicsRootDescriptorTable(void* i, uint32_t rootParameterIndex, uint64_t baseDescriptorPtr) {
//         static_cast<ID3D12GraphicsCommandList*>(i)->SetGraphicsRootDescriptorTable(rootParameterIndex, D3D12_GPU_DESCRIPTOR_HANDLE{ baseDescriptorPtr });
//     }
//     void Ebitengine_ID3D12GraphicsCommandList_SetPipelineState(void* i, void* pPipelineState) {
//         static_cast<ID3D12GraphicsCommandList*>(i)->SetPipelineState(static_cast<ID3D12PipelineState*>(pPipelineState));
//     }
// }

// #include <stdint.h>
//
// void Ebitengine_ID3D12GraphicsCommandList_DrawIndexedInstanced(void* i, uint32_t indexCountPerInstance, uint32_t instanceCount, uint32_t startIndexLocation, int32_t baseVertexLocation, uint32_t startInstanceLocation);
// void Ebitengine_ID3D12GraphicsCommandList_IASetIndexBuffer(void* i, void* pView);
// void Ebitengine_ID3D12GraphicsCommandList_IASetPrimitiveTopology(void* i, int32_t primitiveTopology);
// void Ebitengine_ID3D12GraphicsCommandList_IASetVertexBuffers(void* i, uint32_t startSlot, uint32_t numViews, void* pViews);
// void Ebitengine_ID3D12GraphicsCommandList_OMSetStencilRef(void* i, uint32_t stencilRef);
// void Ebitengine_ID3D12GraphicsCommandList_SetGraphicsRootDescriptorTable(void* i, uint32_t rootParameterIndex, uint64_t baseDescriptorPtr);
// void Ebitengine_ID3D12GraphicsCommandList_SetPipelineState(void* i, void* pPipelineState);
import "C"

import (
	"unsafe"
)

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

func _ID3D12GraphicsCommandList_IASetVertexBuffers(i *_ID3D12GraphicsCommandList, startSlot uint32, pViews []_D3D12_VERTEX_BUFFER_VIEW) {
	C.Ebitengine_ID3D12GraphicsCommandList_IASetVertexBuffers(unsafe.Pointer(i), C.uint32_t(startSlot), C.uint32_t(len(pViews)), unsafe.Pointer(&pViews[0]))
}

func _ID3D12GraphicsCommandList_OMSetStencilRef(i *_ID3D12GraphicsCommandList, stencilRef uint32) {
	C.Ebitengine_ID3D12GraphicsCommandList_OMSetStencilRef(unsafe.Pointer(i), C.uint32_t(stencilRef))
}

func _ID3D12GraphicsCommandList_SetGraphicsRootDescriptorTable(i *_ID3D12GraphicsCommandList, rootParameterIndex uint32, baseDescriptor _D3D12_GPU_DESCRIPTOR_HANDLE) {
	C.Ebitengine_ID3D12GraphicsCommandList_SetGraphicsRootDescriptorTable(unsafe.Pointer(i), C.uint32_t(rootParameterIndex), C.uint64_t(baseDescriptor.ptr))
}

func _ID3D12GraphicsCommandList_SetPipelineState(i *_ID3D12GraphicsCommandList, pPipelineState *_ID3D12PipelineState) {
	C.Ebitengine_ID3D12GraphicsCommandList_SetPipelineState(unsafe.Pointer(i), unsafe.Pointer(pPipelineState))
}
