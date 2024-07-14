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

package wav

import (
	"fmt"
	"io"
)

// sectionReader is similar to io.SectionReader but takes an io.Reader instead of io.ReaderAt.
type sectionReader struct {
	src    io.Reader
	offset int64
	size   int64

	pos int64
}

// newSectionReader creates a new sectionReader.
func newSectionReader(src io.Reader, offset int64, size int64) *sectionReader {
	return &sectionReader{
		src:    src,
		offset: offset,
		size:   size,
	}
}

// Read is implementation of io.Reader's Read.
func (s *sectionReader) Read(p []byte) (int, error) {
	if s.pos >= s.size {
		return 0, io.EOF
	}
	if s.pos+int64(len(p)) > s.size {
		p = p[:s.size-s.pos]
	}
	n, err := s.src.Read(p)
	s.pos += int64(n)
	return n, err
}

// Seek is implementation of io.Seeker's Seek.
//
// If the underlying source is not an io.Seeker, Seek panics.
func (s *sectionReader) Seek(offset int64, whence int) (int64, error) {
	seeker, ok := s.src.(io.Seeker)
	if !ok {
		panic("wav: s.src must be io.Seeker but not")
	}

	switch whence {
	case io.SeekStart:
		offset += s.offset
	case io.SeekCurrent:
	case io.SeekEnd:
		offset += s.offset + s.size
		whence = io.SeekStart
	}
	n, err := seeker.Seek(offset, whence)
	if err != nil {
		return 0, err
	}
	s.pos = n - s.offset
	if s.pos < 0 || s.pos > s.size {
		return 0, fmt.Errorf("wav: position is out of range")
	}
	return s.pos, nil
}
