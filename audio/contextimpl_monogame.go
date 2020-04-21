// Copyright 2020 The Ebiten Authors
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

// +build monogame

package audio

import (
	"io"
	"time"
)

func newContextImpl(sampleRate int, initCh chan struct{}) context {
	return &mockContext{
		sampleRate: sampleRate,
	}
}

type mockContext struct {
	sampleRate int
}

func (*mockContext) Close() error {
	return nil
}

func (m *mockContext) NewPlayer() io.WriteCloser {
	return &mockPlayer{
		sampleRate: m.sampleRate,
	}
}

type mockPlayer struct {
	sampleRate int
}

func (m *mockPlayer) Write(bs []byte) (int, error) {
	bytesPerSec := 4 * m.sampleRate

	n := len(bs)
	time.Sleep(time.Duration(n) * time.Second / time.Duration(bytesPerSec))
	return n, nil
}

func (*mockPlayer) Close() error {
	return nil
}
