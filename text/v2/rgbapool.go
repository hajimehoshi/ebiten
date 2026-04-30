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

package text

import (
	"image"
	"sync"
)

var theRGBAPool = sync.Pool{
	New: func() any {
		return &image.RGBA{}
	},
}

// newPooledRGBA returns an *image.RGBA whose Pix buffer capacity is reused from a pool.
// The returned image must be passed to releasePooledRGBA when it is no longer used.
// The Pix buffer is guaranteed to be zero-filled.
func newPooledRGBA(w, h int) *image.RGBA {
	size := 4 * w * h
	rgba := theRGBAPool.Get().(*image.RGBA)
	if cap(rgba.Pix) < size {
		rgba.Pix = make([]byte, size)
	} else {
		rgba.Pix = rgba.Pix[:size]
	}
	rgba.Stride = 4 * w
	rgba.Rect = image.Rect(0, 0, w, h)
	return rgba
}

// releasePooledRGBA returns an image obtained from newPooledRGBA to the pool.
func releasePooledRGBA(rgba *image.RGBA) {
	clear(rgba.Pix)
	rgba.Pix = rgba.Pix[:0]
	theRGBAPool.Put(rgba)
}
