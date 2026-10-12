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

package graphicscommand

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/thread"
)

// errReadPixelsAborted indicates that a pixel read-back was completed without reading the pixels
// because the graphics driver is not usable anymore.
var errReadPixelsAborted = errors.New("graphicscommand: the pixel read-back was aborted")

// PixelReadback is the completion result of a pixel read-back.
type PixelReadback struct {
	done chan struct{}

	// err is immutable after done closes. Receiving from done publishes it and the destination pixels.
	err error
}

// NewCompletedPixelReadback returns a completed read-back with the given error.
func NewCompletedPixelReadback(err error) *PixelReadback {
	r := &PixelReadback{
		done: make(chan struct{}),
		err:  err,
	}
	close(r.done)
	return r
}

// Poll reports whether the read-back has completed and its error, if any.
func (r *PixelReadback) Poll() (bool, error) {
	select {
	case <-r.done:
		return true, r.err
	default:
		return false, nil
	}
}

// FlushAndPollPixelReadbacks submits queued commands and checks completion on the render thread.
// The caller must hold the atlas backend mutex.
func FlushAndPollPixelReadbacks(graphicsDriver graphicsdriver.Graphics) error {
	// Intermediate flushes retain their queues until the end of the frame.
	// Only submit when there are commands so waiting does not accumulate empty queues.
	if q := theCommandQueueManager.current; q != nil && len(q.commands) != 0 {
		if err := theCommandQueueManager.flush(graphicsDriver, graphicsdriver.FlushModeIntermediate); err != nil {
			return err
		}
	}
	thread.Call(theRenderThread, func() {
		theCommandQueueManager.pollReadPixels()
	})
	if err := theCommandQueueManager.error(); err != nil {
		theCommandQueueManager.abortReadPixels(err)
		return err
	}
	return nil
}

// TryPollPixelReadbacks requests a completion check if the render loop is accepting work.
func TryPollPixelReadbacks() {
	// Only render-thread-owned pending requests are accessed, so no atlas lock is needed.
	// A busy, inactive, or stopped loop leaves completion to a later poll or frame flush.
	thread.TryCallAsync(theRenderThread, (*commandQueueManager).pollReadPixels, &theCommandQueueManager)
}

// FlushAndWaitPixelReadbacks submits queued commands and completes the given read-backs.
// The caller must hold the atlas backend mutex.
func FlushAndWaitPixelReadbacks(graphicsDriver graphicsdriver.Graphics, results []*PixelReadback) error {
	if !slices.ContainsFunc(results, func(result *PixelReadback) bool {
		done, _ := result.Poll()
		return !done
	}) {
		return nil
	}
	if err := FlushAndPollPixelReadbacks(graphicsDriver); err != nil {
		return err
	}
	thread.Call(theRenderThread, func() {
		theCommandQueueManager.waitReadPixels(results)
	})
	return nil
}

func (c *commandQueueManager) waitReadPixels(results []*PixelReadback) {
	for _, req := range c.pendingReadPixels {
		if slices.Contains(results, req.result) {
			req.finish(req.readback.Wait())
		}
	}
	c.pendingReadPixels = slices.DeleteFunc(c.pendingReadPixels, func(req *readPixelsRequest) bool {
		return req.finished
	})
}

// readPixelsRequest holds the arguments, result, and driver resources for a pixel read-back.
//
// The readback and finished fields of readPixelsRequest are accessed only on the render thread.
type readPixelsRequest struct {
	args []graphicsdriver.PixelsArgs

	result *PixelReadback

	// readback is the graphics driver's read-back, or nil before it starts and after it finishes.
	readback graphicsdriver.PixelReadback

	// finished reports whether the request has been completed.
	finished bool
}

// newReadPixelsRequest creates a pixel read-back request with a completion result.
func newReadPixelsRequest(args []graphicsdriver.PixelsArgs) *readPixelsRequest {
	return &readPixelsRequest{
		args: args,
		result: &PixelReadback{
			done: make(chan struct{}),
		},
	}
}

// finish releases the driver read-back and retained arguments and publishes the result.
//
// finish must be called on the render thread, and must be called at most once.
func (r *readPixelsRequest) finish(err error) {
	if r.finished {
		panic("graphicscommand: the read-back must not be finished twice")
	}
	r.finished = true
	if r.readback != nil {
		r.readback.Dispose()
		r.readback = nil
	}
	r.args = nil
	r.result.err = err
	close(r.result.done)
}

// readPixelsAsyncCommand starts a pixel read-back at its position in the command queue.
type readPixelsAsyncCommand struct {
	img *Image
	req *readPixelsRequest
}

func (c *readPixelsAsyncCommand) String() string {
	var args []string
	for _, a := range c.req.args {
		args = append(args, fmt.Sprintf("region: %s", a.Region.String()))
	}
	return fmt.Sprintf("read-pixels-async: image: %d, args: %s", c.img.id, strings.Join(args, ", "))
}

// Exec records the read-back into the graphics driver's command stream at this position, so the
// read pixels include the drawing commands preceding this command and exclude the following ones.
func (c *readPixelsAsyncCommand) Exec(commandQueue *commandQueue, graphicsDriver graphicsdriver.Graphics, indexOffset int) error {
	if c.img.image == nil {
		// The image is not created yet, which happens when the newImageCommand failed.
		c.req.finish(errors.New("graphicscommand: the image is not available at ReadPixelsAsync"))
		return nil
	}

	readback, err := c.img.image.ReadPixelsAsync(c.req.args)
	if err != nil {
		c.req.finish(err)
		return nil
	}
	c.req.readback = readback
	// Track the read-back so that a later flush can finish it.
	theCommandQueueManager.addReadPixelsRequest(c.req)
	return nil
}

func (c *readPixelsAsyncCommand) NeedsSync() bool {
	return false
}

// abort completes the read-back with err without reading the pixels.
//
// abort reports that this command has not run at all, which happens when an earlier command in the
// same flush failed.
//
// abort must be called on the render thread, and must be called at most once.
func (c *readPixelsAsyncCommand) abort(err error) {
	if c.req.finished {
		// Exec has already finished the request.
		return
	}
	if c.req.readback != nil {
		// Exec has started the read-back, and the manager owns it.
		return
	}
	c.req.finish(err)
}

// addReadPixelsRequest tracks a submitted read-back until it finishes.
//
// addReadPixelsRequest must be called on the render thread.
func (c *commandQueueManager) addReadPixelsRequest(req *readPixelsRequest) {
	c.pendingReadPixels = append(c.pendingReadPixels, req)
}

// pollReadPixels finishes completed read-backs and releases their resources.
//
// pollReadPixels must be called on the render thread.
func (c *commandQueueManager) pollReadPixels() {
	// Polling in the submission order lets a finished read-back complete its request first, which
	// keeps the completion order as close to the submission order as possible.
	for _, req := range c.pendingReadPixels {
		done, err := req.readback.Poll()
		if err != nil {
			req.finish(err)
			continue
		}
		if done {
			req.finish(nil)
		}
	}
	c.pendingReadPixels = slices.DeleteFunc(c.pendingReadPixels, func(req *readPixelsRequest) bool {
		return req.finished
	})
}

// abortPendingReadPixels completes with err the submitted read-backs that have not finished.
// It must be called on the render thread.
func (c *commandQueueManager) abortPendingReadPixels(err error) {
	for _, req := range c.pendingReadPixels {
		req.finish(err)
	}
	c.pendingReadPixels = slices.Delete(c.pendingReadPixels, 0, len(c.pendingReadPixels))
}

// abortQueuedReadPixels completes with err the read-backs in q whose commands have not run.
// It must be called on the render thread.
func (q *commandQueue) abortQueuedReadPixels(err error) {
	if q == nil {
		return
	}
	for _, cmd := range q.commands {
		if c, ok := cmd.(*readPixelsAsyncCommand); ok {
			c.abort(err)
		}
	}
}

// abortReadPixels completes the read-backs pending or queued so far with err.
// The caller must hold the atlas backend mutex.
func (c *commandQueueManager) abortReadPixels(err error) {
	// Commands are recorded under the atlas backend mutex, so the current queue cannot change
	// while the render thread walks it.
	q := c.current
	thread.Call(theRenderThread, func() {
		q.abortQueuedReadPixels(err)
		c.abortPendingReadPixels(err)
	})
}
