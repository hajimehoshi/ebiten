// -*- objc -*-

#import "ebiten_controller.h"

@implementation EbitenController {
}

- (void)applicationDidFinishLaunching:(NSNotification*)aNotification {
  (void)aNotification;
  [[NSNotificationCenter defaultCenter] addObserver:self 
                                           selector:@selector(windowClosing:) 
                                               name:NSWindowWillCloseNotification 
                                             object:nil];
}

- (BOOL)applicationShouldTerminateAfterLastWindowClosed:
  (NSApplication*)theApplication {
  (void)theApplication;
  return YES;
}

- (void)windowClosing:(NSNotification*)aNotification {
  (void)aNotification;
  [NSApp terminate:nil];
}

@end
