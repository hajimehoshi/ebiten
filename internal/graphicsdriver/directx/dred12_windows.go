// Copyright 2026 The Ebitengine Authors
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
	"log/slog"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Device Removed Extended Data (DRED): records auto-breadcrumbs (the GPU's command history) and the
// page-fault address that preceded a device removal. Enabled via the EBITENGINE_DIRECTX dred token to
// diagnose a DXGI_ERROR_DEVICE_REMOVED — a non-zero page-fault VA means the GPU dereferenced a bad
// address (a bug in the recorded command stream), while no page fault points at a driver-internal
// failure.

var (
	_IID_ID3D12DeviceRemovedExtendedDataSettings = windows.GUID{Data1: 0x82bc481c, Data2: 0x6b9b, Data3: 0x4030, Data4: [...]byte{0xae, 0xdb, 0x7e, 0xe3, 0xd1, 0xdf, 0x1e, 0x63}}
	_IID_ID3D12DeviceRemovedExtendedData         = windows.GUID{Data1: 0x98931d33, Data2: 0x5ae8, Data3: 0x4791, Data4: [...]byte{0xaa, 0x3c, 0x1a, 0x73, 0xa2, 0x93, 0x4e, 0x71}}
)

type _D3D12_DRED_ENABLEMENT int32

const (
	_D3D12_DRED_ENABLEMENT_FORCED_ON _D3D12_DRED_ENABLEMENT = 2
)

type _ID3D12DeviceRemovedExtendedDataSettings struct {
	vtbl *_ID3D12DeviceRemovedExtendedDataSettings_Vtbl
}

type _ID3D12DeviceRemovedExtendedDataSettings_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	SetAutoBreadcrumbsEnablement uintptr
	SetPageFaultEnablement       uintptr
	SetWatsonDumpEnablement      uintptr
}

func (i *_ID3D12DeviceRemovedExtendedDataSettings) SetAutoBreadcrumbsEnablement(e _D3D12_DRED_ENABLEMENT) {
	_, _, _ = syscall.Syscall(i.vtbl.SetAutoBreadcrumbsEnablement, 2, uintptr(unsafe.Pointer(i)), uintptr(e), 0)
}

func (i *_ID3D12DeviceRemovedExtendedDataSettings) SetPageFaultEnablement(e _D3D12_DRED_ENABLEMENT) {
	_, _, _ = syscall.Syscall(i.vtbl.SetPageFaultEnablement, 2, uintptr(unsafe.Pointer(i)), uintptr(e), 0)
}

func (i *_ID3D12DeviceRemovedExtendedDataSettings) Release() {
	_, _, _ = syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
}

func _D3D12GetDREDSettings() (*_ID3D12DeviceRemovedExtendedDataSettings, error) {
	var settings *_ID3D12DeviceRemovedExtendedDataSettings
	r, _, _ := procD3D12GetDebugInterface.Call(
		uintptr(unsafe.Pointer(&_IID_ID3D12DeviceRemovedExtendedDataSettings)),
		uintptr(unsafe.Pointer(&settings)))
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: D3D12GetDebugInterface for DRED settings failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return settings, nil
}

type _D3D12_DRED_ALLOCATION_NODE struct {
	ObjectNameA    *byte
	ObjectNameW    *uint16
	AllocationType int32
	pNext          *_D3D12_DRED_ALLOCATION_NODE
}

type _D3D12_DRED_PAGE_FAULT_OUTPUT struct {
	PageFaultVA                    uint64
	pHeadExistingAllocationNode    *_D3D12_DRED_ALLOCATION_NODE
	pHeadRecentFreedAllocationNode *_D3D12_DRED_ALLOCATION_NODE
}

type _D3D12_AUTO_BREADCRUMB_NODE struct {
	pCommandListDebugNameA  *byte
	pCommandListDebugNameW  *uint16
	pCommandQueueDebugNameA *byte
	pCommandQueueDebugNameW *uint16
	pCommandList            uintptr
	pCommandQueue           uintptr
	BreadcrumbCount         uint32
	pLastBreadcrumbValue    *uint32
	pCommandHistory         *int32
	pNext                   *_D3D12_AUTO_BREADCRUMB_NODE
}

type _D3D12_DRED_AUTO_BREADCRUMBS_OUTPUT struct {
	pHeadAutoBreadcrumbNode *_D3D12_AUTO_BREADCRUMB_NODE
}

type _ID3D12DeviceRemovedExtendedData struct {
	vtbl *_ID3D12DeviceRemovedExtendedData_Vtbl
}

type _ID3D12DeviceRemovedExtendedData_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	GetAutoBreadcrumbsOutput     uintptr
	GetPageFaultAllocationOutput uintptr
}

func (i *_ID3D12DeviceRemovedExtendedData) GetAutoBreadcrumbsOutput(output *_D3D12_DRED_AUTO_BREADCRUMBS_OUTPUT) error {
	r, _, _ := syscall.Syscall(i.vtbl.GetAutoBreadcrumbsOutput, 2, uintptr(unsafe.Pointer(i)), uintptr(unsafe.Pointer(output)), 0)
	if uint32(r) != uint32(windows.S_OK) {
		return fmt.Errorf("directx: ID3D12DeviceRemovedExtendedData::GetAutoBreadcrumbsOutput failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return nil
}

func (i *_ID3D12DeviceRemovedExtendedData) GetPageFaultAllocationOutput(output *_D3D12_DRED_PAGE_FAULT_OUTPUT) error {
	r, _, _ := syscall.Syscall(i.vtbl.GetPageFaultAllocationOutput, 2, uintptr(unsafe.Pointer(i)), uintptr(unsafe.Pointer(output)), 0)
	if uint32(r) != uint32(windows.S_OK) {
		return fmt.Errorf("directx: ID3D12DeviceRemovedExtendedData::GetPageFaultAllocationOutput failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return nil
}

func (i *_ID3D12DeviceRemovedExtendedData) Release() {
	_, _, _ = syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
}

func dredAllocationName(w *uint16, a *byte) string {
	if w != nil {
		return windows.UTF16PtrToString(w)
	}
	if a != nil {
		return windows.BytePtrToString(a)
	}
	return "<unnamed>"
}

// dredBreadcrumbOpName names the common D3D12_AUTO_BREADCRUMB_OP values; others are printed numerically.
func dredBreadcrumbOpName(op int32) string {
	switch op {
	case 3:
		return "DrawInstanced"
	case 4:
		return "DrawIndexedInstanced"
	case 6:
		return "Dispatch"
	case 7:
		return "CopyBufferRegion"
	case 8:
		return "CopyTextureRegion"
	case 9:
		return "CopyResource"
	case 12:
		return "ClearRenderTargetView"
	case 14:
		return "ClearDepthStencilView"
	case 15:
		return "ResourceBarrier"
	case 17:
		return "Present"
	default:
		return fmt.Sprintf("op(%d)", op)
	}
}

// dumpDRED logs the page-fault address and the GPU's last breadcrumb after a device removal.
// It is a no-op unless DRED was enabled at device creation.
func (g *graphics12) dumpDRED() {
	p, err := g.device.QueryInterface(&_IID_ID3D12DeviceRemovedExtendedData)
	if err != nil {
		slog.Error("directx: DRED data interface is unavailable", "error", err)
		return
	}
	dred := (*_ID3D12DeviceRemovedExtendedData)(p)
	defer dred.Release()

	var pf _D3D12_DRED_PAGE_FAULT_OUTPUT
	if err := dred.GetPageFaultAllocationOutput(&pf); err != nil {
		slog.Error("directx: DRED page-fault output is unavailable", "error", err)
	} else {
		slog.Error("directx: DRED page fault", "gpu_va", fmt.Sprintf("0x%x", pf.PageFaultVA))
		for n := pf.pHeadExistingAllocationNode; n != nil; n = n.pNext {
			slog.Error("directx: DRED existing allocation near fault",
				"name", dredAllocationName(n.ObjectNameW, n.ObjectNameA), "type", n.AllocationType)
		}
		for n := pf.pHeadRecentFreedAllocationNode; n != nil; n = n.pNext {
			slog.Error("directx: DRED recently-freed allocation near fault",
				"name", dredAllocationName(n.ObjectNameW, n.ObjectNameA), "type", n.AllocationType)
		}
	}

	var bc _D3D12_DRED_AUTO_BREADCRUMBS_OUTPUT
	if err := dred.GetAutoBreadcrumbsOutput(&bc); err != nil {
		slog.Error("directx: DRED breadcrumb output is unavailable", "error", err)
		return
	}
	for n := bc.pHeadAutoBreadcrumbNode; n != nil; n = n.pNext {
		var last uint32
		if n.pLastBreadcrumbValue != nil {
			last = *n.pLastBreadcrumbValue
		}
		// The op at index last is the first one the GPU had not finished, i.e. where it faulted.
		faulting := "<none>"
		if n.pCommandHistory != nil && last < n.BreadcrumbCount {
			ops := unsafe.Slice(n.pCommandHistory, n.BreadcrumbCount)
			faulting = dredBreadcrumbOpName(ops[last])
		}
		slog.Error("directx: DRED breadcrumb",
			"completed", last, "total", n.BreadcrumbCount, "faulting_op", faulting)
	}
}
