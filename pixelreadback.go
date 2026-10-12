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

package ebiten

import (
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// PixelReadback represents a pixel read started by [Image.ReadPixelsAsync].
type PixelReadback struct {
	result *graphicscommand.PixelReadback
}

// Poll reports whether the read has completed and its error, if any, without waiting for the GPU.
// Once done is true, Ebitengine no longer accesses the destination buffer.
// Repeated calls return the same completed result. Poll is concurrent-safe.
func (r *PixelReadback) Poll() (done bool, err error) {
	return r.result.Poll()
}

// Wait waits for the read to complete and returns its error, if any.
// It can be called inside Update or Draw without advancing the game to another frame.
// When Wait returns, Ebitengine no longer accesses the destination buffer.
// Repeated calls return the same result. Wait is concurrent-safe unless single-thread mode is enabled.
func (r *PixelReadback) Wait() error {
	for {
		if done, err := r.Poll(); done {
			return err
		}
		ui.Get().FlushAndPollPixelReadbacks()
		if done, err := r.Poll(); done {
			return err
		}
		// WebGL fences can only become signaled after returning to the browser's event loop.
		// A timer suspends this goroutine without holding rendering locks.
		time.Sleep(time.Millisecond)
	}
}
