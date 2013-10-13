// -*- objc -*-

#include "ebiten_opengl_view.h"

void ebiten_EbitenOpenGLView_Initialized(void);
void ebiten_EbitenOpenGLView_Updating(void);
void ebiten_EbitenOpenGLView_InputUpdated(int x, int y);

// Reference:
//   http://developer.apple.com/library/mac/#qa/qa1385/_index.html
//   http://www.alecjacobson.com/weblog/?p=2185

// TODO: Use NSViewController?

static CVReturn
EbitenDisplayLinkCallback(CVDisplayLinkRef displayLink,
                          CVTimeStamp const* now,
                          CVTimeStamp const* outputTime,
                          CVOptionFlags flagsIn,
                          CVOptionFlags* flagsOut,
                          void* displayLinkContext) {
  (void)displayLink;
  (void)now;
  (void)flagsIn;
  (void)flagsOut;
  @autoreleasepool {
    EbitenOpenGLView* view = (__bridge EbitenOpenGLView*)displayLinkContext;
    return [view getFrameForTime:outputTime];
  }
}

@implementation EbitenOpenGLView {
@private
  CVDisplayLinkRef displayLink_;
  size_t screenWidth_;
  size_t screenHeight_;
  size_t screenScale_;
}

- (void)dealloc {
  CVDisplayLinkRelease(self->displayLink_);
  // Do not call [super dealloc] because of ARC.
}

- (void)prepareOpenGL {
  [super prepareOpenGL];
  NSOpenGLContext* openGLContext = [self openGLContext];
  assert(openGLContext != nil);
  GLint swapInterval = 1;
  [openGLContext setValues:&swapInterval
              forParameter:NSOpenGLCPSwapInterval]; 
  CVDisplayLinkCreateWithActiveCGDisplays(&self->displayLink_);
  CVDisplayLinkSetOutputCallback(self->displayLink_,
                                   &EbitenDisplayLinkCallback,
                                   (__bridge void*)self);
  CGLContextObj cglContext = (CGLContextObj)[openGLContext CGLContextObj];
  CGLPixelFormatObj cglPixelFormat =
    (CGLPixelFormatObj)[[self pixelFormat] CGLPixelFormatObj];
  CVDisplayLinkSetCurrentCGDisplayFromOpenGLContext(self->displayLink_,
                                                      cglContext,
                                                      cglPixelFormat);
  CVDisplayLinkStart(self->displayLink_);

  ebiten_EbitenOpenGLView_Initialized();
}

- (CVReturn)getFrameForTime:(CVTimeStamp const*)outputTime {
  (void)outputTime;
  NSOpenGLContext* context = [self openGLContext];
  assert(context != nil);
  [context makeCurrentContext];
  {
    CGLLockContext((CGLContextObj)[context CGLContextObj]);
    ebiten_EbitenOpenGLView_Updating();
    [context flushBuffer];
    CGLUnlockContext((CGLContextObj)[context CGLContextObj]);
  }
  return kCVReturnSuccess;
}

- (BOOL)isFlipped {
  return YES;
}

- (void)mouseDown:(NSEvent*)theEvent {
  NSPoint location = [self convertPoint:[theEvent locationInWindow]
                               fromView:nil];
  int x = location.x / self->screenScale_;
  int y = location.y / self->screenScale_;
  if (x < 0) {
    x = 0;
  } else if (self->screenWidth_<= x) {
    x = self->screenWidth_ - 1;
  }
  if (y < 0) {
    y = 0;
  } else if (self->screenHeight_<= y) {
    y = self->screenHeight_ - 1;
  }
  ebiten_EbitenOpenGLView_InputUpdated(x, y);
}

- (void)mouseUp:(NSEvent*)theEvent {
  (void)theEvent;
  ebiten_EbitenOpenGLView_InputUpdated(-1, -1);
}

- (void)mouseDragged:(NSEvent*)theEvent {
  NSPoint location = [self convertPoint:[theEvent locationInWindow]
                               fromView:nil];
  int x = location.x / self->screenScale_;
  int y = location.y / self->screenScale_;
  if (x < 0) {
    x = 0;
  } else if (self->screenWidth_<= x) {
    x = self->screenWidth_ - 1;
  }
  if (y < 0) {
    y = 0;
  } else if (self->screenHeight_<= y) {
    y = self->screenHeight_ - 1;
  }
  ebiten_EbitenOpenGLView_InputUpdated(x, y);
}

- (void)setScreenWidth:(size_t)screenWidth
          screenHeight:(size_t)screenHeight
           screenScale:(size_t)screenScale {
  self->screenWidth_ = screenWidth;
  self->screenHeight_ = screenHeight;
  self->screenScale_ = screenScale;
}

@end
