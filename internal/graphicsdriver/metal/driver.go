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

const source = `#include <metal_stdlib>

#define FILTER_NEAREST ({{.FilterNearest}})
#define FILTER_LINEAR ({{.FilterLinear}})
#define FILTER_SCREEN ({{.FilterScreen}})

using namespace metal;

struct VertexIn {
  packed_float2 position;
  packed_float4 tex;
  packed_float4 color;
};

struct VertexOut {
  float4 position [[position]];
  float2 tex;
  float2 tex_min;
  float2 tex_max;
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
    .tex = float2(in.tex[0], in.tex[1]),
    .color = in.color,
  };

  if (in.tex[2] >= 0 && in.tex[3] >= 0) {
    out.tex_min = float2(min(in.tex[0], in.tex[2]), min(in.tex[1], in.tex[3]));
    out.tex_max = float2(max(in.tex[0], in.tex[2]), max(in.tex[1], in.tex[3]));
  } else {
    out.tex_min = float2(0);
    out.tex_max = float2(1);
  }

  return out;
}

fragment float4 FragmentShader(VertexOut v [[stage_in]],
                               texture2d<float> texture [[texture(0)]],
                               constant float4x4& color_matrix_body [[buffer(2)]],
                               constant float4& color_matrix_translation [[buffer(3)]],
                               constant uint8_t& filter [[buffer(4)]],
                               constant float& scale [[buffer(5)]]) {
  constexpr sampler texture_sampler(filter::nearest);
  float2 source_size = 1;
  while (source_size.x < texture.get_width()) {
    source_size.x *= 2;
  }
  while (source_size.y < texture.get_height()) {
    source_size.y *= 2;
  }
  const float2 texel_size = 1 / source_size;

  float4 c;

  switch (filter) {
  case FILTER_NEAREST: {
    c = texture.sample(texture_sampler, v.tex);
    if (v.tex.x < v.tex_min.x ||
        v.tex.y < v.tex_min.y ||
        (v.tex_max.x - texel_size.x / 512.0) <= v.tex.x ||
        (v.tex_max.y - texel_size.y / 512.0) <= v.tex.y) {
      c = 0;
    }
    break;
  }

  case FILTER_LINEAR: {
    float2 p0 = v.tex - texel_size / 2.0;
    float2 p1 = v.tex + texel_size / 2.0;

    float4 c0 = texture.sample(texture_sampler, p0);
    float4 c1 = texture.sample(texture_sampler, float2(p1.x, p0.y));
    float4 c2 = texture.sample(texture_sampler, float2(p0.x, p1.y));
    float4 c3 = texture.sample(texture_sampler, p1);

    if (p0.x < v.tex_min.x) {
      c0 = 0;
      c2 = 0;
    }
    if (p0.y < v.tex_min.y) {
      c0 = 0;
      c1 = 0;
    }
    if ((v.tex_max.x - texel_size.x / 512.0) <= p1.x) {
      c1 = 0;
      c3 = 0;
    }
    if ((v.tex_max.y - texel_size.y / 512.0) <= p1.y) {
      c2 = 0;
      c3 = 0;
    }

    float2 rate = fract(p0 * source_size);
    c = mix(mix(c0, c1, rate.x), mix(c2, c3, rate.x), rate.y);
    break;
  }

  case FILTER_SCREEN: {
    float2 p0 = v.tex - texel_size / 2.0 / scale;
    float2 p1 = v.tex + texel_size / 2.0 / scale;

    float4 c0 = texture.sample(texture_sampler, p0);
    float4 c1 = texture.sample(texture_sampler, float2(p1.x, p0.y));
    float4 c2 = texture.sample(texture_sampler, float2(p0.x, p1.y));
    float4 c3 = texture.sample(texture_sampler, p1);

    float2 rate_center = float2(1.0, 1.0) - texel_size / 2.0 / scale;
    float2 rate = clamp(((fract(p0 * source_size) - rate_center) * scale) + rate_center, 0.0, 1.0);
    c = mix(mix(c0, c1, rate.x), mix(c2, c3, rate.x), rate.y);
    break;
  }

  default:
    // Not reached.
    discard_fragment();
    return float4(0);
  }

  if (0 < c.a) {
    c.rgb /= c.a;
  }
  c = (color_matrix_body * c) + color_matrix_translation;
  c *= v.color;
  c = clamp(c, 0.0, 1.0);
  c.rgb *= c.a;
  return c;
}
`

type Driver struct {
	window uintptr

	device    mtl.Device
	ml        ca.MetalLayer
	screenRPS mtl.RenderPipelineState
	rpss      map[graphics.CompositeMode]mtl.RenderPipelineState
	cq        mtl.CommandQueue
	cb        mtl.CommandBuffer

	screenDrawable ca.MetalDrawable

	vb mtl.Buffer
	ib mtl.Buffer

	src *Image
	dst *Image

	maxImageSize int
}

var theDriver Driver

func Get() *Driver {
	return &theDriver
}

func (d *Driver) SetWindow(window uintptr) {
	// Note that [NSApp mainWindow] returns nil when the window is borderless.
	// Then the window is needed to be given.
	d.window = window
}

func (d *Driver) SetVertices(vertices []float32, indices []uint16) {
	mainthread.Run(func() error {
		if d.vb != (mtl.Buffer{}) {
			d.vb.Release()
		}
		if d.ib != (mtl.Buffer{}) {
			d.ib.Release()
		}
		d.vb = d.device.MakeBuffer(unsafe.Pointer(&vertices[0]), unsafe.Sizeof(vertices[0])*uintptr(len(vertices)), mtl.ResourceStorageModeManaged)
		d.ib = d.device.MakeBuffer(unsafe.Pointer(&indices[0]), unsafe.Sizeof(indices[0])*uintptr(len(indices)), mtl.ResourceStorageModeManaged)
		return nil
	})
}

func (d *Driver) Flush() {
	mainthread.Run(func() error {
		if d.cb == (mtl.CommandBuffer{}) {
			return nil
		}

		if d.screenDrawable != (ca.MetalDrawable{}) {
			d.cb.PresentDrawable(d.screenDrawable)
		}
		d.cb.Commit()
		d.cb.WaitUntilCompleted()

		d.cb = mtl.CommandBuffer{}
		d.screenDrawable = ca.MetalDrawable{}

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
		panic(fmt.Sprintf("metal: width (%d) must be equal or more than 1", width))
	}
	if height < 1 {
		panic(fmt.Sprintf("metal: height (%d) must be equal or more than 1", height))
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
		Width:       graphics.NextPowerOf2Int(width),
		Height:      graphics.NextPowerOf2Int(height),
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
			d.rpss = map[graphics.CompositeMode]mtl.RenderPipelineState{}
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
		d.ml.SetDisplaySyncEnabled(true)

		replaces := map[string]string{
			"{{.FilterNearest}}": fmt.Sprintf("%d", graphics.FilterNearest),
			"{{.FilterLinear}}":  fmt.Sprintf("%d", graphics.FilterLinear),
			"{{.FilterScreen}}":  fmt.Sprintf("%d", graphics.FilterScreen),
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
		fs, err := lib.MakeFunction("FragmentShader")
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
				panic("not reached")
			}
		}

		for c := graphics.CompositeModeSourceOver; c <= graphics.CompositeModeMax; c++ {
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
			d.rpss[c] = rps
		}

		d.cq = d.device.MakeCommandQueue()
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (d *Driver) Draw(indexLen int, indexOffset int, mode graphics.CompositeMode, colorM *affine.ColorM, filter graphics.Filter) error {
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
					return err
				}
				d.screenDrawable = drawable
			}
			t = d.screenDrawable.Texture()
		} else {
			d.screenDrawable = ca.MetalDrawable{}
			t = d.dst.texture
		}
		rpd.ColorAttachments[0].Texture = t
		rpd.ColorAttachments[0].ClearColor = mtl.ClearColor{}

		w, h := d.dst.viewportSize()

		if d.cb == (mtl.CommandBuffer{}) {
			d.cb = d.cq.MakeCommandBuffer()
		}
		rce := d.cb.MakeRenderCommandEncoder(rpd)

		if d.dst.screen {
			rce.SetRenderPipelineState(d.screenRPS)
		} else {
			rce.SetRenderPipelineState(d.rpss[mode])
		}
		rce.SetViewport(mtl.Viewport{0, 0, float64(w), float64(h), -1, 1})
		rce.SetVertexBuffer(d.vb, 0, 0)

		viewportSize := [...]float32{float32(w), float32(h)}
		rce.SetVertexBytes(unsafe.Pointer(&viewportSize[0]), unsafe.Sizeof(viewportSize), 1)
		esBody, esTranslate := colorM.UnsafeElements()

		rce.SetFragmentBytes(unsafe.Pointer(&esBody[0]), unsafe.Sizeof(esBody[0])*uintptr(len(esBody)), 2)
		rce.SetFragmentBytes(unsafe.Pointer(&esTranslate[0]), unsafe.Sizeof(esTranslate[0])*uintptr(len(esTranslate)), 3)

		f := uint8(filter)
		rce.SetFragmentBytes(unsafe.Pointer(&f), 1, 4)

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
	d.src = nil
}

func (d *Driver) SetVsyncEnabled(enabled bool) {
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

func (i *Image) viewportSize() (int, int) {
	if i.screen {
		return i.width, i.height
	}
	return graphics.NextPowerOf2Int(i.width), graphics.NextPowerOf2Int(i.height)
}

func (i *Image) Dispose() {
	i.texture.Release()
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
			panic("not reached")
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
	i.driver.Flush()
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
	i.driver.dst = i
}

func (i *Image) SetAsSource() {
	i.driver.src = i
}

func (i *Image) ReplacePixels(pixels []byte, x, y, width, height int) {
	i.driver.Flush()

	mainthread.Run(func() error {
		i.texture.ReplaceRegion(mtl.Region{
			Origin: mtl.Origin{x, y, 0},
			Size:   mtl.Size{width, height, 1},
		}, 0, unsafe.Pointer(&pixels[0]), 4*width)
		return nil
	})
}
