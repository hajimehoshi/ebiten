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

package opengl

import (
	"image"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl/gl"
)

type readbackContext struct {
	gl.Context
	buffers     map[uint32][]byte
	bound       uint32
	created     uint32
	deleted     []uint32
	value       byte
	wait        uint32
	mapFailed   bool
	unmapFailed bool
	fenceFailed bool
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
	c.deleted = append(c.deleted, b)
	delete(c.buffers, b)
}

func (c *readbackContext) BindFramebuffer(target, framebuffer uint32) {
}

func (c *readbackContext) Flush() {
}

func (c *readbackContext) FenceSync(condition, flags uint32) uintptr {
	if c.fenceFailed {
		return 0
	}
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
	if c.mapFailed {
		return nil
	}
	return c.buffers[c.bound][offset : offset+length]
}

func (c *readbackContext) UnmapBuffer(uint32) bool {
	return !c.unmapFailed
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

func readbackTestImage(c *readbackContext) *Image {
	return &Image{
		graphics:    &Graphics{context: context{ctx: c}},
		framebuffer: &framebuffer{native: 1},
	}
}

func startReadback(t *testing.T, img *Image, pixels []byte) graphicsdriver.PixelsReadback {
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
	img := readbackTestImage(c)
	defer img.graphics.clearPixelPackBuffers()
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

func TestReadbackDoesNotReuseFailedBuffer(t *testing.T) {
	for _, name := range []string{"abort", "wait", "map", "unmap", "fence"} {
		t.Run(name, func(t *testing.T) {
			c := &readbackContext{wait: gl.ALREADY_SIGNALED}
			img := readbackTestImage(c)
			defer img.graphics.clearPixelPackBuffers()
			pixels := make([]byte, 4)
			if name == "fence" {
				c.fenceFailed = true
				if _, err := img.ReadPixelsAsync([]graphicsdriver.PixelsArgs{{
					Region: image.Rect(0, 0, 1, 1),
					Pixels: pixels,
				}}); err == nil {
					t.Error("fence creation unexpectedly succeeded")
				}
				c.fenceFailed = false
			} else {
				r := startReadback(t, img, pixels)
				c.wait = gl.TIMEOUT_EXPIRED
				if done, err := r.Poll(); done || err != nil {
					t.Errorf("pending read: got (%v, %v)", done, err)
				}
				if name != "abort" {
					c.wait = gl.ALREADY_SIGNALED
					c.mapFailed = name == "map"
					c.unmapFailed = name == "unmap"
					if name == "wait" {
						c.wait = gl.WAIT_FAILED
					}
					if _, err := r.Poll(); err == nil {
						t.Error("read unexpectedly succeeded")
					}
				}
				r.Dispose()
			}
			if !slices.Equal(c.deleted, []uint32{1}) {
				t.Errorf("deleted buffers: got %v, want [1]", c.deleted)
			}
			c.wait = gl.ALREADY_SIGNALED
			c.mapFailed, c.unmapFailed = false, false
			finishReadback(t, startReadback(t, img, pixels))
			if c.created != 2 {
				t.Errorf("failed buffer was reused: created %d buffers", c.created)
			}
		})
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
			count: maxPixelPackBuffers + 1,
		},
		{
			name:  "bytes",
			size:  maxPixelPackBufferBytes / 2,
			count: 3,
		},
		{
			name:  "oversized",
			size:  maxPixelPackBufferBytes + 1,
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
			g := readbackTestImage(c).graphics
			for j := 1; j <= tc.count; j++ {
				g.releasePixelPackBuffer(buffer(j), tc.size)
			}
			if !slices.Equal(c.deleted, []uint32{1}) {
				t.Errorf("deleted buffers: got %v, want [1]", c.deleted)
			}
			g.clearPixelPackBuffers()
			want := make([]uint32, tc.count)
			for j := range want {
				want[j] = uint32(j + 1)
			}
			if !slices.Equal(c.deleted, want) {
				t.Errorf("cleared buffers: got %v, want %v", c.deleted, want)
			}
		})
	}
}

func TestPixelPackBufferCacheReset(t *testing.T) {
	c := &readbackContext{wait: gl.ALREADY_SIGNALED}
	img := readbackTestImage(c)
	finishReadback(t, startReadback(t, img, make([]byte, 4)))
	if err := img.graphics.Reset(); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(c.deleted, []uint32{1}) {
		t.Errorf("deleted buffers: got %v, want [1]", c.deleted)
	}
	finishReadback(t, startReadback(t, img, make([]byte, 8)))
	finishReadback(t, startReadback(t, img, make([]byte, 4)))
	if c.created != 3 {
		t.Errorf("reset or size change reused an incompatible buffer: created %d buffers, want 3", c.created)
	}
	img.graphics.clearPixelPackBuffers()
	if !slices.Equal(c.deleted, []uint32{1, 2, 3}) {
		t.Errorf("deleted buffers: got %v, want [1 2 3]", c.deleted)
	}
}
