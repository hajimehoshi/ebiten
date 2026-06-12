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

package vmhost_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// TestVMGuestEndpointOption runs a guest built WITHOUT the ebitenginevm build tag, activated through
// RunGameOptions.VMGuestEndpoint rather than EBITENGINE_VM_ENDPOINT.
func TestVMGuestEndpointOption(t *testing.T) {
	guest := startGuest(t, "./testdata/options", activateByOptions, "unix")

	const w, h = 64, 48
	scale := ebiten.Monitor().DeviceScaleFactor()
	pw, ph := int(w*scale), int(h*scale)
	outsideScreen := ebiten.NewImage(pw, ph)

	if err := guest.SetOutsideScreen(outsideScreen); err != nil {
		t.Fatal(err)
	}
	if err := guest.AdvanceTick(); err != nil {
		t.Fatalf("advancing a tick failed: %v", err)
	}
	guest.AdvanceFrame()
	// AdvanceFrame defers its errors to the next AdvanceTick.
	if err := guest.AdvanceTick(); err != nil {
		t.Fatalf("rendering the guest frame failed: %v", err)
	}

	pixels := make([]byte, 4*pw*ph)
	outsideScreen.ReadPixels(pixels)
	i := 4 * ((ph/2)*pw + pw/2)
	if got, want := [4]byte(pixels[i:i+4]), [4]byte{0x80, 0x20, 0x60, 0xff}; got != want {
		t.Errorf("the guest fill color = %v; want %v", got, want)
	}
}
