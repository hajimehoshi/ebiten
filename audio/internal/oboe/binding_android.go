// Copyright 2021 The Ebiten Authors
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

package oboe

// Disable AAudio (#1634).
// AAudio doesn't care about plugging in/out of a headphone.
// See https://github.com/google/oboe/blob/master/docs/notes/disconnect.md

// #cgo CXXFLAGS: -std=c++17 -DOBOE_ENABLE_AAUDIO=0
// #cgo LDFLAGS: -llog -lOpenSLES -static-libstdc++
//
// #include "binding_android.h"
import "C"

import (
	"fmt"
	"reflect"
	"unsafe"
)

var theReadFunc func(buf []float32)

func Play(sampleRate, channelNum, bitDepthInBytes int, readFunc func(buf []float32)) error {
	// Play can invoke the callback. Set the callback before Play.
	theReadFunc = readFunc
	if msg := C.ebiten_oboe_Play(C.int(sampleRate), C.int(channelNum), C.int(bitDepthInBytes)); msg != nil {
		return fmt.Errorf("oboe: Play failed: %s", C.GoString(msg))
	}
	return nil
}

func Suspend() error {
	if msg := C.ebiten_oboe_Suspend(); msg != nil {
		return fmt.Errorf("oboe: Suspend failed: %s", C.GoString(msg))
	}
	return nil
}

func Resume() error {
	if msg := C.ebiten_oboe_Resume(); msg != nil {
		return fmt.Errorf("oboe: Resume failed: %s", C.GoString(msg))
	}
	return nil
}

//export ebiten_oboe_read
func ebiten_oboe_read(buf *C.float, len C.size_t) {
	s := []float32{}
	h := (*reflect.SliceHeader)(unsafe.Pointer(&s))
	h.Data = uintptr(unsafe.Pointer(buf))
	h.Len = int(len)
	h.Cap = int(len)
	for i := range s {
		s[i] = 0
	}
	theReadFunc(s)
}
