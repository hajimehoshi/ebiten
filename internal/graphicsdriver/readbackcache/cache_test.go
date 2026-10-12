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

package readbackcache_test

import (
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/readbackcache"
)

func TestPut(t *testing.T) {
	const n = readbackcache.MaxCount
	var c readbackcache.Cache[int]
	for i := range n {
		if dropped := c.Put(i, 1); len(dropped) != 0 {
			t.Errorf("c.Put(%d, 1) dropped %v, want none", i, dropped)
		}
	}

	if got, want := c.Put(n, 1), []int{0}; !slices.Equal(got, want) {
		t.Errorf("c.Put(%d, 1) dropped %v, want %v", n, got, want)
	}

	want := make([]int, n)
	for i := range want {
		want[i] = i + 1
	}
	if got := c.Put(n+1, readbackcache.MaxSize); !slices.Equal(got, want) {
		t.Errorf("c.Put(%d, %d) dropped %v, want %v", n+1, readbackcache.MaxSize, got, want)
	}

	if got, want := c.Put(n+2, readbackcache.MaxSize+1), []int{n + 2}; !slices.Equal(got, want) {
		t.Errorf("c.Put(%d, %d) dropped %v, want %v", n+2, readbackcache.MaxSize+1, got, want)
	}

	for _, size := range []int{1, readbackcache.MaxSize + 1} {
		if v, ok := c.Take(size); ok {
			t.Errorf("c.Take(%d) = (%d, true), want (0, false)", size, v)
		}
	}
	if v, ok := c.Take(readbackcache.MaxSize); !ok || v != n+1 {
		t.Errorf("c.Take(%d) = (%d, %t), want (%d, true)", readbackcache.MaxSize, v, ok, n+1)
	}
}
