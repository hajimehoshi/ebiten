// -*- objc -*-

#include <stdlib.h>
#include <OpenGL/gl.h>

#import "ebiten_controller.h"
#import "ebiten_window.h"

void StartApplication() {
  EbitenController* controller = [[EbitenController alloc] init];
  NSApplication* app = [NSApplication sharedApplication];
  [app setActivationPolicy:NSApplicationActivationPolicyRegular];
  [app setDelegate:controller];
  [app finishLaunching];
  [app activateIgnoringOtherApps:YES];
}

void* CreateGLContext(void* sharedGLContext) {
  NSOpenGLPixelFormatAttribute attributes[] = {
    NSOpenGLPFAWindow,
    NSOpenGLPFADoubleBuffer,
    NSOpenGLPFAAccelerated,
    NSOpenGLPFADepthSize, 32,
    0,
  };
  NSOpenGLPixelFormat* format = [[NSOpenGLPixelFormat alloc]
                                  initWithAttributes:attributes];
  NSOpenGLContext* glContext =
    [[NSOpenGLContext alloc] initWithFormat:format
                               shareContext:(NSOpenGLContext*)sharedGLContext];
  [format release];
  return glContext;
}

void SetCurrentGLContext(void* glContext) {
  [(NSOpenGLContext*)glContext makeCurrentContext];
}

// This takes the ownership of glContext.
void* CreateWindow(size_t width, size_t height, const char* title, void* glContext) {
  NSSize size = NSMakeSize(width, height);
  EbitenWindow* window = [[EbitenWindow alloc]
                            initWithSize:size
                               glContext:(NSOpenGLContext*)glContext];
  [window setTitle: [[NSString alloc] initWithUTF8String:title]];
  [window makeKeyAndOrderFront:nil];

  [(NSOpenGLContext*)glContext setView:[window contentView]];

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
  [[(EbitenWindow*)window glContext] makeCurrentContext];
  glClear(GL_COLOR_BUFFER_BIT);
}

void EndDrawing(void* window) {
  [[(EbitenWindow*)window glContext] flushBuffer];
}
