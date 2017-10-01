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

// InfiniteLoop represents a loop which never ends.
type InfiniteLoop struct {
	stream ReadSeekCloser
	size   int64
}

// NewInfiniteLoop creates a new infinite loop stream with a stream and size in bytes.
func NewInfiniteLoop(stream ReadSeekCloser, size int64) *InfiniteLoop {
	return &InfiniteLoop{
		stream: stream,
		size:   size,
	}
}

// Read is implementation of ReadSeekCloser's Read.
func (i *InfiniteLoop) Read(b []byte) (int, error) {
	n, err := i.stream.Read(b)
	if err == io.EOF {
		if _, err := i.Seek(0, io.SeekStart); err != nil {
			return 0, err
		}
		err = nil
	}
	return n, err
}

// Seek is implementation of ReadSeekCloser's Seek.
func (i *InfiniteLoop) Seek(offset int64, whence int) (int64, error) {
	next := int64(0)
	switch whence {
	case io.SeekStart:
		next = offset
	case io.SeekCurrent:
		current, err := i.stream.Seek(0, io.SeekCurrent)
		if err != nil {
			return 0, err
		}
		next = current + offset
	case io.SeekEnd:
		return 0, fmt.Errorf("audio: whence must be 0 or 1 for InfiniteLoop")
	}
	next %= i.size
	pos, err := i.stream.Seek(next, io.SeekStart)
	if err != nil {
		return 0, err
	}
	return pos, nil
}

// Close is implementation of ReadSeekCloser's Close.
func (l *InfiniteLoop) Close() error {
	return l.stream.Close()
}
