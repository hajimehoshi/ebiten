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

import "golang.org/x/sys/windows"

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
