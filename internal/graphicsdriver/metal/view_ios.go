// Copyright 2019 The Ebiten Authors
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

// +build darwin
// +build ios

package metal

import (
	"github.com/hajimehoshi/ebiten/internal/graphicsdriver/metal/ca"
	"github.com/hajimehoshi/ebiten/internal/graphicsdriver/metal/mtl"
)

type view struct {
}

func (v *view) setWindow(window uintptr) {
	panic("metal: setWindow cannot be called on iOS")
}

func (v *view) setDrawableSize(width, height int) {
	// Do nothing
}

func (v *view) getMTLDevice() mtl.Device {
	// TODO: Implement this
	return mtl.Device{}
}

func (v *view) setDisplaySyncEnabled(enabled bool) {
	// Do nothing
}

func (v *view) colorPixelFormat() mtl.PixelFormat {
	// TODO: Implement this
	return 0
}

func (v *view) reset() error {
	// Do nothing
	return nil
}

func (v *view) update() {
	// Do nothing
}

func (v *view) drawable() ca.MetalDrawable {
	// TODO: Implemnt this
	return ca.MetalDrawable{}
}
