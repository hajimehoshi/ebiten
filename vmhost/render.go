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
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
	"github.com/hajimehoshi/ebiten/v2/internal/vmprotocol"
)

// frameRenderer replays a guest's recorded graphics commands onto the host GPU by re-issuing them at
// the internal ui.Image layer, through the host's ordinary ebiten rendering stack.
//
// The methods must be called within the host's frame.
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
	// one completed frame to the next (at the AdvanceFrame composite), never showing a partially drawn
	// state.
	screen *hostImage
}

type hostImage struct {
	img    *ui.Image
	width  int
	height int
}

type hostShader struct {
	shader *ui.Shader
	// texels reports whether the shader's unit is texels (so the preserved source-region values are
	// normalized) rather than pixels.
	texels bool
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

// render replays the given commands. Image and shader identities persist across calls.
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
		// Framing, vsync, and read-back are owned by the host's own ebiten loop.
		return nil

	case vmprotocol.GraphicsCommandKindNewImage, vmprotocol.GraphicsCommandKindNewScreenFramebufferImage:
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
		f.shaders[c.ShaderID] = &hostShader{
			shader: ui.NewShader(ir, ""),
			texels: ir.Unit == shaderir.Texels,
		}
		return nil

	case vmprotocol.GraphicsCommandKindSetVertices:
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

		var srcs [graphics.ShaderSrcImageCount]*ui.Image
		var srcRegions [graphics.ShaderSrcImageCount]image.Rectangle
		var src0TexW, src0TexH float32
		for i, s := range c.Srcs {
			var texW, texH float32
			if s != graphicsdriver.InvalidImageID {
				src, ok := f.images[s]
				if !ok {
					return fmt.Errorf("vmhost: DrawTriangles references unknown src image %d", s)
				}
				srcs[i] = src.img
				if shader.texels {
					// For a texel-unit shader the source region and the vertex source coordinates are
					// normalized by the source texture's internal size. The recorded texture-size uniform
					// is unreliable (it is left zero when the shader does not read it), so use the
					// mirrored image's own internal size, which matches the guest's.
					texW = float32(graphics.InternalImageSize(src.width))
					texH = float32(graphics.InternalImageSize(src.height))
				}
			}
			if i == 0 {
				src0TexW, src0TexH = texW, texH
			}
			// The source region is encoded in the preserved-uniform prefix for every slot, including
			// source-less ones: a source-less draw still carries a region (e.g. DrawRectShader's
			// rectangle, which the shader reads via imageSrc0Size/Origin), so recover it regardless of
			// whether a source image is bound.
			region, ok := srcRegionFromUniforms(c.Uniforms, i, texW, texH)
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
		for _, dr := range c.DstRegions {
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

			f.vtxBuf = append(f.vtxBuf[:0], f.vertices[int(lo)*graphics.VertexFloatCount:(int(hi)+1)*graphics.VertexFloatCount]...)
			// ui.Image takes source coordinates in pixels and normalizes them itself, but the guest recorded
			// them already normalized by the first source's texture size (the driver-layer form for a
			// texel-unit shader). Un-normalize them back to pixels.
			if src0TexW > 0 && src0TexH > 0 {
				for v := 0; v+3 < len(f.vtxBuf); v += graphics.VertexFloatCount {
					f.vtxBuf[v+2] *= src0TexW
					f.vtxBuf[v+3] *= src0TexH
				}
			}

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
		for i := range c.Regions {
			hi.img.WritePixels(c.Pixels[i], c.Regions[i])
		}
		return nil

	case vmprotocol.GraphicsCommandKindDisposeImage:
		if hi, ok := f.images[c.ImageID]; ok {
			hi.img.Deallocate()
			delete(f.images, c.ImageID)
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
// preserved-uniform prefix. For a texel-unit shader the stored origin/size are normalized by the source
// texture's internal size; pass that size in texW/texH to scale them back to pixels, or 0 to read them
// as pixels directly (pixel-unit shaders and source-less slots). It returns false if the prefix is
// absent.
func srcRegionFromUniforms(uniforms []uint32, i int, texW, texH float32) (image.Rectangle, bool) {
	if len(uniforms) < graphics.PreservedUniformDwordCount {
		return image.Rectangle{}, false
	}

	// Each origins/sizes entry is two float32s per source image.
	ox := math.Float32frombits(uniforms[graphics.SourceImageRegionOriginUniformDwordIndex+2*i])
	oy := math.Float32frombits(uniforms[graphics.SourceImageRegionOriginUniformDwordIndex+2*i+1])
	sw := math.Float32frombits(uniforms[graphics.SourceImageRegionSizeUniformDwordIndex+2*i])
	sh := math.Float32frombits(uniforms[graphics.SourceImageRegionSizeUniformDwordIndex+2*i+1])
	if texW > 0 && texH > 0 {
		ox *= texW
		sw *= texW
		oy *= texH
		sh *= texH
	}

	// The values are exactly integral: pixel-unit regions hold integer pixel coordinates, and
	// texel-unit values are integers divided by a power-of-two texture size, which the multiplication
	// above inverts exactly.
	return image.Rect(int(ox), int(oy), int(ox+sw), int(oy+sh)), true
}

// readPixels reads back the given regions of a guest image, identified by its recorded ID, into the
// caller-prepared buffers: one premultiplied-alpha RGBA buffer per region, each sized 4*Dx*Dy of its
// region. It must be called within the host's frame.
func (f *frameRenderer) readPixels(pixels [][]byte, id graphicsdriver.ImageID, regions []image.Rectangle) error {
	if len(pixels) != len(regions) {
		return fmt.Errorf("vmhost: the number of pixel buffers (%d) does not match the number of regions (%d)",
			len(pixels), len(regions))
	}
	hi, ok := f.images[id]
	if !ok {
		return fmt.Errorf("vmhost: ReadPixels references unknown image %d", id)
	}
	for i, r := range regions {
		hi.img.ReadPixels(pixels[i], r)
	}
	return nil
}
