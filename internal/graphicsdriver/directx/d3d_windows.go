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
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Reference:
// * https://github.com/wine-mirror/wine/blob/master/include/d3dcommon.idl

type _D3DCOMPILE uint32

const (
	_D3DCOMPILE_OPTIMIZATION_LEVEL3 _D3DCOMPILE = (1 << 15)
)

type _D3D_DRIVER_TYPE int32

const (
	_D3D_DRIVER_TYPE_UNKNOWN _D3D_DRIVER_TYPE = iota
	_D3D_DRIVER_TYPE_HARDWARE
	_D3D_DRIVER_TYPE_REFERENCE
	_D3D_DRIVER_TYPE_NULL
	_D3D_DRIVER_TYPE_SOFTWARE
	_D3D_DRIVER_TYPE_WARP
)

type _D3D_FEATURE_LEVEL int32

const (
	_D3D_FEATURE_LEVEL_9_1  _D3D_FEATURE_LEVEL = 0x9100
	_D3D_FEATURE_LEVEL_9_2  _D3D_FEATURE_LEVEL = 0x9200
	_D3D_FEATURE_LEVEL_9_3  _D3D_FEATURE_LEVEL = 0x9300
	_D3D_FEATURE_LEVEL_10_0 _D3D_FEATURE_LEVEL = 0xa000
	_D3D_FEATURE_LEVEL_10_1 _D3D_FEATURE_LEVEL = 0xa100
	_D3D_FEATURE_LEVEL_11_0 _D3D_FEATURE_LEVEL = 0xb000
	_D3D_FEATURE_LEVEL_11_1 _D3D_FEATURE_LEVEL = 0xb100
	_D3D_FEATURE_LEVEL_12_0 _D3D_FEATURE_LEVEL = 0xc000
	_D3D_FEATURE_LEVEL_12_1 _D3D_FEATURE_LEVEL = 0xc100
	_D3D_FEATURE_LEVEL_12_2 _D3D_FEATURE_LEVEL = 0xc200
)

type _D3D_PRIMITIVE_TOPOLOGY int32

const (
	_D3D_PRIMITIVE_TOPOLOGY_TRIANGLELIST _D3D_PRIMITIVE_TOPOLOGY = 4
)

type _D3D_ROOT_SIGNATURE_VERSION int32

const (
	_D3D_ROOT_SIGNATURE_VERSION_1_0 _D3D_ROOT_SIGNATURE_VERSION = 0x1
)

var (
	procD3DCompile    *windows.LazyProc
	procD3DCreateBlob *windows.LazyProc
)

func init() {
	var d3dcompiler *windows.LazyDLL

	// Enumerate possible DLL names for d3dcompiler_*.dll.
	// https://walbourn.github.io/hlsl-fxc-and-d3dcompile/
	for _, name := range []string{"d3dcompiler_47.dll", "d3dcompiler_46.dll", "d3dcompiler_43.dll"} {
		dll := windows.NewLazySystemDLL(name)
		if err := dll.Load(); err != nil {
			continue
		}
		d3dcompiler = dll
		break
	}

	if d3dcompiler == nil {
		return
	}

	procD3DCompile = d3dcompiler.NewProc("D3DCompile")
	procD3DCreateBlob = d3dcompiler.NewProc("D3DCreateBlob")
}

func isD3DCompilerDLLAvailable() bool {
	return procD3DCompile != nil
}

func _D3DCompile(srcData []byte, sourceName string, pDefines []_D3D_SHADER_MACRO, pInclude unsafe.Pointer, entryPoint string, target string, flags1 uint32, flags2 uint32) (*_ID3DBlob, error) {
	if !isD3DCompilerDLLAvailable() {
		return nil, fmt.Errorf("directx: d3dcompiler_*.dll is missing in this environment")
	}

	// TODO: Define _ID3DInclude for pInclude, but is it possible in Go?

	var defs unsafe.Pointer
	if len(pDefines) > 0 {
		defs = unsafe.Pointer(&pDefines[0])
	}
	sourceNameBytes := append([]byte(sourceName), 0)
	entryPointBytes := append([]byte(entryPoint), 0)
	targetBytes := append([]byte(target), 0)
	var code *_ID3DBlob
	var errorMsgs *_ID3DBlob
	r, _, _ := procD3DCompile.Call(
		uintptr(unsafe.Pointer(&srcData[0])), uintptr(len(srcData)), uintptr(unsafe.Pointer(&sourceNameBytes[0])),
		uintptr(defs), uintptr(unsafe.Pointer(pInclude)), uintptr(unsafe.Pointer(&entryPointBytes[0])),
		uintptr(unsafe.Pointer(&targetBytes[0])), uintptr(flags1), uintptr(flags2),
		uintptr(unsafe.Pointer(&code)), uintptr(unsafe.Pointer(&errorMsgs)))
	runtime.KeepAlive(pDefines)
	runtime.KeepAlive(pInclude)
	runtime.KeepAlive(sourceNameBytes)
	runtime.KeepAlive(entryPointBytes)
	runtime.KeepAlive(targetBytes)
	if uint32(r) != uint32(windows.S_OK) {
		if errorMsgs != nil {
			defer errorMsgs.Release()
			return nil, fmt.Errorf("directx: D3DCompile failed: %s: %w", errorMsgs.String(), handleError(windows.Handle(uint32(r))))
		}
		return nil, fmt.Errorf("directx: D3DCompile failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return code, nil
}

func _D3DCreateBlob(size uint) (*_ID3DBlob, error) {
	if !isD3DCompilerDLLAvailable() {
		return nil, fmt.Errorf("directx: d3dcompiler_*.dll is missing in this environment")
	}

	var blob *_ID3DBlob
	r, _, _ := procD3DCreateBlob.Call(uintptr(size), uintptr(unsafe.Pointer(&blob)))
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("directx: D3DCreateBlob failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return blob, nil
}

type _D3D_SHADER_MACRO struct {
	Name       *byte
	Definition *byte
}

type _ID3DBlob struct {
	vtbl *_ID3DBlob_Vtbl
}

type _ID3DBlob_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	GetBufferPointer uintptr
	GetBufferSize    uintptr
}

func (i *_ID3DBlob) AddRef() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.AddRef, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}

func (i *_ID3DBlob) GetBufferPointer() unsafe.Pointer {
	r, _, _ := syscall.Syscall(i.vtbl.GetBufferPointer, 1, uintptr(unsafe.Pointer(i)),
		0, 0)
	return unsafe.Pointer(r)
}

func (i *_ID3DBlob) GetBufferSize() uintptr {
	r, _, _ := syscall.Syscall(i.vtbl.GetBufferSize, 1, uintptr(unsafe.Pointer(i)),
		0, 0)
	return r
}

func (i *_ID3DBlob) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}

func (i *_ID3DBlob) String() string {
	return string(unsafe.Slice((*byte)(unsafe.Pointer(i.GetBufferPointer())), i.GetBufferSize()))
}
