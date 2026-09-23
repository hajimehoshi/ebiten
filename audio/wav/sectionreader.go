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

	"github.com/hajimehoshi/ebiten/v2/internal/mathutil"
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

// Read is an implementation of io.Reader's Read.
func (s *sectionReader) Read(p []byte) (int, error) {
	if s.pos >= s.size {
		return 0, io.EOF
	}
	if int64(len(p)) > s.size-s.pos {
		p = p[:s.size-s.pos]
	}
	n, err := s.src.Read(p)
	s.pos += int64(n)
	return n, err
}

// Seek is an implementation of io.Seeker's Seek.
//
// If the underlying source is not an io.Seeker, Seek returns an error.
func (s *sectionReader) Seek(offset int64, whence int) (int64, error) {
	seeker, ok := s.src.(io.Seeker)
	if !ok {
		return 0, fmt.Errorf("wav: source must be io.Seeker: %w", errors.ErrUnsupported)
	}

	var base int64
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		base = s.pos
	case io.SeekEnd:
		base = s.size
	default:
		return 0, fmt.Errorf("wav: whence must be io.SeekStart, io.SeekCurrent, or io.SeekEnd but was %d", whence)
	}
	pos, ok := mathutil.AddForSeek(base, offset)
	if !ok {
		return 0, fmt.Errorf("wav: invalid seek position")
	}
	if pos > s.size {
		return 0, fmt.Errorf("wav: position must be in [0, %d] but was %d", s.size, pos)
	}
	sourcePos, ok := mathutil.AddForSeek(pos, s.offset)
	if !ok {
		return 0, fmt.Errorf("wav: source position overflows int64")
	}

	if _, err := seeker.Seek(sourcePos, io.SeekStart); err != nil {
		return 0, err
	}
	s.pos = pos
	return s.pos, nil
}
