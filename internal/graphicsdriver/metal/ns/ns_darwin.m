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

#include "ns_darwin.h"
#import <Cocoa/Cocoa.h>

void *Window_ContentView(uintptr_t window) {
  return ((NSWindow *)window).contentView;
}

void View_SetLayer(void *view, void *layer) {
  ((NSView *)view).layer = (CALayer *)layer;
}

void View_SetWantsLayer(void *view, unsigned char wantsLayer) {
  ((NSView *)view).wantsLayer = (BOOL)wantsLayer;
}

uint8_t View_IsInFullScreenMode(void *view) {
  return ((NSView *)view).isInFullScreenMode;
}
