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

type _D3DCOMPILE uint32

const (
	_D3DCOMPILE_OPTIMIZATION_LEVEL3 _D3DCOMPILE = (1 << 15)
)

type _D3D_FEATURE_LEVEL int32

const (
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
	// https://github.com/MicrosoftDocs/sdk-api/blob/docs/sdk-api-src/content/appnotify/nf-appnotify-registerappstatechangenotification.md
	d3dcompiler = windows.NewLazySystemDLL("d3dcompiler_47.dll")

	procD3DCompile = d3dcompiler.NewProc("D3DCompile")
)

func _D3DCompile(srcData []byte, sourceName string, pDefines []_D3D_SHADER_MACRO, pInclude unsafe.Pointer, entryPoint string, target string, flags1 uint32, flags2 uint32) (*_ID3DBlob, error) {
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

func (i *_ID3DBlob) GetBufferPointer() uintptr {
	r, _, _ := syscall.Syscall(i.vtbl.GetBufferPointer, 1, uintptr(unsafe.Pointer(i)),
		0, 0)
	return r
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
