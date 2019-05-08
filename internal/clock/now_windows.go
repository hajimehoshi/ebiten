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

package clock

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32 = windows.NewLazySystemDLL("kernel32")
)

var (
	procQueryPerformanceFrequency = kernel32.NewProc("QueryPerformanceFrequency")
	procQueryPerformanceCounter   = kernel32.NewProc("QueryPerformanceCounter")
)

func queryPerformanceFrequency() int64 {
	var freq int64
	// TODO: Should the returned value be checked?
	_, _, e := procQueryPerformanceFrequency.Call(uintptr(unsafe.Pointer(&freq)))
	if e != nil && e.(windows.Errno) != 0 {
		panic(fmt.Sprintf("clock: QueryPerformanceFrequency failed: errno: %d", e.(windows.Errno)))
	}
	return freq
}

func queryPerformanceCounter() int64 {
	var ctr int64
	// TODO: Should the returned value be checked?
	_, _, e := procQueryPerformanceCounter.Call(uintptr(unsafe.Pointer(&ctr)))
	if e != nil && e.(windows.Errno) != 0 {
		panic(fmt.Sprintf("clock: QueryPerformanceCounter failed: errno: %d", e.(windows.Errno)))
	}
	return ctr
}

var (
	freq    = queryPerformanceFrequency()
	initCtr = queryPerformanceCounter()
)

func now() int64 {
	// Use the time duration instead of the current counter to avoid overflow.
	duration := queryPerformanceCounter() - initCtr

	// Use float64 not to overflow int64 values (#862).
	return int64(float64(duration) / float64(freq) * 1e9)
}
