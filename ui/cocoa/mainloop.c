// -*- objc -*-

#include <stdlib.h>
#include <OpenGL/gl.h>

#import "ebiten_controller.h"
#import "ebiten_window.h"

static NSOpenGLContext* glContext_;

void StartApplication() {
  EbitenController* controller = [[EbitenController alloc] init];
  NSApplication* app = [NSApplication sharedApplication];
  [app setActivationPolicy:NSApplicationActivationPolicyRegular];
  [app setDelegate:controller];
  [app finishLaunching];
  [app activateIgnoringOtherApps:YES];
}

void* CreateGLContext() {
  NSOpenGLPixelFormatAttribute attributes[] = {
    NSOpenGLPFAWindow,
    NSOpenGLPFADoubleBuffer,
    NSOpenGLPFAAccelerated,
    NSOpenGLPFADepthSize, 32,
    0,
  };
  NSOpenGLPixelFormat* format = [[NSOpenGLPixelFormat alloc]
                                  initWithAttributes:attributes];
  NSOpenGLContext* glContext = [[NSOpenGLContext alloc] initWithFormat:format
                                                          shareContext:nil];
  [format release];
  return glContext;
}

void* CreateWindow(size_t width, size_t height, const char* title) {
  NSSize size = NSMakeSize(width, height);
  EbitenWindow* window = [[EbitenWindow alloc]
                            initWithSize:size];
  [window setTitle: [[NSString alloc] initWithUTF8String:title]];
  [window makeKeyAndOrderFront:nil];
  glContext_ = CreateGLContext();
  [glContext_ makeCurrentContext];
  return window;
}

void PollEvents(void) {
  for (;;) {
    NSEvent* event = [NSApp nextEventMatchingMask:NSAnyEventMask
                                        untilDate:[NSDate distantPast]
                                           inMode:NSDefaultRunLoopMode
                                          dequeue:YES];
    if (event == nil) {
      break;
    }
    [NSApp sendEvent:event];
  }
}

void BeginDrawing(void* window) {
  [glContext_ setView:[(EbitenWindow*)window contentView]];
  glClear(GL_COLOR_BUFFER_BIT);
}

void EndDrawing(void* window) {
  [glContext_ flushBuffer];
}
