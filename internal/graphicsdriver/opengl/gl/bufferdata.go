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

//go:build !playstation5

package gl

import (
	"errors"
	"unsafe"
)

// errReadBufferDataUnavailable indicates that the context provides neither glGetBufferSubData nor
// buffer mapping.
var errReadBufferDataUnavailable = errors.New("gl: reading the content of a buffer is not supported by this context")

// ReadBufferData copies len(dst) bytes from the buffer bound to target, starting at offset, into
// dst.
//
// A buffer must be bound to target.
//
// ReadBufferData must be called only when HasFenceSync reports true.
func (c *defaultContext) ReadBufferData(target uint32, offset int, dst []byte) error {
	if len(dst) == 0 {
		return nil
	}

	// glGetBufferSubData is the simplest way, but it is not available everywhere. Notably, some
	// OpenGL ES implementations, like Mesa's libGLESv2, don't export it at all.
	if c.hasGetBufferSubData {
		c.getBufferSubData(target, offset, dst)
		return nil
	}

	// Buffer mapping is the alternative, but WebGL 2 forbids it.
	if !c.hasBufferMapping {
		return errReadBufferDataUnavailable
	}

	src := c.mapBufferRange(target, offset, len(dst), MAP_READ_BIT)
	if src == nil {
		return errors.New("gl: mapping a buffer failed")
	}
	copy(dst, src)
	// src is no longer valid after the buffer is unmapped, so it must not be used after this.
	if !c.unmapBuffer(target) {
		// The buffer contents were lost while the buffer was mapped.
		return errors.New("gl: unmapping a buffer reported that its content was lost")
	}
	return nil
}

// HasFenceSync reports whether the functions needed for an asynchronous read of pixels are all
// available.
func (c *defaultContext) HasFenceSync() bool {
	return c.hasFenceSync && (c.hasGetBufferSubData || c.hasBufferMapping)
}

// pointerFromUintptr reinterprets a pointer value returned by a foreign function as an
// unsafe.Pointer.
//
// A direct unsafe.Pointer(p) conversion of a uintptr is rejected by go vet as a possible misuse,
// as the compiler cannot know that p does not hold a Go pointer. Taking the address of the local
// copy first makes the reinterpretation explicit, which is also how purego handles this.
func pointerFromUintptr(p uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&p))
}
