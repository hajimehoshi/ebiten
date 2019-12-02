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

#include "ca.h"
#import <QuartzCore/QuartzCore.h>

void *MakeMetalLayer() {
  CAMetalLayer *layer = [[CAMetalLayer alloc] init];
  // TODO: Expose a function to set color space.
  // TODO: Enable colorspace on iOS: this will be available as of iOS 13.0.
#if !TARGET_OS_IPHONE
  CGColorSpaceRef colorspace =
      CGColorSpaceCreateWithName(kCGColorSpaceDisplayP3);
  layer.colorspace = colorspace;
  CGColorSpaceRelease(colorspace);
#endif
  return layer;
}

uint16_t MetalLayer_PixelFormat(void *metalLayer) {
  return ((CAMetalLayer *)metalLayer).pixelFormat;
}

void MetalLayer_SetDevice(void *metalLayer, void *device) {
  ((CAMetalLayer *)metalLayer).device = (id<MTLDevice>)device;
}

void MetalLayer_SetOpaque(void *metalLayer, unsigned char opaque) {
  ((CAMetalLayer *)metalLayer).opaque = (BOOL)opaque;
}

const char *MetalLayer_SetPixelFormat(void *metalLayer, uint16_t pixelFormat) {
  @try {
    ((CAMetalLayer *)metalLayer).pixelFormat = (MTLPixelFormat)pixelFormat;
  } @catch (NSException *exception) {
    return exception.reason.UTF8String;
  }
  return NULL;
}

const char *MetalLayer_SetMaximumDrawableCount(void *metalLayer,
                                               uint_t maximumDrawableCount) {
  // @available syntax is not available for old Xcode (#781)
  //
  // If possible, we'd want to write the guard like:
  //
  //     if (@available(macOS 10.13.2, *)) { ...

  @try {
    if ([(CAMetalLayer *)metalLayer
            respondsToSelector:@selector(setMaximumDrawableCount:)]) {
      [((CAMetalLayer *)metalLayer)
          setMaximumDrawableCount:(NSUInteger)maximumDrawableCount];
    }
  } @catch (NSException *exception) {
    return exception.reason.UTF8String;
  }
  return NULL;
}

void MetalLayer_SetDisplaySyncEnabled(void *metalLayer,
                                      uint8_t displaySyncEnabled) {
  // @available syntax is not available for old Xcode (#781)
  //
  // If possible, we'd want to write the guard like:
  //
  //     if (@available(macOS 10.13, *)) { ...

#if !TARGET_OS_IPHONE
  if ([(CAMetalLayer *)metalLayer
          respondsToSelector:@selector(setDisplaySyncEnabled:)]) {
    [((CAMetalLayer *)metalLayer) setDisplaySyncEnabled:displaySyncEnabled];
  }
#endif
}

void MetalLayer_SetDrawableSize(void *metalLayer, double width, double height) {
  ((CAMetalLayer *)metalLayer).drawableSize = (CGSize){width, height};
}

void *MetalLayer_NextDrawable(void *metalLayer) {
  return [(CAMetalLayer *)metalLayer nextDrawable];
}

void *MetalDrawable_Texture(void *metalDrawable) {
  return ((id<CAMetalDrawable>)metalDrawable).texture;
}
