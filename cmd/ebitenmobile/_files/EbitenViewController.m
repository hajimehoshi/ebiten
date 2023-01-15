// Copyright 2022 The Ebitengine Authors
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

#import <TargetConditionals.h>

#import <stdint.h>
#import <UIKit/UIKit.h>
#import <GLKit/GLKit.h>

#import "Ebitenmobileview.objc.h"

@interface {{.PrefixUpper}}EbitenViewController : UIViewController<EbitenmobileviewRenderRequester>
@end

@implementation {{.PrefixUpper}}EbitenViewController {
  UIView*        metalView_;
  GLKView*       glkView_;
  bool           started_;
  bool           active_;
  bool           error_;
  CADisplayLink* displayLink_;
  bool           explicitRendering_;
  NSThread*      renderThread_;
}

- (UIView*)metalView {
  if (!metalView_) {
    metalView_ = [[UIView alloc] init];
    metalView_.multipleTouchEnabled = YES;
  }
  return metalView_;
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

  if (!started_) {
    @synchronized(self) {
      active_ = true;
    }
    started_ = true;
  }

  NSError* err = nil;
  BOOL isGL = NO;
  EbitenmobileviewIsGL(&isGL, &err);
  if (err != nil) {
    [self onErrorOnGameUpdate:err];
    @synchronized(self) {
      error_ = true;
    }
    return;
  }

  if (isGL) {
    self.glkView.delegate = (id<GLKViewDelegate>)(self);
    [self.view addSubview: self.glkView];
  } else {
    [self.view addSubview: self.metalView];
    EbitenmobileviewSetUIView((uintptr_t)(self.metalView), &err);
    if (err != nil) {
      [self onErrorOnGameUpdate:err];
      @synchronized(self) {
        error_ = true;
      }
      return;
    }
  }

  renderThread_ = [[NSThread alloc] initWithTarget:self
                                          selector:@selector(initRenderer)
                                            object:nil];
  [renderThread_ start];
}

- (void)initRenderer {
  NSError* err = nil;
  BOOL isGL = NO;
  EbitenmobileviewIsGL(&isGL, &err);
  if (err != nil) {
    [self performSelectorOnMainThread:@selector(onErrorOnGameUpdate:)
                           withObject:err
                        waitUntilDone:NO];
    @synchronized(self) {
      error_ = true;
    }
    return;
  }

  if (isGL) {
    EAGLContext *context = [[EAGLContext alloc] initWithAPI:kEAGLRenderingAPIOpenGLES2];
    [self glkView].context = context;

    [EAGLContext setCurrentContext:context];
  }

  displayLink_ = [CADisplayLink displayLinkWithTarget:self selector:@selector(drawFrame)];
  [displayLink_ addToRunLoop:[NSRunLoop currentRunLoop] forMode:NSDefaultRunLoopMode];
  EbitenmobileviewSetRenderRequester(self);

  // Run the loop. This will never return.
  [[NSRunLoop currentRunLoop] run];
}

- (void)viewWillLayoutSubviews {
  NSError* err = nil;
  BOOL isGL = NO;
  EbitenmobileviewIsGL(&isGL, &err);
  if (err != nil) {
    [self onErrorOnGameUpdate:err];
    @synchronized(self) {
      error_ = true;
    }
    return;
  }

  CGRect viewRect = [[self view] frame];
  if (isGL) {
    [[self glkView] setFrame:viewRect];
  } else {
    [[self metalView] setFrame:viewRect];
  }
}

- (void)viewDidLayoutSubviews {
  [super viewDidLayoutSubviews];
  CGRect viewRect = [[self view] frame];

  EbitenmobileviewLayout(viewRect.size.width, viewRect.size.height);
}

- (void)didReceiveMemoryWarning {
  [super didReceiveMemoryWarning];
  // Dispose of any resources that can be recreated.
  // TODO: Notify this to Go world?
}

- (void)drawFrame{
  @synchronized(self) {
    if (!active_) {
      return;
    }
  }

  NSError* err = nil;
  BOOL isGL = NO;
  EbitenmobileviewIsGL(&isGL, &err);
  if (err != nil) {
    [self performSelectorOnMainThread:@selector(onErrorOnGameUpdate:)
                           withObject:err
                        waitUntilDone:NO];
    @synchronized(self) {
      error_ = true;
    }
    return;
  }

  if (isGL) {
    dispatch_async(dispatch_get_main_queue(), ^{
      [[self glkView] setNeedsDisplay];
    });
  } else {
    [self updateEbiten];
  }

  @synchronized(self) {
    if (explicitRendering_) {
      [displayLink_ setPaused:YES];
    }
  }
}

- (void)glkView:(GLKView*)view drawInRect:(CGRect)rect {
  [self updateEbiten];
}

- (void)updateEbiten {
  @synchronized(self) {
    if (error_) {
      return;
    }
  }

  NSError* err = nil;
  EbitenmobileviewUpdate(&err);
  if (err != nil) {
    [self performSelectorOnMainThread:@selector(onErrorOnGameUpdate:)
                           withObject:err
                        waitUntilDone:NO];
    @synchronized(self) {
      error_ = true;
    }
  }
}

- (void)onErrorOnGameUpdate:(NSError*)err {
  NSLog(@"Error: %@", err);
}

- (void)updateTouches:(NSSet*)touches {
  NSError* err = nil;
  BOOL isGL = NO;
  EbitenmobileviewIsGL(&isGL, &err);
  if (err != nil) {
    [self onErrorOnGameUpdate:err];
    @synchronized(self) {
      error_ = true;
    }
    return;
  }

  for (UITouch* touch in touches) {
    if (isGL) {
      if (touch.view != [self glkView]) {
        continue;
      }
    } else {
      if (touch.view != [self metalView]) {
        continue;
      }
    }
    CGPoint location = [touch locationInView:touch.view];
    EbitenmobileviewUpdateTouchesOnIOS(touch.phase, (uintptr_t)touch, location.x, location.y);
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

- (void)suspendGame {
  NSAssert(started_, @"suspendGame must not be called before viewDidLoad is called");

  @synchronized(self) {
    active_ = false;
  }

  NSError* err = nil;
  EbitenmobileviewSuspend(&err);
  if (err != nil) {
    [self onErrorOnGameUpdate:err];
  }
}

- (void)resumeGame {
  NSAssert(started_, @"resumeGame must not be called before viewDidLoad is called");

  @synchronized(self) {
    active_ = true;
  }

  NSError* err = nil;
  EbitenmobileviewResume(&err);
  if (err != nil) {
    [self onErrorOnGameUpdate:err];
  }
}

- (void)setExplicitRenderingMode:(BOOL)explicitRendering {
  @synchronized(self) {
    explicitRendering_ = explicitRendering;
    if (explicitRendering_) {
      [displayLink_ setPaused:YES];
    }
  }
}

- (void)requestRenderIfNeeded {
  @synchronized(self) {
    if (explicitRendering_) {
      // Resume the callback temporarily.
      // This is paused again soon in drawFrame.
      [displayLink_ setPaused:NO];
    }
  }
}

@end
