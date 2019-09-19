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
)

type command struct {
	f func()
}

var (
	delayedCommandsFlushable bool

	// delayedCommands represents a queue for image operations that are ordered before the game starts
	// (BeginFrame). Before the game starts, the package shareable doesn't determine the minimum/maximum texture
	// sizes (#879).
	//
	// TODO: Flush the commands only when necessary (#921).
	delayedCommands  []*command
	delayedCommandsM sync.Mutex
)

func makeDelayedCommandFlushable() {
	delayedCommandsM.Lock()
	delayedCommandsFlushable = true
	delayedCommandsM.Unlock()
}

func enqueueDelayedCommand(f func()) {
	delayedCommandsM.Lock()
	delayedCommands = append(delayedCommands, &command{
		f: f,
	})
	delayedCommandsM.Unlock()
}

func flushDelayedCommands() bool {
	delayedCommandsM.Lock()
	defer delayedCommandsM.Unlock()

	if !delayedCommandsFlushable {
		return false
	}

	for _, c := range delayedCommands {
		c.f()
	}
	delayedCommands = nil
	return true
}
