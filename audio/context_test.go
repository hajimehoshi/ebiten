// Copyright 2018 The Ebiten Authors
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
)

type (
	dummyContext struct{}
	dummyPlayer  struct{}
)

func (d *dummyContext) NewPlayer() io.WriteCloser {
	return &dummyPlayer{}
}

func (d *dummyContext) Close() error {
	return nil
}

func (p *dummyPlayer) Write(b []byte) (int, error) {
	return len(b), nil
}

func (p *dummyPlayer) Close() error {
	return nil
}

func init() {
	contextForTesting = &dummyContext{}
}

type dummyHook struct {
	update func() error
}

func (h *dummyHook) OnSuspendAudio(f func()) {
}

func (h *dummyHook) OnResumeAudio(f func()) {
}

func (h *dummyHook) AppendHookOnBeforeUpdate(f func() error) {
	h.update = f
}

func init() {
	hookForTesting = &dummyHook{}
}

func UpdateForTesting() error {
	return hookForTesting.(*dummyHook).update()
}
