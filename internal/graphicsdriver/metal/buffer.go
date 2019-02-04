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

package metal

import (
	"sort"
	"unsafe"

	"github.com/hajimehoshi/ebiten/internal/graphicsdriver/metal/mtl"
)

// #cgo LDFLAGS: -framework CoreFoundation
//
// #import <CoreFoundation/CoreFoundation.h>
//
// static int count(void* obj) {
//   // TODO: Don't rely on the number of ref counts. CFGetRetainCount should be used only for debugging.
//   // Note that checking whether MTLCommandBuffer's status is completed or not does not work, because the
//   // CommandBuffer might still be used even in such situation.
//   return CFGetRetainCount(obj);
// }
import "C"

type buffer struct {
	b   mtl.Buffer
	len uintptr
}

func (b *buffer) used() bool {
	// If the count is 2 or more, the buffer is actually retained outside.
	// If the count is 1, the buffer is retained only by the buffer pool.
	// The count cannot be 0 since the object is already freed in this case.
	return C.count(b.b.Native()) > 1
}

var bufferPool = map[*buffer]struct{}{}

func getBuffer(device mtl.Device, data unsafe.Pointer, lengthInBytes uintptr) *buffer {
	for buf := range bufferPool {
		if buf.used() {
			continue
		}
		if buf.len < lengthInBytes {
			continue
		}
		buf.b.CopyToContents(data, lengthInBytes)
		buf.b.Retain()
		return buf
	}

	gcBufferPool()

	buf := &buffer{
		b:   device.MakeBufferWithBytes(data, lengthInBytes, mtl.ResourceStorageModeManaged),
		len: lengthInBytes,
	}
	buf.b.Retain()
	bufferPool[buf] = struct{}{}
	return buf
}

func putBuffer(buf *buffer) {
	buf.b.Release()
	// The buffer will be actually released after all the current command buffers are finished.
	gcBufferPool()
}

func gcBufferPool() {
	const threshold = 16

	if len(bufferPool) < threshold {
		return
	}

	toRemove := []*buffer{}
	for buf := range bufferPool {
		if buf.used() {
			continue
		}
		toRemove = append(toRemove, buf)
	}
	sort.Slice(toRemove, func(a, b int) bool {
		return toRemove[a].len < toRemove[b].len
	})

	l := len(toRemove)
	if l > len(bufferPool)-threshold {
		l = len(bufferPool) - threshold
	}
	for _, buf := range toRemove[:l] {
		buf.b.Release()
		delete(bufferPool, buf)
	}
}
