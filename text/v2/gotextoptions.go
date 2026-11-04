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
	"encoding/binary"
	"math"
	"slices"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/shaping"
)

// variations keeps small option sets inline. Larger sets use an immutable shared
// slice, copied before an effective mutation. No stored slice points into inline.
type variations struct {
	inline   [4]font.Variation
	overflow []font.Variation
	count    int
	key      string
}

func (o *variations) values() []font.Variation {
	if o.count <= len(o.inline) {
		return o.inline[:o.count]
	}
	return o.overflow
}

func (o *variations) set(tag font.Tag, value float32) {
	values := o.values()
	idx := len(values)
	for i, entry := range values {
		if entry.Tag < tag {
			continue
		}
		if entry.Tag > tag {
			idx = i
			break
		}
		if entry.Value == value {
			return
		}
		if o.count > len(o.inline) {
			values = slices.Clone(values)
			o.overflow = values
		}
		values[i].Value = value
		o.key = encodeVariations(values)
		return
	}

	// Keep tags sorted so insertion order does not affect the cache key.
	if o.count < len(o.inline) {
		values = o.inline[:o.count+1]
		copy(values[idx+1:], values[idx:o.count])
	} else {
		next := make([]font.Variation, o.count+1)
		copy(next, values[:idx])
		copy(next[idx+1:], values[idx:])
		values = next
		o.overflow = values
	}
	values[idx] = font.Variation{Tag: tag, Value: value}
	o.count++
	o.key = encodeVariations(values)
}

func (o *variations) remove(tag font.Tag) {
	values := o.values()
	for i, entry := range values {
		if entry.Tag < tag {
			continue
		}
		if entry.Tag > tag {
			return
		}
		n := o.count - 1
		if n <= len(o.inline) {
			copy(o.inline[:i], values[:i])
			copy(o.inline[i:n], values[i+1:])
			clear(o.inline[n:])
			o.overflow = nil
		} else {
			next := make([]font.Variation, n)
			copy(next, values[:i])
			copy(next[i:], values[i+1:])
			o.overflow = next
		}
		o.count = n
		o.key = encodeVariations(o.values())
		return
	}
}

// features keeps small option sets inline. Larger sets use an immutable shared
// slice, copied before an effective mutation. No stored slice points into inline.
type features struct {
	inline   [4]shaping.FontFeature
	overflow []shaping.FontFeature
	count    int
	key      string
}

func (o *features) values() []shaping.FontFeature {
	if o.count <= len(o.inline) {
		return o.inline[:o.count]
	}
	return o.overflow
}

func (o *features) set(tag font.Tag, value uint32) {
	values := o.values()
	idx := len(values)
	for i, entry := range values {
		if entry.Tag < tag {
			continue
		}
		if entry.Tag > tag {
			idx = i
			break
		}
		if entry.Value == value {
			return
		}
		if o.count > len(o.inline) {
			values = slices.Clone(values)
			o.overflow = values
		}
		values[i].Value = value
		o.key = encodeFeatures(values)
		return
	}

	// Keep tags sorted so insertion order does not affect the cache key.
	if o.count < len(o.inline) {
		values = o.inline[:o.count+1]
		copy(values[idx+1:], values[idx:o.count])
	} else {
		next := make([]shaping.FontFeature, o.count+1)
		copy(next, values[:idx])
		copy(next[idx+1:], values[idx:])
		values = next
		o.overflow = values
	}
	values[idx] = shaping.FontFeature{Tag: tag, Value: value}
	o.count++
	o.key = encodeFeatures(values)
}

func (o *features) remove(tag font.Tag) {
	values := o.values()
	for i, entry := range values {
		if entry.Tag < tag {
			continue
		}
		if entry.Tag > tag {
			return
		}
		n := o.count - 1
		if n <= len(o.inline) {
			copy(o.inline[:i], values[:i])
			copy(o.inline[i:n], values[i+1:])
			clear(o.inline[n:])
			o.overflow = nil
		} else {
			next := make([]shaping.FontFeature, n)
			copy(next, values[:i])
			copy(next[i:], values[i+1:])
			o.overflow = next
		}
		o.count = n
		o.key = encodeFeatures(o.values())
		return
	}
}

// encodeVariations returns a binary string encoding of variations
// suitable for use as a cache key. It is the canonical form referenced
// by glyphDataCacheKey and glyphRenderDataCacheKey.
func encodeVariations(variations []font.Variation) string {
	if len(variations) == 0 {
		return ""
	}
	buf := make([]byte, 0, 8*len(variations))
	for _, t := range variations {
		buf = binary.LittleEndian.AppendUint32(buf, uint32(t.Tag))
		buf = binary.LittleEndian.AppendUint32(buf, math.Float32bits(t.Value))
	}
	return string(buf)
}

// encodeFeatures returns a binary string encoding of features
// suitable for use as a cache key.
func encodeFeatures(features []shaping.FontFeature) string {
	if len(features) == 0 {
		return ""
	}
	buf := make([]byte, 0, 8*len(features))
	for _, t := range features {
		buf = binary.LittleEndian.AppendUint32(buf, uint32(t.Tag))
		buf = binary.LittleEndian.AppendUint32(buf, t.Value)
	}
	return string(buf)
}
