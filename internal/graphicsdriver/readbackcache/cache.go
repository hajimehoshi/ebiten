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

// Package readbackcache provides a cache of idle graphics resources for pixel read-backs.
package readbackcache

import (
	"slices"
)

const (
	// maxSize is the maximum total size of the cached resources in bytes.
	maxSize = 64 << 20

	// maxCount is the maximum number of the cached resources.
	maxCount = 16
)

type entry[K comparable, V any] struct {
	key   K
	value V
	size  int
}

// Cache holds idle resources for pixel read-backs for reuse, keyed by K.
// Resources put with equal keys must be interchangeable.
//
// A Cache is bounded both by the number of its resources and by their total size.
// The zero value is an empty cache.
type Cache[K comparable, V any] struct {
	// entries are the cached resources, oldest first.
	entries []entry[K, V]

	// totalSize is the sum of the sizes of entries in bytes.
	totalSize int
}

// Take removes the oldest resource with key from the cache and returns it.
// Take returns false if no resource has the key.
func (c *Cache[K, V]) Take(key K) (V, bool) {
	for i, e := range c.entries {
		if e.key != key {
			continue
		}
		c.entries = slices.Delete(c.entries, i, i+1)
		c.totalSize -= e.size
		return e.value, true
	}
	var zero V
	return zero, false
}

// Put adds value, a resource of size bytes, to the cache with key.
//
// Put returns the resources dropped from the cache, which the caller must release.
// They include value itself if value is empty or too large to cache.
func (c *Cache[K, V]) Put(key K, value V, size int) []V {
	if size == 0 || size > maxSize {
		return []V{value}
	}
	var dropped []V
	// Evict the oldest entries so changing read sizes cannot fill the cache with unusable resources.
	for len(c.entries) >= maxCount || c.totalSize > maxSize-size {
		oldest := c.entries[0]
		dropped = append(dropped, oldest.value)
		c.totalSize -= oldest.size
		c.entries = slices.Delete(c.entries, 0, 1)
	}
	c.entries = append(c.entries, entry[K, V]{
		key:   key,
		value: value,
		size:  size,
	})
	c.totalSize += size
	return dropped
}

// Clear removes every resource from the cache and returns them, which the caller must release.
func (c *Cache[K, V]) Clear() []V {
	values := make([]V, 0, len(c.entries))
	for _, e := range c.entries {
		values = append(values, e.value)
	}
	c.entries = slices.Delete(c.entries, 0, len(c.entries))
	c.totalSize = 0
	return values
}
