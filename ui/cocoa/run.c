// -*- objc -*-

#include <stdlib.h>

#import "ebiten_controller.h"
#import "ebiten_opengl_view.h"
#import "ebiten_window.h"

static NSWindow* generateWindow(size_t width, size_t height, size_t scale, const char* title) {
  EbitenWindow* window = [[EbitenWindow alloc]
                           initWithSize:NSMakeSize(width * scale, height * scale)];
  assert(window != nil);

  NSRect const rect = NSMakeRect(0, 0, width * scale, height * scale);
  NSOpenGLPixelFormatAttribute const attributes[] = {
    NSOpenGLPFAWindow,
    NSOpenGLPFADoubleBuffer,
    NSOpenGLPFAAccelerated,
    NSOpenGLPFADepthSize, 32,
    0,
  };
  NSOpenGLPixelFormat* format = [[NSOpenGLPixelFormat alloc]
                                  initWithAttributes:attributes];
  EbitenOpenGLView* glView =
    [[EbitenOpenGLView alloc] initWithFrame:rect
                                pixelFormat:format];
  [window setContentView:glView];
  [window setTitle: [[NSString alloc] initWithUTF8String:title]];
  //[window makeFirstResponder:glView];

  return window;
}

void Run(size_t width, size_t height, size_t scale, const char* title) {
  @autoreleasepool {
    NSWindow* window = generateWindow(width, height, scale, title);
    EbitenController* controller = [[EbitenController alloc]
                                    initWithWindow:window];
    NSApplication* app = [NSApplication sharedApplication];
    [app setActivationPolicy:NSApplicationActivationPolicyRegular];
    [app setDelegate:controller];
    [app finishLaunching];
    [app activateIgnoringOtherApps:YES];
    [app run];
  }
}
