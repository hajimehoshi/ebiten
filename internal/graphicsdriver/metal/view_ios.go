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

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework UIKit
//
// #import <UIKit/UIKit.h>
//
// static void addSublayer(void* view, void* sublayer) {
//   CALayer* layer = ((UIView*)view).layer;
//   [layer addSublayer:(CALayer*)sublayer];
// }
//
// static void setFrame(void* cametal, void* uiview) {
//   CGSize size = ((UIView*)uiview).frame.size;
//   ((CALayer*)cametal).frame = CGRectMake(0, 0, size.width, size.height);
// }
import "C"

import (
	"unsafe"

	"github.com/hajimehoshi/ebiten/internal/graphicsdriver/metal/mtl"
)

func (v *view) setWindow(window uintptr) {
	panic("metal: setWindow is not available on iOS")
}

func (v *view) setUIView(uiview uintptr) {
	v.uiview = uiview
}

func (v *view) update() {
	v.once.Do(func() {
		if v.ml.Layer() == nil {
			panic("metal: CAMetalLayer is not initialized yet")
		}
		C.addSublayer(unsafe.Pointer(v.uiview), v.ml.Layer())
	})
	C.setFrame(v.ml.Layer(), unsafe.Pointer(v.uiview))
}

const (
	storageMode         = mtl.StorageModeShared
	resourceStorageMode = mtl.ResourceStorageModeShared
)
