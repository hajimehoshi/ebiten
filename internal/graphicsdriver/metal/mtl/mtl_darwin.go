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

// Package mtl provides access to Apple's Metal API (https://developer.apple.com/documentation/metal).
//
// Package mtl requires macOS version 10.12 or newer.
//
// This package is in very early stages of development.
// The API will change when opportunities for improvement are discovered; it is not yet frozen.
// Less than 20% of the Metal API surface is implemented.
// Current functionality is sufficient to render very basic geometry.
package mtl

import (
	"errors"
	"fmt"
	"reflect"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
)

// FeatureSet defines a specific platform, hardware, and software configuration.
//
// Reference: https://developer.apple.com/documentation/metal/mtlfeatureset.
type FeatureSet uint16

// The device feature sets that define specific platform, hardware, and software configurations.
const (
	MacOSGPUFamily1V1          FeatureSet = 10000 // The GPU family 1, version 1 feature set for macOS.
	MacOSGPUFamily1V2          FeatureSet = 10001 // The GPU family 1, version 2 feature set for macOS.
	MacOSReadWriteTextureTier2 FeatureSet = 10002 // The read-write texture, tier 2 feature set for macOS.
	MacOSGPUFamily1V3          FeatureSet = 10003 // The GPU family 1, version 3 feature set for macOS.
	MacOSGPUFamily1V4          FeatureSet = 10004 // The GPU family 1, version 4 feature set for macOS.
	MacOSGPUFamily2V1          FeatureSet = 10005 // The GPU family 2, version 1 feature set for macOS.
)

const (
	FeatureSet_iOS_GPUFamily1_v1           FeatureSet = 0
	FeatureSet_iOS_GPUFamily1_v2           FeatureSet = 2
	FeatureSet_iOS_GPUFamily1_v3           FeatureSet = 5
	FeatureSet_iOS_GPUFamily1_v4           FeatureSet = 8
	FeatureSet_iOS_GPUFamily1_v5           FeatureSet = 12
	FeatureSet_iOS_GPUFamily2_v1           FeatureSet = 1
	FeatureSet_iOS_GPUFamily2_v2           FeatureSet = 3
	FeatureSet_iOS_GPUFamily2_v3           FeatureSet = 6
	FeatureSet_iOS_GPUFamily2_v4           FeatureSet = 9
	FeatureSet_iOS_GPUFamily2_v5           FeatureSet = 13
	FeatureSet_iOS_GPUFamily3_v1           FeatureSet = 4
	FeatureSet_iOS_GPUFamily3_v2           FeatureSet = 7
	FeatureSet_iOS_GPUFamily3_v3           FeatureSet = 10
	FeatureSet_iOS_GPUFamily3_v4           FeatureSet = 14
	FeatureSet_iOS_GPUFamily4_v1           FeatureSet = 11
	FeatureSet_iOS_GPUFamily4_v2           FeatureSet = 15
	FeatureSet_iOS_GPUFamily5_v1           FeatureSet = 16
	FeatureSet_tvOS_GPUFamily1_v1          FeatureSet = 30000
	FeatureSet_tvOS_GPUFamily1_v2          FeatureSet = 30001
	FeatureSet_tvOS_GPUFamily1_v3          FeatureSet = 30002
	FeatureSet_tvOS_GPUFamily1_v4          FeatureSet = 30004
	FeatureSet_tvOS_GPUFamily2_v1          FeatureSet = 30003
	FeatureSet_tvOS_GPUFamily2_v2          FeatureSet = 30005
	FeatureSet_macOS_GPUFamily1_v1         FeatureSet = 10000
	FeatureSet_macOS_GPUFamily1_v2         FeatureSet = 10001
	FeatureSet_macOS_GPUFamily1_v3         FeatureSet = 10003
	FeatureSet_macOS_GPUFamily1_v4         FeatureSet = 10004
	FeatureSet_macOS_GPUFamily2_v1         FeatureSet = 10005
	FeatureSet_macOS_ReadWriteTextureTier2 FeatureSet = 10002
)

// TextureType defines The dimension of each image, including whether multiple images are arranged into an array or
// a cube.
//
// Reference: https://developer.apple.com/documentation/metal/mtltexturetype
type TextureType uint16

const (
	TextureType2D TextureType = 2
)

// PixelFormat defines data formats that describe the organization
// and characteristics of individual pixels in a texture.
//
// Reference: https://developer.apple.com/documentation/metal/mtlpixelformat.
type PixelFormat uint16

// The data formats that describe the organization and characteristics
// of individual pixels in a texture.
const (
	PixelFormatRGBA8UNorm     PixelFormat = 70  // Ordinary format with four 8-bit normalized unsigned integer components in RGBA order.
	PixelFormatRGBA8UNormSRGB PixelFormat = 71  // Ordinary format with four 8-bit normalized unsigned integer components in RGBA order with conversion between sRGB and linear space.
	PixelFormatBGRA8UNorm     PixelFormat = 80  // Ordinary format with four 8-bit normalized unsigned integer components in BGRA order.
	PixelFormatBGRA8UNormSRGB PixelFormat = 81  // Ordinary format with four 8-bit normalized unsigned integer components in BGRA order with conversion between sRGB and linear space.
	PixelFormatStencil8       PixelFormat = 253 // A pixel format with an 8-bit unsigned integer component, used for a stencil render target.
)

// PrimitiveType defines geometric primitive types for drawing commands.
//
// Reference: https://developer.apple.com/documentation/metal/mtlprimitivetype.
type PrimitiveType uint8

// Geometric primitive types for drawing commands.
const (
	PrimitiveTypePoint         PrimitiveType = 0
	PrimitiveTypeLine          PrimitiveType = 1
	PrimitiveTypeLineStrip     PrimitiveType = 2
	PrimitiveTypeTriangle      PrimitiveType = 3
	PrimitiveTypeTriangleStrip PrimitiveType = 4
)

// LoadAction defines actions performed at the start of a rendering pass
// for a render command encoder.
//
// Reference: https://developer.apple.com/documentation/metal/mtlloadaction.
type LoadAction uint8

// Actions performed at the start of a rendering pass for a render command encoder.
const (
	LoadActionDontCare LoadAction = 0
	LoadActionLoad     LoadAction = 1
	LoadActionClear    LoadAction = 2
)

// StoreAction defines actions performed at the end of a rendering pass
// for a render command encoder.
//
// Reference: https://developer.apple.com/documentation/metal/mtlstoreaction.
type StoreAction uint8

// Actions performed at the end of a rendering pass for a render command encoder.
const (
	StoreActionDontCare                   StoreAction = 0
	StoreActionStore                      StoreAction = 1
	StoreActionMultisampleResolve         StoreAction = 2
	StoreActionStoreAndMultisampleResolve StoreAction = 3
	StoreActionUnknown                    StoreAction = 4
	StoreActionCustomSampleDepthStore     StoreAction = 5
)

// StorageMode defines defines the memory location and access permissions of a resource.
//
// Reference: https://developer.apple.com/documentation/metal/mtlstoragemode.
type StorageMode uint8

const (
	// StorageModeShared indicates that the resource is stored in system memory
	// accessible to both the CPU and the GPU.
	StorageModeShared StorageMode = 0

	// StorageModeManaged indicates that the resource exists as a synchronized
	// memory pair with one copy stored in system memory accessible to the CPU
	// and another copy stored in video memory accessible to the GPU.
	StorageModeManaged StorageMode = 1

	// StorageModePrivate indicates that the resource is stored in memory
	// only accessible to the GPU. In iOS and tvOS, the resource is stored in
	// system memory. In macOS, the resource is stored in video memory.
	StorageModePrivate StorageMode = 2

	// StorageModeMemoryless indicates that the resource is stored in on-tile memory,
	// without CPU or GPU memory backing. The contents of the on-tile memory are undefined
	// and do not persist; the only way to populate the resource is to render into it.
	// Memoryless resources are limited to temporary render targets (i.e., Textures configured
	// with a TextureDescriptor and used with a RenderPassAttachmentDescriptor).
	StorageModeMemoryless StorageMode = 3
)

// ResourceOptions defines optional arguments used to create
// and influence behavior of buffer and texture objects.
//
// Reference: https://developer.apple.com/documentation/metal/mtlresourceoptions.
type ResourceOptions uint16

const (
	// ResourceCPUCacheModeDefaultCache is the default CPU cache mode for the resource.
	// Guarantees that read and write operations are executed in the expected order.
	ResourceCPUCacheModeDefaultCache ResourceOptions = ResourceOptions(CPUCacheModeDefaultCache) << resourceCPUCacheModeShift

	// ResourceCPUCacheModeWriteCombined is a write-combined CPU cache mode for the resource.
	// Optimized for resources that the CPU will write into, but never read.
	ResourceCPUCacheModeWriteCombined ResourceOptions = ResourceOptions(CPUCacheModeWriteCombined) << resourceCPUCacheModeShift

	// ResourceStorageModeShared indicates that the resource is stored in system memory
	// accessible to both the CPU and the GPU.
	ResourceStorageModeShared ResourceOptions = ResourceOptions(StorageModeShared) << resourceStorageModeShift

	// ResourceStorageModeManaged indicates that the resource exists as a synchronized
	// memory pair with one copy stored in system memory accessible to the CPU
	// and another copy stored in video memory accessible to the GPU.
	ResourceStorageModeManaged ResourceOptions = ResourceOptions(StorageModeManaged) << resourceStorageModeShift

	// ResourceStorageModePrivate indicates that the resource is stored in memory
	// only accessible to the GPU. In iOS and tvOS, the resource is stored
	// in system memory. In macOS, the resource is stored in video memory.
	ResourceStorageModePrivate ResourceOptions = ResourceOptions(StorageModePrivate) << resourceStorageModeShift

	// ResourceStorageModeMemoryless indicates that the resource is stored in on-tile memory,
	// without CPU or GPU memory backing. The contents of the on-tile memory are undefined
	// and do not persist; the only way to populate the resource is to render into it.
	// Memoryless resources are limited to temporary render targets (i.e., Textures configured
	// with a TextureDescriptor and used with a RenderPassAttachmentDescriptor).
	ResourceStorageModeMemoryless ResourceOptions = ResourceOptions(StorageModeMemoryless) << resourceStorageModeShift

	// ResourceHazardTrackingModeUntracked indicates that the command encoder dependencies
	// for this resource are tracked manually with Fence objects. This value is always set
	// for resources sub-allocated from a Heap object and may optionally be specified for
	// non-heap resources.
	ResourceHazardTrackingModeUntracked ResourceOptions = 1 << resourceHazardTrackingModeShift
)

const (
	resourceCPUCacheModeShift       = 0
	resourceStorageModeShift        = 4
	resourceHazardTrackingModeShift = 8
)

// CPUCacheMode is the CPU cache mode that defines the CPU mapping of a resource.
//
// Reference: https://developer.apple.com/documentation/metal/mtlcpucachemode.
type CPUCacheMode uint8

const (
	// CPUCacheModeDefaultCache is the default CPU cache mode for the resource.
	// Guarantees that read and write operations are executed in the expected order.
	CPUCacheModeDefaultCache CPUCacheMode = 0

	// CPUCacheModeWriteCombined is a write-combined CPU cache mode for the resource.
	// Optimized for resources that the CPU will write into, but never read.
	CPUCacheModeWriteCombined CPUCacheMode = 1
)

// IndexType is the index type for an index buffer that references vertices of geometric primitives.
//
// Reference: https://developer.apple.com/documentation/metal/mtlstoragemode
type IndexType uint8

const (
	// IndexTypeUInt16 is a 16-bit unsigned integer used as a primitive index.
	IndexTypeUInt16 IndexType = 0

	// IndexTypeUInt32 is a 32-bit unsigned integer used as a primitive index.
	IndexTypeUInt32 IndexType = 1
)

type TextureUsage uint8

const (
	TextureUsageUnknown         TextureUsage = 0x0000
	TextureUsageShaderRead      TextureUsage = 0x0001
	TextureUsageShaderWrite     TextureUsage = 0x0002
	TextureUsageRenderTarget    TextureUsage = 0x0004
	TextureUsagePixelFormatView TextureUsage = 0x0008
)

type BlendFactor uint8

const (
	BlendFactorZero                     BlendFactor = 0
	BlendFactorOne                      BlendFactor = 1
	BlendFactorSourceColor              BlendFactor = 2
	BlendFactorOneMinusSourceColor      BlendFactor = 3
	BlendFactorSourceAlpha              BlendFactor = 4
	BlendFactorOneMinusSourceAlpha      BlendFactor = 5
	BlendFactorDestinationColor         BlendFactor = 6
	BlendFactorOneMinusDestinationColor BlendFactor = 7
	BlendFactorDestinationAlpha         BlendFactor = 8
	BlendFactorOneMinusDestinationAlpha BlendFactor = 9
	BlendFactorSourceAlphaSaturated     BlendFactor = 10
	BlendFactorBlendColor               BlendFactor = 11
	BlendFactorOneMinusBlendColor       BlendFactor = 12
	BlendFactorBlendAlpha               BlendFactor = 13
	BlendFactorOneMinusBlendAlpha       BlendFactor = 14
	BlendFactorSource1Color             BlendFactor = 15
	BlendFactorOneMinusSource1Color     BlendFactor = 16
	BlendFactorSource1Alpha             BlendFactor = 17
	BlendFactorOneMinusSource1Alpha     BlendFactor = 18
)

type ColorWriteMask uint8

const (
	ColorWriteMaskNone  ColorWriteMask = 0
	ColorWriteMaskRed   ColorWriteMask = 0x1 << 3
	ColorWriteMaskGreen ColorWriteMask = 0x1 << 2
	ColorWriteMaskBlue  ColorWriteMask = 0x1 << 1
	ColorWriteMaskAlpha ColorWriteMask = 0x1 << 0
	ColorWriteMaskAll   ColorWriteMask = 0xf
)

type StencilOperation uint8

const (
	StencilOperationKeep           StencilOperation = 0
	StencilOperationZero           StencilOperation = 1
	StencilOperationReplace        StencilOperation = 2
	StencilOperationIncrementClamp StencilOperation = 3
	StencilOperationDecrementClamp StencilOperation = 4
	StencilOperationInvert         StencilOperation = 5
	StencilOperationIncrementWrap  StencilOperation = 6
	StencilOperationDecrementWrap  StencilOperation = 7
)

type CompareFunction uint8

const (
	CompareFunctionNever        CompareFunction = 0
	CompareFunctionLess         CompareFunction = 1
	CompareFunctionEqual        CompareFunction = 2
	CompareFunctionLessEqual    CompareFunction = 3
	CompareFunctionGreater      CompareFunction = 4
	CompareFunctionNotEqual     CompareFunction = 5
	CompareFunctionGreaterEqual CompareFunction = 6
	CompareFunctionAlways       CompareFunction = 7
)

type CommandBufferStatus uint8

const (
	CommandBufferStatusNotEnqueued CommandBufferStatus = 0 //The command buffer is not enqueued yet.
	CommandBufferStatusEnqueued    CommandBufferStatus = 1 // The command buffer is enqueued.
	CommandBufferStatusCommitted   CommandBufferStatus = 2 // The command buffer is committed for execution.
	CommandBufferStatusScheduled   CommandBufferStatus = 3 // The command buffer is scheduled.
	CommandBufferStatusCompleted   CommandBufferStatus = 4 // The command buffer completed execution successfully.
	CommandBufferStatusError       CommandBufferStatus = 5 // Execution of the command buffer was aborted due to an error during execution.
)

var (
	metal                         = purego.Dlopen("Metal.framework/Metal", purego.RTLD_GLOBAL)
	_MTLCreateSystemDefaultDevice = purego.Dlsym(metal, "MTLCreateSystemDefaultDevice")
)

// Resource represents a memory allocation for storing specialized data
// that is accessible to the GPU.
//
// Reference: https://developer.apple.com/documentation/metal/mtlresource.
type Resource interface {
	// resource returns the underlying id<MTLResource> pointer.
	resource() unsafe.Pointer
}

// RenderPipelineDescriptor configures new RenderPipelineState objects.
//
// Reference: https://developer.apple.com/documentation/metal/mtlrenderpipelinedescriptor.
type RenderPipelineDescriptor struct {
	// VertexFunction is a programmable function that processes individual vertices in a rendering pass.
	VertexFunction Function

	// FragmentFunction is a programmable function that processes individual fragments in a rendering pass.
	FragmentFunction Function

	// ColorAttachments is an array of attachments that store color data.
	ColorAttachments [1]RenderPipelineColorAttachmentDescriptor

	// StencilAttachmentPixelFormat is the pixel format of the attachment that stores stencil data.
	StencilAttachmentPixelFormat PixelFormat
}

// RenderPipelineColorAttachmentDescriptor describes a color render target that specifies
// the color configuration and color operations associated with a render pipeline.
//
// Reference: https://developer.apple.com/documentation/metal/mtlrenderpipelinecolorattachmentdescriptor.
type RenderPipelineColorAttachmentDescriptor struct {
	// PixelFormat is the pixel format of the color attachment's texture.
	PixelFormat PixelFormat

	BlendingEnabled bool

	DestinationAlphaBlendFactor BlendFactor
	DestinationRGBBlendFactor   BlendFactor
	SourceAlphaBlendFactor      BlendFactor
	SourceRGBBlendFactor        BlendFactor

	WriteMask ColorWriteMask
}

// RenderPassDescriptor describes a group of render targets that serve as
// the output destination for pixels generated by a render pass.
//
// Reference: https://developer.apple.com/documentation/metal/mtlrenderpassdescriptor.
type RenderPassDescriptor struct {
	// ColorAttachments is array of state information for attachments that store color data.
	ColorAttachments [1]RenderPassColorAttachmentDescriptor

	// StencilAttachment is state information for an attachment that stores stencil data.
	StencilAttachment RenderPassStencilAttachment
}

// RenderPassColorAttachmentDescriptor describes a color render target that serves
// as the output destination for color pixels generated by a render pass.
//
// Reference: https://developer.apple.com/documentation/metal/mtlrenderpasscolorattachmentdescriptor.
type RenderPassColorAttachmentDescriptor struct {
	RenderPassAttachmentDescriptor
	ClearColor ClearColor
}

// RenderPassStencilAttachment describes a stencil render target that serves as the output
// destination for stencil pixels generated by a render pass.
//
// Reference: https://developer.apple.com/documentation/metal/mtlrenderpassstencilattachmentdescriptor
type RenderPassStencilAttachment struct {
	RenderPassAttachmentDescriptor
}

// RenderPassAttachmentDescriptor describes a render target that serves
// as the output destination for pixels generated by a render pass.
//
// Reference: https://developer.apple.com/documentation/metal/mtlrenderpassattachmentdescriptor.
type RenderPassAttachmentDescriptor struct {
	LoadAction  LoadAction
	StoreAction StoreAction
	Texture     Texture
}

// ClearColor is an RGBA value used for a color pixel.
//
// Reference: https://developer.apple.com/documentation/metal/mtlclearcolor.
type ClearColor struct {
	Red, Green, Blue, Alpha float64
}

// TextureDescriptor configures new Texture objects.
//
// Reference: https://developer.apple.com/documentation/metal/mtltexturedescriptor.
type TextureDescriptor struct {
	TextureType TextureType
	PixelFormat PixelFormat
	Width       int
	Height      int
	StorageMode StorageMode
	Usage       TextureUsage
}

// Device is abstract representation of the GPU that
// serves as the primary interface for a Metal app.
//
// Reference: https://developer.apple.com/documentation/metal/mtldevice.
type Device struct {
	device objc.ID

	// Headless indicates whether a device is configured as headless.
	Headless bool

	// LowPower indicates whether a device is low-power.
	LowPower bool

	// Name is the name of the device.
	Name string
}

var (
	class_MTLRenderPipelineDescriptor = objc.GetClass("MTLRenderPipelineDescriptor")
	class_MTLTextureDescriptor        = objc.GetClass("MTLTextureDescriptor")
	class_MTLDepthStencilDescriptor   = objc.GetClass("MTLDepthStencilDescriptor")
	class_MTLRenderPassDescriptor     = objc.GetClass("MTLRenderPassDescriptor")
)

var (
	sel_class                                                                                                                         = objc.RegisterName("class")
	sel_length                                                                                                                        = objc.RegisterName("length")
	sel_isHeadless                                                                                                                    = objc.RegisterName("isHeadless")
	sel_isLowPower                                                                                                                    = objc.RegisterName("isLowPower")
	sel_name                                                                                                                          = objc.RegisterName("name")
	sel_supportsFeatureSet                                                                                                            = objc.RegisterName("supportsFeatureSet:")
	sel_newCommandQueue                                                                                                               = objc.RegisterName("newCommandQueue")
	sel_newLibraryWithSource_options_error                                                                                            = objc.RegisterName("newLibraryWithSource:options:error:")
	sel_release                                                                                                                       = objc.RegisterName("release")
	sel_retain                                                                                                                        = objc.RegisterName("retain")
	sel_new                                                                                                                           = objc.RegisterName("new")
	sel_localizedDescription                                                                                                          = objc.RegisterName("localizedDescription")
	sel_setVertexFunction                                                                                                             = objc.RegisterName("setVertexFunction:")
	sel_setFragmentFunction                                                                                                           = objc.RegisterName("setFragmentFunction:")
	sel_colorAttachments                                                                                                              = objc.RegisterName("colorAttachments")
	sel_objectAtIndexedSubscript                                                                                                      = objc.RegisterName("objectAtIndexedSubscript:")
	sel_setPixelFormat                                                                                                                = objc.RegisterName("setPixelFormat:")
	sel_setBlendingEnabled                                                                                                            = objc.RegisterName("setBlendingEnabled:")
	sel_setDestinationAlphaBlendFactor                                                                                                = objc.RegisterName("setDestinationAlphaBlendFactor:")
	sel_setDestinationRGBBlendFactor                                                                                                  = objc.RegisterName("setDestinationRGBBlendFactor:")
	sel_setSourceAlphaBlendFactor                                                                                                     = objc.RegisterName("setSourceAlphaBlendFactor:")
	sel_setSourceRGBBlendFactor                                                                                                       = objc.RegisterName("setSourceRGBBlendFactor:")
	sel_setWriteMask                                                                                                                  = objc.RegisterName("setWriteMask:")
	sel_setStencilAttachmentPixelFormat                                                                                               = objc.RegisterName("setStencilAttachmentPixelFormat:")
	sel_newRenderPipelineStateWithDescriptor_error                                                                                    = objc.RegisterName("newRenderPipelineStateWithDescriptor:error:")
	sel_newBufferWithBytes_length_options                                                                                             = objc.RegisterName("newBufferWithBytes:length:options:")
	sel_newBufferWithLength_options                                                                                                   = objc.RegisterName("newBufferWithLength:options:")
	sel_setTextureType                                                                                                                = objc.RegisterName("setTextureType:")
	sel_didModifyRange                                                                                                                = objc.RegisterName("didModifyRange:")
	sel_setWidth                                                                                                                      = objc.RegisterName("setWidth:")
	sel_setHeight                                                                                                                     = objc.RegisterName("setHeight:")
	sel_width                                                                                                                         = objc.RegisterName("width")
	sel_height                                                                                                                        = objc.RegisterName("height")
	sel_contents                                                                                                                      = objc.RegisterName("contents")
	sel_setStorageMode                                                                                                                = objc.RegisterName("setStorageMode:")
	sel_setUsage                                                                                                                      = objc.RegisterName("setUsage:")
	sel_newTextureWithDescriptor                                                                                                      = objc.RegisterName("newTextureWithDescriptor:")
	sel_commandBuffer                                                                                                                 = objc.RegisterName("commandBuffer")
	sel_status                                                                                                                        = objc.RegisterName("status")
	sel_presentDrawable                                                                                                               = objc.RegisterName("presentDrawable:")
	sel_commit                                                                                                                        = objc.RegisterName("commit")
	sel_waitUntilCompleted                                                                                                            = objc.RegisterName("waitUntilCompleted")
	sel_waitUntilScheduled                                                                                                            = objc.RegisterName("waitUntilScheduled")
	sel_renderCommandEncoderWithDescriptor                                                                                            = objc.RegisterName("renderCommandEncoderWithDescriptor:")
	sel_stencilAttachment                                                                                                             = objc.RegisterName("stencilAttachment")
	sel_setLoadAction                                                                                                                 = objc.RegisterName("setLoadAction:")
	sel_setStoreAction                                                                                                                = objc.RegisterName("setStoreAction:")
	sel_setTexture                                                                                                                    = objc.RegisterName("setTexture:")
	sel_setClearColor                                                                                                                 = objc.RegisterName("setClearColor:")
	sel_blitCommandEncoder                                                                                                            = objc.RegisterName("blitCommandEncoder")
	sel_endEncoding                                                                                                                   = objc.RegisterName("endEncoding")
	sel_setRenderPipelineState                                                                                                        = objc.RegisterName("setRenderPipelineState:")
	sel_setViewport                                                                                                                   = objc.RegisterName("setViewport:")
	sel_setScissorRect                                                                                                                = objc.RegisterName("setScissorRect:")
	sel_setVertexBuffer_offset_atIndex                                                                                                = objc.RegisterName("setVertexBuffer:offset:atIndex:")
	sel_setVertexBytes_length_atIndex                                                                                                 = objc.RegisterName("setVertexBytes:length:atIndex:")
	sel_setFragmentBytes_length_atIndex                                                                                               = objc.RegisterName("setFragmentBytes:length:atIndex:")
	sel_setFragmentTexture_atIndex                                                                                                    = objc.RegisterName("setFragmentTexture:atIndex:")
	sel_setBlendColorRedGreenBlueAlpha                                                                                                = objc.RegisterName("setBlendColorRed:green:blue:alpha:")
	sel_setDepthStencilState                                                                                                          = objc.RegisterName("setDepthStencilState:")
	sel_drawPrimitives_vertexStart_vertexCount                                                                                        = objc.RegisterName("drawPrimitives:vertexStart:vertexCount:")
	sel_drawIndexedPrimitives_indexCount_indexType_indexBuffer_indexBufferOffset                                                      = objc.RegisterName("drawIndexedPrimitives:indexCount:indexType:indexBuffer:indexBufferOffset:")
	sel_synchronizeResource                                                                                                           = objc.RegisterName("synchronizeResource:")
	sel_synchronizeTexture_slice_level                                                                                                = objc.RegisterName("synchronizeTexture:slice:level:")
	sel_copyFromTexture_sourceSlice_sourceLevel_sourceOrigin_sourceSize_toTexture_destinationSlice_destinationLevel_destinationOrigin = objc.RegisterName("copyFromTexture:sourceSlice:sourceLevel:sourceOrigin:sourceSize:toTexture:destinationSlice:destinationLevel:destinationOrigin:")
	sel_newFunctionWithName                                                                                                           = objc.RegisterName("newFunctionWithName:")
	sel_backFaceStencil                                                                                                               = objc.RegisterName("backFaceStencil")
	sel_frontFaceStencil                                                                                                              = objc.RegisterName("frontFaceStencil")
	sel_setStencilFailureOperation                                                                                                    = objc.RegisterName("setStencilFailureOperation:")
	sel_setDepthFailureOperation                                                                                                      = objc.RegisterName("setDepthFailureOperation:")
	sel_setDepthStencilPassOperation                                                                                                  = objc.RegisterName("setDepthStencilPassOperation:")
	sel_setStencilCompareFunction                                                                                                     = objc.RegisterName("setStencilCompareFunction:")
	sel_newDepthStencilStateWithDescriptor                                                                                            = objc.RegisterName("newDepthStencilStateWithDescriptor:")
	sel_replaceRegion_mipmapLevel_withBytes_bytesPerRow                                                                               = objc.RegisterName("replaceRegion:mipmapLevel:withBytes:bytesPerRow:")
	sel_getBytes_bytesPerRow_fromRegion_mipmapLevel                                                                                   = objc.RegisterName("getBytes:bytesPerRow:fromRegion:mipmapLevel:")
)

// CreateSystemDefaultDevice returns the preferred system default Metal device.
//
// Reference: https://developer.apple.com/documentation/metal/1433401-mtlcreatesystemdefaultdevice.
func CreateSystemDefaultDevice() (Device, bool) {
	d, _, _ := purego.SyscallN(_MTLCreateSystemDefaultDevice)
	if d == 0 {
		return Device{}, false
	}
	var (
		headless bool
		lowPower bool
		name     string
	)
	if !cocoa.IsIOS {
		headless = int(objc.ID(d).Send(sel_isHeadless)) != 0
		lowPower = int(objc.ID(d).Send(sel_isLowPower)) != 0
	}
	name = cocoa.NSString{ID: objc.ID(d).Send(sel_name)}.String()

	return Device{
		device:   objc.ID(d),
		Headless: headless,
		LowPower: lowPower,
		Name:     name,
	}, true
}

// Device returns the underlying id<MTLDevice> pointer.
func (d Device) Device() unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&d.device)) }

// SupportsFeatureSet reports whether device d supports feature set fs.
//
// Reference: https://developer.apple.com/documentation/metal/mtldevice/1433418-supportsfeatureset.
func (d Device) SupportsFeatureSet(fs FeatureSet) bool {
	return d.device.Send(sel_supportsFeatureSet, uintptr(fs)) != 0
}

// MakeCommandQueue creates a serial command submission queue.
//
// Reference: https://developer.apple.com/documentation/metal/mtldevice/1433388-makecommandqueue.
func (d Device) MakeCommandQueue() CommandQueue {
	return CommandQueue{d.device.Send(sel_newCommandQueue)}
}

// MakeLibrary creates a new library that contains
// the functions stored in the specified source string.
//
// Reference: https://developer.apple.com/documentation/metal/mtldevice/1433431-makelibrary.
func (d Device) MakeLibrary(source string, opt CompileOptions) (Library, error) {
	var err cocoa.NSError
	l := d.device.Send(
		sel_newLibraryWithSource_options_error,
		cocoa.NSString_alloc().InitWithUTF8String(source).ID,
		0,
		unsafe.Pointer(&err),
	)
	if l == 0 {
		return Library{}, errors.New(cocoa.NSString{ID: err.Send(sel_localizedDescription)}.String())
	}

	return Library{l}, nil
}

// MakeRenderPipelineState creates a render pipeline state object.
//
// Reference: https://developer.apple.com/documentation/metal/mtldevice/1433369-makerenderpipelinestate.
func (d Device) MakeRenderPipelineState(rpd RenderPipelineDescriptor) (RenderPipelineState, error) {
	renderPipelineDescriptor := objc.ID(class_MTLRenderPipelineDescriptor).Send(sel_new)
	renderPipelineDescriptor.Send(sel_setVertexFunction, rpd.VertexFunction.function)
	renderPipelineDescriptor.Send(sel_setFragmentFunction, rpd.FragmentFunction.function)
	colorAttachments0 := renderPipelineDescriptor.Send(sel_colorAttachments).Send(sel_objectAtIndexedSubscript, 0)
	colorAttachments0.Send(sel_setPixelFormat, uintptr(rpd.ColorAttachments[0].PixelFormat))
	colorAttachments0.Send(sel_setBlendingEnabled, rpd.ColorAttachments[0].BlendingEnabled)
	colorAttachments0.Send(sel_setDestinationAlphaBlendFactor, uintptr(rpd.ColorAttachments[0].DestinationAlphaBlendFactor))
	colorAttachments0.Send(sel_setDestinationRGBBlendFactor, uintptr(rpd.ColorAttachments[0].DestinationRGBBlendFactor))
	colorAttachments0.Send(sel_setSourceAlphaBlendFactor, uintptr(rpd.ColorAttachments[0].SourceAlphaBlendFactor))
	colorAttachments0.Send(sel_setSourceRGBBlendFactor, uintptr(rpd.ColorAttachments[0].SourceRGBBlendFactor))
	colorAttachments0.Send(sel_setWriteMask, uintptr(rpd.ColorAttachments[0].WriteMask))
	renderPipelineDescriptor.Send(sel_setStencilAttachmentPixelFormat, uintptr(rpd.StencilAttachmentPixelFormat))
	var err cocoa.NSError
	renderPipelineState := d.device.Send(sel_newRenderPipelineStateWithDescriptor_error,
		renderPipelineDescriptor,
		unsafe.Pointer(&err),
	)
	renderPipelineDescriptor.Send(sel_release)
	if renderPipelineState == 0 {
		return RenderPipelineState{}, errors.New(cocoa.NSString{ID: err.Send(sel_localizedDescription)}.String())
	}

	return RenderPipelineState{renderPipelineState}, nil
}

// MakeBufferWithBytes allocates a new buffer of a given length
// and initializes its contents by copying existing data into it.
//
// Reference: https://developer.apple.com/documentation/metal/mtldevice/1433429-makebuffer.
func (d Device) MakeBufferWithBytes(bytes unsafe.Pointer, length uintptr, opt ResourceOptions) Buffer {
	return Buffer{d.device.Send(sel_newBufferWithBytes_length_options, bytes, length, uintptr(opt))}
}

// MakeBufferWithLength allocates a new zero-filled buffer of a given length.
//
// Reference: https://developer.apple.com/documentation/metal/mtldevice/1433375-newbufferwithlength
func (d Device) MakeBufferWithLength(length uintptr, opt ResourceOptions) Buffer {
	return Buffer{d.device.Send(sel_newBufferWithLength_options, length, uintptr(opt))}
}

// MakeTexture creates a texture object with privately owned storage
// that contains texture state.
//
// Reference: https://developer.apple.com/documentation/metal/mtldevice/1433425-maketexture.
func (d Device) MakeTexture(td TextureDescriptor) Texture {
	textureDescriptor := objc.ID(class_MTLTextureDescriptor).Send(sel_new)
	textureDescriptor.Send(sel_setTextureType, uintptr(td.TextureType))
	textureDescriptor.Send(sel_setPixelFormat, uintptr(td.PixelFormat))
	textureDescriptor.Send(sel_setWidth, uintptr(td.Width))
	textureDescriptor.Send(sel_setHeight, uintptr(td.Height))
	textureDescriptor.Send(sel_setStorageMode, uintptr(td.StorageMode))
	textureDescriptor.Send(sel_setUsage, uintptr(td.Usage))
	texture := d.device.Send(sel_newTextureWithDescriptor, textureDescriptor)
	textureDescriptor.Send(sel_release)
	return Texture{
		texture: texture,
	}
}

// MakeDepthStencilState creates a new object that contains depth and stencil test state.
//
// Reference: https://developer.apple.com/documentation/metal/mtldevice/1433412-makedepthstencilstate
func (d Device) MakeDepthStencilState(dsd DepthStencilDescriptor) DepthStencilState {
	depthStencilDescriptor := objc.ID(class_MTLDepthStencilDescriptor).Send(sel_new)
	backFaceStencil := depthStencilDescriptor.Send(sel_backFaceStencil)
	backFaceStencil.Send(sel_setStencilFailureOperation, uintptr(dsd.BackFaceStencil.StencilFailureOperation))
	backFaceStencil.Send(sel_setDepthFailureOperation, uintptr(dsd.BackFaceStencil.DepthFailureOperation))
	backFaceStencil.Send(sel_setDepthStencilPassOperation, uintptr(dsd.BackFaceStencil.DepthStencilPassOperation))
	backFaceStencil.Send(sel_setStencilCompareFunction, uintptr(dsd.BackFaceStencil.StencilCompareFunction)) //TODO
	frontFaceStencil := depthStencilDescriptor.Send(sel_frontFaceStencil)
	frontFaceStencil.Send(sel_setStencilFailureOperation, uintptr(dsd.FrontFaceStencil.StencilFailureOperation))
	frontFaceStencil.Send(sel_setDepthFailureOperation, uintptr(dsd.FrontFaceStencil.DepthFailureOperation))
	frontFaceStencil.Send(sel_setDepthStencilPassOperation, uintptr(dsd.FrontFaceStencil.DepthStencilPassOperation))
	frontFaceStencil.Send(sel_setStencilCompareFunction, uintptr(dsd.FrontFaceStencil.StencilCompareFunction))
	depthStencilState := d.device.Send(sel_newDepthStencilStateWithDescriptor, depthStencilDescriptor)
	depthStencilDescriptor.Send(sel_release)
	return DepthStencilState{
		depthStencilState: depthStencilState,
	}
}

// CompileOptions specifies optional compilation settings for
// the graphics or compute functions within a library.
//
// Reference: https://developer.apple.com/documentation/metal/mtlcompileoptions.
type CompileOptions struct {
	// TODO.
}

// Drawable is a displayable resource that can be rendered or written to.
//
// Reference: https://developer.apple.com/documentation/metal/mtldrawable.
type Drawable interface {
	// Drawable returns the underlying id<MTLDrawable> pointer.
	Drawable() unsafe.Pointer
}

// CommandQueue is a queue that organizes the order
// in which command buffers are executed by the GPU.
//
// Reference: https://developer.apple.com/documentation/metal/mtlcommandqueue.
type CommandQueue struct {
	commandQueue objc.ID
}

func (cq CommandQueue) Release() {
	cq.commandQueue.Send(sel_release)
}

// MakeCommandBuffer creates a command buffer.
//
// Reference: https://developer.apple.com/documentation/metal/mtlcommandqueue/1508686-makecommandbuffer.
func (cq CommandQueue) MakeCommandBuffer() CommandBuffer {
	return CommandBuffer{cq.commandQueue.Send(sel_commandBuffer)}
}

// CommandBuffer is a container that stores encoded commands
// that are committed to and executed by the GPU.
//
// Reference: https://developer.apple.com/documentation/metal/mtlcommandbuffer.
type CommandBuffer struct {
	commandBuffer objc.ID
}

func (cb CommandBuffer) Retain() {
	cb.commandBuffer.Send(sel_retain)
}

func (cb CommandBuffer) Release() {
	cb.commandBuffer.Send(sel_release)
}

// Status returns the current stage in the lifetime of the command buffer.
//
// Reference: https://developer.apple.com/documentation/metal/mtlcommandbuffer/1443048-status
func (cb CommandBuffer) Status() CommandBufferStatus {
	return CommandBufferStatus(cb.commandBuffer.Send(sel_status))
}

// PresentDrawable registers a drawable presentation to occur as soon as possible.
//
// Reference: https://developer.apple.com/documentation/metal/mtlcommandbuffer/1443029-presentdrawable.
func (cb CommandBuffer) PresentDrawable(d Drawable) {
	cb.commandBuffer.Send(sel_presentDrawable, d.Drawable())
}

// Commit commits this command buffer for execution as soon as possible.
//
// Reference: https://developer.apple.com/documentation/metal/mtlcommandbuffer/1443003-commit.
func (cb CommandBuffer) Commit() {
	cb.commandBuffer.Send(sel_commit)
}

// WaitUntilCompleted waits for the execution of this command buffer to complete.
//
// Reference: https://developer.apple.com/documentation/metal/mtlcommandbuffer/1443039-waituntilcompleted.
func (cb CommandBuffer) WaitUntilCompleted() {
	cb.commandBuffer.Send(sel_waitUntilCompleted)
}

// WaitUntilScheduled blocks execution of the current thread until the command buffer is scheduled.
//
// Reference: https://developer.apple.com/documentation/metal/mtlcommandbuffer/1443036-waituntilscheduled.
func (cb CommandBuffer) WaitUntilScheduled() {
	cb.commandBuffer.Send(sel_waitUntilScheduled)
}

// MakeRenderCommandEncoder creates an encoder object that can
// encode graphics rendering commands into this command buffer.
//
// Reference: https://developer.apple.com/documentation/metal/mtlcommandbuffer/1442999-makerendercommandencoder.
func (cb CommandBuffer) MakeRenderCommandEncoder(rpd RenderPassDescriptor) RenderCommandEncoder {
	var renderPassDescriptor = objc.ID(class_MTLRenderPassDescriptor).Send(sel_new)
	var colorAttachments0 = renderPassDescriptor.Send(sel_colorAttachments).Send(sel_objectAtIndexedSubscript, 0)
	colorAttachments0.Send(sel_setLoadAction, int(rpd.ColorAttachments[0].LoadAction))
	colorAttachments0.Send(sel_setStoreAction, int(rpd.ColorAttachments[0].StoreAction))
	colorAttachments0.Send(sel_setTexture, rpd.ColorAttachments[0].Texture.texture)
	sig := cocoa.NSMethodSignature_instanceMethodSignatureForSelector(colorAttachments0.Send(sel_class), sel_setClearColor)
	inv := cocoa.NSInvocation_invocationWithMethodSignature(sig)
	inv.SetTarget(colorAttachments0)
	inv.SetSelector(sel_setClearColor)
	inv.SetArgumentAtIndex(unsafe.Pointer(&rpd.ColorAttachments[0].ClearColor), 2)
	inv.Invoke()
	var stencilAttachment = renderPassDescriptor.Send(sel_stencilAttachment)
	stencilAttachment.Send(sel_setLoadAction, int(rpd.StencilAttachment.LoadAction))
	stencilAttachment.Send(sel_setStoreAction, int(rpd.StencilAttachment.StoreAction))
	stencilAttachment.Send(sel_setTexture, rpd.StencilAttachment.Texture.texture)
	var rce = cb.commandBuffer.Send(sel_renderCommandEncoderWithDescriptor, renderPassDescriptor)
	renderPassDescriptor.Send(sel_release)
	return RenderCommandEncoder{CommandEncoder{rce}}
}

// MakeBlitCommandEncoder creates an encoder object that can encode
// memory operation (blit) commands into this command buffer.
//
// Reference: https://developer.apple.com/documentation/metal/mtlcommandbuffer/1443001-makeblitcommandencoder.
func (cb CommandBuffer) MakeBlitCommandEncoder() BlitCommandEncoder {
	ce := cb.commandBuffer.Send(sel_blitCommandEncoder)
	return BlitCommandEncoder{CommandEncoder{ce}}
}

// CommandEncoder is an encoder that writes sequential GPU commands
// into a command buffer.
//
// Reference: https://developer.apple.com/documentation/metal/mtlcommandencoder.
type CommandEncoder struct {
	commandEncoder objc.ID
}

// EndEncoding declares that all command generation from this encoder is completed.
//
// Reference: https://developer.apple.com/documentation/metal/mtlcommandencoder/1458038-endencoding.
func (ce CommandEncoder) EndEncoding() {
	ce.commandEncoder.Send(sel_endEncoding)
}

// RenderCommandEncoder is an encoder that specifies graphics-rendering commands
// and executes graphics functions.
//
// Reference: https://developer.apple.com/documentation/metal/mtlrendercommandencoder.
type RenderCommandEncoder struct {
	CommandEncoder
}

func (rce RenderCommandEncoder) Release() {
	rce.commandEncoder.Send(sel_release)
}

// SetRenderPipelineState sets the current render pipeline state object.
//
// Reference: https://developer.apple.com/documentation/metal/mtlrendercommandencoder/1515811-setrenderpipelinestate.
func (rce RenderCommandEncoder) SetRenderPipelineState(rps RenderPipelineState) {
	rce.commandEncoder.Send(sel_setRenderPipelineState, rps.renderPipelineState)
}

func (rce RenderCommandEncoder) SetViewport(viewport Viewport) {
	inv := cocoa.NSInvocation_invocationWithMethodSignature(cocoa.NSMethodSignature_signatureWithObjCTypes("v@:{MTLViewport=dddddd}"))
	inv.SetTarget(rce.commandEncoder)
	inv.SetSelector(sel_setViewport)
	inv.SetArgumentAtIndex(unsafe.Pointer(&viewport), 2)
	inv.Invoke()
}

// SetScissorRect sets the scissor rectangle for a fragment scissor test.
//
// Reference: https://developer.apple.com/documentation/metal/mtlrendercommandencoder/1515583-setscissorrect
func (rce RenderCommandEncoder) SetScissorRect(scissorRect ScissorRect) {
	inv := cocoa.NSInvocation_invocationWithMethodSignature(cocoa.NSMethodSignature_signatureWithObjCTypes("v@:{MTLScissorRect=qqqq}"))
	inv.SetTarget(rce.commandEncoder)
	inv.SetSelector(sel_setScissorRect)
	inv.SetArgumentAtIndex(unsafe.Pointer(&scissorRect), 2)
	inv.Invoke()
}

// SetVertexBuffer sets a buffer for the vertex shader function at an index
// in the buffer argument table with an offset that specifies the start of the data.
//
// Reference: https://developer.apple.com/documentation/metal/mtlrendercommandencoder/1515829-setvertexbuffer.
func (rce RenderCommandEncoder) SetVertexBuffer(buf Buffer, offset, index int) {
	rce.commandEncoder.Send(sel_setVertexBuffer_offset_atIndex, buf.buffer, offset, index)
}

// SetVertexBytes sets a block of data for the vertex function.
//
// Reference: https://developer.apple.com/documentation/metal/mtlrendercommandencoder/1515846-setvertexbytes.
func (rce RenderCommandEncoder) SetVertexBytes(bytes unsafe.Pointer, length uintptr, index int) {
	rce.commandEncoder.Send(sel_setVertexBytes_length_atIndex, bytes, length, index)
}

func (rce RenderCommandEncoder) SetFragmentBytes(bytes unsafe.Pointer, length uintptr, index int) {
	rce.commandEncoder.Send(sel_setFragmentBytes_length_atIndex, bytes, length, index)
}

// SetFragmentTexture sets a texture for the fragment function at an index in the texture argument table.
//
// Reference: https://developer.apple.com/documentation/metal/mtlrendercommandencoder/1515390-setfragmenttexture
func (rce RenderCommandEncoder) SetFragmentTexture(texture Texture, index int) {
	rce.commandEncoder.Send(sel_setFragmentTexture_atIndex, texture.texture, index)
}

func (rce RenderCommandEncoder) SetBlendColor(red, green, blue, alpha float32) {
	inv := cocoa.NSInvocation_invocationWithMethodSignature(cocoa.NSMethodSignature_signatureWithObjCTypes("v@:ffff"))
	inv.SetTarget(rce.commandEncoder)
	inv.SetSelector(sel_setBlendColorRedGreenBlueAlpha)
	inv.SetArgumentAtIndex(unsafe.Pointer(&red), 2)
	inv.SetArgumentAtIndex(unsafe.Pointer(&green), 3)
	inv.SetArgumentAtIndex(unsafe.Pointer(&blue), 4)
	inv.SetArgumentAtIndex(unsafe.Pointer(&alpha), 5)
	inv.Invoke()
}

// SetDepthStencilState sets the depth and stencil test state.
//
// Reference: https://developer.apple.com/documentation/metal/mtlrendercommandencoder/1516119-setdepthstencilstate
func (rce RenderCommandEncoder) SetDepthStencilState(depthStencilState DepthStencilState) {
	rce.commandEncoder.Send(sel_setDepthStencilState, depthStencilState.depthStencilState)
}

// DrawPrimitives renders one instance of primitives using vertex data
// in contiguous array elements.
//
// Reference: https://developer.apple.com/documentation/metal/mtlrendercommandencoder/1516326-drawprimitives.
func (rce RenderCommandEncoder) DrawPrimitives(typ PrimitiveType, vertexStart, vertexCount int) {
	rce.commandEncoder.Send(sel_drawPrimitives_vertexStart_vertexCount, uintptr(typ), vertexStart, vertexCount)
}

// DrawIndexedPrimitives encodes a command to render one instance of primitives using an index list specified in a buffer.
//
// Reference: https://developer.apple.com/documentation/metal/mtlrendercommandencoder/1515542-drawindexedprimitives
func (rce RenderCommandEncoder) DrawIndexedPrimitives(typ PrimitiveType, indexCount int, indexType IndexType, indexBuffer Buffer, indexBufferOffset int) {
	rce.commandEncoder.Send(
		sel_drawIndexedPrimitives_indexCount_indexType_indexBuffer_indexBufferOffset,
		uintptr(typ), indexCount, uintptr(indexType), indexBuffer.buffer, indexBufferOffset)
}

// BlitCommandEncoder is an encoder that specifies resource copy
// and resource synchronization commands.
//
// Reference: https://developer.apple.com/documentation/metal/mtlblitcommandencoder.
type BlitCommandEncoder struct {
	CommandEncoder
}

// Synchronize flushes any copy of the specified resource from its corresponding
// Device caches and, if needed, invalidates any CPU caches.
//
// Reference: https://developer.apple.com/documentation/metal/mtlblitcommandencoder/1400775-synchronize.
func (bce BlitCommandEncoder) Synchronize(resource Resource) {
	if cocoa.IsIOS {
		return
	}
	bce.commandEncoder.Send(sel_synchronizeResource, resource.resource())
}

func (bce BlitCommandEncoder) SynchronizeTexture(texture Texture, slice int, level int) {
	if cocoa.IsIOS {
		return
	}
	bce.commandEncoder.Send(sel_synchronizeTexture_slice_level, texture.texture, slice, level)
}

func (bce BlitCommandEncoder) CopyFromTexture(sourceTexture Texture, sourceSlice int, sourceLevel int, sourceOrigin Origin, sourceSize Size, destinationTexture Texture, destinationSlice int, destinationLevel int, destinationOrigin Origin) {
	inv := cocoa.NSInvocation_invocationWithMethodSignature(cocoa.NSMethodSignature_signatureWithObjCTypes("v@:@QQ{MTLOrigin=qqq}{MTLSize=qqq}@QQ{MTLOrigin=qqq}"))
	inv.SetTarget(bce.commandEncoder)
	inv.SetSelector(sel_copyFromTexture_sourceSlice_sourceLevel_sourceOrigin_sourceSize_toTexture_destinationSlice_destinationLevel_destinationOrigin)
	inv.SetArgumentAtIndex(unsafe.Pointer(&sourceTexture), 2)
	inv.SetArgumentAtIndex(unsafe.Pointer(&sourceSlice), 3)
	inv.SetArgumentAtIndex(unsafe.Pointer(&sourceLevel), 4)
	inv.SetArgumentAtIndex(unsafe.Pointer(&sourceOrigin), 5)
	inv.SetArgumentAtIndex(unsafe.Pointer(&sourceSize), 6)
	inv.SetArgumentAtIndex(unsafe.Pointer(&destinationTexture), 7)
	inv.SetArgumentAtIndex(unsafe.Pointer(&destinationSlice), 8)
	inv.SetArgumentAtIndex(unsafe.Pointer(&destinationLevel), 9)
	inv.SetArgumentAtIndex(unsafe.Pointer(&destinationOrigin), 10)
	inv.Invoke()
}

// Library is a collection of compiled graphics or compute functions.
//
// Reference: https://developer.apple.com/documentation/metal/mtllibrary.
type Library struct {
	library objc.ID
}

// MakeFunction returns a pre-compiled, non-specialized function.
//
// Reference: https://developer.apple.com/documentation/metal/mtllibrary/1515524-makefunction.
func (l Library) MakeFunction(name string) (Function, error) {
	f := l.library.Send(sel_newFunctionWithName,
		cocoa.NSString_alloc().InitWithUTF8String(name).ID,
	)
	if f == 0 {
		return Function{}, fmt.Errorf("function %q not found", name)
	}
	return Function{f}, nil
}

// Texture is a memory allocation for storing formatted
// image data that is accessible to the GPU.
//
// Reference: https://developer.apple.com/documentation/metal/mtltexture.
type Texture struct {
	texture objc.ID
}

// NewTexture returns a Texture that wraps an existing id<MTLTexture> pointer.
func NewTexture(texture objc.ID) Texture {
	return Texture{texture: texture}
}

// resource implements the Resource interface.
func (t Texture) resource() unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&t.texture)) }

func (t Texture) Release() {
	t.texture.Send(sel_release)
}

// GetBytes copies a block of pixels from the storage allocation of texture
// slice zero into system memory at a specified address.
//
// Reference: https://developer.apple.com/documentation/metal/mtltexture/1515751-getbytes.
func (t Texture) GetBytes(pixelBytes *byte, bytesPerRow uintptr, region Region, level int) {
	inv := cocoa.NSInvocation_invocationWithMethodSignature(cocoa.NSMethodSignature_signatureWithObjCTypes("v@:^vQ{MTLRegion={MTLOrigin=qqq}{MTLSize=qqq}}Q"))
	inv.SetTarget(t.texture)
	inv.SetSelector(sel_getBytes_bytesPerRow_fromRegion_mipmapLevel)
	inv.SetArgumentAtIndex(unsafe.Pointer(&pixelBytes), 2)
	inv.SetArgumentAtIndex(unsafe.Pointer(&bytesPerRow), 3)
	inv.SetArgumentAtIndex(unsafe.Pointer(&region), 4)
	inv.SetArgumentAtIndex(unsafe.Pointer(&level), 5)
	inv.Invoke()
}

// ReplaceRegion copies a block of pixels from the caller's pointer into the storage allocation for slice 0 of a texture.
//
// Reference: https://developer.apple.com/documentation/metal/mtltexture/1515464-replaceregion
func (t Texture) ReplaceRegion(region Region, level int, pixelBytes unsafe.Pointer, bytesPerRow int) {
	inv := cocoa.NSInvocation_invocationWithMethodSignature(cocoa.NSMethodSignature_signatureWithObjCTypes("v@:{MTLRegion={MTLOrigin=qqq}{MTLSize=qqq}}Q^vQ"))
	inv.SetTarget(t.texture)
	inv.SetSelector(sel_replaceRegion_mipmapLevel_withBytes_bytesPerRow)
	inv.SetArgumentAtIndex(unsafe.Pointer(&region), 2)
	inv.SetArgumentAtIndex(unsafe.Pointer(&level), 3)
	inv.SetArgumentAtIndex(unsafe.Pointer(&pixelBytes), 4)
	inv.SetArgumentAtIndex(unsafe.Pointer(&bytesPerRow), 5)
	inv.Invoke()
}

// Width is the width of the texture image for the base level mipmap, in pixels.
//
// Reference: https://developer.apple.com/documentation/metal/mtltexture/1515339-width
func (t Texture) Width() int {
	return int(t.texture.Send(sel_width))
}

// Height is the height of the texture image for the base level mipmap, in pixels.
//
// Reference: https://developer.apple.com/documentation/metal/mtltexture/1515938-height
func (t Texture) Height() int {
	return int(t.texture.Send(sel_height))
}

// Buffer is a memory allocation for storing unformatted data
// that is accessible to the GPU.
//
// Reference: https://developer.apple.com/documentation/metal/mtlbuffer.
type Buffer struct {
	buffer objc.ID
}

func (b Buffer) resource() unsafe.Pointer { return *(*unsafe.Pointer)(unsafe.Pointer(&b.buffer)) }

func (b Buffer) Length() uintptr {
	return uintptr(b.buffer.Send(sel_length))
}

func (b Buffer) CopyToContents(data unsafe.Pointer, lengthInBytes uintptr) {
	contents := b.buffer.Send(sel_contents)
	// use unsafe.Slice when ebitengine reaches 1.17
	var contentSlice []byte
	contentHeader := (*reflect.SliceHeader)(unsafe.Pointer(&contentSlice))
	contentHeader.Data = uintptr(contents)
	contentHeader.Len = int(lengthInBytes)
	contentHeader.Cap = int(lengthInBytes)
	var dataSlice []byte
	dataHeader := (*reflect.SliceHeader)(unsafe.Pointer(&dataSlice))
	dataHeader.Data = uintptr(data)
	dataHeader.Len = int(lengthInBytes)
	dataHeader.Cap = int(lengthInBytes)
	copy(contentSlice, dataSlice)
	if !cocoa.IsIOS {
		b.buffer.Send(sel_didModifyRange, 0, lengthInBytes)
	}
}

func (b Buffer) Retain() {
	b.buffer.Send(sel_retain)
}

func (b Buffer) Release() {
	b.buffer.Send(sel_release)
}

func (b Buffer) Native() unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&b.buffer))
}

// Function represents a programmable graphics or compute function executed by the GPU.
//
// Reference: https://developer.apple.com/documentation/metal/mtlfunction.
type Function struct {
	function objc.ID
}

func (f Function) Release() {
	f.function.Send(sel_release)
}

// RenderPipelineState contains the graphics functions
// and configuration state used in a render pass.
//
// Reference: https://developer.apple.com/documentation/metal/mtlrenderpipelinestate.
type RenderPipelineState struct {
	renderPipelineState objc.ID
}

func (r RenderPipelineState) Release() {
	r.renderPipelineState.Send(sel_release)
}

// Region is a rectangular block of pixels in an image or texture,
// defined by its upper-left corner and its size.
//
// Reference: https://developer.apple.com/documentation/metal/mtlregion.
type Region struct {
	Origin Origin // The location of the upper-left corner of the block.
	Size   Size   // The size of the block.
}

// Origin represents the location of a pixel in an image or texture relative
// to the upper-left corner, whose coordinates are (0, 0).
//
// Reference: https://developer.apple.com/documentation/metal/mtlorigin.
type Origin struct{ X, Y, Z int }

// Size represents the set of dimensions that declare the size of an object,
// such as an image, texture, threadgroup, or grid.
//
// Reference: https://developer.apple.com/documentation/metal/mtlsize.
type Size struct{ Width, Height, Depth int }

// RegionMake2D returns a 2D, rectangular region for image or texture data.
//
// Reference: https://developer.apple.com/documentation/metal/1515675-mtlregionmake2d.
func RegionMake2D(x, y, width, height int) Region {
	return Region{
		Origin: Origin{x, y, 0},
		Size:   Size{width, height, 1},
	}
}

type Viewport struct {
	OriginX float64
	OriginY float64
	Width   float64
	Height  float64
	ZNear   float64
	ZFar    float64
}

// ScissorRect represents a rectangle for the scissor fragment test.
//
// Reference: https://developer.apple.com/documentation/metal/mtlscissorrect
type ScissorRect struct {
	X      int
	Y      int
	Width  int
	Height int
}

// DepthStencilState is a depth and stencil state object that specifies the depth and stencil configuration and operations used in a render pass.
//
// Reference: https://developer.apple.com/documentation/metal/mtldepthstencilstate
type DepthStencilState struct {
	depthStencilState objc.ID
}

func (d DepthStencilState) Release() {
	d.depthStencilState.Send(sel_release)
}

// DepthStencilDescriptor is an object that configures new MTLDepthStencilState objects.
//
// Reference: https://developer.apple.com/documentation/metal/mtldepthstencildescriptor
type DepthStencilDescriptor struct {
	// BackFaceStencil is the stencil descriptor for back-facing primitives.
	BackFaceStencil StencilDescriptor

	// FrontFaceStencil is The stencil descriptor for front-facing primitives.
	FrontFaceStencil StencilDescriptor
}

// StencilDescriptor is an object that defines the front-facing or back-facing stencil operations of a depth and stencil state object.
//
// Reference: https://developer.apple.com/documentation/metal/mtlstencildescriptor
type StencilDescriptor struct {
	// StencilFailureOperation is the operation that is performed to update the values in the stencil attachment when the stencil test fails.
	StencilFailureOperation StencilOperation

	// DepthFailureOperation is the operation that is performed to update the values in the stencil attachment when the stencil test passes, but the depth test fails.
	DepthFailureOperation StencilOperation

	// DepthStencilPassOperation is the operation that is performed to update the values in the stencil attachment when both the stencil test and the depth test pass.
	DepthStencilPassOperation StencilOperation

	// StencilCompareFunction is the comparison that is performed between the masked reference value and a masked value in the stencil attachment.
	StencilCompareFunction CompareFunction
}
