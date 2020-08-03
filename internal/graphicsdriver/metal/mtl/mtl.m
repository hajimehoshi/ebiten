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

#include "mtl.h"
#import <Metal/Metal.h>
#include <stdlib.h>

struct Device CreateSystemDefaultDevice() {
  id<MTLDevice> device = MTLCreateSystemDefaultDevice();
  if (!device) {
    struct Device d;
    d.Device = NULL;
    return d;
  }

  struct Device d;
  d.Device = device;
#if !TARGET_OS_IPHONE
  d.Headless = device.headless;
  d.LowPower = device.lowPower;
#else
  d.Headless = 0;
  d.LowPower = 0;
#endif
  d.Name = device.name.UTF8String;
  return d;
}

uint8_t Device_SupportsFeatureSet(void *device, uint16_t featureSet) {
  return [(id<MTLDevice>)device supportsFeatureSet:featureSet];
}

void *Device_MakeCommandQueue(void *device) {
  return [(id<MTLDevice>)device newCommandQueue];
}

struct Library Device_MakeLibrary(void *device, const char *source,
                                  size_t sourceLength) {
  NSError *error;
  id<MTLLibrary> library = [(id<MTLDevice>)device
      newLibraryWithSource:[[NSString alloc] initWithBytes:source
                                                    length:sourceLength
                                                  encoding:NSUTF8StringEncoding]
                   options:NULL // TODO.
                     error:&error];

  struct Library l;
  l.Library = library;
  if (!library) {
    l.Error = error.localizedDescription.UTF8String;
  }
  return l;
}

struct RenderPipelineState
Device_MakeRenderPipelineState(void *device,
                               struct RenderPipelineDescriptor descriptor) {
  MTLRenderPipelineDescriptor *renderPipelineDescriptor =
      [[MTLRenderPipelineDescriptor alloc] init];
  renderPipelineDescriptor.vertexFunction = descriptor.VertexFunction;
  renderPipelineDescriptor.fragmentFunction = descriptor.FragmentFunction;
  renderPipelineDescriptor.colorAttachments[0].pixelFormat =
      descriptor.ColorAttachment0PixelFormat;
  renderPipelineDescriptor.colorAttachments[0].blendingEnabled =
      descriptor.ColorAttachment0BlendingEnabled;
  renderPipelineDescriptor.colorAttachments[0].destinationAlphaBlendFactor =
      descriptor.ColorAttachment0DestinationAlphaBlendFactor;
  renderPipelineDescriptor.colorAttachments[0].destinationRGBBlendFactor =
      descriptor.ColorAttachment0DestinationRGBBlendFactor;
  renderPipelineDescriptor.colorAttachments[0].sourceAlphaBlendFactor =
      descriptor.ColorAttachment0SourceAlphaBlendFactor;
  renderPipelineDescriptor.colorAttachments[0].sourceRGBBlendFactor =
      descriptor.ColorAttachment0SourceRGBBlendFactor;
  NSError *error;
  id<MTLRenderPipelineState> renderPipelineState = [(id<MTLDevice>)device
      newRenderPipelineStateWithDescriptor:renderPipelineDescriptor
                                     error:&error];
  [renderPipelineDescriptor release];
  struct RenderPipelineState rps;
  rps.RenderPipelineState = renderPipelineState;
  if (!renderPipelineState) {
    rps.Error = error.localizedDescription.UTF8String;
  }
  return rps;
}

void *Device_MakeBufferWithBytes(void *device, const void *bytes, size_t length,
                                 uint16_t options) {
  return [(id<MTLDevice>)device newBufferWithBytes:(const void *)bytes
                                            length:(NSUInteger)length
                                           options:(MTLResourceOptions)options];
}

void *Device_MakeBufferWithLength(void *device, size_t length,
                                  uint16_t options) {
  return
      [(id<MTLDevice>)device newBufferWithLength:(NSUInteger)length
                                         options:(MTLResourceOptions)options];
}

void *Device_MakeTexture(void *device, struct TextureDescriptor descriptor) {
  MTLTextureDescriptor *textureDescriptor = [[MTLTextureDescriptor alloc] init];
  textureDescriptor.textureType = descriptor.TextureType;
  textureDescriptor.pixelFormat = descriptor.PixelFormat;
  textureDescriptor.width = descriptor.Width;
  textureDescriptor.height = descriptor.Height;
  textureDescriptor.storageMode = descriptor.StorageMode;
  textureDescriptor.usage = descriptor.Usage;
  id<MTLTexture> texture =
      [(id<MTLDevice>)device newTextureWithDescriptor:textureDescriptor];
  [textureDescriptor release];
  return texture;
}

void CommandQueue_Release(void *commandQueue) {
  [(id<MTLCommandQueue>)commandQueue release];
}

void *CommandQueue_MakeCommandBuffer(void *commandQueue) {
  return [(id<MTLCommandQueue>)commandQueue commandBuffer];
}

void CommandBuffer_PresentDrawable(void *commandBuffer, void *drawable) {
  [(id<MTLCommandBuffer>)commandBuffer
      presentDrawable:(id<MTLDrawable>)drawable];
}

void CommandBuffer_Commit(void *commandBuffer) {
  [(id<MTLCommandBuffer>)commandBuffer commit];
}

void CommandBuffer_WaitUntilCompleted(void *commandBuffer) {
  [(id<MTLCommandBuffer>)commandBuffer waitUntilCompleted];
}

void *
CommandBuffer_MakeRenderCommandEncoder(void *commandBuffer,
                                       struct RenderPassDescriptor descriptor) {
  MTLRenderPassDescriptor *renderPassDescriptor =
      [[MTLRenderPassDescriptor alloc] init];
  renderPassDescriptor.colorAttachments[0].loadAction =
      descriptor.ColorAttachment0LoadAction;
  renderPassDescriptor.colorAttachments[0].storeAction =
      descriptor.ColorAttachment0StoreAction;
  renderPassDescriptor.colorAttachments[0].clearColor =
      MTLClearColorMake(descriptor.ColorAttachment0ClearColor.Red,
                        descriptor.ColorAttachment0ClearColor.Green,
                        descriptor.ColorAttachment0ClearColor.Blue,
                        descriptor.ColorAttachment0ClearColor.Alpha);
  renderPassDescriptor.colorAttachments[0].texture =
      (id<MTLTexture>)descriptor.ColorAttachment0Texture;
  id<MTLRenderCommandEncoder> rce = [(id<MTLCommandBuffer>)commandBuffer
      renderCommandEncoderWithDescriptor:renderPassDescriptor];
  [renderPassDescriptor release];
  return rce;
}

void *CommandBuffer_MakeBlitCommandEncoder(void *commandBuffer) {
  return [(id<MTLCommandBuffer>)commandBuffer blitCommandEncoder];
}

void CommandEncoder_EndEncoding(void *commandEncoder) {
  [(id<MTLCommandEncoder>)commandEncoder endEncoding];
}

void RenderCommandEncoder_Release(void *renderCommandEncoder) {
  [(id<MTLRenderCommandEncoder>)renderCommandEncoder release];
}

void RenderCommandEncoder_SetRenderPipelineState(void *renderCommandEncoder,
                                                 void *renderPipelineState) {
  [(id<MTLRenderCommandEncoder>)renderCommandEncoder
      setRenderPipelineState:(id<MTLRenderPipelineState>)renderPipelineState];
}

void RenderCommandEncoder_SetViewport(void *renderCommandEncoder,
                                      struct Viewport viewport) {
  [(id<MTLRenderCommandEncoder>)renderCommandEncoder
      setViewport:(MTLViewport){
                      viewport.OriginX,
                      viewport.OriginY,
                      viewport.Width,
                      viewport.Height,
                      viewport.ZNear,
                      viewport.ZFar,
                  }];
}

void RenderCommandEncoder_SetVertexBuffer(void *renderCommandEncoder,
                                          void *buffer, uint_t offset,
                                          uint_t index) {
  [(id<MTLRenderCommandEncoder>)renderCommandEncoder
      setVertexBuffer:(id<MTLBuffer>)buffer
               offset:(NSUInteger)offset
              atIndex:(NSUInteger)index];
}

void RenderCommandEncoder_SetVertexBytes(void *renderCommandEncoder,
                                         const void *bytes, size_t length,
                                         uint_t index) {
  [(id<MTLRenderCommandEncoder>)renderCommandEncoder
      setVertexBytes:bytes
              length:(NSUInteger)length
             atIndex:(NSUInteger)index];
}

void RenderCommandEncoder_SetFragmentBytes(void *renderCommandEncoder,
                                           const void *bytes, size_t length,
                                           uint_t index) {
  [(id<MTLRenderCommandEncoder>)renderCommandEncoder
      setFragmentBytes:bytes
                length:(NSUInteger)length
               atIndex:(NSUInteger)index];
}

void RenderCommandEncoder_SetFragmentTexture(void *renderCommandEncoder,
                                             void *texture, uint_t index) {
  [(id<MTLRenderCommandEncoder>)renderCommandEncoder
      setFragmentTexture:(id<MTLTexture>)texture
                 atIndex:(NSUInteger)index];
}

void RenderCommandEncoder_SetBlendColor(void *renderCommandEncoder, float red,
                                        float green, float blue, float alpha) {
  [(id<MTLRenderCommandEncoder>)renderCommandEncoder setBlendColorRed:red
                                                                green:green
                                                                 blue:blue
                                                                alpha:alpha];
}

void RenderCommandEncoder_DrawPrimitives(void *renderCommandEncoder,
                                         uint8_t primitiveType,
                                         uint_t vertexStart,
                                         uint_t vertexCount) {
  [(id<MTLRenderCommandEncoder>)renderCommandEncoder
      drawPrimitives:(MTLPrimitiveType)primitiveType
         vertexStart:(NSUInteger)vertexStart
         vertexCount:(NSUInteger)vertexCount];
}

void RenderCommandEncoder_DrawIndexedPrimitives(
    void *renderCommandEncoder, uint8_t primitiveType, uint_t indexCount,
    uint8_t indexType, void *indexBuffer, uint_t indexBufferOffset) {
  [(id<MTLRenderCommandEncoder>)renderCommandEncoder
      drawIndexedPrimitives:(MTLPrimitiveType)primitiveType
                 indexCount:(NSUInteger)indexCount
                  indexType:(MTLIndexType)indexType
                indexBuffer:(id<MTLBuffer>)indexBuffer
          indexBufferOffset:(NSUInteger)indexBufferOffset];
}

void BlitCommandEncoder_Synchronize(void *blitCommandEncoder, void *resource) {
#if !TARGET_OS_IPHONE
  [(id<MTLBlitCommandEncoder>)blitCommandEncoder
      synchronizeResource:(id<MTLResource>)resource];
#endif
}

void BlitCommandEncoder_SynchronizeTexture(void *blitCommandEncoder,
                                           void *texture, uint_t slice,
                                           uint_t level) {
#if !TARGET_OS_IPHONE
  [(id<MTLBlitCommandEncoder>)blitCommandEncoder
      synchronizeTexture:(id<MTLTexture>)texture
                   slice:(NSUInteger)slice
                   level:(NSUInteger)level];
#endif
}

void BlitCommandEncoder_CopyFromTexture(
    void *blitCommandEncoder, void *sourceTexture, uint_t sourceSlice,
    uint_t sourceLevel, struct Origin sourceOrigin, struct Size sourceSize,
    void *destinationTexture, uint_t destinationSlice, uint_t destinationLevel,
    struct Origin destinationOrigin) {
  [(id<MTLBlitCommandEncoder>)blitCommandEncoder
        copyFromTexture:(id<MTLTexture>)sourceTexture
            sourceSlice:(NSUInteger)sourceSlice
            sourceLevel:(NSUInteger)sourceLevel
           sourceOrigin:(MTLOrigin){sourceOrigin.X, sourceOrigin.Y,
                                    sourceOrigin.Z}
             sourceSize:(MTLSize){sourceSize.Width, sourceSize.Height,
                                  sourceSize.Depth}
              toTexture:(id<MTLTexture>)destinationTexture
       destinationSlice:(NSUInteger)destinationSlice
       destinationLevel:(NSUInteger)destinationLevel
      destinationOrigin:(MTLOrigin){destinationOrigin.X, destinationOrigin.Y,
                                    destinationOrigin.Z}];
}

void *Library_MakeFunction(void *library, const char *name) {
  return [(id<MTLLibrary>)library
      newFunctionWithName:[NSString stringWithUTF8String:name]];
}

void Texture_Release(void *texture) { [(id<MTLTexture>)texture release]; }

void Texture_GetBytes(void *texture, void *pixelBytes, size_t bytesPerRow,
                      struct Region region, uint_t level) {
  [(id<MTLTexture>)texture
         getBytes:(void *)pixelBytes
      bytesPerRow:(NSUInteger)bytesPerRow
       fromRegion:(MTLRegion) {
         {region.Origin.X, region.Origin.Y, region.Origin.Z}, {
           region.Size.Width, region.Size.Height, region.Size.Depth
         }
       }
      mipmapLevel:(NSUInteger)level];
}

void Texture_ReplaceRegion(void *texture, struct Region region, uint_t level,
                           void *bytes, uint_t bytesPerRow) {
  [(id<MTLTexture>)texture replaceRegion:(MTLRegion) {
    {region.Origin.X, region.Origin.Y, region.Origin.Z}, {
      region.Size.Width, region.Size.Height, region.Size.Depth
    }
  }
                             mipmapLevel:(NSUInteger)level
                               withBytes:bytes
                             bytesPerRow:(NSUInteger)bytesPerRow];
}

int Texture_Width(void *texture) { return [(id<MTLTexture>)texture width]; }

int Texture_Height(void *texture) { return [(id<MTLTexture>)texture height]; }

void Buffer_CopyToContents(void *buffer, void *data, size_t lengthInBytes) {
  memcpy(((id<MTLBuffer>)buffer).contents, data, lengthInBytes);
}

void Buffer_Retain(void *buffer) { [(id<MTLBuffer>)buffer retain]; }

void Buffer_Release(void *buffer) { [(id<MTLBuffer>)buffer release]; }

void Function_Release(void *function) { [(id<MTLFunction>)function release]; }

void RenderPipelineState_Release(void *renderPipelineState) {
  [(id<MTLRenderPipelineState>)renderPipelineState release];
}
