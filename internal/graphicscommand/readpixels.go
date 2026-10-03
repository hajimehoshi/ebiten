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

// readPixelsRequest is a pixel read-back that has been enqueued into the command queue and is not
// finished yet.
type readPixelsRequest struct {
	// args is the destination of the read pixels. The slices are owned by the caller of
	// ReadPixelsAsync and must not be touched until the result is published.
	args []graphicsdriver.PixelsArgs

	// result receives exactly one value: nil when the read-back is finished, or an error. The
	// channel is buffered so that publishing the result never blocks, even when the caller ignores
	// the channel.
	result chan error

	// readback is the graphics driver's read-back. This is nil while the enqueued command has not
	// been executed yet, or when the graphics driver doesn't support an asynchronous read-back and
	// the pixels have been read synchronously instead.
	//
	// This is written and read only on the render thread.
	readback graphicsdriver.PixelsReadback

	// finished represents whether the result has been published. This is used to ensure that the
	// result is published exactly once.
	//
	// This is written and read only on the render thread.
	finished bool
}

// newReadPixelsRequest returns a new read-back request for the given arguments.
func newReadPixelsRequest(args []graphicsdriver.PixelsArgs) *readPixelsRequest {
	return &readPixelsRequest{
		args:   args,
		result: make(chan error, 1),
	}
}

// publish completes the read-back with the given result.
//
// publish must be called on the render thread, and must be called at most once.
func (r *readPixelsRequest) publish(err error) {
	if r.finished {
		panic("graphicscommand: the read-back must not be published twice")
	}
	r.finished = true
	r.args = nil
	r.result <- err
}

// discard releases the graphics driver's resources for the read-back.
//
// discard must be called on the render thread, and must be called at most once.
func (r *readPixelsRequest) discard() {
	if r.readback == nil {
		return
	}
	r.readback.Discard()
	r.readback = nil
}

// readPixelsAsyncCommand represents a command to read pixels without blocking the caller until the
// GPU finishes.
type readPixelsAsyncCommand struct {
	// manager is the manager that owns this command. The read-back is tracked by this manager, as
	// the manager polls the pending read-backs at a flush.
	manager *commandQueueManager

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

// Exec executes the readPixelsAsyncCommand.
//
// Exec records the read-back into the graphics driver's command stream at this position, so the
// read pixels include the drawing commands preceding this command and exclude the following ones.
//
// A graphics driver that doesn't implement graphicsdriver.AsyncPixelsReader reads the pixels
// synchronously here. The command still doesn't wait for the GPU on the calling thread's behalf,
// so the caller of ReadPixelsAsync is not blocked.
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
		c.manager.addReadPixelsRequest(c.req)
		return nil
	}

	// A graphics driver that doesn't implement graphicsdriver.AsyncPixelsReader reads the pixels
	// synchronously. The result is published right away and the read-back is not tracked.
	err := c.img.image.ReadPixels(c.req.args)
	c.req.publish(err)
	return nil
}

// NeedsSync reports whether the command must be executed synchronously.
//
// The command must not need a synchronous flush: that is the whole point of an asynchronous
// read-back. The GPU work is tracked by the graphics driver's read-back instead.
func (c *readPixelsAsyncCommand) NeedsSync() bool {
	return false
}

// abort completes the read-back with err without reading the pixels.
//
// abort reports that this command has not run at all, which happens when an earlier command in the
// same flush failed. Without this, the caller of ReadPixelsAsync would wait forever.
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

// addReadPixelsRequest adds a read-back request to the pending read-backs.
//
// addReadPixelsRequest must be called on the render thread.
func (c *commandQueueManager) addReadPixelsRequest(req *readPixelsRequest) {
	c.pendingReadPixels = append(c.pendingReadPixels, req)
}

// pollReadPixels checks the pending read-backs and publishes the results of the finished ones.
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

// abortReadPixels completes all the pending read-backs with err without reading the pixels.
//
// abortReadPixels must be called on the render thread. A pending request must always receive a
// result, even when the graphics driver is not usable anymore, so that the callers of
// ReadPixelsAsync never wait forever.
func (c *commandQueueManager) abortReadPixels(err error) {
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

// readPixelsAsync enqueues a read-back for the image and returns the channel to receive the result.
//
// The read-back is enqueued in drawing order and this does not wait for the GPU.
func (c *commandQueueManager) readPixelsAsync(img *Image, args []graphicsdriver.PixelsArgs) <-chan error {
	img.flushBufferedWritePixels()
	req := newReadPixelsRequest(args)
	c.enqueueCommand(&readPixelsAsyncCommand{
		manager: c,
		img:     img,
		req:     req,
	})
	return req.result
}

// abortReadPixels completes requests whose commands have not executed yet.
// It must be called on the render thread, after any previously submitted queues.
func (q *commandQueue) abortReadPixels(err error) {
	if q == nil {
		return
	}
	for _, cmd := range q.commands {
		if c, ok := cmd.(*readPixelsAsyncCommand); ok {
			c.abort(err)
		}
	}
}

// stopReadPixels drains render-thread work before aborting the unsubmitted queue.
// The caller must hold the atlas backend mutex to prevent concurrent command recording.
func (c *commandQueueManager) stopReadPixels(err error) {
	q := c.current
	thread.Call(theRenderThread, func() {
		q.abortReadPixels(err)
		c.abortReadPixels(err)
	})
}
