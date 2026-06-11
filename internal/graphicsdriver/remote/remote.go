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

// Package remote provides a graphicsdriver.Graphics implementation that forwards the command stream
// to a Host instead of rasterizing it.
//
// It is the graphics device of a virtualization guest: the guest runs unmodified against this driver,
// never touching a GPU. Recorded commands (vmprotocol.GraphicsCommand) are flushed to the Host, which
// renders them on real hardware, and ReadPixels round-trips to the Host synchronously.
package remote

import (
	"fmt"
	"image"
	"slices"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/color"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
	"github.com/hajimehoshi/ebiten/v2/internal/vmprotocol"
)

var (
	_ graphicsdriver.Graphics = (*Graphics)(nil)
	_ graphicsdriver.Image    = (*Image)(nil)
	_ graphicsdriver.Shader   = (*Shader)(nil)
)

// Host receives a guest's recorded graphics output.
type Host interface {
	// SendGraphicsCommands forwards a flushed batch of graphics commands for the host to render.
	// cmds and the data it references are reused after the call returns and must not be retained.
	SendGraphicsCommands(cmds []vmprotocol.GraphicsCommand) error

	// ReadPixels returns the requested regions of the image, reproduced from the commands already
	// forwarded.
	ReadPixels(id graphicsdriver.ImageID, regions []image.Rectangle) ([][]byte, error)

	// MaxImageSize reports the host graphics driver's maximum image size.
	MaxImageSize() (int, error)

	// ColorSpace reports the host graphics driver's color space.
	ColorSpace() (color.ColorSpace, error)
}

// Graphics records the command stream emitted by the guest and forwards it to a Host. It is not safe
// for concurrent use: the guest drives it from a single goroutine.
type Graphics struct {
	commands []vmprotocol.GraphicsCommand

	// The recorded commands' payload slices are subslices of these reusable buffers, so a command is
	// valid only until the next Flush truncates them.
	verticesBuf   []float32
	indicesBuf    []uint32
	uniformsBuf   []uint32
	dstRegionsBuf []graphicsdriver.DstRegion
	regionsBuf    []image.Rectangle
	pixelsBuf     []byte
	pixelsListBuf [][]byte

	nextImageID  graphicsdriver.ImageID
	nextShaderID graphicsdriver.ShaderID

	host Host

	// maxImageSize caches the host's maximum image size, queried from the host exactly once via
	// maxImageSizeOnce. It never changes for a session.
	maxImageSize     int
	maxImageSizeOnce sync.Once

	// colorSpace caches the host's color space, queried from the host exactly once via
	// colorSpaceOnce. It never changes for a session.
	colorSpace     color.ColorSpace
	colorSpaceOnce sync.Once
}

// NewGraphics returns a remote graphics driver.
func NewGraphics() *Graphics {
	return &Graphics{}
}

// SetHost installs the destination for recorded commands and read-back requests. It must be set
// before the guest is driven.
func (g *Graphics) SetHost(h Host) {
	g.host = h
}

func (g *Graphics) record(c vmprotocol.GraphicsCommand) {
	g.commands = append(g.commands, c)
}

// appendToBuf appends a copy of src to *buf and returns the appended portion, which stays valid until
// *buf is truncated.
func appendToBuf[E any](buf *[]E, src []E) []E {
	start := len(*buf)
	*buf = append(*buf, src...)
	return (*buf)[start:len(*buf):len(*buf)]
}

// Flush forwards the commands recorded since the last flush to the host and clears them. With no
// host attached, the recorded commands are discarded.
func (g *Graphics) Flush() error {
	var err error
	if g.host != nil && len(g.commands) > 0 {
		err = g.host.SendGraphicsCommands(g.commands)
	}
	// Zero the command elements so the data they reference is released, and truncate the payload
	// buffers, keeping every backing array for the next batch.
	g.commands = slices.Delete(g.commands, 0, len(g.commands))
	g.verticesBuf = g.verticesBuf[:0]
	g.indicesBuf = g.indicesBuf[:0]
	g.uniformsBuf = g.uniformsBuf[:0]
	g.dstRegionsBuf = g.dstRegionsBuf[:0]
	g.regionsBuf = g.regionsBuf[:0]
	g.pixelsBuf = g.pixelsBuf[:0]
	g.pixelsListBuf = slices.Delete(g.pixelsListBuf, 0, len(g.pixelsListBuf))
	return err
}

func (g *Graphics) Initialize() error {
	g.record(vmprotocol.GraphicsCommand{
		Kind: vmprotocol.GraphicsCommandKindInitialize,
	})
	return nil
}

// ColorSpace returns the host graphics driver's color space: the guest's pixels are rendered by the
// host's driver, so its color space is the one in effect. It panics if no host is attached.
func (g *Graphics) ColorSpace() color.ColorSpace {
	// A call happens only while the guest is driven, by which point a host is attached.
	if g.host == nil {
		panic("remote: ColorSpace was called before a host was attached")
	}
	g.colorSpaceOnce.Do(func() {
		c, err := g.host.ColorSpace()
		if err != nil {
			panic(fmt.Sprintf("remote: querying the host's color space failed: %v", err))
		}
		g.colorSpace = c
	})
	return g.colorSpace
}

func (g *Graphics) Begin() error {
	g.record(vmprotocol.GraphicsCommand{
		Kind: vmprotocol.GraphicsCommandKindBegin,
	})
	return nil
}

func (g *Graphics) End(present bool) error {
	g.record(vmprotocol.GraphicsCommand{
		Kind:    vmprotocol.GraphicsCommandKindEnd,
		Present: present,
	})
	return nil
}

func (g *Graphics) SetTransparent(transparent bool) {
	g.record(vmprotocol.GraphicsCommand{
		Kind:        vmprotocol.GraphicsCommandKindSetTransparent,
		Transparent: transparent,
	})
}

func (g *Graphics) SetVertices(vertices []float32, indices []uint32) error {
	g.record(vmprotocol.GraphicsCommand{
		Kind:     vmprotocol.GraphicsCommandKindSetVertices,
		Vertices: appendToBuf(&g.verticesBuf, vertices),
		Indices:  appendToBuf(&g.indicesBuf, indices),
	})
	return nil
}

func (g *Graphics) NewImage(width, height int) (graphicsdriver.Image, error) {
	g.nextImageID++
	id := g.nextImageID
	g.record(vmprotocol.GraphicsCommand{
		Kind:    vmprotocol.GraphicsCommandKindNewImage,
		ImageID: id,
		Width:   width,
		Height:  height,
	})
	return &Image{id: id, graphics: g, width: width, height: height}, nil
}

func (g *Graphics) NewScreenFramebufferImage(width, height int) (graphicsdriver.Image, error) {
	g.nextImageID++
	id := g.nextImageID
	g.record(vmprotocol.GraphicsCommand{
		Kind:    vmprotocol.GraphicsCommandKindNewScreenFramebufferImage,
		ImageID: id,
		Width:   width,
		Height:  height,
		Screen:  true,
	})
	return &Image{id: id, graphics: g, width: width, height: height, screen: true}, nil
}

func (g *Graphics) SetVsyncEnabled(enabled bool) {
	g.record(vmprotocol.GraphicsCommand{
		Kind:         vmprotocol.GraphicsCommandKindSetVsyncEnabled,
		VsyncEnabled: enabled,
	})
}

func (g *Graphics) NeedsClearingScreen() bool {
	// The host mirrors the guest's screen as a persistent image rather than a swapchain framebuffer that
	// is recycled every frame, so it must be cleared explicitly or each frame's draw accumulates.
	return true
}

// MaxImageSize returns the host graphics driver's maximum image size. It panics if no host is
// attached.
func (g *Graphics) MaxImageSize() int {
	// The first call happens during the guest's first frame, by which point a host is attached.
	if g.host == nil {
		panic("remote: MaxImageSize was called before a host was attached")
	}
	g.maxImageSizeOnce.Do(func() {
		n, err := g.host.MaxImageSize()
		if err != nil {
			panic(fmt.Sprintf("remote: querying the host's maximum image size failed: %v", err))
		}
		g.maxImageSize = n
	})
	return g.maxImageSize
}

func (g *Graphics) NewShader(program *shaderir.Program) (graphicsdriver.Shader, error) {
	g.nextShaderID++
	id := g.nextShaderID
	// The host cannot use the compiled program directly, so forward the Kage source it was built
	// from (retained on the program at compile time) for the host to recompile.
	g.record(vmprotocol.GraphicsCommand{
		Kind:         vmprotocol.GraphicsCommandKindNewShader,
		ShaderID:     id,
		ShaderSource: program.FragmentSource,
	})
	return &Shader{id: id, graphics: g}, nil
}

func (g *Graphics) DrawTriangles(dst graphicsdriver.ImageID, srcs [graphics.ShaderSrcImageCount]graphicsdriver.ImageID, shader graphicsdriver.ShaderID, dstRegions []graphicsdriver.DstRegion, indexOffset int, blend graphicsdriver.Blend, uniforms []uint32) error {
	g.record(vmprotocol.GraphicsCommand{
		Kind:        vmprotocol.GraphicsCommandKindDrawTriangles,
		Dst:         dst,
		Srcs:        srcs,
		ShaderID:    shader,
		DstRegions:  appendToBuf(&g.dstRegionsBuf, dstRegions),
		IndexOffset: indexOffset,
		Blend:       blend,
		Uniforms:    appendToBuf(&g.uniformsBuf, uniforms),
	})
	return nil
}

// Image is a render target that holds no pixels of its own.
type Image struct {
	id       graphicsdriver.ImageID
	graphics *Graphics
	width    int
	height   int
	screen   bool
}

func (i *Image) ID() graphicsdriver.ImageID {
	return i.id
}

func (i *Image) Dispose() {
	i.graphics.record(vmprotocol.GraphicsCommand{
		Kind:    vmprotocol.GraphicsCommandKindDisposeImage,
		ImageID: i.id,
	})
}

func (i *Image) ReadPixels(args []graphicsdriver.PixelsArgs) error {
	regions := make([]image.Rectangle, len(args))
	for j, a := range args {
		regions[j] = a.Region
	}

	g := i.graphics
	if g.host == nil {
		g.record(vmprotocol.GraphicsCommand{
			Kind:    vmprotocol.GraphicsCommandKindReadPixels,
			ImageID: i.id,
			Regions: regions,
		})
		// No host is attached, so the requested pixels stay as the caller allocated them (zeroed).
		return nil
	}

	// The command queue is flushed before a read-back, so the commands recorded so far reproduce this
	// image. Forward them, then ask the host to read the image back.
	if err := g.Flush(); err != nil {
		return err
	}
	pixels, err := g.host.ReadPixels(i.id, regions)
	if err != nil {
		return err
	}
	for j := range args {
		if j < len(pixels) {
			copy(args[j].Pixels, pixels[j])
		}
	}
	return nil
}

func (i *Image) WritePixels(args []graphicsdriver.PixelsArgs) error {
	g := i.graphics
	regionsStart := len(g.regionsBuf)
	pixelsStart := len(g.pixelsListBuf)
	for _, a := range args {
		g.regionsBuf = append(g.regionsBuf, a.Region)
		g.pixelsListBuf = append(g.pixelsListBuf, appendToBuf(&g.pixelsBuf, a.Pixels))
	}
	g.record(vmprotocol.GraphicsCommand{
		Kind:    vmprotocol.GraphicsCommandKindWritePixels,
		ImageID: i.id,
		Regions: g.regionsBuf[regionsStart:len(g.regionsBuf):len(g.regionsBuf)],
		Pixels:  g.pixelsListBuf[pixelsStart:len(g.pixelsListBuf):len(g.pixelsListBuf)],
	})
	return nil
}

// Shader is a shader handle with no program of its own.
type Shader struct {
	id       graphicsdriver.ShaderID
	graphics *Graphics
}

func (s *Shader) ID() graphicsdriver.ShaderID {
	return s.id
}

func (s *Shader) Dispose() {
	s.graphics.record(vmprotocol.GraphicsCommand{
		Kind:     vmprotocol.GraphicsCommandKindDisposeShader,
		ShaderID: s.id,
	})
}
