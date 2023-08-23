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

// smallImageSet is a set for images.
// smallImageSet uses a slice assuming the number of items is not so big (smaller than 100 or so).
// If the number of items is big, a map might be better than a slice.
type smallImageSet struct {
	s []*Image

	tmp []*Image
}

func (s *smallImageSet) add(image *Image) {
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

func (s *smallImageSet) remove(image *Image) {
	for i, img := range s.s {
		if img == image {
			copy(s.s[i:], s.s[i+1:])
			s.s[len(s.s)-1] = nil
			s.s = s.s[:len(s.s)-1]
			return
		}
	}
}

func (s *smallImageSet) forEach(f func(*Image)) {
	// Copy images to a temporary buffer since f might modify the original slice s.s (#2729).
	s.tmp = append(s.tmp, s.s...)

	for _, img := range s.tmp {
		f(img)
	}

	// Clear the temporary buffer.
	for i := range s.tmp {
		s.tmp[i] = nil
	}
	s.tmp = s.tmp[:0]
}

func (s *smallImageSet) clear() {
	for i := range s.s {
		s.s[i] = nil
	}
	s.s = s.s[:0]
}
