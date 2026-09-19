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

package gbm

import (
	"testing"
	"unsafe"
)

const (
	pointerSize                     = unsafe.Sizeof(uintptr(0))
	drmModeResSize                  = unsafe.Sizeof(drmModeRes{})
	drmModeResCountConnectorsOffset = unsafe.Offsetof(drmModeRes{}.countConnectors)
	drmModeResConnectorsOffset      = unsafe.Offsetof(drmModeRes{}.connectors)
	drmModeConnectorSize            = unsafe.Sizeof(drmModeConnector{})
	drmModeConnectorModesOffset     = unsafe.Offsetof(drmModeConnector{}.modes)
	drmModeModeInfoSize             = unsafe.Sizeof(drmModeModeInfo{})
	drmModeModeInfoVRefreshOffset   = unsafe.Offsetof(drmModeModeInfo{}.vrefresh)

	wantDRMModeResSize                  = 8*pointerSize + 16
	wantDRMModeResCountConnectorsOffset = 4 * pointerSize
	wantDRMModeResConnectorsOffset      = 5 * pointerSize
	wantDRMModeConnectorSize            = 7*pointerSize + 32
	wantDRMModeConnectorModesOffset     = pointerSize + 32
)

// Compile-time libdrm ABI checks.
var (
	_ [wantDRMModeResSize - drmModeResSize]byte
	_ [drmModeResSize - wantDRMModeResSize]byte
	_ [wantDRMModeResCountConnectorsOffset - drmModeResCountConnectorsOffset]byte
	_ [drmModeResCountConnectorsOffset - wantDRMModeResCountConnectorsOffset]byte
	_ [wantDRMModeResConnectorsOffset - drmModeResConnectorsOffset]byte
	_ [drmModeResConnectorsOffset - wantDRMModeResConnectorsOffset]byte
	_ [wantDRMModeConnectorSize - drmModeConnectorSize]byte
	_ [drmModeConnectorSize - wantDRMModeConnectorSize]byte
	_ [wantDRMModeConnectorModesOffset - drmModeConnectorModesOffset]byte
	_ [drmModeConnectorModesOffset - wantDRMModeConnectorModesOffset]byte
	_ [68 - drmModeModeInfoSize]byte
	_ [drmModeModeInfoSize - 68]byte
	_ [24 - drmModeModeInfoVRefreshOffset]byte
	_ [drmModeModeInfoVRefreshOffset - 24]byte
)

func TestCRTCForEncoder(t *testing.T) {
	crtcs := []uint32{10, 20, 30}
	for _, test := range []struct {
		name string
		enc  drmModeEncoder
		want uint32
	}{
		{
			name: "current CRTC",
			enc:  drmModeEncoder{crtcID: 40},
			want: 40,
		},
		{
			name: "first compatible CRTC",
			enc:  drmModeEncoder{possibleCrtcs: 1 << 1},
			want: 20,
		},
		{
			name: "no compatible CRTC",
			enc:  drmModeEncoder{},
			want: 0,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := crtcForEncoder(&test.enc, crtcs); got != test.want {
				t.Errorf("crtcForEncoder(...) = %d, want %d", got, test.want)
			}
		})
	}
}
