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
	"unsafe"
)

const (
	linuxInputEventPointerSize = unsafe.Sizeof(uintptr(0))
	linuxInputEventSize        = unsafe.Sizeof(linuxInputEvent{})
	linuxInputEventTypeOffset  = unsafe.Offsetof(linuxInputEvent{}.typ)
	wantLinuxInputEventSize    = 2*linuxInputEventPointerSize + 8
	wantLinuxInputEventTypeOff = 2 * linuxInputEventPointerSize
)

// Compile-time input_event ABI checks.
var (
	_ [wantLinuxInputEventSize - linuxInputEventSize]byte
	_ [linuxInputEventSize - wantLinuxInputEventSize]byte
	_ [wantLinuxInputEventTypeOff - linuxInputEventTypeOffset]byte
	_ [linuxInputEventTypeOffset - wantLinuxInputEventTypeOff]byte
)
