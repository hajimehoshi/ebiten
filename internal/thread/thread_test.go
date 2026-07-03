// Copyright 2026 The Ebiten Authors
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

package thread_test

import (
	"context"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/thread"
)

func TestNestedLoop(t *testing.T) {
	th := thread.NewOSThread()

	ctx := t.Context()
	go func() {
		_ = th.Loop(ctx)
	}()

	var values []int
	th.Call(func() {
		values = append(values, 1)

		// While this function blocks the thread, another goroutine's Call must be
		// processed by NestedLoop.
		nestedCtx, nestedCancel := context.WithCancel(context.Background())
		go func() {
			defer nestedCancel()
			th.Call(func() {
				values = append(values, 2)
			})
		}()
		_ = th.NestedLoop(nestedCtx)

		values = append(values, 3)
	})
	th.Call(func() {
		values = append(values, 4)
	})

	if got, want := values, []int{1, 2, 3, 4}; !slices.Equal(got, want) {
		t.Errorf("got: %v, want: %v", got, want)
	}
}
