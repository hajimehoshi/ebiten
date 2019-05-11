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
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/internal/graphicsdriver/metal/ca"
	"github.com/hajimehoshi/ebiten/internal/graphicsdriver/metal/mtl"
	"github.com/hajimehoshi/ebiten/internal/graphicsdriver/metal/ns"
	"github.com/hajimehoshi/ebiten/internal/mainthread"
)

// #cgo CFLAGS: -x objective-c -mmacosx-version-min=10.11
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
    float4(0, -2.0 / viewport_size.y, 0, 0),
    float4(0, 0, 1, 0),
    float4(-1, 1, 0, 1)
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
	filter        graphics.Filter
	address       graphics.Address
	compositeMode graphics.CompositeMode
}

type Driver struct {
	window uintptr

	device    mtl.Device
	ml        ca.MetalLayer
	screenRPS mtl.RenderPipelineState
	rpss      map[rpsKey]mtl.RenderPipelineState
	cq        mtl.CommandQueue
	cb        mtl.CommandBuffer

	screenDrawable ca.MetalDrawable

	vb mtl.Buffer
	ib mtl.Buffer

	src *Image
	dst *Image

	maxImageSize int

	pool unsafe.Pointer
}

var theDriver Driver

func Get() *Driver {
	return &theDriver
}

func (d *Driver) Begin() {
	mainthread.Run(func() error {
		// NSAutoreleasePool is required to release drawable correctly (#847).
		// https://developer.apple.com/library/archive/documentation/3DDrawing/Conceptual/MTLBestPracticesGuide/Drawables.html
		d.pool = C.allocAutoreleasePool()
		return nil
	})
}

func (d *Driver) End() {
	d.flush(false, true)
	mainthread.Run(func() error {
		d.screenDrawable = ca.MetalDrawable{}
		C.releaseAutoreleasePool(d.pool)
		d.pool = nil
		return nil
	})
}

func (d *Driver) SetWindow(window uintptr) {
	mainthread.Run(func() error {
		// Note that [NSApp mainWindow] returns nil when the window is borderless.
		// Then the window is needed to be given.
		d.window = window
		return nil
	})
}

func (d *Driver) SetVertices(vertices []float32, indices []uint16) {
	mainthread.Run(func() error {
		if d.vb != (mtl.Buffer{}) {
			d.vb.Release()
		}
		if d.ib != (mtl.Buffer{}) {
			d.ib.Release()
		}
		d.vb = d.device.MakeBufferWithBytes(unsafe.Pointer(&vertices[0]), unsafe.Sizeof(vertices[0])*uintptr(len(vertices)), mtl.ResourceStorageModeManaged)
		d.ib = d.device.MakeBufferWithBytes(unsafe.Pointer(&indices[0]), unsafe.Sizeof(indices[0])*uintptr(len(indices)), mtl.ResourceStorageModeManaged)
		return nil
	})
}

func (d *Driver) Flush() {
	// On Metal, flushing command buffers only once is enough. Do not call flush.
}

func (d *Driver) flush(wait bool, present bool) {
	mainthread.Run(func() error {
		if d.cb == (mtl.CommandBuffer{}) {
			return nil
		}

		if present && d.screenDrawable != (ca.MetalDrawable{}) {
			d.cb.PresentDrawable(d.screenDrawable)
		}
		d.cb.Commit()
		if wait {
			d.cb.WaitUntilCompleted()
		}

		d.cb = mtl.CommandBuffer{}

		return nil
	})
}

func (d *Driver) checkSize(width, height int) {
	m := 0
	mainthread.Run(func() error {
		if d.maxImageSize == 0 {
			d.maxImageSize = 4096
			// https://developer.apple.com/metal/Metal-Feature-Set-Tables.pdf
			switch {
			case d.device.SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily5_v1):
				d.maxImageSize = 16384
			case d.device.SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily4_v1):
				d.maxImageSize = 16384
			case d.device.SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily3_v1):
				d.maxImageSize = 16384
			case d.device.SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily2_v2):
				d.maxImageSize = 8192
			case d.device.SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily2_v1):
				d.maxImageSize = 4096
			case d.device.SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily1_v2):
				d.maxImageSize = 8192
			case d.device.SupportsFeatureSet(mtl.FeatureSet_iOS_GPUFamily1_v1):
				d.maxImageSize = 4096
			case d.device.SupportsFeatureSet(mtl.FeatureSet_tvOS_GPUFamily2_v1):
				d.maxImageSize = 16384
			case d.device.SupportsFeatureSet(mtl.FeatureSet_tvOS_GPUFamily1_v1):
				d.maxImageSize = 8192
			case d.device.SupportsFeatureSet(mtl.FeatureSet_macOS_GPUFamily1_v1):
				d.maxImageSize = 16384
			default:
				panic("metal: there is no supported feature set")
			}
		}
		m = d.maxImageSize
		return nil
	})

	if width < 1 {
		panic(fmt.Sprintf("metal: width (%d) must be equal or more than %d", width, 1))
	}
	if height < 1 {
		panic(fmt.Sprintf("metal: height (%d) must be equal or more than %d", height, 1))
	}
	if width > m {
		panic(fmt.Sprintf("metal: width (%d) must be less than or equal to %d", width, m))
	}
	if height > m {
		panic(fmt.Sprintf("metal: height (%d) must be less than or equal to %d", height, m))
	}
}

func (d *Driver) NewImage(width, height int) (graphicsdriver.Image, error) {
	d.checkSize(width, height)
	td := mtl.TextureDescriptor{
		PixelFormat: mtl.PixelFormatRGBA8UNorm,
		Width:       graphics.InternalImageSize(width),
		Height:      graphics.InternalImageSize(height),
		StorageMode: mtl.StorageModeManaged,

		// MTLTextureUsageRenderTarget might cause a problematic render result. Not sure the reason.
		// Usage: mtl.TextureUsageShaderRead | mtl.TextureUsageRenderTarget
		Usage: mtl.TextureUsageShaderRead,
	}
	var t mtl.Texture
	mainthread.Run(func() error {
		t = d.device.MakeTexture(td)
		return nil
	})
	return &Image{
		driver:  d,
		width:   width,
		height:  height,
		texture: t,
	}, nil
}

func (d *Driver) NewScreenFramebufferImage(width, height int) (graphicsdriver.Image, error) {
	mainthread.Run(func() error {
		d.ml.SetDrawableSize(width, height)
		return nil
	})
	return &Image{
		driver: d,
		width:  width,
		height: height,
		screen: true,
	}, nil
}

func (d *Driver) Reset() error {
	if err := mainthread.Run(func() error {
		if d.cq != (mtl.CommandQueue{}) {
			d.cq.Release()
			d.cq = mtl.CommandQueue{}
		}

		// TODO: Release existing rpss
		if d.rpss == nil {
			d.rpss = map[rpsKey]mtl.RenderPipelineState{}
		}

		var err error
		d.device, err = mtl.CreateSystemDefaultDevice()
		if err != nil {
			return err
		}

		d.ml = ca.MakeMetalLayer()
		d.ml.SetDevice(d.device)
		// https://developer.apple.com/documentation/quartzcore/cametallayer/1478155-pixelformat
		//
		// The pixel format for a Metal layer must be MTLPixelFormatBGRA8Unorm,
		// MTLPixelFormatBGRA8Unorm_sRGB, MTLPixelFormatRGBA16Float, MTLPixelFormatBGRA10_XR, or
		// MTLPixelFormatBGRA10_XR_sRGB.
		d.ml.SetPixelFormat(mtl.PixelFormatBGRA8UNorm)
		d.ml.SetMaximumDrawableCount(3)

		replaces := map[string]string{
			"{{.FilterNearest}}":      fmt.Sprintf("%d", graphics.FilterNearest),
			"{{.FilterLinear}}":       fmt.Sprintf("%d", graphics.FilterLinear),
			"{{.FilterScreen}}":       fmt.Sprintf("%d", graphics.FilterScreen),
			"{{.AddressClampToZero}}": fmt.Sprintf("%d", graphics.AddressClampToZero),
			"{{.AddressRepeat}}":      fmt.Sprintf("%d", graphics.AddressRepeat),
		}
		src := source
		for k, v := range replaces {
			src = strings.Replace(src, k, v, -1)
		}

		lib, err := d.device.MakeLibrary(src, mtl.CompileOptions{})
		if err != nil {
			return err
		}
		vs, err := lib.MakeFunction("VertexShader")
		if err != nil {
			return err
		}
		fs, err := lib.MakeFunction(
			fmt.Sprintf("FragmentShader_%d_%d_%d", 0, graphics.FilterScreen, graphics.AddressClampToZero))
		if err != nil {
			return err
		}
		rpld := mtl.RenderPipelineDescriptor{
			VertexFunction:   vs,
			FragmentFunction: fs,
		}
		rpld.ColorAttachments[0].PixelFormat = d.ml.PixelFormat()
		rpld.ColorAttachments[0].BlendingEnabled = true
		rpld.ColorAttachments[0].DestinationAlphaBlendFactor = mtl.BlendFactorZero
		rpld.ColorAttachments[0].DestinationRGBBlendFactor = mtl.BlendFactorZero
		rpld.ColorAttachments[0].SourceAlphaBlendFactor = mtl.BlendFactorOne
		rpld.ColorAttachments[0].SourceRGBBlendFactor = mtl.BlendFactorOne
		rps, err := d.device.MakeRenderPipelineState(rpld)
		if err != nil {
			return err
		}
		d.screenRPS = rps

		conv := func(c graphics.Operation) mtl.BlendFactor {
			switch c {
			case graphics.Zero:
				return mtl.BlendFactorZero
			case graphics.One:
				return mtl.BlendFactorOne
			case graphics.SrcAlpha:
				return mtl.BlendFactorSourceAlpha
			case graphics.DstAlpha:
				return mtl.BlendFactorDestinationAlpha
			case graphics.OneMinusSrcAlpha:
				return mtl.BlendFactorOneMinusSourceAlpha
			case graphics.OneMinusDstAlpha:
				return mtl.BlendFactorOneMinusDestinationAlpha
			default:
				panic(fmt.Sprintf("metal: invalid operation: %d", c))
			}
		}

		for _, cm := range []bool{false, true} {
			for _, a := range []graphics.Address{
				graphics.AddressClampToZero,
				graphics.AddressRepeat,
			} {
				for _, f := range []graphics.Filter{
					graphics.FilterNearest,
					graphics.FilterLinear,
				} {
					for c := graphics.CompositeModeSourceOver; c <= graphics.CompositeModeMax; c++ {
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
						rpld.ColorAttachments[0].PixelFormat = mtl.PixelFormatRGBA8UNorm
						rpld.ColorAttachments[0].BlendingEnabled = true

						src, dst := c.Operations()
						rpld.ColorAttachments[0].DestinationAlphaBlendFactor = conv(dst)
						rpld.ColorAttachments[0].DestinationRGBBlendFactor = conv(dst)
						rpld.ColorAttachments[0].SourceAlphaBlendFactor = conv(src)
						rpld.ColorAttachments[0].SourceRGBBlendFactor = conv(src)
						rps, err := d.device.MakeRenderPipelineState(rpld)
						if err != nil {
							return err
						}
						d.rpss[rpsKey{
							useColorM:     cm,
							filter:        f,
							address:       a,
							compositeMode: c,
						}] = rps
					}
				}
			}
		}

		d.cq = d.device.MakeCommandQueue()
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (d *Driver) Draw(indexLen int, indexOffset int, mode graphics.CompositeMode, colorM *affine.ColorM, filter graphics.Filter, address graphics.Address) error {
	if err := mainthread.Run(func() error {
		// NSView can be changed anytime (probably). Set this everyframe.
		cocoaWindow := ns.NewWindow(unsafe.Pointer(d.window))
		cocoaWindow.ContentView().SetLayer(d.ml)
		cocoaWindow.ContentView().SetWantsLayer(true)

		rpd := mtl.RenderPassDescriptor{}
		if d.dst.screen {
			rpd.ColorAttachments[0].LoadAction = mtl.LoadActionDontCare
			rpd.ColorAttachments[0].StoreAction = mtl.StoreActionStore
		} else {
			rpd.ColorAttachments[0].LoadAction = mtl.LoadActionLoad
			rpd.ColorAttachments[0].StoreAction = mtl.StoreActionStore
		}
		var t mtl.Texture
		if d.dst.screen {
			if d.screenDrawable == (ca.MetalDrawable{}) {
				drawable, err := d.ml.NextDrawable()
				if err != nil {
					// Drawable is nil. This can happen at the initial state. Let's wait and see.
					return nil
				}
				d.screenDrawable = drawable
			}
			t = d.screenDrawable.Texture()
		} else {
			t = d.dst.texture
		}
		rpd.ColorAttachments[0].Texture = t
		rpd.ColorAttachments[0].ClearColor = mtl.ClearColor{}

		w, h := d.dst.viewportSize()

		if d.cb == (mtl.CommandBuffer{}) {
			d.cb = d.cq.MakeCommandBuffer()
		}
		rce := d.cb.MakeRenderCommandEncoder(rpd)

		if d.dst.screen && filter == graphics.FilterScreen {
			rce.SetRenderPipelineState(d.screenRPS)
		} else {
			rce.SetRenderPipelineState(d.rpss[rpsKey{
				useColorM:     colorM != nil,
				filter:        filter,
				address:       address,
				compositeMode: mode,
			}])
		}
		rce.SetViewport(mtl.Viewport{0, 0, float64(w), float64(h), -1, 1})
		rce.SetVertexBuffer(d.vb, 0, 0)

		viewportSize := [...]float32{float32(w), float32(h)}
		rce.SetVertexBytes(unsafe.Pointer(&viewportSize[0]), unsafe.Sizeof(viewportSize), 1)

		sourceSize := [...]float32{
			float32(graphics.InternalImageSize(d.src.width)),
			float32(graphics.InternalImageSize(d.src.height)),
		}
		rce.SetFragmentBytes(unsafe.Pointer(&sourceSize[0]), unsafe.Sizeof(sourceSize), 2)

		if colorM != nil {
			esBody, esTranslate := colorM.UnsafeElements()
			rce.SetFragmentBytes(unsafe.Pointer(&esBody[0]), unsafe.Sizeof(esBody[0])*uintptr(len(esBody)), 3)
			rce.SetFragmentBytes(unsafe.Pointer(&esTranslate[0]), unsafe.Sizeof(esTranslate[0])*uintptr(len(esTranslate)), 4)
		}

		scale := float32(d.dst.width) / float32(d.src.width)
		rce.SetFragmentBytes(unsafe.Pointer(&scale), unsafe.Sizeof(scale), 5)

		if d.src != nil {
			rce.SetFragmentTexture(d.src.texture, 0)
		} else {
			rce.SetFragmentTexture(mtl.Texture{}, 0)
		}
		rce.DrawIndexedPrimitives(mtl.PrimitiveTypeTriangle, indexLen, mtl.IndexTypeUInt16, d.ib, indexOffset*2)
		rce.EndEncoding()

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (d *Driver) ResetSource() {
	mainthread.Run(func() error {
		d.src = nil
		return nil
	})
}

func (d *Driver) SetVsyncEnabled(enabled bool) {
	// TODO: Now SetVsyncEnabled is called only from the main thread, and mainthread.Run is not available since
	// recursive function call via Run is forbidden.
	// Fix this to use mainthread.Run to avoid confusion.
	d.ml.SetDisplaySyncEnabled(enabled)
}

func (d *Driver) VDirection() graphicsdriver.VDirection {
	return graphicsdriver.VUpward
}

func (d *Driver) IsGL() bool {
	return false
}

type Image struct {
	driver  *Driver
	width   int
	height  int
	screen  bool
	texture mtl.Texture
}

// viewportSize must be called from the main thread.
func (i *Image) viewportSize() (int, int) {
	if i.screen {
		return i.width, i.height
	}
	return graphics.InternalImageSize(i.width), graphics.InternalImageSize(i.height)
}

func (i *Image) Dispose() {
	mainthread.Run(func() error {
		if i.texture != (mtl.Texture{}) {
			i.texture.Release()
			i.texture = mtl.Texture{}
		}
		return nil
	})
}

func (i *Image) IsInvalidated() bool {
	// TODO: Does Metal cause context lost?
	// https://developer.apple.com/documentation/metal/mtlresource/1515898-setpurgeablestate
	// https://developer.apple.com/documentation/metal/mtldevicenotificationhandler
	return false
}

func (i *Image) syncTexture() {
	mainthread.Run(func() error {
		if i.driver.cb != (mtl.CommandBuffer{}) {
			panic("metal: command buffer must be empty at syncTexture: flush is not called yet?")
		}

		cb := i.driver.cq.MakeCommandBuffer()
		bce := cb.MakeBlitCommandEncoder()
		bce.SynchronizeTexture(i.texture, 0, 0)
		bce.EndEncoding()
		cb.Commit()
		cb.WaitUntilCompleted()
		return nil
	})
}

func (i *Image) Pixels() ([]byte, error) {
	i.driver.flush(true, false)
	i.syncTexture()

	b := make([]byte, 4*i.width*i.height)
	mainthread.Run(func() error {
		i.texture.GetBytes(&b[0], uintptr(4*i.width), mtl.Region{
			Size: mtl.Size{i.width, i.height, 1},
		}, 0)
		return nil
	})
	return b, nil
}

func (i *Image) SetAsDestination() {
	mainthread.Run(func() error {
		i.driver.dst = i
		return nil
	})
}

func (i *Image) SetAsSource() {
	mainthread.Run(func() error {
		i.driver.src = i
		return nil
	})
}

func (i *Image) ReplacePixels(pixels []byte, x, y, width, height int) {
	i.driver.flush(true, false)

	mainthread.Run(func() error {
		i.texture.ReplaceRegion(mtl.Region{
			Origin: mtl.Origin{x, y, 0},
			Size:   mtl.Size{width, height, 1},
		}, 0, unsafe.Pointer(&pixels[0]), 4*width)
		return nil
	})
}
