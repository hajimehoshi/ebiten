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

	"golang.org/x/image/math/fixed"

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

type faceCacheKey struct {
	id             uint64
	goTextFaceSize float64
}

type glyphImageCacheKey struct {
	// id is rune for StdFace, and GID for GoTextFace.
	id uint32

	xoffset fixed.Int26_6
	yoffset fixed.Int26_6

	variations string
}

type glyphImageCacheEntry struct {
	image *ebiten.Image
	atime int64
}

type glyphImageCache struct {
	cache map[faceCacheKey]map[glyphImageCacheKey]*glyphImageCacheEntry
	m     sync.Mutex
}

var theGlyphImageCache glyphImageCache

func (g *glyphImageCache) getOrCreate(face Face, key glyphImageCacheKey, create func() *ebiten.Image) *ebiten.Image {
	g.m.Lock()
	defer g.m.Unlock()

	e, ok := g.cache[face.faceCacheKey()][key]
	if ok {
		e.atime = now()
		return e.image
	}

	if g.cache == nil {
		g.cache = map[faceCacheKey]map[glyphImageCacheKey]*glyphImageCacheEntry{}
	}
	faceCacheKey := face.faceCacheKey()
	if g.cache[faceCacheKey] == nil {
		g.cache[faceCacheKey] = map[glyphImageCacheKey]*glyphImageCacheEntry{}
	}

	img := create()
	e = &glyphImageCacheEntry{
		image: img,
	}
	if img != nil {
		e.atime = now()
	} else {
		// If the glyph image is nil, the entry doesn't have to be removed.
		// Keep this until the face is GCed.
		e.atime = infTime
	}
	g.cache[faceCacheKey][key] = e

	// Clean up old entries.

	// cacheSoftLimit indicates the soft limit of the number of glyphs in the cache.
	// If the number of glyphs exceeds this soft limits, old glyphs are removed.
	// Even after cleaning up the cache, the number of glyphs might still exceed the soft limit, but
	// this is fine.
	cacheSoftLimit := 128 * glyphVariationCount(face)
	if len(g.cache[faceCacheKey]) > cacheSoftLimit {
		for key, e := range g.cache[faceCacheKey] {
			// 60 is an arbitrary number.
			if e.atime >= now()-60 {
				continue
			}
			delete(g.cache[faceCacheKey], key)
		}
	}

	return img
}

func (g *glyphImageCache) clear(comp func(key faceCacheKey) bool) {
	g.m.Lock()
	defer g.m.Unlock()

	for key := range g.cache {
		if comp(key) {
			delete(g.cache, key)
		}
	}
}
