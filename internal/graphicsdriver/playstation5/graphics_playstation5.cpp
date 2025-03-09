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

// The actual implementation will be provided by github.com/hajimehoshi/uwagaki.

#include "graphics_playstation5.h"

extern "C" ebitengine_Error ebitengine_InitializeGraphics(void) { return {}; }

extern "C" ebitengine_Error ebitengine_NewImage(int *image, int width,
                                                int height) {
  return {};
}

extern "C" void ebitengine_ReadPixels(int image, uint8_t *pixels,
                                      ebitengine_Region region) {}

extern "C" ebitengine_Error ebitengine_FlushReadPixels(int image) { return {}; }

extern "C" void ebitengine_WritePixels(int image, const uint8_t *pixels,
                                       ebitengine_Region region) {}

extern "C" ebitengine_Error ebitengine_FlushWritePixels(int image) {
  return {};
}

extern "C" ebitengine_Error
ebitengine_NewScreenFramebufferImage(int *image, int width, int height) {
  return {};
}

extern "C" void ebitengine_DisposeImage(int id) {}

extern "C" void ebitengine_Begin() {}

extern "C" void ebitengine_End(int present) {}

extern "C" void ebitengine_SetVertices(const float *vertices, int vertex_count,
                                       const uint32_t *indices,
                                       int index_count) {}

extern "C" ebitengine_Error
ebitengine_DrawTriangles(int dst, const int *srcs, int src_count, int shader,
                         const ebitengine_DstRegion *dst_regions,
                         int dst_region_count, int index_offset,
                         ebitengine_Blend blend, const uint32_t *uniforms,
                         int uniform_count, int fill_rule) {
  return {};
}

extern "C" ebitengine_Error ebitengine_NewShader(
    int *shader, const char *vertex_header, int vertex_header_size,
    const char *vertex_text, int vertex_text_size, const char *pixel_header,
    int pixel_header_size, const char *pixel_text, int pixel_text_size) {
  return {};
}

extern "C" void ebitengine_DisposeShader(int id) {}
