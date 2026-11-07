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

type faceOption interface {
	font.Variation | shaping.FontFeature
}

func optionTag[T faceOption](entry T) font.Tag {
	switch entry := any(entry).(type) {
	case font.Variation:
		return entry.Tag
	case shaping.FontFeature:
		return entry.Tag
	default:
		panic("text: unexpected face option type")
	}
}

// optionStorage keeps small option sets inline. Larger sets use an immutable
// shared slice, copied before an effective mutation. No stored slice points into
// inline. T is a concrete go-text entry type so values needs no conversion.
type optionStorage[T faceOption] struct {
	inline   [4]T
	overflow []T
	count    int
}

func (o *optionStorage[T]) values() []T {
	if o.count <= len(o.inline) {
		return o.inline[:o.count]
	}
	return o.overflow
}

// set reports whether the entries changed.
func (o *optionStorage[T]) set(entry T) bool {
	tag := optionTag(entry)
	values := o.values()
	idx := len(values)
	for i, current := range values {
		currentTag := optionTag(current)
		if currentTag < tag {
			continue
		}
		if currentTag > tag {
			idx = i
			break
		}
		if current == entry {
			return false
		}
		if o.count > len(o.inline) {
			values = slices.Clone(values)
			o.overflow = values
		}
		values[i] = entry
		return true
	}

	// Keep tags sorted so insertion order does not affect the cache key.
	if o.count < len(o.inline) {
		values = o.inline[:o.count+1]
		copy(values[idx+1:], values[idx:o.count])
	} else {
		next := make([]T, o.count+1)
		copy(next, values[:idx])
		copy(next[idx+1:], values[idx:])
		values = next
		o.overflow = values
	}
	values[idx] = entry
	o.count++
	return true
}

// remove reports whether the entries changed.
func (o *optionStorage[T]) remove(tag font.Tag) bool {
	values := o.values()
	for i, entry := range values {
		entryTag := optionTag(entry)
		if entryTag < tag {
			continue
		}
		if entryTag > tag {
			return false
		}
		n := o.count - 1
		if n <= len(o.inline) {
			copy(o.inline[:i], values[:i])
			copy(o.inline[i:n], values[i+1:])
			clear(o.inline[n:])
			o.overflow = nil
		} else {
			next := make([]T, n)
			copy(next, values[:i])
			copy(next[i:], values[i+1:])
			o.overflow = next
		}
		o.count = n
		return true
	}
	return false
}

type variations struct {
	optionStorage[font.Variation]
	key string
}

func (o *variations) set(tag font.Tag, value float32) {
	if o.optionStorage.set(font.Variation{Tag: tag, Value: value}) {
		o.key = encodeVariations(o.values())
	}
}

func (o *variations) remove(tag font.Tag) {
	if o.optionStorage.remove(tag) {
		o.key = encodeVariations(o.values())
	}
}

type features struct {
	optionStorage[shaping.FontFeature]
	key string
}

func (o *features) set(tag font.Tag, value uint32) {
	if o.optionStorage.set(shaping.FontFeature{Tag: tag, Value: value}) {
		o.key = encodeFeatures(o.values())
	}
}

func (o *features) remove(tag font.Tag) {
	if o.optionStorage.remove(tag) {
		o.key = encodeFeatures(o.values())
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
