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

// TestGuestEndpointNotInherited asserts that a guest activated through EBITENGINE_VM_ENDPOINT drops
// the variable, so that a process it starts cannot reach the host through the same endpoint. The
// guest paints its answer: white when the variable is gone, red when it survives (see
// testdata/envconsume).
func TestGuestEndpointNotInherited(t *testing.T) {
	guest := startGuest(t, "./testdata/envconsume", activateByEnv, "unix")

	const w, h = 64, 48
	scale := ebiten.Monitor().DeviceScaleFactor()
	pw, ph := int(w*scale), int(h*scale)
	outsideScreen := ebiten.NewImage(pw, ph)
	if err := guest.SetOutsideScreen(outsideScreen); err != nil {
		t.Fatal(err)
	}

	tickAndFrame(t, guest)

	pixels := make([]byte, 4*pw*ph)
	outsideScreen.ReadPixels(pixels)
	i := 4 * ((ph/2)*pw + pw/2)
	want := [4]byte{0xff, 0xff, 0xff, 0xff}
	if got := [4]byte(pixels[i : i+4]); got != want {
		t.Errorf("composited center pixel = %v; want %v (red means the guest still has EBITENGINE_VM_ENDPOINT in its environment)", got, want)
	}
}
