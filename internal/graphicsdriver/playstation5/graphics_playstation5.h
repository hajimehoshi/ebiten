// Copyright 2023 The Ebitengine Authors
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

//go:build playstation5

#ifndef EBITENGINE_INTERNAL_GRAPHICSDRIVER_PLAYSTATION5_GRAPHICS_PLAYSTATION5_H
#define EBITENGINE_INTERNAL_GRAPHICSDRIVER_PLAYSTATION5_GRAPHICS_PLAYSTATION5_H

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

typedef struct ebitengine_Error {
  const char *Message;
  int Code;
} ebitengine_Error;

static bool ebitengine_IsErrorNil(ebitengine_Error *err) {
  return err->Message == NULL && err->Code == 0;
}

typedef struct ebitengine_DstRegion {
  int MinX;
  int MinY;
  int MaxX;
  int MaxY;
  int IndexCount;
} ebitengine_DstRegion;

typedef struct ebitengine_Blend {
  uint8_t BlendFactorSourceRGB;
  uint8_t BlendFactorSourceAlpha;
  uint8_t BlendFactorDestinationRGB;
  uint8_t BlendFactorDestinationAlpha;
  uint8_t BlendOperationRGB;
  uint8_t BlendOperationAlpha;
} ebitengine_Blend;

ebitengine_Error ebitengine_InitializeGraphics(void);
ebitengine_Error ebitengine_NewImage(int *image, int width, int height);
ebitengine_Error ebitengine_NewScreenFramebufferImage(int *image, int width,
                                                      int height);
void ebitengine_DisposeImage(int id);

ebitengine_Error
ebitengine_DrawTriangles(int dst, int *srcs, int srcCount, int shader,
                         ebitengine_DstRegion *dstRegions, int dstRegionCount,
                         int indexOffset, ebitengine_Blend blend,
                         uint32_t *uniforms, int uniformCount, int fillRule);

ebitengine_Error ebitengine_NewShader(int *shader, const char *source);
void ebitengine_DisposeShader(int id);

#ifdef __cplusplus
} // extern "C"
#endif

#endif // EBITENGINE_INTERNAL_GRAPHICSDRIVER_PLAYSTATION5_GRAPHICS_PLAYSTATION5_H
