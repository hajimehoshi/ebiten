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

//go:build darwin && !ios

package metal

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/mtl"
)

var _ graphicsdriver.AsyncPixelsReader = (*Image)(nil)

type readback struct {
	commandBuffer mtl.CommandBuffer
	textures      []mtl.Texture
	args          []graphicsdriver.PixelsArgs
}

// ReadPixelsAsync records texture copies without waiting for the GPU.
func (i *Image) ReadPixelsAsync(args []graphicsdriver.PixelsArgs) (_ graphicsdriver.PixelsReadback, err error) {
	// Validate everything before recording commands or allocating resources.
	for _, arg := range args {
		if got, want := len(arg.Pixels), 4*arg.Region.Dx()*arg.Region.Dy(); got != want {
			return nil, fmt.Errorf("metal: len(Pixels) must be %d but %d at ReadPixelsAsync", want, got)
		}
	}
	r := &readback{args: args}
	defer func() {
		if err != nil {
			r.Dispose()
		}
	}()
	if len(args) == 0 {
		return r, nil
	}
	g := i.graphics
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
	for _, arg := range args {
		if arg.Region.Empty() {
			r.textures = append(r.textures, mtl.Texture{})
			continue
		}
		// A separate managed texture preserves the captured pixels across later
		// writes or disposal. Synchronize it before exposing its CPU copy.
		texture, err := g.view.getMTLDevice().NewTextureWithDescriptor(mtl.TextureDescriptor{
			TextureType: mtl.TextureType2D,
			PixelFormat: mtl.PixelFormatRGBA8UNorm,
			Width:       arg.Region.Dx(),
			Height:      arg.Region.Dy(),
			StorageMode: mtl.StorageModeManaged,
		})
		if err != nil {
			return nil, fmt.Errorf("metal: device.NewTextureWithDescriptor failed: %w", err)
		}
		r.textures = append(r.textures, texture)
		bce.CopyFromTexture(i.texture, 0, 0,
			mtl.Origin{X: arg.Region.Min.X, Y: arg.Region.Min.Y},
			mtl.Size{Width: arg.Region.Dx(), Height: arg.Region.Dy(), Depth: 1},
			texture, 0, 0, mtl.Origin{})
		bce.SynchronizeTexture(texture, 0, 0)
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
		// Copy below only after Metal has synchronized the managed textures.
	case mtl.CommandBufferStatusError:
		return true, fmt.Errorf("metal: asynchronous pixel read-back command buffer failed")
	default:
		return false, nil
	}
	for n, arg := range r.args {
		if arg.Region.Empty() {
			continue
		}
		if err := r.textures[n].GetBytes(arg.Pixels, 4*arg.Region.Dx(), mtl.Region{
			Size: mtl.Size{Width: arg.Region.Dx(), Height: arg.Region.Dy(), Depth: 1},
		}, 0); err != nil {
			return true, err
		}
	}
	return true, nil
}

// Dispose releases the staging textures and command buffer.
func (r *readback) Dispose() {
	// Metal command buffers retain encoded resources until execution finishes,
	// including when a pending read-back is discarded during shutdown.
	for _, texture := range r.textures {
		if texture != (mtl.Texture{}) {
			texture.Release()
		}
	}
	r.textures = nil
	r.args = nil
	if r.commandBuffer != (mtl.CommandBuffer{}) {
		r.commandBuffer.Release()
		r.commandBuffer = mtl.CommandBuffer{}
	}
}
