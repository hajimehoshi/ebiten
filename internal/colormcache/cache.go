// Copyright 2020 The Ebiten Authors
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

package colormcache

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten"
)

var (
	monotonicClock int64
)

func now() int64 {
	monotonicClock++
	return monotonicClock
}

const (
	cacheLimit = 512 // This is an arbitrary number.
)

type colorMCacheKey uint32

type colorMCacheEntry struct {
	m     ebiten.ColorM
	atime int64
}

var (
	colorMCache = map[colorMCacheKey]*colorMCacheEntry{}
	emptyColorM ebiten.ColorM
)

func init() {
	emptyColorM.Scale(0, 0, 0, 0)
}

func ColorToColorM(clr color.Color) ebiten.ColorM {
	// RGBA() is in [0 - 0xffff]. Adjust them in [0 - 0xff].
	cr, cg, cb, ca := clr.RGBA()
	cr /= 0x101
	cg /= 0x101
	cb /= 0x101
	ca /= 0x101
	if ca == 0 {
		return emptyColorM
	}

	key := colorMCacheKey(uint32(cr) | (uint32(cg) << 8) | (uint32(cb) << 16) | (uint32(ca) << 24))
	e, ok := colorMCache[key]
	if ok {
		e.atime = now()
		return e.m
	}

	if len(colorMCache) > cacheLimit {
		oldest := int64(math.MaxInt64)
		oldestKey := colorMCacheKey(0)
		for key, c := range colorMCache {
			if c.atime < oldest {
				oldestKey = key
				oldest = c.atime
			}
		}
		delete(colorMCache, oldestKey)
	}

	cm := ebiten.ColorM{}
	rf := float64(cr) / float64(ca)
	gf := float64(cg) / float64(ca)
	bf := float64(cb) / float64(ca)
	af := float64(ca) / 0xff
	cm.Scale(rf, gf, bf, af)
	e = &colorMCacheEntry{
		m:     cm,
		atime: now(),
	}
	colorMCache[key] = e

	return e.m
}
