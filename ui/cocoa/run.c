// -*- objc -*-

#include <stdlib.h>

#import "ebiten_controller.h"
#import "ebiten_window.h"

void Run(size_t width, size_t height, size_t scale, const char* title) {
  @autoreleasepool {
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
    [app run];
  }
}
