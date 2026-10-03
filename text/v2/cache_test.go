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

package text_test

import (
	"testing"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

func TestCacheNonExpiringEntry(t *testing.T) {
	c := text.NewCacheForTesting[string, int](1)
	var calls int
	permanent := func() (int, bool) {
		calls++
		return 42, false
	}
	if got := c.GetOrCreateAt("permanent", 0, permanent); got != 42 {
		t.Errorf("initial value = %d, want 42", got)
	}
	c.GetOrCreateAt("permanent", 1, permanent)
	c.GetOrCreateAt("old", 1, func() (int, bool) { return 1, true })
	c.GetOrCreateAt("pressure", 62, func() (int, bool) { return 2, true })
	if got := c.GetOrCreateAt("permanent", 62, permanent); got != 42 {
		t.Errorf("value after eviction pressure = %d, want 42", got)
	}
	if calls != 1 {
		t.Errorf("non-expiring value created %d times, want 1", calls)
	}
}
