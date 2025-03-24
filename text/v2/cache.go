// Copyright 2024 The Ebitengine Authors
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

package text

import (
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
)

const infTick = math.MaxInt64

type cacheValue[Value any] struct {
	value Value

	// atime is the last time when the value was accessed.
	atime int64
}

type cache[Key comparable, Value any] struct {
	// softLimit indicates the soft limit of the number of values in the cache.
	softLimit int

	values map[Key]*cacheValue[Value]

	// atime is the last time when the cache was accessed.
	atime int64

	m sync.Mutex
}

func newCache[Key comparable, Value any](softLimit int) *cache[Key, Value] {
	return &cache[Key, Value]{
		softLimit: softLimit,
	}
}

func (c *cache[Key, Value]) getOrCreate(key Key, create func() (Value, bool)) Value {
	n := ebiten.Tick()

	c.m.Lock()
	defer c.m.Unlock()

	e, ok := c.values[key]
	if ok {
		e.atime = n
		return e.value
	}

	if c.values == nil {
		c.values = map[Key]*cacheValue[Value]{}
	}

	ent, canExpire := create()
	e = &cacheValue[Value]{
		value: ent,
		atime: infTick,
	}
	if canExpire {
		e.atime = n
	}
	c.values[key] = e

	// Clean up old entries.
	if c.atime < n {
		// If the number of values exceeds the soft limits, old values are removed.
		// Even after cleaning up the cache, the number of values might still exceed the soft limit,
		// but this is fine.
		if len(c.values) > c.softLimit {
			for key, e := range c.values {
				// 60 is an arbitrary number.
				if e.atime >= n-60 {
					continue
				}
				delete(c.values, key)
			}
		}
	}

	c.atime = n

	return e.value
}
