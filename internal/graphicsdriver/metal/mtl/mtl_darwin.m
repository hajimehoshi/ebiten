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

#include "mtl_darwin.h"
#import <Metal/Metal.h>
#include <stdlib.h>

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
  renderPassDescriptor.stencilAttachment.loadAction =
      descriptor.StencilAttachmentLoadAction;
  renderPassDescriptor.stencilAttachment.storeAction =
      descriptor.StencilAttachmentStoreAction;
  renderPassDescriptor.stencilAttachment.texture =
      (id<MTLTexture>)descriptor.StencilAttachmentTexture;
  id<MTLRenderCommandEncoder> rce = [(id<MTLCommandBuffer>)commandBuffer
      renderCommandEncoderWithDescriptor:renderPassDescriptor];
  [renderPassDescriptor release];
  return rce;
}

void RenderCommandEncoder_SetViewport(void *renderCommandEncoder,
                                      struct Viewport viewport) {
  [(id<MTLRenderCommandEncoder>)renderCommandEncoder
      setViewport:(MTLViewport){
                      .originX = viewport.OriginX,
                      .originY = viewport.OriginY,
                      .width = viewport.Width,
                      .height = viewport.Height,
                      .znear = viewport.ZNear,
                      .zfar = viewport.ZFar,
                  }];
}

void RenderCommandEncoder_SetScissorRect(void *renderCommandEncoder,
                                         struct ScissorRect scissorRect) {
  [(id<MTLRenderCommandEncoder>)renderCommandEncoder
      setScissorRect:(MTLScissorRect){
                         .x = scissorRect.X,
                         .y = scissorRect.Y,
                         .width = scissorRect.Width,
                         .height = scissorRect.Height,
                     }];
}

void RenderCommandEncoder_SetBlendColor(void *renderCommandEncoder, float red,
                                        float green, float blue, float alpha) {
  [(id<MTLRenderCommandEncoder>)renderCommandEncoder setBlendColorRed:red
                                                                green:green
                                                                 blue:blue
                                                                alpha:alpha];
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
           sourceOrigin:(MTLOrigin){.x = sourceOrigin.X,
                                    .y = sourceOrigin.Y,
                                    .z = sourceOrigin.Z}
             sourceSize:(MTLSize){.width = sourceSize.Width,
                                  .height = sourceSize.Height,
                                  .depth = sourceSize.Depth}
              toTexture:(id<MTLTexture>)destinationTexture
       destinationSlice:(NSUInteger)destinationSlice
       destinationLevel:(NSUInteger)destinationLevel
      destinationOrigin:(MTLOrigin){.x = destinationOrigin.X,
                                    .y = destinationOrigin.Y,
                                    .z = destinationOrigin.Z}];
}

void Texture_GetBytes(void *texture, void *pixelBytes, size_t bytesPerRow,
                      struct Region region, uint_t level) {
  [(id<MTLTexture>)texture getBytes:(void *)pixelBytes
                        bytesPerRow:(NSUInteger)bytesPerRow
                         fromRegion:(MTLRegion) {
                           .origin = {.x = region.Origin.X,
                                      .y = region.Origin.Y,
                                      .z = region.Origin.Z},
                           .size = {
                             .width = region.Size.Width,
                             .height = region.Size.Height,
                             .depth = region.Size.Depth
                           }
                         }
                        mipmapLevel:(NSUInteger)level];
}

void Texture_ReplaceRegion(void *texture, struct Region region, uint_t level,
                           void *bytes, uint_t bytesPerRow) {
  [(id<MTLTexture>)texture replaceRegion:(MTLRegion) {
    .origin = {.x = region.Origin.X,
               .y = region.Origin.Y,
               .z = region.Origin.Z},
    .size = {
      .width = region.Size.Width,
      .height = region.Size.Height,
      .depth = region.Size.Depth
    }
  }
                             mipmapLevel:(NSUInteger)level
                               withBytes:bytes
                             bytesPerRow:(NSUInteger)bytesPerRow];
}
