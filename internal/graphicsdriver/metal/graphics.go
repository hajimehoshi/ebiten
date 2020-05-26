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

// +build darwin

package metal

import (
	"fmt"
	"strings"
	"unsafe"

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/graphicsdriver/metal/ca"
	"github.com/hajimehoshi/ebiten/internal/graphicsdriver/metal/mtl"
	"github.com/hajimehoshi/ebiten/internal/shaderir"
	"github.com/hajimehoshi/ebiten/internal/thread"
)

// #cgo CFLAGS: -x objective-c
// #cgo !ios CFLAGS: -mmacosx-version-min=10.11
// #cgo LDFLAGS: -framework Foundation
//
// #import <Foundation/Foundation.h>
//
// static void* allocAutoreleasePool() {
//   return [[NSAutoreleasePool alloc] init];
// }
//
// static void releaseAutoreleasePool(void* pool) {
//   [(NSAutoreleasePool*)pool release];
// }
import "C"

const source = `#include <metal_stdlib>

#define FILTER_NEAREST {{.FilterNearest}}
#define FILTER_LINEAR {{.FilterLinear}}
#define FILTER_SCREEN {{.FilterScreen}}

#define ADDRESS_CLAMP_TO_ZERO {{.AddressClampToZero}}
#define ADDRESS_REPEAT {{.AddressRepeat}}

using namespace metal;

struct VertexIn {
  packed_float2 position;
  packed_float2 tex;
  packed_float4 tex_region;
  packed_float4 color;
};

struct VertexOut {
  float4 position [[position]];
  float2 tex;
  float4 tex_region;
  float4 color;
};

vertex VertexOut VertexShader(
  uint vid [[vertex_id]],
  device VertexIn* vertices [[buffer(0)]],
  constant float2& viewport_size [[buffer(1)]]
) {
  float4x4 projectionMatrix = float4x4(
    float4(2.0 / viewport_size.x, 0, 0, 0),
    float4(0, 2.0 / viewport_size.y, 0, 0),
    float4(0, 0, 1, 0),
    float4(-1, -1, 0, 1)
  );

  VertexIn in = vertices[vid];
  VertexOut out = {
    .position = projectionMatrix * float4(in.position, 0, 1),
    .tex = in.tex,
    .tex_region = in.tex_region,
    .color = in.color,
  };

  return out;
}

// AdjustTexels adjust texels.
// See #669, #759
float2 AdjustTexel(float2 source_size, float2 p0, float2 p1) {
  const float2 texel_size = 1.0 / source_size;
  if (fract((p1.x-p0.x)*source_size.x) == 0.0) {
    p1.x -= texel_size.x / 512.0;
  }
  if (fract((p1.y-p0.y)*source_size.y) == 0.0) {
    p1.y -= texel_size.y / 512.0;
  }
  return p1;
}

float FloorMod(float x, float y) {
  if (x < 0.0) {
    return y - (-x - y * floor(-x/y));
  }
  return x - y * floor(x/y);
}

template<uint8_t address>
float2 AdjustTexelByAddress(float2 p, float4 tex_region);

template<>
inline float2 AdjustTexelByAddress<ADDRESS_CLAMP_TO_ZERO>(float2 p, float4 tex_region) {
  return p;
}

template<>
inline float2 AdjustTexelByAddress<ADDRESS_REPEAT>(float2 p, float4 tex_region) {
  float2 o = float2(tex_region[0], tex_region[1]);
  float2 size = float2(tex_region[2] - tex_region[0], tex_region[3] - tex_region[1]);
  return float2(FloorMod((p.x - o.x), size.x) + o.x, FloorMod((p.y - o.y), size.y) + o.y);
}

template<uint8_t filter, uint8_t address>
struct ColorFromTexel;

template<uint8_t address>
struct ColorFromTexel<FILTER_NEAREST, address> {
  inline float4 Do(VertexOut v, texture2d<float> texture, constant float2& source_size, float scale) {
    float2 p = AdjustTexelByAddress<address>(v.tex, v.tex_region);
    if (v.tex_region[0] <= p.x &&
        v.tex_region[1] <= p.y &&
        p.x < v.tex_region[2] &&
        p.y < v.tex_region[3]) {
      constexpr sampler texture_sampler(filter::nearest);
      return texture.sample(texture_sampler, p);
    }
    return 0.0;
  }
};

template<uint8_t address>
struct ColorFromTexel<FILTER_LINEAR, address> {
  inline float4 Do(VertexOut v, texture2d<float> texture, constant float2& source_size, float scale) {
    constexpr sampler texture_sampler(filter::nearest);
    const float2 texel_size = 1 / source_size;

    float2 p0 = v.tex - texel_size / 2.0;
    float2 p1 = v.tex + texel_size / 2.0;
    p1 = AdjustTexel(source_size, p0, p1);
    p0 = AdjustTexelByAddress<address>(p0, v.tex_region);
    p1 = AdjustTexelByAddress<address>(p1, v.tex_region);

    float4 c0 = texture.sample(texture_sampler, p0);
    float4 c1 = texture.sample(texture_sampler, float2(p1.x, p0.y));
    float4 c2 = texture.sample(texture_sampler, float2(p0.x, p1.y));
    float4 c3 = texture.sample(texture_sampler, p1);

    if (p0.x < v.tex_region[0]) {
      c0 = 0;
      c2 = 0;
    }
    if (p0.y < v.tex_region[1]) {
      c0 = 0;
      c1 = 0;
    }
    if (v.tex_region[2] <= p1.x) {
      c1 = 0;
      c3 = 0;
    }
    if (v.tex_region[3] <= p1.y) {
      c2 = 0;
      c3 = 0;
    }

    float2 rate = fract(p0 * source_size);
    return mix(mix(c0, c1, rate.x), mix(c2, c3, rate.x), rate.y);
  }
};

template<uint8_t address>
struct ColorFromTexel<FILTER_SCREEN, address> {
  inline float4 Do(VertexOut v, texture2d<float> texture, constant float2& source_size, float scale) {
    constexpr sampler texture_sampler(filter::nearest);
    const float2 texel_size = 1 / source_size;

    float2 p0 = v.tex - texel_size / 2.0 / scale;
    float2 p1 = v.tex + texel_size / 2.0 / scale;
    p1 = AdjustTexel(source_size, p0, p1);

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
      constant float& scale) {
    float4 c = ColorFromTexel<filter, address>().Do(v, texture, source_size, scale);
    if (useColorM) {
      c.rgb /= c.a + (1.0 - sign(c.a));
      c = (color_matrix_body * c) + color_matrix_translation;
      c *= v.color;
      c.rgb *= c.a;
    } else {
      float4 s = v.color;
      c *= float4(s.r, s.g, s.b, 1.0) * s.a;
    }
    c = min(c, c.a);
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
      constant float& scale) {
    return ColorFromTexel<FILTER_SCREEN, address>().Do(v, texture, source_size, scale);
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
      constant float& scale [[buffer(5)]]) { \
    return FragmentShaderImpl<useColorM, filter, address>().Do( \
        v, texture, source_size, color_matrix_body, color_matrix_translation, scale); \
  }

FragmentShaderFunc(0, FILTER_NEAREST, ADDRESS_CLAMP_TO_ZERO)
FragmentShaderFunc(0, FILTER_LINEAR, ADDRESS_CLAMP_TO_ZERO)
FragmentShaderFunc(0, FILTER_NEAREST, ADDRESS_REPEAT)
FragmentShaderFunc(0, FILTER_LINEAR, ADDRESS_REPEAT)
FragmentShaderFunc(1, FILTER_NEAREST, ADDRESS_CLAMP_TO_ZERO)
FragmentShaderFunc(1, FILTER_LINEAR, ADDRESS_CLAMP_TO_ZERO)
FragmentShaderFunc(1, FILTER_NEAREST, ADDRESS_REPEAT)
FragmentShaderFunc(1, FILTER_LINEAR, ADDRESS_REPEAT)

FragmentShaderFunc(0, FILTER_SCREEN, ADDRESS_CLAMP_TO_ZERO)

#undef FragmentShaderFuncName
`

type rpsKey struct {
	useColorM     bool
	filter        driver.Filter
	address       driver.Address
	compositeMode driver.CompositeMode
	screen        bool
}

type Graphics struct {
	view view

	screenRPS mtl.RenderPipelineState
	rpss      map[rpsKey]mtl.RenderPipelineState
	cq        mtl.CommandQueue
	cb        mtl.CommandBuffer

	screenDrawable ca.MetalDrawable

	vb mtl.Buffer
	ib mtl.Buffer

	images      map[driver.ImageID]*Image
	nextImageID driver.ImageID

	src *Image
	dst *Image

	transparent  bool
	maxImageSize int
	drawCalled   bool

	t *thread.Thread

	pool unsafe.Pointer
}

var theGraphics Graphics

func Get() *Graphics {
	return &theGraphics
}

func (g *Graphics) SetThread(thread *thread.Thread) {
	g.t = thread
}

func (g *Graphics) Begin() {
	g.t.Call(func() error {
		// NSAutoreleasePool is required to release drawable correctly (#847).
		// https://developer.apple.com/library/archive/documentation/3DDrawing/Conceptual/MTLBestPracticesGuide/Drawables.html
		g.pool = C.allocAutoreleasePool()
		return nil
	})
}

func (g *Graphics) End() {
	g.flush(false, true)
	g.t.Call(func() error {
		g.screenDrawable = ca.MetalDrawable{}
		C.releaseAutoreleasePool(g.pool)
		g.pool = nil
		return nil
	})
}

func (g *Graphics) SetWindow(window unsafe.Pointer) {
	g.t.Call(func() error {
		// Note that [NSApp mainWindow] returns nil when the window is borderless.
		// Then the window is needed to be given explicitly.
		g.view.setWindow(window)
		return nil
	})
}

func (g *Graphics) SetUIView(uiview uintptr) {
	// TODO: Should this be called on the main thread?
	g.view.setUIView(uiview)
}

func (g *Graphics) SetVertices(vertices []float32, indices []uint16) {
	g.t.Call(func() error {
		if g.vb != (mtl.Buffer{}) {
			g.vb.Release()
		}
		if g.ib != (mtl.Buffer{}) {
			g.ib.Release()
		}
		g.vb = g.view.getMTLDevice().MakeBufferWithBytes(unsafe.Pointer(&vertices[0]), unsafe.Sizeof(vertices[0])*uintptr(len(vertices)), resourceStorageMode)
		g.ib = g.view.getMTLDevice().MakeBufferWithBytes(unsafe.Pointer(&indices[0]), unsafe.Sizeof(indices[0])*uintptr(len(indices)), resourceStorageMode)
		return nil
	})
}

func (g *Graphics) flush(wait bool, present bool) {
	g.t.Call(func() error {
		if g.cb == (mtl.CommandBuffer{}) {
			return nil
		}

		if present && g.screenDrawable != (ca.MetalDrawable{}) {
			g.cb.PresentDrawable(g.screenDrawable)
		}
		g.cb.Commit()
		if wait {
			g.cb.WaitUntilCompleted()
		}

		g.cb = mtl.CommandBuffer{}

		return nil
	})
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

func (g *Graphics) genNextImageID() driver.ImageID {
	id := g.nextImageID
	g.nextImageID++
	return id
}

func (g *Graphics) NewImage(width, height int) (driver.Image, error) {
	g.checkSize(width, height)
	td := mtl.TextureDescriptor{
		PixelFormat: mtl.PixelFormatRGBA8UNorm,
		Width:       graphics.InternalImageSize(width),
		Height:      graphics.InternalImageSize(height),
		StorageMode: storageMode,
		Usage:       textureUsage,
	}
	var t mtl.Texture
	g.t.Call(func() error {
		t = g.view.getMTLDevice().MakeTexture(td)
		return nil
	})
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

func (g *Graphics) NewScreenFramebufferImage(width, height int) (driver.Image, error) {
	g.t.Call(func() error {
		g.view.setDrawableSize(width, height)
		return nil
	})
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
		g.images = map[driver.ImageID]*Image{}
	}
	if _, ok := g.images[img.id]; ok {
		panic(fmt.Sprintf("opengl: image ID %d was already registered", img.id))
	}
	g.images[img.id] = img
}

func (g *Graphics) removeImage(img *Image) {
	delete(g.images, img.id)
}

func (g *Graphics) SetTransparent(transparent bool) {
	g.transparent = transparent
}

func (g *Graphics) Reset() error {
	if err := g.t.Call(func() error {
		if g.cq != (mtl.CommandQueue{}) {
			g.cq.Release()
			g.cq = mtl.CommandQueue{}
		}

		// TODO: Release existing rpss
		if g.rpss == nil {
			g.rpss = map[rpsKey]mtl.RenderPipelineState{}
		}

		if err := g.view.reset(); err != nil {
			return err
		}
		if g.transparent {
			g.view.ml.SetOpaque(false)
		}

		replaces := map[string]string{
			"{{.FilterNearest}}":      fmt.Sprintf("%d", driver.FilterNearest),
			"{{.FilterLinear}}":       fmt.Sprintf("%d", driver.FilterLinear),
			"{{.FilterScreen}}":       fmt.Sprintf("%d", driver.FilterScreen),
			"{{.AddressClampToZero}}": fmt.Sprintf("%d", driver.AddressClampToZero),
			"{{.AddressRepeat}}":      fmt.Sprintf("%d", driver.AddressRepeat),
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
			fmt.Sprintf("FragmentShader_%d_%d_%d", 0, driver.FilterScreen, driver.AddressClampToZero))
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
		rps, err := g.view.getMTLDevice().MakeRenderPipelineState(rpld)
		if err != nil {
			return err
		}
		g.screenRPS = rps

		conv := func(c driver.Operation) mtl.BlendFactor {
			switch c {
			case driver.Zero:
				return mtl.BlendFactorZero
			case driver.One:
				return mtl.BlendFactorOne
			case driver.SrcAlpha:
				return mtl.BlendFactorSourceAlpha
			case driver.DstAlpha:
				return mtl.BlendFactorDestinationAlpha
			case driver.OneMinusSrcAlpha:
				return mtl.BlendFactorOneMinusSourceAlpha
			case driver.OneMinusDstAlpha:
				return mtl.BlendFactorOneMinusDestinationAlpha
			default:
				panic(fmt.Sprintf("metal: invalid operation: %d", c))
			}
		}

		for _, screen := range []bool{false, true} {
			for _, cm := range []bool{false, true} {
				for _, a := range []driver.Address{
					driver.AddressClampToZero,
					driver.AddressRepeat,
				} {
					for _, f := range []driver.Filter{
						driver.FilterNearest,
						driver.FilterLinear,
					} {
						for c := driver.CompositeModeSourceOver; c <= driver.CompositeModeMax; c++ {
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

							pix := mtl.PixelFormatRGBA8UNorm
							if screen {
								pix = g.view.colorPixelFormat()
							}
							rpld.ColorAttachments[0].PixelFormat = pix
							rpld.ColorAttachments[0].BlendingEnabled = true

							src, dst := c.Operations()
							rpld.ColorAttachments[0].DestinationAlphaBlendFactor = conv(dst)
							rpld.ColorAttachments[0].DestinationRGBBlendFactor = conv(dst)
							rpld.ColorAttachments[0].SourceAlphaBlendFactor = conv(src)
							rpld.ColorAttachments[0].SourceRGBBlendFactor = conv(src)
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
							}] = rps
						}
					}
				}
			}
		}

		g.cq = g.view.getMTLDevice().MakeCommandQueue()
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (g *Graphics) Draw(dstID, srcID driver.ImageID, indexLen int, indexOffset int, mode driver.CompositeMode, colorM *affine.ColorM, filter driver.Filter, address driver.Address) error {
	dst := g.images[dstID]
	src := g.images[srcID]

	g.drawCalled = true

	if err := g.t.Call(func() error {
		g.view.update()

		rpd := mtl.RenderPassDescriptor{}
		// Even though the destination pixels are not used, mtl.LoadActionDontCare might cause glitches
		// (#1019). Always using mtl.LoadActionLoad is safe.
		rpd.ColorAttachments[0].LoadAction = mtl.LoadActionLoad
		rpd.ColorAttachments[0].StoreAction = mtl.StoreActionStore

		var t mtl.Texture
		if dst.screen {
			if g.screenDrawable == (ca.MetalDrawable{}) {
				drawable := g.view.drawable()
				if drawable == (ca.MetalDrawable{}) {
					return nil
				}
				g.screenDrawable = drawable
			}
			t = g.screenDrawable.Texture()
		} else {
			t = dst.texture
		}
		rpd.ColorAttachments[0].Texture = t
		rpd.ColorAttachments[0].ClearColor = mtl.ClearColor{}

		if g.cb == (mtl.CommandBuffer{}) {
			g.cb = g.cq.MakeCommandBuffer()
		}
		rce := g.cb.MakeRenderCommandEncoder(rpd)

		if dst.screen && filter == driver.FilterScreen {
			rce.SetRenderPipelineState(g.screenRPS)
		} else {
			rce.SetRenderPipelineState(g.rpss[rpsKey{
				screen:        dst.screen,
				useColorM:     colorM != nil,
				filter:        filter,
				address:       address,
				compositeMode: mode,
			}])
		}
		// In Metal, the NDC's Y direction (upward) and the framebuffer's Y direction (downward) don't
		// match. Then, the Y direction must be inverted.
		w, h := dst.viewportSize()
		rce.SetViewport(mtl.Viewport{
			OriginX: 0,
			OriginY: float64(h),
			Width:   float64(w),
			Height:  -float64(h),
			ZNear:   -1,
			ZFar:    1,
		})
		rce.SetVertexBuffer(g.vb, 0, 0)

		viewportSize := [...]float32{float32(w), float32(h)}
		rce.SetVertexBytes(unsafe.Pointer(&viewportSize[0]), unsafe.Sizeof(viewportSize), 1)

		sourceSize := [...]float32{
			float32(graphics.InternalImageSize(src.width)),
			float32(graphics.InternalImageSize(src.height)),
		}
		rce.SetFragmentBytes(unsafe.Pointer(&sourceSize[0]), unsafe.Sizeof(sourceSize), 2)

		esBody, esTranslate := colorM.UnsafeElements()
		rce.SetFragmentBytes(unsafe.Pointer(&esBody[0]), unsafe.Sizeof(esBody[0])*uintptr(len(esBody)), 3)
		rce.SetFragmentBytes(unsafe.Pointer(&esTranslate[0]), unsafe.Sizeof(esTranslate[0])*uintptr(len(esTranslate)), 4)

		scale := float32(dst.width) / float32(src.width)
		rce.SetFragmentBytes(unsafe.Pointer(&scale), unsafe.Sizeof(scale), 5)

		if src != nil {
			rce.SetFragmentTexture(src.texture, 0)
		} else {
			rce.SetFragmentTexture(mtl.Texture{}, 0)
		}
		rce.DrawIndexedPrimitives(mtl.PrimitiveTypeTriangle, indexLen, mtl.IndexTypeUInt16, g.ib, indexOffset*2)
		rce.EndEncoding()

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (g *Graphics) SetVsyncEnabled(enabled bool) {
	g.view.setDisplaySyncEnabled(enabled)
}

func (g *Graphics) FramebufferYDirection() driver.YDirection {
	return driver.Downward
}

func (g *Graphics) NeedsRestoring() bool {
	return false
}

func (g *Graphics) IsGL() bool {
	return false
}

func (g *Graphics) HasHighPrecisionFloat() bool {
	return true
}

func (g *Graphics) MaxImageSize() int {
	m := 0
	g.t.Call(func() error {
		if g.maxImageSize == 0 {
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
		}
		m = g.maxImageSize
		return nil
	})
	return m
}

func (g *Graphics) NewShader(program *shaderir.Program) (driver.Shader, error) {
	panic("metal: NewShader is not implemented")
}

func (g *Graphics) DrawShader(dst driver.ImageID, shader driver.ShaderID, indexLen int, indexOffset int, mode driver.CompositeMode, uniforms []interface{}) error {
	panic("metal: DrawShader is not implemented")
}

type Image struct {
	id       driver.ImageID
	graphics *Graphics
	width    int
	height   int
	screen   bool
	texture  mtl.Texture
}

func (i *Image) ID() driver.ImageID {
	return i.id
}

// viewportSize must be called from the main thread.
func (i *Image) viewportSize() (int, int) {
	if i.screen {
		return i.width, i.height
	}
	return graphics.InternalImageSize(i.width), graphics.InternalImageSize(i.height)
}

func (i *Image) Dispose() {
	i.graphics.t.Call(func() error {
		if i.texture != (mtl.Texture{}) {
			i.texture.Release()
			i.texture = mtl.Texture{}
		}
		return nil
	})
	i.graphics.removeImage(i)
}

func (i *Image) IsInvalidated() bool {
	// TODO: Does Metal cause context lost?
	// https://developer.apple.com/documentation/metal/mtlresource/1515898-setpurgeablestate
	// https://developer.apple.com/documentation/metal/mtldevicenotificationhandler
	return false
}

func (i *Image) syncTexture() {
	i.graphics.t.Call(func() error {
		if i.graphics.cb != (mtl.CommandBuffer{}) {
			panic("metal: command buffer must be empty at syncTexture: flush is not called yet?")
		}

		cb := i.graphics.cq.MakeCommandBuffer()
		bce := cb.MakeBlitCommandEncoder()
		bce.SynchronizeTexture(i.texture, 0, 0)
		bce.EndEncoding()
		cb.Commit()
		cb.WaitUntilCompleted()
		return nil
	})
}

func (i *Image) Pixels() ([]byte, error) {
	i.graphics.flush(true, false)
	i.syncTexture()

	b := make([]byte, 4*i.width*i.height)
	i.graphics.t.Call(func() error {
		i.texture.GetBytes(&b[0], uintptr(4*i.width), mtl.Region{
			Size: mtl.Size{Width: i.width, Height: i.height, Depth: 1},
		}, 0)
		return nil
	})
	return b, nil
}

func (i *Image) ReplacePixels(args []*driver.ReplacePixelsArgs) {
	g := i.graphics
	if g.drawCalled {
		g.flush(true, false)
		g.drawCalled = false
	}

	g.t.Call(func() error {
		for _, a := range args {
			i.texture.ReplaceRegion(mtl.Region{
				Origin: mtl.Origin{X: a.X, Y: a.Y, Z: 0},
				Size:   mtl.Size{Width: a.Width, Height: a.Height, Depth: 1},
			}, 0, unsafe.Pointer(&a.Pixels[0]), 4*a.Width)
		}
		return nil
	})
}
