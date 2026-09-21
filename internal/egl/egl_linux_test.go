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

package egl

import (
	"testing"
	"unsafe"
)

func TestChooseConfigs(t *testing.T) {
	const count = 40
	c := &Context{display: 1}
	queries := 0
	c.api.ChooseConfig = func(display uintptr, attribList *int32, configs *uintptr, configSize int32, numConfig *int32) bool {
		queries++
		if queries == 1 {
			if configs != nil || configSize != 0 {
				t.Error("first call must query the config count")
			}
			*numConfig = count
			return true
		}
		if configSize != count {
			t.Errorf("configSize = %d, want %d", configSize, count)
		}
		out := unsafe.Slice(configs, int(configSize))
		for i := range out {
			out[i] = uintptr(i + 1)
		}
		*numConfig = count
		return true
	}
	configs, err := c.ChooseConfigs([]int32{None})
	if err != nil {
		t.Fatal(err)
	}
	if queries != 2 || len(configs) != count || configs[count-1] != count {
		t.Errorf("queries = %d, configs = %v", queries, configs)
	}
}
