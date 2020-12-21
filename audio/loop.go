// Copyright 2017 The Ebiten Authors
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

package audio

import (
	"fmt"
	"io"
)

// InfiniteLoop represents a looped stream which never ends.
type InfiniteLoop struct {
	src     io.ReadSeeker
	lstart  int64
	llength int64
	pos     int64
}

// NewInfiniteLoop creates a new infinite loop stream with a source stream and length in bytes.
func NewInfiniteLoop(src io.ReadSeeker, length int64) *InfiniteLoop {
	return NewInfiniteLoopWithIntro(src, 0, length)
}

// NewInfiniteLoopWithIntro creates a new infinite loop stream with an intro part.
// NewInfiniteLoopWithIntro accepts a source stream src, introLength in bytes and loopLength in bytes.
func NewInfiniteLoopWithIntro(src io.ReadSeeker, introLength int64, loopLength int64) *InfiniteLoop {
	return &InfiniteLoop{
		src:     src,
		lstart:  introLength,
		llength: loopLength,
		pos:     -1,
	}
}

func (i *InfiniteLoop) length() int64 {
	return i.lstart + i.llength
}

func (i *InfiniteLoop) ensurePos() error {
	if i.pos >= 0 {
		return nil
	}
	pos, err := i.src.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	if pos >= i.length() {
		return fmt.Errorf("audio: stream position must be less than the specified length")
	}
	i.pos = pos
	return nil
}

// Read is implementation of ReadSeekCloser's Read.
func (i *InfiniteLoop) Read(b []byte) (int, error) {
	if err := i.ensurePos(); err != nil {
		return 0, err
	}

	if i.pos+int64(len(b)) > i.length() {
		b = b[:i.length()-i.pos]
	}

	n, err := i.src.Read(b)
	i.pos += int64(n)
	if i.pos > i.length() {
		panic(fmt.Sprintf("audio: position must be <= length but not at (*InfiniteLoop).Read: pos: %d, length: %d", i.pos, i.length()))
	}

	if err != nil && err != io.EOF {
		return 0, err
	}

	if err == io.EOF || i.pos == i.length() {
		pos, err := i.Seek(i.lstart, io.SeekStart)
		if err != nil {
			return 0, err
		}
		i.pos = pos
	}
	return n, nil
}

// Seek is implementation of ReadSeekCloser's Seek.
func (i *InfiniteLoop) Seek(offset int64, whence int) (int64, error) {
	if err := i.ensurePos(); err != nil {
		return 0, err
	}

	next := int64(0)
	switch whence {
	case io.SeekStart:
		next = offset
	case io.SeekCurrent:
		next = i.pos + offset
	case io.SeekEnd:
		return 0, fmt.Errorf("audio: whence must be io.SeekStart or io.SeekCurrent for InfiniteLoop")
	}
	if next < 0 {
		return 0, fmt.Errorf("audio: position must >= 0")
	}
	if next >= i.lstart {
		next = ((next - i.lstart) % i.llength) + i.lstart
	}
	// Ignore the new position returned by Seek since the source position might not be match with the position
	// managed by this.
	if _, err := i.src.Seek(next, io.SeekStart); err != nil {
		return 0, err
	}
	i.pos = next
	return i.pos, nil
}
