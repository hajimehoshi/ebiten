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

//go:build !js && !playstation5

package opengl_test

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl/gl"
)

type readbackContext struct {
	gl.Context
	buffers map[uint32][]byte
	bound   uint32
	created uint32
	deleted int
	value   byte
	wait    uint32
}

func (c *readbackContext) CreateBuffer() uint32 {
	c.created++
	if c.buffers == nil {
		c.buffers = map[uint32][]byte{}
	}
	return c.created
}

func (c *readbackContext) BindBuffer(target, b uint32) {
	c.bound = b
}

func (c *readbackContext) BufferInit(target uint32, size int, usage uint32) {
	c.buffers[c.bound] = make([]byte, size)
}

func (c *readbackContext) DeleteBuffer(b uint32) {
	c.deleted++
	delete(c.buffers, b)
}

func (c *readbackContext) BindFramebuffer(target, framebuffer uint32) {
}

func (c *readbackContext) Flush() {
}

func (c *readbackContext) FenceSync(condition, flags uint32) uintptr {
	return 1
}

func (c *readbackContext) DeleteSync(uintptr) {
}

func (c *readbackContext) ClientWaitSync(uintptr, uint32, uint64) uint32 {
	return c.wait
}

func (c *readbackContext) ReadPixels([]byte, int32, int32, int32, int32, uint32, uint32) {
	for i := range c.buffers[c.bound] {
		c.buffers[c.bound][i] = c.value
	}
}

func (c *readbackContext) MapBufferRange(target uint32, offset, length int, access uint32) []byte {
	return c.buffers[c.bound][offset : offset+length]
}

func (c *readbackContext) UnmapBuffer(uint32) bool {
	return true
}

func (c *readbackContext) LoadFunctions() error {
	return nil
}

func (c *readbackContext) GetInteger(uint32) int {
	return 0
}

func (c *readbackContext) Enable(uint32) {
}

func (c *readbackContext) BlendFuncSeparate(uint32, uint32, uint32, uint32) {
}

func (c *readbackContext) BlendEquationSeparate(uint32, uint32) {
}

func (c *readbackContext) UseProgram(uint32) {
}

func readbackTestImage(t *testing.T, c *readbackContext) (*opengl.Image, *opengl.Graphics) {
	t.Helper()
	img, g := opengl.NewReadbackTestImage(c)
	t.Cleanup(func() {
		if err := g.Reset(); err != nil {
			t.Error(err)
		}
	})
	return img, g
}

func startReadback(t *testing.T, img *opengl.Image, pixels []byte) graphicsdriver.PixelsReadback {
	t.Helper()
	r, err := img.ReadPixelsAsync([]graphicsdriver.PixelsArgs{{
		Region: image.Rect(0, 0, len(pixels)/4, 1),
		Pixels: pixels,
	}})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func finishReadback(t *testing.T, r graphicsdriver.PixelsReadback) {
	t.Helper()
	done, err := r.Poll()
	if err != nil {
		t.Fatal(err)
	}
	if !done {
		t.Fatal("read-back did not complete")
	}
	r.Dispose()
}

func TestReadbackReusesCompletedBuffer(t *testing.T) {
	c := &readbackContext{wait: gl.ALREADY_SIGNALED}
	img, _ := readbackTestImage(t, c)
	first, second := make([]byte, 4), make([]byte, 4)
	c.value = 1
	r1 := startReadback(t, img, first)
	c.value = 2
	r2 := startReadback(t, img, second)
	if c.created != 2 {
		t.Errorf("pending reads created %d buffers, want 2", c.created)
	}
	finishReadback(t, r1)
	c.value = 3
	third := make([]byte, 4)
	r3 := startReadback(t, img, third)
	if c.created != 2 {
		t.Errorf("completed buffer was not reused: created %d buffers", c.created)
	}
	finishReadback(t, r2)
	finishReadback(t, r3)
	for j, pixels := range [][]byte{first, second, third} {
		for _, v := range pixels {
			if v != byte(j+1) {
				t.Errorf("read %d: got %d, want %d", j, v, j+1)
			}
		}
	}
}

func TestReadbackDoesNotReuseAbortedBuffer(t *testing.T) {
	c := &readbackContext{wait: gl.TIMEOUT_EXPIRED}
	img, _ := readbackTestImage(t, c)
	pixels := make([]byte, 4)
	r := startReadback(t, img, pixels)
	if done, err := r.Poll(); done || err != nil {
		t.Errorf("pending read: got (%v, %v)", done, err)
	}
	r.Dispose()
	c.wait = gl.ALREADY_SIGNALED
	finishReadback(t, startReadback(t, img, pixels))
	if c.created != 2 {
		t.Errorf("aborted buffer was reused: created %d buffers, want 2", c.created)
	}
}

func TestPixelPackBufferCacheLimits(t *testing.T) {
	for _, tc := range []struct {
		name  string
		size  int
		count int
	}{
		{
			name:  "count",
			size:  4,
			count: opengl.MaxPixelPackBuffers + 1,
		},
		{
			name:  "bytes",
			size:  opengl.MaxPixelPackBufferBytes / 2,
			count: 3,
		},
		{
			name:  "oversized",
			size:  opengl.MaxPixelPackBufferBytes + 1,
			count: 1,
		},
		{
			name:  "empty",
			size:  0,
			count: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &readbackContext{}
			_, g := readbackTestImage(t, c)
			for range tc.count {
				opengl.ReleasePixelPackBufferForTesting(g, c.CreateBuffer(), tc.size)
			}
			if c.deleted != 1 {
				t.Errorf("limit deleted %d buffers, want 1", c.deleted)
			}
			if err := g.Reset(); err != nil {
				t.Fatal(err)
			}
			if c.deleted != tc.count {
				t.Errorf("reset deleted %d buffers, want %d", c.deleted, tc.count)
			}
		})
	}
}

func TestPixelPackBufferCacheReset(t *testing.T) {
	c := &readbackContext{wait: gl.ALREADY_SIGNALED}
	img, g := readbackTestImage(t, c)
	finishReadback(t, startReadback(t, img, make([]byte, 4)))
	if err := g.Reset(); err != nil {
		t.Fatal(err)
	}
	finishReadback(t, startReadback(t, img, make([]byte, 8)))
	finishReadback(t, startReadback(t, img, make([]byte, 4)))
	if c.created != 3 {
		t.Errorf("reset or size change reused an incompatible buffer: created %d buffers, want 3", c.created)
	}
}
