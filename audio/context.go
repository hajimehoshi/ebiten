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

type context interface {
	NewPlayer() io.WriteCloser
	io.Closer
}

var contextForTesting context

func newContext(sampleRate int) context {
	if contextForTesting != nil {
		return contextForTesting
	}

	ch := make(chan struct{})
	var once sync.Once
	hooks.AppendHookOnBeforeUpdate(func() error {
		once.Do(func() {
			close(ch)
		})
		return nil
	})
	return newContextImpl(sampleRate, ch)
}

type hook interface {
	OnSuspendAudio(f func())
	OnResumeAudio(f func())
	AppendHookOnBeforeUpdate(f func() error)
}

var hookForTesting hook

func getHook() hook {
	if hookForTesting != nil {
		return hookForTesting
	}
	return &hookImpl{}
}

type hookImpl struct{}

func (h *hookImpl) OnSuspendAudio(f func()) {
	hooks.OnSuspendAudio(f)
}

func (h *hookImpl) OnResumeAudio(f func()) {
	hooks.OnResumeAudio(f)
}

func (h *hookImpl) AppendHookOnBeforeUpdate(f func() error) {
	hooks.AppendHookOnBeforeUpdate(f)
}
