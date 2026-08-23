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

package gamepad

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const _CM_LOCATE_DEVNODE_NORMAL = 0

// devpkeyDeviceInstanceID is DEVPKEY_Device_InstanceId from devpkey.h.
var devpkeyDeviceInstanceID = windows.DEVPROPKEY{
	FmtID: windows.DEVPROPGUID{
		Data1: 0x78c34fc8,
		Data2: 0x104a,
		Data3: 0x4aca,
		Data4: [8]byte{0x9e, 0xa4, 0x52, 0x4d, 0x52, 0x99, 0x6e, 0x57},
	},
	PID: 256,
}

var (
	cfgmgr32 = windows.NewLazySystemDLL("cfgmgr32.dll")

	procCMGetDeviceInterfacePropertyW = cfgmgr32.NewProc("CM_Get_Device_Interface_PropertyW")
	procCMLocateDevNodeW              = cfgmgr32.NewProc("CM_Locate_DevNodeW")
	procCMGetParent                   = cfgmgr32.NewProc("CM_Get_Parent")
	procCMGetDeviceIDW                = cfgmgr32.NewProc("CM_Get_Device_IDW")
)

// parentDeviceInstanceID resolves a device interface path to its device node
// and returns the device instance ID of the node's parent. A device instance
// ID begins with the name of the enumerator that created the device, so the
// parent's ID names the bus the device is connected through.
// See https://learn.microsoft.com/en-us/windows-hardware/drivers/install/device-instance-ids.
func parentDeviceInstanceID(interfacePath string) (string, error) {
	for _, p := range []*windows.LazyProc{procCMGetDeviceInterfacePropertyW, procCMLocateDevNodeW, procCMGetParent, procCMGetDeviceIDW} {
		if err := p.Find(); err != nil {
			return "", err
		}
	}

	pathPtr, err := windows.UTF16PtrFromString(interfacePath)
	if err != nil {
		return "", err
	}

	var propType windows.DEVPROPTYPE
	var size uint32
	if r, _, _ := procCMGetDeviceInterfacePropertyW.Call(uintptr(unsafe.Pointer(pathPtr)), uintptr(unsafe.Pointer(&devpkeyDeviceInstanceID)),
		uintptr(unsafe.Pointer(&propType)), 0, uintptr(unsafe.Pointer(&size)), 0); windows.CONFIGRET(r) != windows.CR_BUFFER_SMALL {
		return "", fmt.Errorf("gamepad: CM_Get_Device_Interface_PropertyW failed: CONFIGRET(%d)", r)
	}
	if size == 0 || size%2 != 0 {
		return "", fmt.Errorf("gamepad: CM_Get_Device_Interface_PropertyW returned an invalid size: %d", size)
	}

	instanceID := make([]uint16, size/2)
	if r, _, _ := procCMGetDeviceInterfacePropertyW.Call(uintptr(unsafe.Pointer(pathPtr)), uintptr(unsafe.Pointer(&devpkeyDeviceInstanceID)),
		uintptr(unsafe.Pointer(&propType)), uintptr(unsafe.Pointer(&instanceID[0])), uintptr(unsafe.Pointer(&size)), 0); windows.CONFIGRET(r) != windows.CR_SUCCESS {
		return "", fmt.Errorf("gamepad: CM_Get_Device_Interface_PropertyW failed: CONFIGRET(%d)", r)
	}
	if propType != windows.DEVPROP_TYPE_STRING {
		return "", fmt.Errorf("gamepad: CM_Get_Device_Interface_PropertyW returned an unexpected property type: %d", propType)
	}

	var devInst windows.DEVINST
	if r, _, _ := procCMLocateDevNodeW.Call(uintptr(unsafe.Pointer(&devInst)), uintptr(unsafe.Pointer(&instanceID[0])),
		_CM_LOCATE_DEVNODE_NORMAL); windows.CONFIGRET(r) != windows.CR_SUCCESS {
		return "", fmt.Errorf("gamepad: CM_Locate_DevNodeW failed: CONFIGRET(%d)", r)
	}

	var parent windows.DEVINST
	if r, _, _ := procCMGetParent.Call(uintptr(unsafe.Pointer(&parent)), uintptr(devInst), 0); windows.CONFIGRET(r) != windows.CR_SUCCESS {
		return "", fmt.Errorf("gamepad: CM_Get_Parent failed: CONFIGRET(%d)", r)
	}

	var parentID [windows.MAX_DEVICE_ID_LEN + 1]uint16
	if r, _, _ := procCMGetDeviceIDW.Call(uintptr(parent), uintptr(unsafe.Pointer(&parentID[0])),
		uintptr(len(parentID)), 0); windows.CONFIGRET(r) != windows.CR_SUCCESS {
		return "", fmt.Errorf("gamepad: CM_Get_Device_IDW failed: CONFIGRET(%d)", r)
	}

	return windows.UTF16ToString(parentID[:]), nil
}
