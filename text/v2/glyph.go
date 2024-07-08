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

package text

import (
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/internal/hook"
)

var monotonicClock int64

const infTime = math.MaxInt64

func now() int64 {
	return monotonicClock
}

func init() {
	hook.AppendHookOnBeforeUpdate(func() error {
		monotonicClock++
		return nil
	})
}

type glyphImageCacheEntry struct {
	image *ebiten.Image
	atime int64
}

type glyphImageCache[Key comparable] struct {
	cache map[Key]*glyphImageCacheEntry
	atime int64
	m     sync.Mutex
}

func (g *glyphImageCache[Key]) getOrCreate(face Face, key Key, create func() *ebiten.Image) *ebiten.Image {
	g.m.Lock()
	defer g.m.Unlock()

	n := now()

	e, ok := g.cache[key]
	if ok {
		e.atime = n
		return e.image
	}

	if g.cache == nil {
		g.cache = map[Key]*glyphImageCacheEntry{}
	}

	img := create()
	e = &glyphImageCacheEntry{
		image: img,
	}
	if img != nil {
		e.atime = n
	} else {
		// If the glyph image is nil, the entry doesn't have to be removed.
		// Keep this until the face is GCed.
		e.atime = infTime
	}
	g.cache[key] = e

	// Clean up old entries.
	if g.atime < n {
		// cacheSoftLimit indicates the soft limit of the number of glyphs in the cache.
		// If the number of glyphs exceeds this soft limits, old glyphs are removed.
		// Even after cleaning up the cache, the number of glyphs might still exceed the soft limit, but
		// this is fine.
		cacheSoftLimit := 128 * glyphVariationCount(face)
		if len(g.cache) > cacheSoftLimit {
			for key, e := range g.cache {
				// 60 is an arbitrary number.
				if e.atime >= now()-60 {
					continue
				}
				delete(g.cache, key)
			}
		}
	}

	g.atime = n

	return img
}
