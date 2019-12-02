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

#include <stdint.h>

typedef unsigned long uint_t;

void *MakeMetalLayer();

uint16_t MetalLayer_PixelFormat(void *metalLayer);
void MetalLayer_SetDevice(void *metalLayer, void *device);
void MetalLayer_SetOpaque(void *metalLayer, unsigned char opaque);
const char *MetalLayer_SetPixelFormat(void *metalLayer, uint16_t pixelFormat);
const char *MetalLayer_SetMaximumDrawableCount(void *metalLayer,
                                               uint_t maximumDrawableCount);
void MetalLayer_SetDisplaySyncEnabled(void *metalLayer,
                                      uint8_t displaySyncEnabled);
void MetalLayer_SetDrawableSize(void *metalLayer, double width, double height);
void *MetalLayer_NextDrawable(void *metalLayer);

void *MetalDrawable_Texture(void *drawable);
