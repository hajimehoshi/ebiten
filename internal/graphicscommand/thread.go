// Copyright 2020 The Ebiten Authors
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

package graphicscommand

import (
	"github.com/hajimehoshi/ebiten/v2/internal/thread"
)

var theRenderThread thread.Thread = thread.NewNoopThread()

// SetRenderThread must be called from the rendering thread where e.g. OpenGL works.
//
// TODO: Create thread in this package instead of setting it externally.
func SetRenderThread(thread thread.Thread) {
	theRenderThread = thread
}

// runOnRenderThread calls f on the rendering thread.
func runOnRenderThread(f func()) {
	theRenderThread.Call(f)
}
