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

package buffered

import (
	"sync"
	"sync/atomic"
)

var (
	// delayedCommands represents a queue for image operations that are ordered before the game starts
	// (BeginFrame). Before the game starts, the package shareable doesn't determine the minimum/maximum texture
	// sizes (#879).
	delayedCommands = []func() error{}

	delayedCommandsM       sync.Mutex
	delayedCommandsFlushed uint32
)

func flushDelayedCommands() error {
	fs := getDelayedFuncsAndClear()
	for _, f := range fs {
		if err := f(); err != nil {
			return err
		}
	}
	return nil

}

func getDelayedFuncsAndClear() []func() error {
	if atomic.LoadUint32(&delayedCommandsFlushed) == 0 {
		// Outline the slow-path to expect the fast-path is inlined.
		return getDelayedFuncsAndClearSlow()
	}
	return nil
}

func getDelayedFuncsAndClearSlow() []func() error {
	delayedCommandsM.Lock()
	defer delayedCommandsM.Unlock()

	if delayedCommandsFlushed == 0 {
		defer atomic.StoreUint32(&delayedCommandsFlushed, 1)

		fs := make([]func() error, len(delayedCommands))
		copy(fs, delayedCommands)
		delayedCommands = nil
		return fs
	}

	return nil
}

// maybeCanAddDelayedCommand returns false if the delayed commands cannot be added.
// Otherwise, maybeCanAddDelayedCommand's returning value is not determined.
// For example, maybeCanAddDelayedCommand can return true even when flusing is being processed.
func maybeCanAddDelayedCommand() bool {
	return atomic.LoadUint32(&delayedCommandsFlushed) == 0
}

func tryAddDelayedCommand(f func() error) bool {
	delayedCommandsM.Lock()
	defer delayedCommandsM.Unlock()

	if delayedCommandsFlushed == 0 {
		delayedCommands = append(delayedCommands, func() error {
			return f()
		})
		return true
	}

	return false
}

func checkDelayedCommandsFlushed(fname string) {
	if atomic.LoadUint32(&delayedCommandsFlushed) == 0 {
		panic("buffered: the command queue is not available yet at " + fname)
	}
}
