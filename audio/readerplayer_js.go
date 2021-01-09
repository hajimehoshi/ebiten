// Copyright 2021 The Ebiten Authors
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
	"io"
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2/audio/internal/go2cpp"
)

func isReaderContextAvailable() bool {
	return js.Global().Get("go2cpp").Truthy()
}

func newReaderDriverImpl(sampleRate int) readerDriver {
	return &go2cppDriverWrapper{go2cpp.NewContext(sampleRate)}
}

type go2cppDriverWrapper struct {
	c *go2cpp.Context
}

func (w *go2cppDriverWrapper) NewPlayer(r io.Reader) readerDriverPlayer {
	return w.c.NewPlayer(r)
}

func (w *go2cppDriverWrapper) Close() error {
	return w.c.Close()
}
