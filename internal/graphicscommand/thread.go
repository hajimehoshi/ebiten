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
	"context"

	"github.com/hajimehoshi/ebiten/v2/internal/thread"
)

var theRenderThread thread.Thread = thread.NewNoopThread()

// SetOSThreadAsRenderThread sets an OS thread as the rendering thread e.g. for OpenGL.
func SetOSThreadAsRenderThread() {
	theRenderThread = thread.NewOSThread()
}

func LoopRenderThread(ctx context.Context) {
	_ = theRenderThread.Loop(ctx)
}

// runOnRenderThread calls f with arg on the rendering thread and returns its result.
func runOnRenderThread[A, R any](f func(A) R, arg A) R {
	return thread.CallWithArgAndResult(theRenderThread, f, arg)
}

// runOnRenderThreadAsync queues f with arg on the rendering thread.
func runOnRenderThreadAsync[A any](f func(A), arg A) {
	// As the render thread's task channel is unbuffered,
	// CallAsync blocks when the previously-queued task is not executed yet.
	// This blocking is expected as double-buffering is used.
	thread.CallAsync(theRenderThread, f, arg)
}

// WaitForRenderThread blocks until all the tasks queued on the rendering thread so far have been executed.
func WaitForRenderThread() {
	// The rendering thread executes tasks in order, so an empty task finishes after all the previously queued ones.
	thread.Call(theRenderThread, func() {})
}

// AbortReadPixels completes pending and queued pixel read-backs with an error.
// Their results have been delivered when AbortReadPixels returns.
// The caller must hold the atlas backend mutex.
func AbortReadPixels() {
	theCommandQueueManager.abortReadPixels(errReadPixelsAborted)
}
