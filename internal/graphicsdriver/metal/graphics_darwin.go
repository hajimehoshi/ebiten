// Copyright 2018 The Ebiten Authors
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

package metal

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/ca"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/mtl"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

const source = `#include <metal_stdlib>

#define FILTER_NEAREST {{.FilterNearest}}
#define FILTER_LINEAR {{.FilterLinear}}
#define FILTER_SCREEN {{.FilterScreen}}

#define ADDRESS_CLAMP_TO_ZERO {{.AddressClampToZero}}
#define ADDRESS_REPEAT {{.AddressRepeat}}
#define ADDRESS_UNSAFE {{.AddressUnsafe}}

using namespace metal;

struct VertexIn {
  float2 position;
  float2 tex;
  float4 color;
};

struct VertexOut {
  float4 position [[position]];
  float2 tex;
  float4 color;
};

vertex VertexOut VertexShader(
  uint vid [[vertex_id]],
  const device VertexIn* vertices [[buffer(0)]],
  constant float2& viewport_size [[buffer(1)]]
) {
  // In Metal, the NDC's Y direction (upward) and the framebuffer's Y direction (downward) don't
  // match. Then, the Y direction must be inverted.
  float4x4 projectionMatrix = float4x4(
    float4(2.0 / viewport_size.x, 0, 0, 0),
    float4(0, -2.0 / viewport_size.y, 0, 0),
    float4(0, 0, 1, 0),
    float4(-1, 1, 0, 1)
  );

  VertexIn in = vertices[vid];
  VertexOut out = {
    .position = projectionMatrix * float4(in.position, 0, 1),
    .tex = in.tex,
    // Fragment shader wants premultiplied alpha.
    .color = float4(in.color.rgb, 1) * in.color.a,
  };

  return out;
}

float EuclideanMod(float x, float y) {
  // Assume that y is always positive.
  return x - y * floor(x/y);
}

template<uint8_t address>
float2 AdjustTexelByAddress(float2 p, float4 source_region);

template<>
inline float2 AdjustTexelByAddress<ADDRESS_CLAMP_TO_ZERO>(float2 p, float4 source_region) {
  return p;
}

template<>
inline float2 AdjustTexelByAddress<ADDRESS_REPEAT>(float2 p, float4 source_region) {
  float2 o = float2(source_region[0], source_region[1]);
  float2 size = float2(source_region[2] - source_region[0], source_region[3] - source_region[1]);
  return float2(EuclideanMod((p.x - o.x), size.x) + o.x, EuclideanMod((p.y - o.y), size.y) + o.y);
}

template<uint8_t filter, uint8_t address>
struct ColorFromTexel;

constexpr sampler texture_sampler{filter::nearest};

template<>
struct ColorFromTexel<FILTER_NEAREST, ADDRESS_UNSAFE> {
  inline float4 Do(VertexOut v, texture2d<float> texture, constant float2& source_size, constant float4& source_region, float scale) {
    float2 p = v.tex;
    return texture.sample(texture_sampler, p);
  }
};

template<uint8_t address>
struct ColorFromTexel<FILTER_NEAREST, address> {
  inline float4 Do(VertexOut v, texture2d<float> texture, constant float2& source_size, constant float4& source_region, float scale) {
    float2 p = AdjustTexelByAddress<address>(v.tex, source_region);
    if (source_region[0] <= p.x &&
        source_region[1] <= p.y &&
        p.x < source_region[2] &&
        p.y < source_region[3]) {
      return texture.sample(texture_sampler, p);
    }
    return 0.0;
  }
};

template<>
struct ColorFromTexel<FILTER_LINEAR, ADDRESS_UNSAFE> {
  inline float4 Do(VertexOut v, texture2d<float> texture, constant float2& source_size, constant float4& source_region, float scale) {
    const float2 texel_size = 1 / source_size;

    // Shift 1/512 [texel] to avoid the tie-breaking issue.
    // As all the vertex positions are aligned to 1/16 [pixel], this shiting should work in most cases.
    float2 p0 = v.tex - texel_size / 2.0 + (texel_size / 512.0);
    float2 p1 = v.tex + texel_size / 2.0 + (texel_size / 512.0);

    float4 c0 = texture.sample(texture_sampler, p0);
    float4 c1 = texture.sample(texture_sampler, float2(p1.x, p0.y));
    float4 c2 = texture.sample(texture_sampler, float2(p0.x, p1.y));
    float4 c3 = texture.sample(texture_sampler, p1);

    float2 rate = fract(p0 * source_size);
    return mix(mix(c0, c1, rate.x), mix(c2, c3, rate.x), rate.y);
  }
};

template<uint8_t address>
struct ColorFromTexel<FILTER_LINEAR, address> {
  inline float4 Do(VertexOut v, texture2d<float> texture, constant float2& source_size, constant float4& source_region, float scale) {
    const float2 texel_size = 1 / source_size;

    // Shift 1/512 [texel] to avoid the tie-breaking issue.
    // As all the vertex positions are aligned to 1/16 [pixel], this shiting should work in most cases.
    float2 p0 = v.tex - texel_size / 2.0 + (texel_size / 512.0);
    float2 p1 = v.tex + texel_size / 2.0 + (texel_size / 512.0);
    p0 = AdjustTexelByAddress<address>(p0, source_region);
    p1 = AdjustTexelByAddress<address>(p1, source_region);

    float4 c0 = texture.sample(texture_sampler, p0);
    float4 c1 = texture.sample(texture_sampler, float2(p1.x, p0.y));
    float4 c2 = texture.sample(texture_sampler, float2(p0.x, p1.y));
    float4 c3 = texture.sample(texture_sampler, p1);

    if (p0.x < source_region[0]) {
      c0 = 0;
      c2 = 0;
    }
    if (p0.y < source_region[1]) {
      c0 = 0;
      c1 = 0;
    }
    if (source_region[2] <= p1.x) {
      c1 = 0;
      c3 = 0;
    }
    if (source_region[3] <= p1.y) {
      c2 = 0;
      c3 = 0;
    }

    float2 rate = fract(p0 * source_size);
    return mix(mix(c0, c1, rate.x), mix(c2, c3, rate.x), rate.y);
  }
};

template<uint8_t address>
struct ColorFromTexel<FILTER_SCREEN, address> {
  inline float4 Do(VertexOut v, texture2d<float> texture, constant float2& source_size, constant float4& source_region, float scale) {
    const float2 texel_size = 1 / source_size;

    float2 p0 = v.tex - texel_size / 2.0 / scale + (texel_size / 512.0);
    float2 p1 = v.tex + texel_size / 2.0 / scale + (texel_size / 512.0);

    float4 c0 = texture.sample(texture_sampler, p0);
    float4 c1 = texture.sample(texture_sampler, float2(p1.x, p0.y));
    float4 c2 = texture.sample(texture_sampler, float2(p0.x, p1.y));
    float4 c3 = texture.sample(texture_sampler, p1);

    float2 rate_center = float2(1.0, 1.0) - texel_size / 2.0 / scale;
    float2 rate = clamp(((fract(p0 * source_size) - rate_center) * scale) + rate_center, 0.0, 1.0);
    return mix(mix(c0, c1, rate.x), mix(c2, c3, rate.x), rate.y);
  }
};

template<bool useColorM, uint8_t filter, uint8_t address>
struct FragmentShaderImpl {
  inline float4 Do(
      VertexOut v,
      texture2d<float> texture,
      constant float2& source_size,
      constant float4x4& color_matrix_body,
      constant float4& color_matrix_translation,
      constant float4& source_region,
      constant float& scale) {
    float4 c = ColorFromTexel<filter, address>().Do(v, texture, source_size, source_region, scale);
    if (useColorM) {
      c.rgb /= c.a + (1.0 - sign(c.a));
      c = (color_matrix_body * c) + color_matrix_translation;
      c.rgb *= c.a;
      c *= v.color;
      c.rgb = min(c.rgb, c.a);
    } else {
      c *= v.color;
    }
    return c;
  }
};

template<bool useColorM, uint8_t address>
struct FragmentShaderImpl<useColorM, FILTER_SCREEN, address> {
  inline float4 Do(
      VertexOut v,
      texture2d<float> texture,
      constant float2& source_size,
      constant float4x4& color_matrix_body,
      constant float4& color_matrix_translation,
      constant float4& source_region,
      constant float& scale) {
    return ColorFromTexel<FILTER_SCREEN, address>().Do(v, texture, source_size, source_region, scale);
  }
};

// Define Foo and FooCp macros to force macro replacement.
// See "6.10.3.1 Argument substitution" in ISO/IEC 9899.

#define FragmentShaderFunc(useColorM, filter, address) \
  FragmentShaderFuncCp(useColorM, filter, address)

#define FragmentShaderFuncCp(useColorM, filter, address) \
  fragment float4 FragmentShader_##useColorM##_##filter##_##address( \
      VertexOut v [[stage_in]], \
      texture2d<float> texture [[texture(0)]], \
      constant float2& source_size [[buffer(2)]], \
      constant float4x4& color_matrix_body [[buffer(3)]], \
      constant float4& color_matrix_translation [[buffer(4)]], \
      constant float4& source_region [[buffer(5)]], \
      constant float& scale [[buffer(6)]]) { \
    return FragmentShaderImpl<useColorM, filter, address>().Do( \
        v, texture, source_size, color_matrix_body, color_matrix_translation, source_region, scale); \
  }

FragmentShaderFunc(0, FILTER_NEAREST, ADDRESS_CLAMP_TO_ZERO)
FragmentShaderFunc(0, FILTER_LINEAR, ADDRESS_CLAMP_TO_ZERO)
FragmentShaderFunc(0, FILTER_NEAREST, ADDRESS_REPEAT)
FragmentShaderFunc(0, FILTER_LINEAR, ADDRESS_REPEAT)
FragmentShaderFunc(0, FILTER_NEAREST, ADDRESS_UNSAFE)
FragmentShaderFunc(0, FILTER_LINEAR, ADDRESS_UNSAFE)
FragmentShaderFunc(1, FILTER_NEAREST, ADDRESS_CLAMP_TO_ZERO)
FragmentShaderFunc(1, FILTER_LINEAR, ADDRESS_CLAMP_TO_ZERO)
FragmentShaderFunc(1, FILTER_NEAREST, ADDRESS_REPEAT)
FragmentShaderFunc(1, FILTER_LINEAR, ADDRESS_REPEAT)
FragmentShaderFunc(1, FILTER_NEAREST, ADDRESS_UNSAFE)
FragmentShaderFunc(1, FILTER_LINEAR, ADDRESS_UNSAFE)

FragmentShaderFunc(0, FILTER_SCREEN, ADDRESS_UNSAFE)

#undef FragmentShaderFuncName
`

type rpsKey struct {
	useColorM     bool
	filter        graphicsdriver.Filter
	address       graphicsdriver.Address
	compositeMode graphicsdriver.CompositeMode
	stencilMode   stencilMode
	screen        bool
}

type Graphics struct {
	view view

	screenRPS mtl.RenderPipelineState
	rpss      map[rpsKey]mtl.RenderPipelineState
	cq        mtl.CommandQueue
	cb        mtl.CommandBuffer
	rce       mtl.RenderCommandEncoder
	dsss      map[stencilMode]mtl.DepthStencilState

	screenDrawable ca.MetalDrawable

	buffers       map[mtl.CommandBuffer][]mtl.Buffer
	unusedBuffers map[mtl.Buffer]struct{}

	lastDst         *Image
	lastStencilMode stencilMode

	vb mtl.Buffer
	ib mtl.Buffer

	images      map[graphicsdriver.ImageID]*Image
	nextImageID graphicsdriver.ImageID

	shaders      map[graphicsdriver.ShaderID]*Shader
	nextShaderID graphicsdriver.ShaderID

	transparent  bool
	maxImageSize int
	tmpTextures  []mtl.Texture

	pool cocoa.NSAutoreleasePool
}

type stencilMode int

const (
	prepareStencil stencilMode = iota
	drawWithStencil
	noStencil
)

var creatingSystemDefaultDeviceSucceeded bool

func init() {
	// mtl.CreateSystemDefaultDevice must be called on the main thread (#2147).
	_, ok := mtl.CreateSystemDefaultDevice()
	creatingSystemDefaultDeviceSucceeded = ok
}

// NewGraphics creates an implementation of graphicsdriver.Graphics for Metal.
// The returned graphics value is nil iff the error is not nil.
func NewGraphics() (graphicsdriver.Graphics, error) {
	// On old mac devices like iMac 2011, Metal is not supported (#779).
	// TODO: Is there a better way to check whether Metal is available or not?
	// It seems OK to call MTLCreateSystemDefaultDevice multiple times, so this should be fine.
	if !creatingSystemDefaultDeviceSucceeded {
		return nil, fmt.Errorf("metal: mtl.CreateSystemDefaultDevice failed")
	}

	return &Graphics{}, nil
}

func (g *Graphics) Begin() error {
	// NSAutoreleasePool is required to release drawable correctly (#847).
	// https://developer.apple.com/library/archive/documentation/3DDrawing/Conceptual/MTLBestPracticesGuide/Drawables.html
	g.pool = cocoa.NSAutoreleasePool_new()
	return nil
}

func (g *Graphics) End(present bool) error {
	g.flushIfNeeded(present)
	g.screenDrawable = ca.MetalDrawable{}
	g.pool.Release()
	g.pool.ID = 0
	return nil
}

func (g *Graphics) SetWindow(window uintptr) {
	// Note that [NSApp mainWindow] returns nil when the window is borderless.
	// Then the window is needed to be given explicitly.
	g.view.setWindow(window)
}

func (g *Graphics) SetUIView(uiview uintptr) {
	// TODO: Should this be called on the main thread?
	g.view.setUIView(uiview)
}

func pow2(x uintptr) uintptr {
	var p2 uintptr = 1
	for p2 < x {
		p2 *= 2
	}
	return p2
}

func (g *Graphics) gcBuffers() {
	for cb, bs := range g.buffers {
		// If the command buffer still lives, the buffer must not be updated.
		// TODO: Handle an error?
		if cb.Status() != mtl.CommandBufferStatusCompleted {
			continue
		}

		for _, b := range bs {
			if g.unusedBuffers == nil {
				g.unusedBuffers = map[mtl.Buffer]struct{}{}
			}
			g.unusedBuffers[b] = struct{}{}
		}
		delete(g.buffers, cb)
		cb.Release()
	}

	const maxUnusedBuffers = 10
	if len(g.unusedBuffers) > maxUnusedBuffers {
		bufs := make([]mtl.Buffer, 0, len(g.unusedBuffers))
		for b := range g.unusedBuffers {
			bufs = append(bufs, b)
		}
		sort.Slice(bufs, func(a, b int) bool {
			return bufs[a].Length() > bufs[b].Length()
		})
		for _, b := range bufs[maxUnusedBuffers:] {
			delete(g.unusedBuffers, b)
			b.Release()
		}
	}
}

func (g *Graphics) availableBuffer(length uintptr) mtl.Buffer {
	if g.cb == (mtl.CommandBuffer{}) {
		g.cb = g.cq.MakeCommandBuffer()
	}

	var newBuf mtl.Buffer
	for b := range g.unusedBuffers {
		if b.Length() >= length {
			newBuf = b
			delete(g.unusedBuffers, b)
			break
		}
	}

	if newBuf == (mtl.Buffer{}) {
		newBuf = g.view.getMTLDevice().MakeBufferWithLength(pow2(length), resourceStorageMode)
	}

	if g.buffers == nil {
		g.buffers = map[mtl.CommandBuffer][]mtl.Buffer{}
	}
	if _, ok := g.buffers[g.cb]; !ok {
		g.cb.Retain()
	}
	g.buffers[g.cb] = append(g.buffers[g.cb], newBuf)
	return newBuf
}

func (g *Graphics) SetVertices(vertices []float32, indices []uint16) error {
	vbSize := unsafe.Sizeof(vertices[0]) * uintptr(len(vertices))
	ibSize := unsafe.Sizeof(indices[0]) * uintptr(len(indices))

	g.vb = g.availableBuffer(vbSize)
	g.vb.CopyToContents(unsafe.Pointer(&vertices[0]), vbSize)

	g.ib = g.availableBuffer(ibSize)
	g.ib.CopyToContents(unsafe.Pointer(&indices[0]), ibSize)

	return nil
}

func (g *Graphics) flushIfNeeded(present bool) {
	if g.cb == (mtl.CommandBuffer{}) {
		return
	}
	g.flushRenderCommandEncoderIfNeeded()

	if !g.view.presentsWithTransaction() && present && g.screenDrawable != (ca.MetalDrawable{}) {
		g.cb.PresentDrawable(g.screenDrawable)
	}
	g.cb.Commit()
	if g.view.presentsWithTransaction() && present && g.screenDrawable != (ca.MetalDrawable{}) {
		g.cb.WaitUntilScheduled()
		g.screenDrawable.Present()
	}

	for _, t := range g.tmpTextures {
		t.Release()
	}
	g.tmpTextures = g.tmpTextures[:0]

	g.cb = mtl.CommandBuffer{}
}

func (g *Graphics) checkSize(width, height int) {
	if width < 1 {
		panic(fmt.Sprintf("metal: width (%d) must be equal or more than %d", width, 1))
	}
	if height < 1 {
		panic(fmt.Sprintf("metal: height (%d) must be equal or more than %d", height, 1))
	}
	m := g.MaxImageSize()
	if width > m {
		panic(fmt.Sprintf("metal: width (%d) must be less than or equal to %d", width, m))
	}
	if height > m {
		panic(fmt.Sprintf("metal: height (%d) must be less than or equal to %d", height, m))
	}
}

func (g *Graphics) genNextImageID() graphicsdriver.ImageID {
	g.nextImageID++
	return g.nextImageID
}

func (g *Graphics) genNextShaderID() graphicsdriver.ShaderID {
	g.nextShaderID++
	return g.nextShaderID
}

func (g *Graphics) NewImage(width, height int) (graphicsdriver.Image, error) {
	g.checkSize(width, height)
	td := mtl.TextureDescriptor{
		TextureType: mtl.TextureType2D,
		PixelFormat: mtl.PixelFormatRGBA8UNorm,
		Width:       graphics.InternalImageSize(width),
		Height:      graphics.InternalImageSize(height),
		StorageMode: storageMode,
		Usage:       mtl.TextureUsageShaderRead | mtl.TextureUsageRenderTarget,
	}
	t := g.view.getMTLDevice().MakeTexture(td)
	i := &Image{
		id:       g.genNextImageID(),
		graphics: g,
		width:    width,
		height:   height,
		texture:  t,
	}
	g.addImage(i)
	return i, nil
}

func (g *Graphics) NewScreenFramebufferImage(width, height int) (graphicsdriver.Image, error) {
	g.view.setDrawableSize(width, height)
	i := &Image{
		id:       g.genNextImageID(),
		graphics: g,
		width:    width,
		height:   height,
		screen:   true,
	}
	g.addImage(i)
	return i, nil
}

func (g *Graphics) addImage(img *Image) {
	if g.images == nil {
		g.images = map[graphicsdriver.ImageID]*Image{}
	}
	if _, ok := g.images[img.id]; ok {
		panic(fmt.Sprintf("metal: image ID %d was already registered", img.id))
	}
	g.images[img.id] = img
}

func (g *Graphics) removeImage(img *Image) {
	delete(g.images, img.id)
}

func (g *Graphics) SetTransparent(transparent bool) {
	g.transparent = transparent
}

func operationToBlendFactor(c graphicsdriver.Operation) mtl.BlendFactor {
	switch c {
	case graphicsdriver.Zero:
		return mtl.BlendFactorZero
	case graphicsdriver.One:
		return mtl.BlendFactorOne
	case graphicsdriver.SrcAlpha:
		return mtl.BlendFactorSourceAlpha
	case graphicsdriver.DstAlpha:
		return mtl.BlendFactorDestinationAlpha
	case graphicsdriver.OneMinusSrcAlpha:
		return mtl.BlendFactorOneMinusSourceAlpha
	case graphicsdriver.OneMinusDstAlpha:
		return mtl.BlendFactorOneMinusDestinationAlpha
	case graphicsdriver.DstColor:
		return mtl.BlendFactorDestinationColor
	default:
		panic(fmt.Sprintf("metal: invalid operation: %d", c))
	}
}

func (g *Graphics) Initialize() error {
	// Creating *State objects are expensive and reuse them whenever possible.
	// See https://developer.apple.com/library/archive/documentation/Miscellaneous/Conceptual/MetalProgrammingGuide/Cmd-Submiss/Cmd-Submiss.html

	// TODO: Release existing rpss
	if g.rpss == nil {
		g.rpss = map[rpsKey]mtl.RenderPipelineState{}
	}

	for _, dss := range g.dsss {
		dss.Release()
	}
	if g.dsss == nil {
		g.dsss = map[stencilMode]mtl.DepthStencilState{}
	}

	if err := g.view.initialize(); err != nil {
		return err
	}
	if g.transparent {
		g.view.ml.SetOpaque(false)
	}

	replaces := map[string]string{
		"{{.FilterNearest}}":      fmt.Sprintf("%d", graphicsdriver.FilterNearest),
		"{{.FilterLinear}}":       fmt.Sprintf("%d", graphicsdriver.FilterLinear),
		"{{.FilterScreen}}":       fmt.Sprintf("%d", graphicsdriver.FilterScreen),
		"{{.AddressClampToZero}}": fmt.Sprintf("%d", graphicsdriver.AddressClampToZero),
		"{{.AddressRepeat}}":      fmt.Sprintf("%d", graphicsdriver.AddressRepeat),
		"{{.AddressUnsafe}}":      fmt.Sprintf("%d", graphicsdriver.AddressUnsafe),
	}
	src := source
	for k, v := range replaces {
		src = strings.Replace(src, k, v, -1)
	}

	lib, err := g.view.getMTLDevice().MakeLibrary(src, mtl.CompileOptions{})
	if err != nil {
		return err
	}
	vs, err := lib.MakeFunction("VertexShader")
	if err != nil {
		return err
	}
	fs, err := lib.MakeFunction(
		fmt.Sprintf("FragmentShader_%d_%d_%d", 0, graphicsdriver.FilterScreen, graphicsdriver.AddressUnsafe))
	if err != nil {
		return err
	}
	rpld := mtl.RenderPipelineDescriptor{
		VertexFunction:   vs,
		FragmentFunction: fs,
	}
	rpld.ColorAttachments[0].PixelFormat = g.view.colorPixelFormat()
	rpld.ColorAttachments[0].BlendingEnabled = true
	rpld.ColorAttachments[0].DestinationAlphaBlendFactor = mtl.BlendFactorZero
	rpld.ColorAttachments[0].DestinationRGBBlendFactor = mtl.BlendFactorZero
	rpld.ColorAttachments[0].SourceAlphaBlendFactor = mtl.BlendFactorOne
	rpld.ColorAttachments[0].SourceRGBBlendFactor = mtl.BlendFactorOne
	rpld.ColorAttachments[0].WriteMask = mtl.ColorWriteMaskAll
	rps, err := g.view.getMTLDevice().MakeRenderPipelineState(rpld)
	if err != nil {
		return err
	}
	g.screenRPS = rps

	for _, screen := range []bool{false, true} {
		for _, cm := range []bool{false, true} {
			for _, a := range []graphicsdriver.Address{
				graphicsdriver.AddressClampToZero,
				graphicsdriver.AddressRepeat,
				graphicsdriver.AddressUnsafe,
			} {
				for _, f := range []graphicsdriver.Filter{
					graphicsdriver.FilterNearest,
					graphicsdriver.FilterLinear,
				} {
					for c := graphicsdriver.CompositeModeSourceOver; c <= graphicsdriver.CompositeModeMax; c++ {
						for _, stencil := range []stencilMode{
							prepareStencil,
							drawWithStencil,
							noStencil,
						} {
							cmi := 0
							if cm {
								cmi = 1
							}
							fs, err := lib.MakeFunction(fmt.Sprintf("FragmentShader_%d_%d_%d", cmi, f, a))
							if err != nil {
								return err
							}
							rpld := mtl.RenderPipelineDescriptor{
								VertexFunction:   vs,
								FragmentFunction: fs,
							}
							if stencil != noStencil {
								rpld.StencilAttachmentPixelFormat = mtl.PixelFormatStencil8
							}

							pix := mtl.PixelFormatRGBA8UNorm
							if screen {
								pix = g.view.colorPixelFormat()
							}
							rpld.ColorAttachments[0].PixelFormat = pix
							rpld.ColorAttachments[0].BlendingEnabled = true

							src, dst := c.Operations()
							rpld.ColorAttachments[0].DestinationAlphaBlendFactor = operationToBlendFactor(dst)
							rpld.ColorAttachments[0].DestinationRGBBlendFactor = operationToBlendFactor(dst)
							rpld.ColorAttachments[0].SourceAlphaBlendFactor = operationToBlendFactor(src)
							rpld.ColorAttachments[0].SourceRGBBlendFactor = operationToBlendFactor(src)
							if stencil == prepareStencil {
								rpld.ColorAttachments[0].WriteMask = mtl.ColorWriteMaskNone
							} else {
								rpld.ColorAttachments[0].WriteMask = mtl.ColorWriteMaskAll
							}
							rps, err := g.view.getMTLDevice().MakeRenderPipelineState(rpld)
							if err != nil {
								return err
							}
							g.rpss[rpsKey{
								screen:        screen,
								useColorM:     cm,
								filter:        f,
								address:       a,
								compositeMode: c,
								stencilMode:   stencil,
							}] = rps
						}
					}
				}
			}
		}
	}

	// The stencil reference value is always 0 (default).
	g.dsss[prepareStencil] = g.view.getMTLDevice().MakeDepthStencilState(mtl.DepthStencilDescriptor{
		BackFaceStencil: mtl.StencilDescriptor{
			StencilFailureOperation:   mtl.StencilOperationKeep,
			DepthFailureOperation:     mtl.StencilOperationKeep,
			DepthStencilPassOperation: mtl.StencilOperationInvert,
			StencilCompareFunction:    mtl.CompareFunctionAlways,
		},
		FrontFaceStencil: mtl.StencilDescriptor{
			StencilFailureOperation:   mtl.StencilOperationKeep,
			DepthFailureOperation:     mtl.StencilOperationKeep,
			DepthStencilPassOperation: mtl.StencilOperationInvert,
			StencilCompareFunction:    mtl.CompareFunctionAlways,
		},
	})
	g.dsss[drawWithStencil] = g.view.getMTLDevice().MakeDepthStencilState(mtl.DepthStencilDescriptor{
		BackFaceStencil: mtl.StencilDescriptor{
			StencilFailureOperation:   mtl.StencilOperationKeep,
			DepthFailureOperation:     mtl.StencilOperationKeep,
			DepthStencilPassOperation: mtl.StencilOperationKeep,
			StencilCompareFunction:    mtl.CompareFunctionNotEqual,
		},
		FrontFaceStencil: mtl.StencilDescriptor{
			StencilFailureOperation:   mtl.StencilOperationKeep,
			DepthFailureOperation:     mtl.StencilOperationKeep,
			DepthStencilPassOperation: mtl.StencilOperationKeep,
			StencilCompareFunction:    mtl.CompareFunctionNotEqual,
		},
	})
	g.dsss[noStencil] = g.view.getMTLDevice().MakeDepthStencilState(mtl.DepthStencilDescriptor{
		BackFaceStencil: mtl.StencilDescriptor{
			StencilFailureOperation:   mtl.StencilOperationKeep,
			DepthFailureOperation:     mtl.StencilOperationKeep,
			DepthStencilPassOperation: mtl.StencilOperationKeep,
			StencilCompareFunction:    mtl.CompareFunctionAlways,
		},
		FrontFaceStencil: mtl.StencilDescriptor{
			StencilFailureOperation:   mtl.StencilOperationKeep,
			DepthFailureOperation:     mtl.StencilOperationKeep,
			DepthStencilPassOperation: mtl.StencilOperationKeep,
			StencilCompareFunction:    mtl.CompareFunctionAlways,
		},
	})

	g.cq = g.view.getMTLDevice().MakeCommandQueue()
	return nil
}

func (g *Graphics) flushRenderCommandEncoderIfNeeded() {
	if g.rce == (mtl.RenderCommandEncoder{}) {
		return
	}
	g.rce.EndEncoding()
	g.rce = mtl.RenderCommandEncoder{}
	g.lastDst = nil
}

func (g *Graphics) draw(rps mtl.RenderPipelineState, dst *Image, dstRegion graphicsdriver.Region, srcs [graphics.ShaderImageCount]*Image, indexLen int, indexOffset int, uniforms [][]float32, stencilMode stencilMode) error {
	// When prepareing a stencil buffer, flush the current render command encoder
	// to make sure the stencil buffer is cleared when loading.
	// TODO: What about clearing the stencil buffer by vertices?
	if g.lastDst != dst || (g.lastStencilMode == noStencil) != (stencilMode == noStencil) || stencilMode == prepareStencil {
		g.flushRenderCommandEncoderIfNeeded()
	}
	g.lastDst = dst
	g.lastStencilMode = stencilMode

	if g.rce == (mtl.RenderCommandEncoder{}) {
		rpd := mtl.RenderPassDescriptor{}
		// Even though the destination pixels are not used, mtl.LoadActionDontCare might cause glitches
		// (#1019). Always using mtl.LoadActionLoad is safe.
		if dst.screen {
			rpd.ColorAttachments[0].LoadAction = mtl.LoadActionClear
		} else {
			rpd.ColorAttachments[0].LoadAction = mtl.LoadActionLoad
		}

		// The store action should always be 'store' even for the screen (#1700).
		rpd.ColorAttachments[0].StoreAction = mtl.StoreActionStore

		t := dst.mtlTexture()
		if t == (mtl.Texture{}) {
			return nil
		}
		rpd.ColorAttachments[0].Texture = t
		rpd.ColorAttachments[0].ClearColor = mtl.ClearColor{}

		if stencilMode == prepareStencil {
			dst.ensureStencil()
			rpd.StencilAttachment.LoadAction = mtl.LoadActionClear
			rpd.StencilAttachment.StoreAction = mtl.StoreActionDontCare
			rpd.StencilAttachment.Texture = dst.stencil
		}

		if g.cb == (mtl.CommandBuffer{}) {
			g.cb = g.cq.MakeCommandBuffer()
		}
		g.rce = g.cb.MakeRenderCommandEncoder(rpd)
	}

	g.rce.SetRenderPipelineState(rps)

	w, h := dst.internalSize()
	g.rce.SetViewport(mtl.Viewport{
		OriginX: 0,
		OriginY: 0,
		Width:   float64(w),
		Height:  float64(h),
		ZNear:   -1,
		ZFar:    1,
	})
	g.rce.SetScissorRect(mtl.ScissorRect{
		X:      int(dstRegion.X),
		Y:      int(dstRegion.Y),
		Width:  int(dstRegion.Width),
		Height: int(dstRegion.Height),
	})
	g.rce.SetVertexBuffer(g.vb, 0, 0)

	for i, u := range uniforms {
		g.rce.SetVertexBytes(unsafe.Pointer(&u[0]), unsafe.Sizeof(u[0])*uintptr(len(u)), i+1)
		g.rce.SetFragmentBytes(unsafe.Pointer(&u[0]), unsafe.Sizeof(u[0])*uintptr(len(u)), i+1)
	}

	for i, src := range srcs {
		if src != nil {
			g.rce.SetFragmentTexture(src.texture, i)
		} else {
			g.rce.SetFragmentTexture(mtl.Texture{}, i)
		}
	}

	g.rce.SetDepthStencilState(g.dsss[stencilMode])

	g.rce.DrawIndexedPrimitives(mtl.PrimitiveTypeTriangle, indexLen, mtl.IndexTypeUInt16, g.ib, indexOffset*2)

	return nil
}

func (g *Graphics) DrawTriangles(dstID graphicsdriver.ImageID, srcIDs [graphics.ShaderImageCount]graphicsdriver.ImageID, offsets [graphics.ShaderImageCount - 1][2]float32, shaderID graphicsdriver.ShaderID, indexLen int, indexOffset int, mode graphicsdriver.CompositeMode, colorM graphicsdriver.ColorM, filter graphicsdriver.Filter, address graphicsdriver.Address, dstRegion, srcRegion graphicsdriver.Region, uniforms [][]float32, evenOdd bool) error {
	dst := g.images[dstID]

	if dst.screen {
		g.view.update()
	}

	var srcs [graphics.ShaderImageCount]*Image
	for i, srcID := range srcIDs {
		srcs[i] = g.images[srcID]
	}

	rpss := map[stencilMode]mtl.RenderPipelineState{}
	var uniformVars [][]float32
	if shaderID == graphicsdriver.InvalidShaderID {
		if dst.screen && filter == graphicsdriver.FilterScreen {
			rpss[noStencil] = g.screenRPS
		} else {
			for _, stencil := range []stencilMode{
				prepareStencil,
				drawWithStencil,
				noStencil,
			} {
				rpss[stencil] = g.rpss[rpsKey{
					screen:        dst.screen,
					useColorM:     !colorM.IsIdentity(),
					filter:        filter,
					address:       address,
					compositeMode: mode,
					stencilMode:   stencil,
				}]
			}
		}

		w, h := dst.internalSize()
		sourceSize := []float32{0, 0}
		if filter != graphicsdriver.FilterNearest {
			w, h := srcs[0].internalSize()
			sourceSize[0] = float32(w)
			sourceSize[1] = float32(h)
		}
		var esBody [16]float32
		var esTranslate [4]float32
		colorM.Elements(&esBody, &esTranslate)
		scale := float32(0)
		if filter == graphicsdriver.FilterScreen {
			scale = float32(dst.width) / float32(srcs[0].width)
		}
		uniformVars = [][]float32{
			{float32(w), float32(h)},
			sourceSize,
			esBody[:],
			esTranslate[:],
			{
				srcRegion.X,
				srcRegion.Y,
				srcRegion.X + srcRegion.Width,
				srcRegion.Y + srcRegion.Height,
			},
			{scale},
		}
	} else {
		for _, stencil := range []stencilMode{
			prepareStencil,
			drawWithStencil,
			noStencil,
		} {
			var err error
			rpss[stencil], err = g.shaders[shaderID].RenderPipelineState(g.view.getMTLDevice(), mode, stencil)
			if err != nil {
				return err
			}
		}

		uniformVars = make([][]float32, graphics.PreservedUniformVariablesCount+len(uniforms))

		// Set the destination texture size.
		dw, dh := dst.internalSize()
		uniformVars[graphics.TextureDestinationSizeUniformVariableIndex] = []float32{float32(dw), float32(dh)}

		// Set the source texture sizes.
		usizes := make([]float32, 2*len(srcs))
		for i, src := range srcs {
			if src != nil {
				w, h := src.internalSize()
				usizes[2*i] = float32(w)
				usizes[2*i+1] = float32(h)
			}
		}
		uniformVars[graphics.TextureSourceSizesUniformVariableIndex] = usizes

		// Set the destination region's origin.
		udorigin := []float32{float32(dstRegion.X) / float32(dw), float32(dstRegion.Y) / float32(dh)}
		uniformVars[graphics.TextureDestinationRegionOriginUniformVariableIndex] = udorigin

		// Set the destination region's size.
		udsize := []float32{float32(dstRegion.Width) / float32(dw), float32(dstRegion.Height) / float32(dh)}
		uniformVars[graphics.TextureDestinationRegionSizeUniformVariableIndex] = udsize

		// Set the source offsets.
		uoffsets := make([]float32, 2*len(offsets))
		for i, offset := range offsets {
			uoffsets[2*i] = offset[0]
			uoffsets[2*i+1] = offset[1]
		}
		uniformVars[graphics.TextureSourceOffsetsUniformVariableIndex] = uoffsets

		// Set the source region's origin of texture0.
		usorigin := []float32{float32(srcRegion.X), float32(srcRegion.Y)}
		uniformVars[graphics.TextureSourceRegionOriginUniformVariableIndex] = usorigin

		// Set the source region's size of texture0.
		ussize := []float32{float32(srcRegion.Width), float32(srcRegion.Height)}
		uniformVars[graphics.TextureSourceRegionSizeUniformVariableIndex] = ussize

		uniformVars[graphics.ProjectionMatrixUniformVariableIndex] = []float32{
			2 / float32(dw), 0, 0, 0,
			0, -2 / float32(dh), 0, 0,
			0, 0, 1, 0,
			-1, 1, 0, 1,
		}

		// Set the additional uniform variables.
		for i, v := range uniforms {
			const offset = graphics.PreservedUniformVariablesCount
			t := g.shaders[shaderID].ir.Uniforms[offset+i]
			switch t.Main {
			case shaderir.Mat3:
				// float3x3 requires 16-byte alignment (#2036).
				v1 := make([]float32, 12)
				copy(v1[0:3], v[0:3])
				copy(v1[4:7], v[3:6])
				copy(v1[8:11], v[6:9])
				uniformVars[offset+i] = v1
			case shaderir.Array:
				switch t.Sub[0].Main {
				case shaderir.Mat3:
					v1 := make([]float32, t.Length*12)
					for j := 0; j < t.Length; j++ {
						offset0 := j * 9
						offset1 := j * 12
						copy(v1[offset1:offset1+3], v[offset0:offset0+3])
						copy(v1[offset1+4:offset1+7], v[offset0+3:offset0+6])
						copy(v1[offset1+8:offset1+11], v[offset0+6:offset0+9])
					}
					uniformVars[offset+i] = v1
				default:
					uniformVars[offset+i] = v
				}
			default:
				uniformVars[offset+i] = v
			}
		}
	}

	if evenOdd {
		if err := g.draw(rpss[prepareStencil], dst, dstRegion, srcs, indexLen, indexOffset, uniformVars, prepareStencil); err != nil {
			return err
		}
		if err := g.draw(rpss[drawWithStencil], dst, dstRegion, srcs, indexLen, indexOffset, uniformVars, drawWithStencil); err != nil {
			return err
		}
	} else {
		if err := g.draw(rpss[noStencil], dst, dstRegion, srcs, indexLen, indexOffset, uniformVars, noStencil); err != nil {
			return err
		}
	}

	return nil
}

func (g *Graphics) SetVsyncEnabled(enabled bool) {
	g.view.setDisplaySyncEnabled(enabled)
}

func (g *Graphics) SetFullscreen(fullscreen bool) {
	g.view.setFullscreen(fullscreen)
}

func (g *Graphics) FramebufferYDirection() graphicsdriver.YDirection {
	return graphicsdriver.Downward
}

func (g *Graphics) NeedsRestoring() bool {
	return false
}

func (g *Graphics) NeedsClearingScreen() bool {
	return false
}

func (g *Graphics) IsGL() bool {
	return false
}

func (g *Graphics) IsDirectX() bool {
	return false
}

func (g *Graphics) MaxImageSize() int {
	if g.maxImageSize != 0 {
		return g.maxImageSize
	}

	g.maxImageSize = 4096
	// https://developer.apple.com/metal/Metal-Feature-Set-Tables.pdf
	switch {
	case g.view.getMTLDevice().SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily5_v1):
		g.maxImageSize = 16384
	case g.view.getMTLDevice().SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily4_v1):
		g.maxImageSize = 16384
	case g.view.getMTLDevice().SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily3_v1):
		g.maxImageSize = 16384
	case g.view.getMTLDevice().SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily2_v2):
		g.maxImageSize = 8192
	case g.view.getMTLDevice().SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily2_v1):
		g.maxImageSize = 4096
	case g.view.getMTLDevice().SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily1_v2):
		g.maxImageSize = 8192
	case g.view.getMTLDevice().SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily1_v1):
		g.maxImageSize = 4096
	case g.view.getMTLDevice().SupportsFeatureSet(mtl.FeatureSet_tvOS_GPUFamily2_v1):
		g.maxImageSize = 16384
	case g.view.getMTLDevice().SupportsFeatureSet(mtl.FeatureSet_tvOS_GPUFamily1_v1):
		g.maxImageSize = 8192
	case g.view.getMTLDevice().SupportsFeatureSet(mtl.FeatureSet_macOS_GPUFamily1_v1):
		g.maxImageSize = 16384
	default:
		panic("metal: there is no supported feature set")
	}
	return g.maxImageSize
}

func (g *Graphics) NewShader(program *shaderir.Program) (graphicsdriver.Shader, error) {
	s, err := newShader(g.view.getMTLDevice(), g.genNextShaderID(), program)
	if err != nil {
		return nil, err
	}
	g.addShader(s)
	return s, nil
}

func (g *Graphics) addShader(shader *Shader) {
	if g.shaders == nil {
		g.shaders = map[graphicsdriver.ShaderID]*Shader{}
	}
	if _, ok := g.shaders[shader.id]; ok {
		panic(fmt.Sprintf("metal: shader ID %d was already registered", shader.id))
	}
	g.shaders[shader.id] = shader
}

func (g *Graphics) removeShader(shader *Shader) {
	delete(g.shaders, shader.id)
}

type Image struct {
	id       graphicsdriver.ImageID
	graphics *Graphics
	width    int
	height   int
	screen   bool
	texture  mtl.Texture
	stencil  mtl.Texture
}

func (i *Image) ID() graphicsdriver.ImageID {
	return i.id
}

func (i *Image) internalSize() (int, int) {
	if i.screen {
		return i.width, i.height
	}
	return graphics.InternalImageSize(i.width), graphics.InternalImageSize(i.height)
}

func (i *Image) Dispose() {
	if i.stencil != (mtl.Texture{}) {
		i.stencil.Release()
		i.stencil = mtl.Texture{}
	}
	if i.texture != (mtl.Texture{}) {
		i.texture.Release()
		i.texture = mtl.Texture{}
	}
	i.graphics.removeImage(i)
}

func (i *Image) IsInvalidated() bool {
	// TODO: Does Metal cause context lost?
	// https://developer.apple.com/documentation/metal/mtlresource/1515898-setpurgeablestate
	// https://developer.apple.com/documentation/metal/mtldevicenotificationhandler
	return false
}

func (i *Image) syncTexture() {
	i.graphics.flushRenderCommandEncoderIfNeeded()

	// Calling SynchronizeTexture is ignored on iOS (see mtl.m), but it looks like committing BlitCommandEncoder
	// is necessary (#1337).
	if i.graphics.cb != (mtl.CommandBuffer{}) {
		panic("metal: command buffer must be empty at syncTexture: flushIfNeeded is not called yet?")
	}

	cb := i.graphics.cq.MakeCommandBuffer()
	bce := cb.MakeBlitCommandEncoder()
	bce.SynchronizeTexture(i.texture, 0, 0)
	bce.EndEncoding()

	cb.Commit()
	// TODO: Are fences available here?
	cb.WaitUntilCompleted()
}

func (i *Image) ReadPixels(buf []byte) error {
	if got, want := len(buf), 4*i.width*i.height; got != want {
		return fmt.Errorf("metal: len(buf) must be %d but %d at ReadPixels", want, got)
	}

	i.graphics.flushIfNeeded(false)
	i.syncTexture()

	i.texture.GetBytes(&buf[0], uintptr(4*i.width), mtl.Region{
		Size: mtl.Size{Width: i.width, Height: i.height, Depth: 1},
	}, 0)
	return nil
}

func (i *Image) WritePixels(args []*graphicsdriver.WritePixelsArgs) error {
	g := i.graphics

	g.flushRenderCommandEncoderIfNeeded()

	// Calculate the smallest texture size to include all the values in args.
	minX := math.MaxInt32
	minY := math.MaxInt32
	maxX := 0
	maxY := 0
	for _, a := range args {
		if minX > a.X {
			minX = a.X
		}
		if maxX < a.X+a.Width {
			maxX = a.X + a.Width
		}
		if minY > a.Y {
			minY = a.Y
		}
		if maxY < a.Y+a.Height {
			maxY = a.Y + a.Height
		}
	}
	w := maxX - minX
	h := maxY - minY

	// Use a temporary texture to send pixels asynchrounsly, whichever the memory is shared (e.g., iOS) or
	// managed (e.g., macOS). A temporary texture is needed since ReplaceRegion tries to sync the pixel
	// data between CPU and GPU, and doing it on the existing texture is inefficient (#1418).
	// The texture cannot be reused until sending the pixels finishes, then create new ones for each call.
	td := mtl.TextureDescriptor{
		TextureType: mtl.TextureType2D,
		PixelFormat: mtl.PixelFormatRGBA8UNorm,
		Width:       w,
		Height:      h,
		StorageMode: storageMode,
		Usage:       mtl.TextureUsageShaderRead | mtl.TextureUsageRenderTarget,
	}
	t := g.view.getMTLDevice().MakeTexture(td)
	g.tmpTextures = append(g.tmpTextures, t)

	for _, a := range args {
		t.ReplaceRegion(mtl.Region{
			Origin: mtl.Origin{X: a.X - minX, Y: a.Y - minY, Z: 0},
			Size:   mtl.Size{Width: a.Width, Height: a.Height, Depth: 1},
		}, 0, unsafe.Pointer(&a.Pixels[0]), 4*a.Width)
	}

	if g.cb == (mtl.CommandBuffer{}) {
		g.cb = i.graphics.cq.MakeCommandBuffer()
	}
	bce := g.cb.MakeBlitCommandEncoder()
	for _, a := range args {
		so := mtl.Origin{X: a.X - minX, Y: a.Y - minY, Z: 0}
		ss := mtl.Size{Width: a.Width, Height: a.Height, Depth: 1}
		do := mtl.Origin{X: a.X, Y: a.Y, Z: 0}
		bce.CopyFromTexture(t, 0, 0, so, ss, i.texture, 0, 0, do)
	}
	bce.EndEncoding()

	return nil
}

func (i *Image) mtlTexture() mtl.Texture {
	if i.screen {
		g := i.graphics
		if g.screenDrawable == (ca.MetalDrawable{}) {
			drawable := g.view.nextDrawable()
			if drawable == (ca.MetalDrawable{}) {
				return mtl.Texture{}
			}
			g.screenDrawable = drawable
			// After nextDrawable, it is expected some command buffers are completed.
			g.gcBuffers()
		}
		return g.screenDrawable.Texture()
	}
	return i.texture
}

func (i *Image) ensureStencil() {
	if i.stencil != (mtl.Texture{}) {
		return
	}

	td := mtl.TextureDescriptor{
		TextureType: mtl.TextureType2D,
		PixelFormat: mtl.PixelFormatStencil8,
		Width:       graphics.InternalImageSize(i.width),
		Height:      graphics.InternalImageSize(i.height),
		StorageMode: mtl.StorageModePrivate,
		Usage:       mtl.TextureUsageRenderTarget,
	}
	i.stencil = i.graphics.view.getMTLDevice().MakeTexture(td)
}
