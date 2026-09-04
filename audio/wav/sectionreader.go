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
	"errors"
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
// If the underlying source is not an io.Seeker, Seek returns an error.
func (s *sectionReader) Seek(offset int64, whence int) (int64, error) {
	seeker, ok := s.src.(io.Seeker)
	if !ok {
		return 0, fmt.Errorf("wav: source must be io.Seeker: %w", errors.ErrUnsupported)
	}

	var pos int64
	switch whence {
	case io.SeekStart:
		pos = offset
	case io.SeekCurrent:
		pos = s.pos + offset
	case io.SeekEnd:
		pos = s.size + offset
	default:
		return 0, fmt.Errorf("wav: whence must be io.SeekStart, io.SeekCurrent, or io.SeekEnd but was %d", whence)
	}
	if pos < 0 || pos > s.size {
		return 0, fmt.Errorf("wav: position must be in [0, %d] but was %d", s.size, pos)
	}

	if _, err := seeker.Seek(pos+s.offset, io.SeekStart); err != nil {
		return 0, err
	}
	s.pos = pos
	return s.pos, nil
}
