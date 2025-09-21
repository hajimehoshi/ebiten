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

package graphics

import (
	"runtime"
	"sync"
)

// ManagedBytes is a managed byte slice.
// The internal byte alice are managed in a pool.
// ManagedBytes is useful when its lifetime is explicit, as the underlying byte slice can be reused for another ManagedBytes later.
// This can reduce allocations and GCs.
type ManagedBytes struct {
	bytes   []byte
	pool    *bytesPool
	cleanup runtime.Cleanup
}

// Len returns the length of the slice.
func (m *ManagedBytes) Len() int {
	return len(m.bytes)
}

// Read reads the byte slice's content to dst.
func (m *ManagedBytes) Read(dst []byte, from, to int) {
	copy(dst, m.bytes[from:to])
}

// Clone creates a new ManagedBytes with the same content.
func (m *ManagedBytes) Clone() *ManagedBytes {
	return NewManagedBytes(len(m.bytes), func(bs []byte) {
		copy(bs, m.bytes)
	})
}

// GetAndRelease returns the raw byte slice and a finalizer.
// A finalizer should be called when you can ensure that the slice is no longer used,
// e.g. when a graphics command using this slice is sent and executed.
//
// After GetAndRelease is called, the underlying byte slice is no longer available.
func (m *ManagedBytes) GetAndRelease() ([]byte, func()) {
	bs := m.bytes
	m.bytes = nil
	return bs, func() {
		m.pool.put(bs)
		m.cleanup.Stop()
	}
}

// Release releases the underlying byte slice.
//
// After Release is called, the underlying byte slice is no longer available.
func (m *ManagedBytes) Release() {
	m.pool.put(m.bytes)
	m.bytes = nil
	m.cleanup.Stop()
}

// NewManagedBytes returns a managed byte slice initialized by the given constructor f.
//
// The byte slice is not zero-cleared at the constructor.
func NewManagedBytes(size int, f func([]byte)) *ManagedBytes {
	bs := theBytesPool.get(size)
	f(bs.bytes)
	return bs
}

type bytesPool struct {
	pool [][]byte

	m sync.Mutex
}

var theBytesPool bytesPool

func (b *bytesPool) get(size int) *ManagedBytes {
	bs := b.getFromCache(size)
	if bs == nil {
		bs = make([]byte, size)
	}
	m := &ManagedBytes{
		bytes: bs,
		pool:  b,
	}
	m.cleanup = runtime.AddCleanup(m, func(bytes []byte) {
		b.put(bytes)
	}, m.bytes)
	return m
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
		return bs[:size]
	}

	return nil
}

func (b *bytesPool) put(bs []byte) {
	if len(bs) == 0 {
		return
	}

	b.m.Lock()
	defer b.m.Unlock()

	b.pool = append(b.pool, bs)

	// GC the pool. The size limitation is arbitrary.
	for len(b.pool) >= 32 || b.totalSize() >= 1024*1024*1024 {
		b.pool = b.pool[1:]
	}
}

func (b *bytesPool) totalSize() int {
	var s int
	for _, bs := range b.pool {
		s += len(bs)
	}
	return s
}
