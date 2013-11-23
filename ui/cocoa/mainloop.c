// -*- objc -*-

#include <stdlib.h>

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

void* CreateWindow(size_t width, size_t height, const char* title) {
  NSSize size = NSMakeSize(width, height);
  EbitenWindow* window = [[EbitenWindow alloc]
                            initWithSize:size];
  [window setTitle: [[NSString alloc] initWithUTF8String:title]];
  [window makeKeyAndOrderFront:nil];
  [window initializeGLContext];
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
  [(EbitenWindow*)window beginDrawing];
}

void EndDrawing(void* window) {
  [(EbitenWindow*)window endDrawing];
}
