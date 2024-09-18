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

// The actual implementation will be provided by -overlay.

#include "graphics_playstation5.h"

extern "C" ebitengine_Error ebitengine_InitializeGraphics(void) { return {}; }

extern "C" ebitengine_Error ebitengine_NewImage(int *image, int width,
                                                int height) {
  return {};
}

extern "C" ebitengine_Error
ebitengine_NewScreenFramebufferImage(int *image, int width, int height) {
  return {};
}

extern "C" void ebitengine_DisposeImage(int id) {}

extern "C" void ebitengine_SetVertices(float *vertices, int vertexCount,
                                       uint32_t *indices, int indexCount) {}

extern "C" ebitengine_Error
ebitengine_DrawTriangles(int dst, int *srcs, int srcCount, int shader,
                         ebitengine_DstRegion *dstRegions, int dstRegionCount,
                         int indexOffset, ebitengine_Blend blend,
                         uint32_t *uniforms, int uniformCount, int fillRule) {
  return {};
}

extern "C" ebitengine_Error ebitengine_NewShader(int *shader,
                                                 const char *source) {
  return {};
}

extern "C" void ebitengine_DisposeShader(int id) {}
