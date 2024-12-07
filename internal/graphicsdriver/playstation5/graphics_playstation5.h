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

int ebitengine_ProjectionMatrixUniformDwordIndex();

typedef struct ebitengine_Error {
  const char *message;
  int code;
} ebitengine_Error;

static bool ebitengine_IsErrorNil(ebitengine_Error *err) {
  return err->message == NULL && err->code == 0;
}

typedef struct ebitengine_Region {
  int min_x;
  int min_y;
  int max_x;
  int max_y;
} ebitengine_Region;

typedef struct ebitengine_DstRegion {
  int min_x;
  int min_y;
  int max_x;
  int max_y;
  int index_count;
} ebitengine_DstRegion;

// kBlendFactor* and kBlendOperation* must be synced with
// internal/graphicsdriver/blend.go.

enum {
  kBlendFactorZero = 0,
  kBlendFactorOne = 1,
  kBlendFactorSourceColor = 2,
  kBlendFactorOneMinusSourceColor = 3,
  kBlendFactorSourceAlpha = 4,
  kBlendFactorOneMinusSourceAlpha = 5,
  kBlendFactorDestinationColor = 6,
  kBlendFactorOneMinusDestinationColor = 7,
  kBlendFactorDestinationAlpha = 8,
  kBlendFactorOneMinusDestinationAlpha = 9,
  kBlendFactorSourceAlphaSaturated = 10,
};

enum {
  kBlendOperationAdd = 0,
  kBlendOperationSubtract = 1,
  kBlendOperationReverseSubtract = 2,
  kBlendOperationMin = 3,
  kBlendOperationMax = 4,
};

typedef struct ebitengine_Blend {
  uint8_t factor_src_rgb;
  uint8_t factor_src_alpha;
  uint8_t factor_dst_rgb;
  uint8_t factor_dst_alpha;
  uint8_t operation_rgb;
  uint8_t operation_alpha;
} ebitengine_Blend;

ebitengine_Error ebitengine_InitializeGraphics(void);
ebitengine_Error ebitengine_NewImage(int *image, int width, int height);
ebitengine_Error ebitengine_NewScreenFramebufferImage(int *image, int width,
                                                      int height);
void ebitengine_ReadPixels(int image, uint8_t *pixels,
                           ebitengine_Region region);
ebitengine_Error ebitengine_FlushReadPixels(int image);
void ebitengine_WritePixels(int image, const uint8_t *pixels,
                            ebitengine_Region region);
ebitengine_Error ebitengine_FlushWritePixels(int image);
void ebitengine_DisposeImage(int id);

void ebitengine_Begin();
void ebitengine_End(int present);
void ebitengine_SetVertices(const float *vertices, int vertex_count,
                            const uint32_t *indices, int index_count);

ebitengine_Error
ebitengine_DrawTriangles(int dst, const int *srcs, int src_count, int shader,
                         const ebitengine_DstRegion *dst_regions,
                         int dst_region_count, int indexOffset,
                         ebitengine_Blend blend, const uint32_t *uniforms,
                         int uniform_count, int fill_rule);

ebitengine_Error ebitengine_NewShader(
    int *shader, const char *vertex_header, int vertex_header_size,
    const char *vertex_text, int vertex_text_size, const char *pixel_header,
    int pixel_header_size, const char *pixel_text, int pixel_text_size);
void ebitengine_DisposeShader(int id);

#ifdef __cplusplus
} // extern "C"
#endif

#endif // EBITENGINE_INTERNAL_GRAPHICSDRIVER_PLAYSTATION5_GRAPHICS_PLAYSTATION5_H
