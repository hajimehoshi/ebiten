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

//go:build !android && !nintendosdk && !playstation5

package gamepad_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
)

func TestSDLID(t *testing.T) {
	for _, tc := range []struct {
		name                          string
		bus, vendor, product, version uint16
		want                          string
	}{
		{
			name:    "8BitDo Pro 2 for Xbox",
			bus:     3,
			vendor:  0x2dc8,
			product: 0x2000,
			version: 0,
			want:    "03000000c82d00000020000000000000",
		},
		{
			name:    "versioned",
			bus:     3,
			vendor:  0x2dc8,
			product: 0x2000,
			version: 0x1234,
			want:    "03000000c82d00000020000034120000",
		},
		{
			name:    "Pad",
			bus:     3,
			vendor:  0,
			product: 0x2000,
			version: 0,
			want:    "03000000506164000000000000000000",
		},
		{
			name:    "ABCDEFGHIJKLM",
			bus:     3,
			vendor:  0x2dc8,
			product: 0,
			version: 1,
			want:    "030000004142434445464748494a4b4c",
		},
	} {
		got := gamepad.SDLIDForTesting(tc.bus, tc.vendor, tc.product, tc.version, tc.name)
		if got != tc.want {
			t.Errorf("%s: got %s, want %s", tc.name, got, tc.want)
		}
		if tc.name == "8BitDo Pro 2 for Xbox" && !gamepaddb.HasStandardLayoutMapping(got) {
			t.Error("zero-version GUID has no standard layout mapping")
		}
	}
}
