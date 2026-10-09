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
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/thread"
)

// errReadPixelsAborted indicates that a pixel read-back was completed without reading the pixels
// because the graphics driver is not usable anymore.
var errReadPixelsAborted = errors.New("graphicscommand: the pixel read-back was aborted")

// readPixelsRequest holds the arguments, result channel, and driver resources for a pixel read-back.
//
// The readback and finished fields of readPixelsRequest are accessed only on the render thread.
type readPixelsRequest struct {
	args []graphicsdriver.PixelsArgs

	// The channel is buffered so publication never blocks the render thread.
	result chan error

	// readback is the graphics driver's read-back. This is nil while the enqueued command has not
	// been executed yet, or when the graphics driver doesn't support an asynchronous read-back and
	// the pixels have been read synchronously instead.
	readback graphicsdriver.PixelsReadback

	// finished prevents publishing a second result.
	finished bool
}

// newReadPixelsRequest creates a pixel read-back request with a buffered result channel.
func newReadPixelsRequest(args []graphicsdriver.PixelsArgs) *readPixelsRequest {
	return &readPixelsRequest{
		args:   args,
		result: make(chan error, 1),
	}
}

// publish sends the error, if any, closes the result channel, and releases the retained arguments.
//
// publish must be called on the render thread, and must be called at most once.
func (r *readPixelsRequest) publish(err error) {
	if r.finished {
		panic("graphicscommand: the read-back must not be published twice")
	}
	r.finished = true
	r.args = nil
	if err != nil {
		r.result <- err
	}
	close(r.result)
}

// discard releases the driver resources for the read-back.
//
// discard must be called on the render thread, and must be called at most once.
func (r *readPixelsRequest) discard() {
	if r.readback == nil {
		return
	}
	r.readback.Discard()
	r.readback = nil
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
		c.req.publish(errors.New("graphicscommand: the image is not available at ReadPixelsAsync"))
		return nil
	}

	if r, ok := c.img.image.(graphicsdriver.AsyncPixelsReader); ok {
		readback, err := r.ReadPixelsAsync(c.req.args)
		if err != nil {
			c.req.publish(err)
			return nil
		}
		c.req.readback = readback
		// Track the read-back so that a later flush can publish its result.
		theCommandQueueManager.addReadPixelsRequest(c.req)
		return nil
	}

	// A graphics driver that doesn't implement graphicsdriver.AsyncPixelsReader reads the pixels
	// synchronously. The result is published right away and the read-back is not tracked.
	err := c.img.image.ReadPixels(c.req.args)
	c.req.publish(err)
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
		// Exec has already published a result.
		return
	}
	if c.req.readback != nil {
		// Exec has started the read-back, and the manager owns it.
		return
	}
	c.req.publish(err)
}

// addReadPixelsRequest tracks a submitted read-back until it finishes.
//
// addReadPixelsRequest must be called on the render thread.
func (c *commandQueueManager) addReadPixelsRequest(req *readPixelsRequest) {
	c.pendingReadPixels = append(c.pendingReadPixels, req)
}

// pollReadPixels publishes results for completed read-backs and releases their resources.
//
// pollReadPixels must be called on the render thread at a flush, after the frame's commands have
// been submitted to the graphics driver.
func (c *commandQueueManager) pollReadPixels() {
	if len(c.pendingReadPixels) == 0 {
		return
	}

	// Polling in the submission order lets a finished read-back publish its result first, which
	// keeps the completion order as close to the submission order as possible.
	pending := c.pendingReadPixels[:0]
	for _, req := range c.pendingReadPixels {
		done, err := req.readback.Poll()
		if !done {
			pending = append(pending, req)
			continue
		}
		if err == nil {
			err = req.readback.Copy(req.args)
		}
		req.readback.Discard()
		req.readback = nil
		req.publish(err)
	}
	clear(c.pendingReadPixels[len(pending):])
	c.pendingReadPixels = pending
}

// abortPendingReadPixels completes all the pending read-backs with err without reading the pixels.
//
// abortPendingReadPixels must be called on the render thread. A pending request must always receive a
// result, even when the graphics driver is not usable anymore, so that the callers of
// ReadPixelsAsync never wait forever.
func (c *commandQueueManager) abortPendingReadPixels(err error) {
	if len(c.pendingReadPixels) == 0 {
		return
	}
	for _, req := range c.pendingReadPixels {
		req.discard()
		req.publish(err)
	}
	clear(c.pendingReadPixels)
	c.pendingReadPixels = nil
}

// abortQueuedReadPixels completes requests whose commands have not executed yet.
// It must be called on the render thread, after any previously submitted queues.
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

// abortReadPixels completes pending and queued pixel read-backs with err on the render thread.
// The caller must hold the atlas backend mutex to prevent concurrent command recording.
func (c *commandQueueManager) abortReadPixels(err error) {
	q := c.current
	thread.Call(theRenderThread, func() {
		q.abortQueuedReadPixels(err)
		c.abortPendingReadPixels(err)
	})
}
