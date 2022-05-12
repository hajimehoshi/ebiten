// Copyright 2022 The Ebiten Authors
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

//go:build darwin
// +build darwin

package syscall

import (
	"unsafe"
)

//go:linkname syscall_syscallX syscall.syscallX
func syscall_syscallX(fn, a1, a2, a3 uintptr) (r1, r2, err uintptr) // from runtime/sys_darwin_64.go

//go:linkname runtime_libcCall runtime.libcCall
//go:linkname runtime_entersyscall runtime.entersyscall
//go:linkname runtime_exitsyscall runtime.exitsyscall
func runtime_libcCall(fn, arg unsafe.Pointer) int32 // from runtime/sys_libc.go
func runtime_entersyscall()                         // from runtime/proc.go
func runtime_exitsyscall()                          // from runtime/proc.go
