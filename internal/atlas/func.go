// Copyright 2023 The Ebitengine Authors
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

package atlas

import (
	"sync"
)

var (
	deferred []func()

	// deferredM is a mutex for the slice operations. This must not be used for other usages.
	deferredM sync.Mutex
)

func appendDeferred(f func()) {
	deferredM.Lock()
	defer deferredM.Unlock()
	deferred = append(deferred, f)
}

func flushDeferred() {
	deferredM.Lock()
	fs := deferred
	deferred = nil
	deferredM.Unlock()

	for _, f := range fs {
		f()
	}
}

type funcsInFrame struct {
	initOnce      sync.Once
	funcsCh       chan func()
	funcsAckCh    chan struct{}
	beginFrameCh  chan struct{}
	endFrameCh    chan struct{}
	endFrameAckCh chan struct{}
}

var theFuncsInFrame funcsInFrame

func (f *funcsInFrame) beginFrame() {
	f.initOnce.Do(func() {
		f.funcsCh = make(chan func())
		f.funcsAckCh = make(chan struct{})
		f.beginFrameCh = make(chan struct{})
		f.endFrameCh = make(chan struct{})
		f.endFrameAckCh = make(chan struct{})
		go func() {
			<-f.beginFrameCh
			for {
				select {
				case fn := <-f.funcsCh:
					fn()
					f.funcsAckCh <- struct{}{}
				case <-f.endFrameCh:
					f.endFrameAckCh <- struct{}{}
					// Wait for the next frame.
					<-f.beginFrameCh
				}
			}
		}()
	})

	f.beginFrameCh <- struct{}{}
}

func (f *funcsInFrame) endFrame() {
	f.endFrameCh <- struct{}{}
	// Ensure that all the queued functions are consumed and the loop is suspended.
	<-f.endFrameAckCh
}

func (f *funcsInFrame) runFuncInFrame(fn func()) {
	f.funcsCh <- fn
	<-f.funcsAckCh
}
