// Copyright 2024 The Ebiten Authors
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

package mtl

import (
	"unsafe"

	"github.com/ebitengine/purego"
)

var libSystem uintptr

var (
	dispatchDataCreate func(buffer unsafe.Pointer, size uint, queue uintptr, destructor uintptr) uintptr
	dispatchRelease    func(obj uintptr)
)

func init() {
	lib, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		panic(err)
	}
	libSystem = lib

	purego.RegisterLibFunc(&dispatchDataCreate, libSystem, "dispatch_data_create")
	purego.RegisterLibFunc(&dispatchRelease, libSystem, "dispatch_release")
}
