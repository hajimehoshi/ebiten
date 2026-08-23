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

const _HIDP_STATUS_SUCCESS = 0x00110000

type _HIDP_CAPS struct {
	Usage                     uint16
	UsagePage                 uint16
	InputReportByteLength     uint16
	OutputReportByteLength    uint16
	FeatureReportByteLength   uint16
	Reserved                  [17]uint16
	NumberLinkCollectionNodes uint16
	NumberInputButtonCaps     uint16
	NumberInputValueCaps      uint16
	NumberInputDataIndices    uint16
	NumberOutputButtonCaps    uint16
	NumberOutputValueCaps     uint16
	NumberOutputDataIndices   uint16
	NumberFeatureButtonCaps   uint16
	NumberFeatureValueCaps    uint16
	NumberFeatureDataIndices  uint16
}

var (
	hid = windows.NewLazySystemDLL("hid.dll")

	procHidDGetPreparsedData  = hid.NewProc("HidD_GetPreparsedData")
	procHidDFreePreparsedData = hid.NewProc("HidD_FreePreparsedData")
	procHidPGetCaps           = hid.NewProc("HidP_GetCaps")
)

// hidOutputReportByteLength returns the maximum output report length in
// bytes, including the report ID byte, for an opened HID device.
func hidOutputReportByteLength(handle windows.Handle) (int, error) {
	for _, p := range []*windows.LazyProc{procHidDGetPreparsedData, procHidDFreePreparsedData, procHidPGetCaps} {
		if err := p.Find(); err != nil {
			return 0, err
		}
	}

	var preparsedData uintptr
	if r, _, _ := procHidDGetPreparsedData.Call(uintptr(handle), uintptr(unsafe.Pointer(&preparsedData))); r == 0 {
		return 0, fmt.Errorf("gamepad: HidD_GetPreparsedData failed")
	}
	defer func() {
		_, _, _ = procHidDFreePreparsedData.Call(preparsedData)
	}()

	var caps _HIDP_CAPS
	if r, _, _ := procHidPGetCaps.Call(preparsedData, uintptr(unsafe.Pointer(&caps))); uint32(r) != _HIDP_STATUS_SUCCESS {
		return 0, fmt.Errorf("gamepad: HidP_GetCaps failed: NTSTATUS(%#08x)", uint32(r))
	}

	return int(caps.OutputReportByteLength), nil
}
