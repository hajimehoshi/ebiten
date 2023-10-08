// Copyright 2023 The Ebitengine Authors
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

package atlas

import (
	"runtime"
	"sync"
	"unsafe"
)

// allocBytesFromPool returns a byte slice with the given size.
// The slice might be obtained from a cache, and might not be zero-cleared.
func allocBytesFromPool(size int) []byte {
	return theBytesPool.get(size)
}

type bytesPool struct {
	pool [][]byte

	m sync.Mutex
}

var theBytesPool bytesPool

func (b *bytesPool) get(size int) []byte {
	if size == 0 {
		return nil
	}

	if bs := b.getFromCache(size); bs != nil {
		return bs
	}

	bs := make([]byte, size)
	b.setFinalizer(bs)
	return bs
}

func (b *bytesPool) getFromCache(size int) []byte {
	b.m.Lock()
	defer b.m.Unlock()

	for i, bs := range b.pool {
		if cap(bs) < size {
			continue
		}

		copy(b.pool[i:], b.pool[i+1:])
		b.pool[len(b.pool)-1] = nil
		b.pool = b.pool[:len(b.pool)-1]

		bs = bs[:size]
		b.setFinalizer(bs)
		return bs
	}

	return nil
}

func (b *bytesPool) setFinalizer(bs []byte) {
	c := cap(bs)
	runtime.SetFinalizer(&bs[0], func(ptr *byte) {
		b.m.Lock()
		defer b.m.Unlock()

		b.pool = append(b.pool, unsafe.Slice(ptr, c))

		// GC the pool. The size limitation is arbitrary.
		for len(b.pool) >= 32 || b.totalSize() >= 1024*1024*1024 {
			b.pool = b.pool[1:]
		}
	})
}

func (b *bytesPool) totalSize() int {
	var s int
	for _, bs := range b.pool {
		s += len(bs)
	}
	return s
}
