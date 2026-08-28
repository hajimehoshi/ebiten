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

package gamepaddb

import (
	"fmt"
	"sync"
	"testing"
)

func TestAndroidDefaultMappingsConcurrent(t *testing.T) {
	SetPlatformForTesting(platformAndroid)
	defer SetPlatformForTesting(0)

	const goroutines = 16
	ids := make([]string, goroutines)
	for g := range goroutines {
		ids[g] = fmt.Sprintf("030000004c0500006802000001%02x0000", g)
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	for g := range goroutines {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			<-start
			for range 1000 {
				_ = HasStandardButton(ids[g], StandardButtonRightBottom)
			}
		}(g)
	}
	close(start)
	wg.Wait()

	for g := range goroutines {
		if !HasStandardButton(ids[g], StandardButtonRightBottom) {
			t.Errorf("the Android default mapping was not registered for %s", ids[g])
		}
	}
}
