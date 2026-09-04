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

package vmhost

import (
	"fmt"
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
	"github.com/hajimehoshi/ebiten/v2/internal/vmprotocol"
)

// frameRenderer replays a guest's recorded graphics commands onto the host GPU by re-issuing them at
// the internal ui.Image layer, through the host's ordinary ebiten rendering stack.
//
// render is called from the session goroutine, while dispose must be called within the host's frame.
type frameRenderer struct {
	images  map[graphicsdriver.ImageID]*hostImage
	shaders map[graphicsdriver.ShaderID]*hostShader

	vertices []float32
	indices  []uint32

	// vtxBuf and idxBuf are reused per draw to forward only the vertices a draw references, rebased to a
	// zero origin. Forwarding the whole shared vertex buffer per draw makes the host's command queue grow
	// quadratically with the number of batched draws.
	vtxBuf []float32
	idxBuf []uint32

	// screen is the mirror of the guest's screen framebuffer. It is a renderer-owned image, not the
	// outside screen: a frame is drawn through many commands, and the outside screen must advance from
	// one completed frame to the next (at CompositeFrame), never showing a partially drawn state.
	screen *hostImage

	// maxImageSize is the host graphics driver's maximum image size, 0 before it is first needed.
	maxImageSize int
}

type hostImage struct {
	img    *ui.Image
	width  int
	height int
}

type hostShader struct {
	shader *ui.Shader

	// uniformDwordCount is the number of uniform dwords the shader takes, including the preserved
	// prefix. A draw carrying a different number cannot be replayed.
	uniformDwordCount int
}

// newFrameRenderer returns an empty renderer.
func newFrameRenderer() *frameRenderer {
	return &frameRenderer{
		images:  map[graphicsdriver.ImageID]*hostImage{},
		shaders: map[graphicsdriver.ShaderID]*hostShader{},
	}
}

// dispose releases every host GPU resource the renderer created. It must be called within the host's
// frame.
func (f *frameRenderer) dispose() {
	for _, hi := range f.images {
		hi.img.Deallocate()
	}
	for _, hs := range f.shaders {
		hs.shader.Deallocate()
	}
	f.images = map[graphicsdriver.ImageID]*hostImage{}
	f.shaders = map[graphicsdriver.ShaderID]*hostShader{}
	f.screen = nil
}

// hostMaxImageSize returns the host graphics driver's maximum image size. It must not be called
// before the host's game starts.
func (f *frameRenderer) hostMaxImageSize() int {
	if f.maxImageSize == 0 {
		f.maxImageSize = ui.Get().GraphicsMaxImageSize()
	}
	return f.maxImageSize
}

// render replays the given commands. Image and shader identities persist across calls.
//
// A command no guest can have produced (a size, a count, or an index the host cannot replay) fails
// with an error: the host's rendering stack panics on such input rather than rejecting it.
func (f *frameRenderer) render(cmds []vmprotocol.GraphicsCommand) error {
	for _, c := range cmds {
		if err := f.renderOne(c); err != nil {
			return err
		}
	}
	return nil
}

func (f *frameRenderer) renderOne(c vmprotocol.GraphicsCommand) error {
	switch c.Kind {
	case vmprotocol.GraphicsCommandKindInitialize,
		vmprotocol.GraphicsCommandKindBegin,
		vmprotocol.GraphicsCommandKindEnd,
		vmprotocol.GraphicsCommandKindSetTransparent,
		vmprotocol.GraphicsCommandKindSetVsyncEnabled,
		vmprotocol.GraphicsCommandKindReadPixels:
		// Framing, vsync, and read-back are owned by the host's own ebiten loop, and the guest clears its
		// own screen to the alpha its transparency setting implies.
		return nil

	case vmprotocol.GraphicsCommandKindNewImage, vmprotocol.GraphicsCommandKindNewScreenFramebufferImage:
		if c.Width <= 0 || c.Height <= 0 {
			return fmt.Errorf("vmhost: %s needs a positive size but got %dx%d", c.Kind, c.Width, c.Height)
		}
		if maxSize := f.hostMaxImageSize(); c.Width > maxSize || c.Height > maxSize {
			return fmt.Errorf("vmhost: %s of %dx%d exceeds the host's maximum image size %d", c.Kind, c.Width, c.Height, maxSize)
		}
		// The guest assigns each image a fresh ID, so a live one being reused would silently orphan the
		// mirror it names.
		if _, ok := f.images[c.ImageID]; ok {
			return fmt.Errorf("vmhost: %s reuses the live image %d", c.Kind, c.ImageID)
		}
		// Mirror each guest backing image (a logical image or an atlas page) as an unmanaged host image
		// of the same size, so the host adds no atlas offset of its own and the recorded coordinates
		// (already relative to this backing image) replay 1:1.
		hi := &hostImage{
			img:    ui.Get().NewImage(c.Width, c.Height, atlas.ImageTypeUnmanaged),
			width:  c.Width,
			height: c.Height,
		}
		f.images[c.ImageID] = hi
		if c.Kind == vmprotocol.GraphicsCommandKindNewScreenFramebufferImage {
			f.screen = hi
		}
		return nil

	case vmprotocol.GraphicsCommandKindNewShader:
		if len(c.ShaderSource) == 0 {
			return fmt.Errorf("vmhost: no shader source forwarded for shader %d", c.ShaderID)
		}
		ir, err := graphics.CompileShader(c.ShaderSource)
		if err != nil {
			return fmt.Errorf("vmhost: recompiling forwarded shader %d failed: %w", c.ShaderID, err)
		}
		var uniformDwordCount int
		for _, u := range ir.Uniforms {
			uniformDwordCount += u.DwordCount()
		}
		f.shaders[c.ShaderID] = &hostShader{
			shader:            ui.NewShader(ir, ""),
			uniformDwordCount: uniformDwordCount,
		}
		return nil

	case vmprotocol.GraphicsCommandKindSetVertices:
		if len(c.Vertices)%graphics.VertexFloatCount != 0 {
			return fmt.Errorf("vmhost: SetVertices got %d floats, which is not a multiple of %d", len(c.Vertices), graphics.VertexFloatCount)
		}
		f.vertices = c.Vertices
		f.indices = c.Indices
		return nil

	case vmprotocol.GraphicsCommandKindDrawTriangles:
		dst, ok := f.images[c.Dst]
		if !ok {
			return fmt.Errorf("vmhost: DrawTriangles references unknown dst image %d", c.Dst)
		}
		shader, ok := f.shaders[c.ShaderID]
		if !ok {
			return fmt.Errorf("vmhost: DrawTriangles references unknown shader %d", c.ShaderID)
		}
		if len(c.Uniforms) != shader.uniformDwordCount {
			return fmt.Errorf("vmhost: DrawTriangles carries %d uniform dwords for shader %d, which takes %d",
				len(c.Uniforms), c.ShaderID, shader.uniformDwordCount)
		}

		var srcs [graphics.ShaderSrcImageCount]*ui.Image
		var srcRegions [graphics.ShaderSrcImageCount]image.Rectangle
		for i, s := range c.Srcs {
			if s != graphicsdriver.InvalidImageID {
				if s == c.Dst {
					return fmt.Errorf("vmhost: DrawTriangles uses image %d as both its dst and a src", s)
				}
				src, ok := f.images[s]
				if !ok {
					return fmt.Errorf("vmhost: DrawTriangles references unknown src image %d", s)
				}
				srcs[i] = src.img
			}
			// The source region is encoded in the preserved-uniform prefix for every slot, including
			// source-less ones: a source-less draw still carries a region (e.g. DrawRectShader's
			// rectangle, which the shader reads via imageSrc0Size/Origin), so recover it regardless of
			// whether a source image is bound.
			region, ok := srcRegionFromUniforms(c.Uniforms, i)
			if !ok {
				return fmt.Errorf("vmhost: DrawTriangles lacks the preserved uniform prefix")
			}
			srcRegions[i] = region
		}

		// User uniforms follow the fixed preserved-uniform prefix; ui.Image re-derives the prefix.
		var uniforms []uint32
		if len(c.Uniforms) > graphics.PreservedUniformDwordCount {
			uniforms = c.Uniforms[graphics.PreservedUniformDwordCount:]
		}

		// The guest recorded one shared vertex buffer (GraphicsCommandKindSetVertices) and many draws that index into
		// ranges of it (the driver-layer form). ui.Image.DrawTriangles instead expects per-draw vertices,
		// appending whatever it is given to the host's command queue, so forwarding the whole shared buffer
		// per draw would make that queue grow quadratically. Forward only the vertices each draw references,
		// rebased to a zero origin.
		offset := c.IndexOffset
		if offset < 0 || offset > len(f.indices) {
			return fmt.Errorf("vmhost: DrawTriangles starts at index %d, outside the %d indices set", offset, len(f.indices))
		}
		vertexCount := len(f.vertices) / graphics.VertexFloatCount
		for _, dr := range c.DstRegions {
			// Comparing against what remains keeps offset+IndexCount from overflowing.
			if dr.IndexCount < 0 || dr.IndexCount > len(f.indices)-offset {
				return fmt.Errorf("vmhost: DrawTriangles takes %d indices from index %d, outside the %d indices set",
					dr.IndexCount, offset, len(f.indices))
			}
			idx := f.indices[offset : offset+dr.IndexCount]
			offset += dr.IndexCount
			if len(idx) == 0 {
				continue
			}

			lo, hi := idx[0], idx[0]
			for _, i := range idx[1:] {
				lo = min(lo, i)
				hi = max(hi, i)
			}
			// The comparison is 64-bit because an index is a uint32, which an int cannot hold on a 32-bit
			// host.
			if uint64(hi) >= uint64(vertexCount) {
				return fmt.Errorf("vmhost: DrawTriangles references vertex %d, outside the %d vertices set", hi, vertexCount)
			}

			f.vtxBuf = append(f.vtxBuf[:0], f.vertices[int(lo)*graphics.VertexFloatCount:(int(hi)+1)*graphics.VertexFloatCount]...)

			f.idxBuf = f.idxBuf[:0]
			for _, i := range idx {
				f.idxBuf = append(f.idxBuf, i-lo)
			}

			dst.img.DrawTriangles(srcs, f.vtxBuf, f.idxBuf, c.Blend, dr.Region, srcRegions, shader.shader, uniforms, true)
		}
		return nil

	case vmprotocol.GraphicsCommandKindWritePixels:
		hi, ok := f.images[c.ImageID]
		if !ok {
			return fmt.Errorf("vmhost: WritePixels references unknown image %d", c.ImageID)
		}
		if len(c.Pixels) != len(c.Regions) {
			return fmt.Errorf("vmhost: WritePixels carries %d pixel buffers for %d regions", len(c.Pixels), len(c.Regions))
		}
		// The whole batch is checked before any of it is written, so a rejected command leaves the mirror
		// untouched.
		bounds := image.Rect(0, 0, hi.width, hi.height)
		for i, r := range c.Regions {
			if r.Empty() || !r.In(bounds) {
				return fmt.Errorf("vmhost: WritePixels region %v is not within image %d's bounds %v", r, c.ImageID, bounds)
			}
			if l := 4 * r.Dx() * r.Dy(); len(c.Pixels[i]) != l {
				return fmt.Errorf("vmhost: WritePixels carries %d pixel bytes for region %v, which takes %d",
					len(c.Pixels[i]), r, l)
			}
		}
		for i := range c.Regions {
			hi.img.WritePixels(c.Pixels[i], c.Regions[i])
		}
		return nil

	case vmprotocol.GraphicsCommandKindDisposeImage:
		if hi, ok := f.images[c.ImageID]; ok {
			hi.img.Deallocate()
			delete(f.images, c.ImageID)
			if f.screen == hi {
				f.screen = nil
			}
		}
		return nil

	case vmprotocol.GraphicsCommandKindDisposeShader:
		if s, ok := f.shaders[c.ShaderID]; ok {
			s.shader.Deallocate()
			delete(f.shaders, c.ShaderID)
		}
		return nil

	default:
		return fmt.Errorf("vmhost: cannot render command %s", c.Kind)
	}
}

// srcRegionFromUniforms recovers source image i's sampling region (in source pixels) from the
// preserved-uniform prefix. It returns false if the prefix is absent.
func srcRegionFromUniforms(uniforms []uint32, i int) (image.Rectangle, bool) {
	if len(uniforms) < graphics.PreservedUniformDwordCount {
		return image.Rectangle{}, false
	}

	// Each origins/sizes entry is two float32s per source image. The values are exactly integral pixel
	// coordinates.
	ox := math.Float32frombits(uniforms[graphics.SourceImageRegionOriginUniformDwordIndex+2*i])
	oy := math.Float32frombits(uniforms[graphics.SourceImageRegionOriginUniformDwordIndex+2*i+1])
	sw := math.Float32frombits(uniforms[graphics.SourceImageRegionSizeUniformDwordIndex+2*i])
	sh := math.Float32frombits(uniforms[graphics.SourceImageRegionSizeUniformDwordIndex+2*i+1])
	return image.Rect(int(ox), int(oy), int(ox+sw), int(oy+sh)), true
}
