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

package loop

import (
	"errors"

	"github.com/hajimehoshi/ebiten/internal/clock"
)

const FPS = clock.FPS

func CurrentFPS() float64 {
	return clock.CurrentFPS()
}

type runContext struct{}

var (
	theRunContext *runContext
	contextInitCh = make(chan struct{})
)

func Start() error {
	// TODO: Need lock here?
	if theRunContext != nil {
		return errors.New("loop: The game is already running")
	}
	theRunContext = &runContext{}
	close(contextInitCh)
	return nil
}

func End() {
	theRunContext = nil
}
