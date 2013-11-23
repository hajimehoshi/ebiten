// -*- objc -*-

#include <stdlib.h>

#import "ebiten_controller.h"
#import "ebiten_window.h"

static EbitenWindow* currentWindow = 0;

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

void Start(size_t width, size_t height, size_t scale, const char* title) {
  NSSize size = NSMakeSize(width * scale, height * scale);
  EbitenWindow* window = [[EbitenWindow alloc]
                            initWithSize:size];
  [window setTitle: [[NSString alloc] initWithUTF8String:title]];
  EbitenController* controller = [[EbitenController alloc]
                                    initWithWindow:window];
  NSApplication* app = [NSApplication sharedApplication];
  [app setActivationPolicy:NSApplicationActivationPolicyRegular];
  [app setDelegate:controller];
  [app finishLaunching];
  [app activateIgnoringOtherApps:YES];

  currentWindow = window;

  PollEvents();

  [window initializeGLContext];
}

void BeginDrawing(void) {
  [currentWindow beginDrawing];
}

void EndDrawing(void) {
  [currentWindow endDrawing];
}
