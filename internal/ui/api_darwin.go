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

package ui

import (
	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

var libSystem uintptr

var dispatchMainQ uintptr

var dispatchSync func(queue uintptr, block objc.Block)

func init() {
	lib, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		panic(err)
	}
	libSystem = lib

	q, err := purego.Dlsym(libSystem, "_dispatch_main_q")
	if err != nil {
		panic(err)
	}
	dispatchMainQ = q

	purego.RegisterLibFunc(&dispatchSync, libSystem, "dispatch_sync")
}
