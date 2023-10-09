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
	delayedCommands = []func(){}

	delayedCommandsM       sync.Mutex
	delayedCommandsFlushed uint32
)

func flushDelayedCommands() {
	if atomic.LoadUint32(&delayedCommandsFlushed) == 0 {
		// Outline the slow-path to expect the fast-path is inlined.
		flushDelayedCommandsSlow()
	}
}

func flushDelayedCommandsSlow() {
	delayedCommandsM.Lock()
	defer delayedCommandsM.Unlock()

	if delayedCommandsFlushed == 0 {
		for _, f := range delayedCommands {
			f()
		}
		delayedCommands = nil
		delayedCommandsFlushed = 1
	}
}

// maybeCanAddDelayedCommand returns false if the delayed commands cannot be added.
// Otherwise, maybeCanAddDelayedCommand's returning value is not determined.
// For example, maybeCanAddDelayedCommand can return true even when flushing is being processed.
func maybeCanAddDelayedCommand() bool {
	return atomic.LoadUint32(&delayedCommandsFlushed) == 0
}

func tryAddDelayedCommand(f func()) bool {
	delayedCommandsM.Lock()
	defer delayedCommandsM.Unlock()

	if delayedCommandsFlushed == 0 {
		delayedCommands = append(delayedCommands, f)
		return true
	}

	return false
}

func checkDelayedCommandsFlushed(fname string) {
	if atomic.LoadUint32(&delayedCommandsFlushed) == 0 {
		panic("buffered: the command queue is not available yet at " + fname)
	}
}
