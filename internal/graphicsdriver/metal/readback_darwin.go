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

//go:build darwin

package metal

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/mtl"
)

var _ graphicsdriver.AsyncPixelsReader = (*Image)(nil)

type readback struct {
	commandBuffer mtl.CommandBuffer
	buffer        mtl.Buffer
	args          []graphicsdriver.PixelsArgs
}

// ReadPixelsAsync records texture copies without waiting for the GPU.
func (i *Image) ReadPixelsAsync(args []graphicsdriver.PixelsArgs) (_ graphicsdriver.PixelsReadback, err error) {
	// Validate everything before recording commands or allocating resources.
	var size uintptr
	for _, arg := range args {
		if got, want := len(arg.Pixels), 4*arg.Region.Dx()*arg.Region.Dy(); got != want {
			return nil, fmt.Errorf("metal: len(Pixels) must be %d but %d at ReadPixelsAsync", want, got)
		}
		size += uintptr(len(arg.Pixels))
	}
	r := &readback{args: args}
	defer func() {
		if err != nil {
			r.Dispose()
		}
	}()
	if size == 0 {
		return r, nil
	}
	g := i.graphics
	// A separate shared buffer preserves the capture across later writes or disposal.
	r.buffer, err = g.view.getMTLDevice().NewBufferWithLength(size, mtl.ResourceStorageModeShared)
	if err != nil {
		return nil, fmt.Errorf("metal: device.NewBufferWithLength failed: %w", err)
	}

	g.flushRenderCommandEncoderIfNeeded()
	if err := g.ensureCommandBuffer(); err != nil {
		return nil, err
	}
	r.commandBuffer = g.cb
	r.commandBuffer.Retain()
	bce, err := g.cb.BlitCommandEncoder()
	if err != nil {
		return nil, fmt.Errorf("metal: cb.BlitCommandEncoder failed: %w", err)
	}
	defer bce.EndEncoding()
	var offset uintptr
	for _, arg := range args {
		if arg.Region.Empty() {
			continue
		}
		bce.CopyFromTextureToBuffer(i.texture, 0, 0,
			mtl.Origin{X: arg.Region.Min.X, Y: arg.Region.Min.Y},
			mtl.Size{Width: arg.Region.Dx(), Height: arg.Region.Dy(), Depth: 1},
			r.buffer, offset, uintptr(4*arg.Region.Dx()), 0)
		offset += uintptr(len(arg.Pixels))
	}
	return r, nil
}

// Poll copies the captured pixels after the command buffer completes.
func (r *readback) Poll() (bool, error) {
	if r.commandBuffer == (mtl.CommandBuffer{}) {
		return true, nil
	}
	switch r.commandBuffer.Status() {
	case mtl.CommandBufferStatusCompleted:
		// Shared storage is CPU-visible after GPU execution completes.
	case mtl.CommandBufferStatusError:
		return true, fmt.Errorf("metal: asynchronous pixel read-back command buffer failed")
	default:
		return false, nil
	}
	var offset uintptr
	for _, arg := range r.args {
		mtl.CopyFromBufferAt(r.buffer, arg.Pixels, offset)
		offset += uintptr(len(arg.Pixels))
	}
	return true, nil
}

// Dispose releases the staging buffer and command buffer.
func (r *readback) Dispose() {
	// Metal command buffers retain encoded resources until execution finishes,
	// including when a pending read-back is discarded during shutdown.
	if r.buffer != (mtl.Buffer{}) {
		r.buffer.Release()
		r.buffer = mtl.Buffer{}
	}
	r.args = nil
	if r.commandBuffer != (mtl.CommandBuffer{}) {
		r.commandBuffer.Release()
		r.commandBuffer = mtl.CommandBuffer{}
	}
}
