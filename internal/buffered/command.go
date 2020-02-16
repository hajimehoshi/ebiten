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

var (
	needsToDelayCommands = true

	// delayedCommands represents a queue for image operations that are ordered before the game starts
	// (BeginFrame). Before the game starts, the package shareable doesn't determine the minimum/maximum texture
	// sizes (#879).
	delayedCommands  []func() error
	delayedCommandsM sync.Mutex
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
	delayedCommandsM.Lock()
	defer delayedCommandsM.Unlock()

	fs := make([]func() error, len(delayedCommands))
	copy(fs, delayedCommands)
	delayedCommands = nil
	needsToDelayCommands = false
	return fs
}
