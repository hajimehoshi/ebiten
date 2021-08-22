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

#include <stddef.h>
#include <stdint.h>

typedef unsigned long uint_t;

struct Device {
  void *Device;
  uint8_t Headless;
  uint8_t LowPower;
  uint8_t Removable;
  uint64_t RegistryID;
  const char *Name;
};

struct Devices {
  struct Device *Devices;
  int Length;
};

struct Library {
  void *Library;
  const char *Error;
};

struct RenderPipelineDescriptor {
  void *VertexFunction;
  void *FragmentFunction;
  uint16_t ColorAttachment0PixelFormat;
  uint8_t ColorAttachment0BlendingEnabled;
  uint8_t ColorAttachment0DestinationAlphaBlendFactor;
  uint8_t ColorAttachment0DestinationRGBBlendFactor;
  uint8_t ColorAttachment0SourceAlphaBlendFactor;
  uint8_t ColorAttachment0SourceRGBBlendFactor;
  uint8_t ColorAttachment0WriteMask;
  uint8_t StencilAttachmentPixelFormat;
};

struct RenderPipelineState {
  void *RenderPipelineState;
  const char *Error;
};

struct ClearColor {
  double Red;
  double Green;
  double Blue;
  double Alpha;
};

struct RenderPassDescriptor {
  uint8_t ColorAttachment0LoadAction;
  uint8_t ColorAttachment0StoreAction;
  struct ClearColor ColorAttachment0ClearColor;
  void *ColorAttachment0Texture;
  uint8_t StencilAttachmentLoadAction;
  uint8_t StencilAttachmentStoreAction;
  void *StencilAttachmentTexture;
};

struct TextureDescriptor {
  uint16_t TextureType;
  uint16_t PixelFormat;
  uint_t Width;
  uint_t Height;
  uint8_t StorageMode;
  uint8_t Usage;
};

struct Origin {
  uint_t X;
  uint_t Y;
  uint_t Z;
};

struct Size {
  uint_t Width;
  uint_t Height;
  uint_t Depth;
};

struct Region {
  struct Origin Origin;
  struct Size Size;
};

struct Viewport {
  double OriginX;
  double OriginY;
  double Width;
  double Height;
  double ZNear;
  double ZFar;
};

struct ScissorRect {
  uint_t X;
  uint_t Y;
  uint_t Width;
  uint_t Height;
};

struct DepthStencilDescriptor {
  uint8_t BackFaceStencilStencilFailureOperation;
  uint8_t BackFaceStencilDepthFailureOperation;
  uint8_t BackFaceStencilDepthStencilPassOperation;
  uint8_t BackFaceStencilStencilCompareFunction;
  uint8_t FrontFaceStencilStencilFailureOperation;
  uint8_t FrontFaceStencilDepthFailureOperation;
  uint8_t FrontFaceStencilDepthStencilPassOperation;
  uint8_t FrontFaceStencilStencilCompareFunction;
};

struct Device CreateSystemDefaultDevice();
struct Devices CopyAllDevices();

uint8_t Device_SupportsFeatureSet(void *device, uint16_t featureSet);
void *Device_MakeCommandQueue(void *device);
struct Library Device_MakeLibrary(void *device, const char *source,
                                  size_t sourceLength);
struct RenderPipelineState
Device_MakeRenderPipelineState(void *device,
                               struct RenderPipelineDescriptor descriptor);
void *Device_MakeBufferWithBytes(void *device, const void *bytes, size_t length,
                                 uint16_t options);
void *Device_MakeBufferWithLength(void *device, size_t length,
                                  uint16_t options);
void *Device_MakeTexture(void *device, struct TextureDescriptor descriptor);
void *Device_MakeDepthStencilState(void *device,
                                   struct DepthStencilDescriptor descriptor);

void CommandQueue_Release(void *commandQueue);
void *CommandQueue_MakeCommandBuffer(void *commandQueue);

void CommandBuffer_Retain(void *commandBuffer);
void CommandBuffer_Release(void *commandBuffer);
uint8_t CommandBuffer_Status(void *commandBuffer);
void CommandBuffer_PresentDrawable(void *commandBuffer, void *drawable);
void CommandBuffer_Commit(void *commandBuffer);
void CommandBuffer_WaitUntilCompleted(void *commandBuffer);
void CommandBuffer_WaitUntilScheduled(void *commandBuffer);
void *
CommandBuffer_MakeRenderCommandEncoder(void *commandBuffer,
                                       struct RenderPassDescriptor descriptor);
void *CommandBuffer_MakeBlitCommandEncoder(void *commandBuffer);

void CommandEncoder_EndEncoding(void *commandEncoder);

void RenderCommandEncoder_Release(void *renderCommandEncoder);
void RenderCommandEncoder_SetRenderPipelineState(void *renderCommandEncoder,
                                                 void *renderPipelineState);
void RenderCommandEncoder_SetViewport(void *renderCommandEncoder,
                                      struct Viewport viewport);
void RenderCommandEncoder_SetScissorRect(void *renderCommandEncoder,
                                         struct ScissorRect scissorRect);
void RenderCommandEncoder_SetVertexBuffer(void *renderCommandEncoder,
                                          void *buffer, uint_t offset,
                                          uint_t index);
void RenderCommandEncoder_SetVertexBytes(void *renderCommandEncoder,
                                         const void *bytes, size_t length,
                                         uint_t index);
void RenderCommandEncoder_SetFragmentBytes(void *renderCommandEncoder,
                                           const void *bytes, size_t length,
                                           uint_t index);
void RenderCommandEncoder_SetBlendColor(void *renderCommandEncoder, float red,
                                        float green, float blue, float alpha);
void RenderCommandEncoder_SetFragmentTexture(void *renderCommandEncoder,
                                             void *texture, uint_t index);
void RenderCommandEncoder_SetDepthStencilState(void *renderCommandEncoder,
                                               void *depthStencilState);
void RenderCommandEncoder_DrawPrimitives(void *renderCommandEncoder,
                                         uint8_t primitiveType,
                                         uint_t vertexStart,
                                         uint_t vertexCount);
void RenderCommandEncoder_DrawIndexedPrimitives(
    void *renderCommandEncoder, uint8_t primitiveType, uint_t indexCount,
    uint8_t indexType, void *indexBuffer, uint_t indexBufferOffset);

void BlitCommandEncoder_Synchronize(void *blitCommandEncoder, void *resource);
void BlitCommandEncoder_SynchronizeTexture(void *blitCommandEncoder,
                                           void *texture, uint_t slice,
                                           uint_t level);
void BlitCommandEncoder_CopyFromTexture(
    void *blitCommandEncoder, void *sourceTexture, uint_t sourceSlice,
    uint_t sourceLevel, struct Origin sourceOrigin, struct Size sourceSize,
    void *destinationTexture, uint_t destinationSlice, uint_t destinationLevel,
    struct Origin destinationOrigin);

void *Library_MakeFunction(void *library, const char *name);

void Texture_Release(void *texture);
void Texture_GetBytes(void *texture, void *pixelBytes, size_t bytesPerRow,
                      struct Region region, uint_t level);
void Texture_ReplaceRegion(void *texture, struct Region region, uint_t level,
                           void *pixelBytes, uint_t bytesPerRow);
int Texture_Width(void *texture);
int Texture_Height(void *texture);

size_t Buffer_Length(void *buffer);
void Buffer_CopyToContents(void *buffer, void *data, size_t lengthInBytes);
void Buffer_Retain(void *buffer);
void Buffer_Release(void *buffer);
void Function_Release(void *function);
void RenderPipelineState_Release(void *renderPipelineState);
void DepthStencilState_Release(void *depthStencilState);
