// Copyright 2016 Hajime Hoshi
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

package vorbis

import (
	"bytes"
	"time"
)

type Stream struct {
	buf        *bytes.Reader
	sampleRate int
}

func (s *Stream) Read(p []byte) (int, error) {
	return s.buf.Read(p)
}

func (s *Stream) Seek(offset int64, whence int) (int64, error) {
	return s.buf.Seek(offset, whence)
}

func (s *Stream) Close() error {
	s.buf = nil
	return nil
}

func (s *Stream) Len() time.Duration {
	const bytesPerSample = 4
	return time.Duration(s.buf.Len()/bytesPerSample) * time.Second / time.Duration(s.sampleRate)
}
