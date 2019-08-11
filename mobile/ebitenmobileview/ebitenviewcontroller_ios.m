// Copyright 2019 The Ebiten Authors
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

// +build ios

#import <UIKit/UIKit.h>
#import <GLKit/GLkit.h>

#import "ebitenviewcontroller_ios.h"

#include "_cgo_export.h"

@implementation EbitenViewController {
  GLKView* glkView_;
}

- (GLKView*)glkView {
  if (!glkView_) {
    glkView_ = [[GLKView alloc] init];
    glkView_.multipleTouchEnabled = YES;
  }
  return glkView_;
}

- (void)viewDidLoad {
  [super viewDidLoad];

  self.glkView.delegate = (id<GLKViewDelegate>)(self);
  [self.view addSubview: self.glkView];

  EAGLContext *context = [[EAGLContext alloc] initWithAPI:kEAGLRenderingAPIOpenGLES2];
  [self glkView].context = context;
	
  [EAGLContext setCurrentContext:context];
	
  CADisplayLink *displayLink = [CADisplayLink displayLinkWithTarget:self selector:@selector(drawFrame)];
  [displayLink addToRunLoop:[NSRunLoop currentRunLoop] forMode:NSDefaultRunLoopMode];
}

- (void)viewDidLayoutSubviews {
  [super viewDidLayoutSubviews];
  CGRect viewRect = [[self view] frame];

  int x, y, width, height;
  ebitenLayout(viewRect.size.width, viewRect.size.height, &x, &y, &width, &height);

  CGRect glkViewRect = CGRectMake(x, y, width, height);
  [[self glkView] setFrame:glkViewRect];
}

- (void)didReceiveMemoryWarning {
  [super didReceiveMemoryWarning];
  // Dispose of any resources that can be recreated.
  // TODO: Notify this to Go world?
}

- (void)drawFrame{
  [[self glkView] setNeedsDisplay];
}

- (void)glkView:(GLKView*)view drawInRect:(CGRect)rect {
  const char* err = ebitenUpdate();
  if (err != nil) {
    NSLog(@"Error: %s", err);
  }
}

- (void)updateTouches:(NSSet*)touches {
  for (UITouch* touch in touches) {
    if (touch.view != [self glkView]) {
      continue;
    }
    CGPoint location = [touch locationInView:touch.view];
    ebitenUpdateTouchesOnIOS(touch.phase, (uintptr_t)touch, location.x, location.y);
  }
}

- (void)touchesBegan:(NSSet*)touches withEvent:(UIEvent*)event {
  [self updateTouches:touches];
}

- (void)touchesMoved:(NSSet*)touches withEvent:(UIEvent*)event {
  [self updateTouches:touches];
}

- (void)touchesEnded:(NSSet*)touches withEvent:(UIEvent*)event {
  [self updateTouches:touches];
}

- (void)touchesCancelled:(NSSet*)touches withEvent:(UIEvent*)event {
  [self updateTouches:touches];
}

@end
