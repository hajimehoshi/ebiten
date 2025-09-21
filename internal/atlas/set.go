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

import "slices"

// imageSmallSet is a small set for images.
// imageSmallSet uses a slice assuming the number of items is not so big (smaller than 100 or so).
// If the number of items is big, a map might be better than a slice.
type imageSmallSet struct {
	s []*Image

	tmp []*Image
}

func (s *imageSmallSet) add(image *Image) {
	if image == nil {
		panic("atlas: nil image cannot be added")
	}
	for _, img := range s.s {
		if img == image {
			return
		}
	}
	s.s = append(s.s, image)
}

func (s *imageSmallSet) remove(image *Image) {
	for i, img := range s.s {
		if img == image {
			copy(s.s[i:], s.s[i+1:])
			s.s[len(s.s)-1] = nil
			s.s = s.s[:len(s.s)-1]
			return
		}
	}
}

func (s *imageSmallSet) forEach(f func(*Image)) {
	// Copy images to a temporary buffer since f might modify the original slice s.s (#2729).
	s.tmp = append(s.tmp, s.s...)
	for _, img := range s.tmp {
		f(img)
	}
	s.tmp = slices.Delete(s.tmp, 0, len(s.tmp))
}

func (s *imageSmallSet) clear() {
	s.s = slices.Delete(s.s, 0, len(s.s))
}
