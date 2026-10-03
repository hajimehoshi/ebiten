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

//go:build !playstation5

package opengl

import (
	"errors"
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl/gl"
)

// syncObject is an OpenGL sync object, created by FenceSync. The zero value means no sync object.
//
// A sync object is a pointer, so this is a uintptr rather than a uint32 like the other object names.
type syncObject uintptr

// readback is a pixel read-back that has been recorded into the command stream and whose pixels are
// copied to the caller once a fence reports the reads are complete.
//
// A readback owns its pixel pack buffers and its fence, so it can outlive the image it read from:
// glReadPixels and glFenceSync have already been recorded, and OpenGL keeps the objects they refer
// to alive until the commands referencing them finish.
type readback struct {
	graphics *Graphics

	// pbos are the pixel pack buffers holding the read pixels, one per read region.
	//
	// Each region uses a separate buffer and is read at offset zero.
	pbos []buffer

	// fence is a fence sync object that is signaled once every glReadPixels above is complete.
	//
	// A single fence is enough as the reads are recorded in order, and thus the fence is recorded
	// after all of them.
	fence syncObject

	// regions are the read regions, in the same order as the arguments of ReadPixelsAsync.
	regions []image.Rectangle
}

// ReadPixelsAsync records pixel reads into pixel pack buffers and returns without waiting for the
// GPU.
//
// The reads are recorded in the command stream at the current position, so the read pixels include
// the preceding drawing commands and exclude the following ones.
func (i *Image) ReadPixelsAsync(args []graphicsdriver.PixelsArgs) (graphicsdriver.PixelsReadback, error) {
	c := &i.graphics.context
	if err := i.ensureFramebuffer(); err != nil {
		return nil, err
	}

	c.bindFramebuffer(i.framebuffer.native)

	r := &readback{
		graphics: i.graphics,
	}
	for _, arg := range args {
		size := 4 * arg.Region.Dx() * arg.Region.Dy()
		if len(arg.Pixels) != size {
			r.Discard()
			return nil, fmt.Errorf("opengl: len(Pixels) must be %d but %d at ReadPixelsAsync", size, len(arg.Pixels))
		}
		b, err := c.newPixelPackBuffer(size)
		if err != nil {
			r.Discard()
			return nil, err
		}
		c.readPixelsToPixelPackBuffer(arg.Region)
		r.pbos = append(r.pbos, b)
		r.regions = append(r.regions, arg.Region)
	}

	f, err := c.newFence()
	if err != nil {
		r.Discard()
		return nil, err
	}
	r.fence = f
	// Submit the fence so polling does not depend on a later frame flush.
	c.ctx.Flush()

	// The read commands must not be affected by the drawing commands that follow, and the pixel
	// pack buffer must not be bound while another command reads into it.
	c.ctx.BindBuffer(gl.PIXEL_PACK_BUFFER, 0)
	return r, nil
}

// Poll reports whether the reads are complete. Poll must not block.
func (r *readback) Poll() (bool, error) {
	// A zero timeout makes this a pure query that never blocks the render thread.
	return r.graphics.context.pollFence(r.fence)
}

// Copy copies the read pixels to args. Copy must be called only after Poll reported done.
func (r *readback) Copy(args []graphicsdriver.PixelsArgs) error {
	if len(args) != len(r.pbos) {
		return fmt.Errorf("opengl: len(args) must be %d but %d at Copy", len(r.pbos), len(args))
	}
	for j, arg := range args {
		if err := r.graphics.context.readPixelsFromPixelPackBuffer(r.pbos[j], r.regions[j], arg.Pixels); err != nil {
			return err
		}
	}
	return nil
}

// Discard releases the resources for the read-back without copying the pixels.
func (r *readback) Discard() {
	c := &r.graphics.context
	if r.fence != 0 {
		c.deleteFence(r.fence)
		r.fence = 0
	}
	for j, b := range r.pbos {
		c.deleteBuffer(b)
		r.pbos[j] = 0
	}
	r.pbos = nil
}

func (c *context) newPixelPackBuffer(size int) (buffer, error) {
	b := c.ctx.CreateBuffer()
	if b <= 0 {
		return 0, errors.New("opengl: creating a pixel pack buffer failed")
	}
	c.ctx.BindBuffer(gl.PIXEL_PACK_BUFFER, b)
	c.ctx.BufferInit(gl.PIXEL_PACK_BUFFER, size, gl.STREAM_READ)
	return buffer(b), nil
}

func (c *context) deleteBuffer(b buffer) {
	if b == 0 {
		return
	}
	c.ctx.DeleteBuffer(uint32(b))
}

// readPixelsToPixelPackBuffer records a read of region into the bound pixel pack buffer.
//
// The framebuffer to read from and a pixel pack buffer to read into must both be bound.
func (c *context) readPixelsToPixelPackBuffer(region image.Rectangle) {
	x := int32(region.Min.X)
	y := int32(region.Min.Y)
	width := int32(region.Dx())
	height := int32(region.Dy())
	// A nil destination means reading into the bound GL_PIXEL_PACK_BUFFER.
	//
	// Unlike a read into client memory, the rows are packed without padding. As the format is
	// RGBA/UNSIGNED_BYTE, a row is always 4-byte aligned anyway, so the layout is the same and no
	// row-stride fixup is necessary.
	c.ctx.ReadPixels(nil, x, y, width, height, gl.RGBA, gl.UNSIGNED_BYTE)
}

// readPixelsFromPixelPackBuffer copies the pixels that region was read into b to buf.
func (c *context) readPixelsFromPixelPackBuffer(b buffer, region image.Rectangle, buf []byte) error {
	if got, want := len(buf), 4*region.Dx()*region.Dy(); got != want {
		return fmt.Errorf("opengl: len(buf) must be %d but %d at readPixelsFromPixelPackBuffer", want, got)
	}
	if len(buf) == 0 {
		return nil
	}
	c.ctx.BindBuffer(gl.PIXEL_PACK_BUFFER, uint32(b))
	err := c.ctx.ReadBufferData(gl.PIXEL_PACK_BUFFER, 0, buf)
	c.ctx.BindBuffer(gl.PIXEL_PACK_BUFFER, 0)
	return err
}

func (c *context) newFence() (syncObject, error) {
	if s := c.ctx.FenceSync(gl.SYNC_GPU_COMMANDS_COMPLETE, 0); s != 0 {
		return syncObject(s), nil
	}
	return 0, errors.New("opengl: creating a fence sync object failed")
}

func (c *context) deleteFence(s syncObject) {
	c.ctx.DeleteSync(uintptr(s))
}

// pollFence reports whether the fence is signaled. pollFence must not block.
func (c *context) pollFence(s syncObject) (bool, error) {
	switch r := c.ctx.ClientWaitSync(uintptr(s), 0, 0); r {
	case gl.ALREADY_SIGNALED, gl.CONDITION_SATISFIED:
		return true, nil
	case gl.TIMEOUT_EXPIRED:
		return false, nil
	case gl.WAIT_FAILED:
		// This happens when the context is lost. The pixels can never become available, so report
		// the read-back as finished with an error instead of waiting forever.
		return true, errors.New("opengl: waiting for a pixel read-back failed: the context might be lost")
	default:
		return true, fmt.Errorf("opengl: unexpected ClientWaitSync result: %d", r)
	}
}
