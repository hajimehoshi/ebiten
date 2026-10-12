// Copyright 2026 The Ebitengine Authors
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

package ui

import (
	"errors"
	"sync"
)

// Serialize session reservations with mode selection before graphics startup.
var vmHostState struct {
	mu           sync.Mutex
	sessions     int
	singleThread bool
}

// RegisterVMHost reserves a host session until UnregisterVMHost is called.
func RegisterVMHost() error {
	vmHostState.mu.Lock()
	defer vmHostState.mu.Unlock()
	if buildTagSingleThread || vmHostState.singleThread {
		return errors.New("vmhost: hosting guests is not supported in single-thread mode")
	}
	vmHostState.sessions++
	return nil
}

func UnregisterVMHost() {
	vmHostState.mu.Lock()
	defer vmHostState.mu.Unlock()
	vmHostState.sessions--
}

func setVMHostThreadMode(singleThread bool) error {
	vmHostState.mu.Lock()
	defer vmHostState.mu.Unlock()
	singleThread = singleThread || buildTagSingleThread
	if singleThread && vmHostState.sessions > 0 {
		return errors.New("vmhost: hosting guests is not supported in single-thread mode")
	}
	// Keep the selected mode after Run returns: graphics resources can outlive the loop.
	vmHostState.singleThread = singleThread
	return nil
}
