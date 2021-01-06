// Copyright 2019 The Ebiten Authors
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
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/hooks"
)

// writerContext represents a context represented as io.WriteClosers.
// The actual implementation is oto.Context.
type writerContext interface {
	NewPlayer() io.WriteCloser
	io.Closer
}

var writerContextForTesting writerContext

func newWriterContext(sampleRate int) writerContext {
	if writerContextForTesting != nil {
		return writerContextForTesting
	}

	ch := make(chan struct{})
	var once sync.Once
	hooks.AppendHookOnBeforeUpdate(func() error {
		once.Do(func() {
			close(ch)
		})
		return nil
	})
	return newOtoContext(sampleRate, ch)
}
