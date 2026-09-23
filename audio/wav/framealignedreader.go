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

package wav

import (
	"errors"
	"fmt"
	"io"
)

// frameAlignedReader reads src to its end, discarding a partial frame at the end.
// frameAlignedReader is used when the size of the 'data' chunk is unknown.
type frameAlignedReader struct {
	src       io.Reader
	frameSize int

	// buf holds the bytes read from src but not returned yet.
	buf []byte
	// pos is the number of bytes returned so far.
	pos int64
	eof bool
}

// newFrameAlignedReader creates a new frameAlignedReader.
func newFrameAlignedReader(src io.Reader, frameSize int) *frameAlignedReader {
	return &frameAlignedReader{
		src:       src,
		frameSize: frameSize,
	}
}

// available returns the number of buffered bytes that complete a frame.
func (r *frameAlignedReader) available() int {
	frameSize := int64(r.frameSize)
	total := r.pos + int64(len(r.buf))
	return int(total/frameSize*frameSize - r.pos)
}

// Read is an implementation of io.Reader's Read.
func (r *frameAlignedReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	// Keep reading until one frame is available so that a source returning less than one frame
	// at a time doesn't make Read return (0, nil).
	l := max(len(p), r.frameSize)
	var readErr error
	for len(r.buf) < l && !r.eof {
		origLen := len(r.buf)
		if cap(r.buf) < l {
			r.buf = append(r.buf, make([]byte, l-origLen)...)
		}
		n, err := r.src.Read(r.buf[origLen:l])
		r.buf = r.buf[:origLen+n]
		if err == io.EOF {
			r.eof = true
			break
		}
		if err != nil {
			readErr = err
			break
		}
		if r.available() > 0 || n == 0 {
			break
		}
	}

	n := copy(p, r.buf[:r.available()])
	r.pos += int64(n)
	r.buf = r.buf[:copy(r.buf, r.buf[n:])]
	if readErr != nil {
		return n, readErr
	}
	if r.eof && r.available() == 0 {
		return n, io.EOF
	}
	return n, nil
}

// Seek is an implementation of io.Seeker's Seek.
//
// Seek always returns an error wrapping errors.ErrUnsupported, as the source is not seekable.
func (r *frameAlignedReader) Seek(offset int64, whence int) (int64, error) {
	return 0, fmt.Errorf("wav: source must be io.Seeker: %w", errors.ErrUnsupported)
}
