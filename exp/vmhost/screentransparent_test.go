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

// The transparent guest requests a transparent screen via RunGameOptions.ScreenTransparent, the opaque
// guest does not (see testdata/screentransparent and testdata/screenopaque). Both leave the whole
// screen uncovered, so CompositeFrame yields a fully transparent frame for the transparent guest and an
// opaque-black one for the non-transparent guest.

package vmhost_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestScreenTransparentComposite(t *testing.T) {
	tests := []struct {
		name    string
		pkgPath string
		want    [4]byte
	}{
		{name: "transparent", pkgPath: "./testdata/screentransparent", want: [4]byte{0, 0, 0, 0}},
		{name: "opaque", pkgPath: "./testdata/screenopaque", want: [4]byte{0, 0, 0, 0xff}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guest := startGuest(t, tt.pkgPath, activateByEnv, "unix")

			const w, h = 64, 48
			scale := ebiten.Monitor().DeviceScaleFactor()
			pw, ph := int(w*scale), int(h*scale)
			outsideScreen := ebiten.NewImage(pw, ph)
			if err := guest.SetOutsideScreen(outsideScreen); err != nil {
				t.Fatal(err)
			}

			// The guest draws nothing, so the whole screen is uncovered: a transparent guest composites to
			// fully transparent, a non-transparent guest over opaque black.
			tickAndFrame(t, guest)

			pixels := make([]byte, 4*pw*ph)
			outsideScreen.ReadPixels(pixels)
			i := 4 * ((ph/2)*pw + pw/2)
			if got := [4]byte(pixels[i : i+4]); got != tt.want {
				t.Errorf("composited center pixel = %v; want %v", got, tt.want)
			}
		})
	}
}
