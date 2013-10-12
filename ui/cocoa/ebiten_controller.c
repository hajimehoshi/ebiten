// -*- objc -*-

#import "ebiten_controller.h"

@implementation EbitenController {
@private
  NSWindow* window_;
}

- (id)initWithWindow:(NSWindow*)window {
  self = [super init];
  if (self != nil) {
    self->window_ = window;
  }
  return self;
}

- (void)initMenu {
  NSString* processName = [[NSProcessInfo processInfo] processName];

  NSMenu* menuBar = [NSMenu new];
  NSMenuItem* rootMenu = [NSMenuItem new];
  [menuBar addItem:rootMenu];

  NSMenu* appMenu = [NSMenu new];
  [appMenu addItemWithTitle:[@"Quit " stringByAppendingString:processName]
                     action:@selector(performClose:)
              keyEquivalent:@"q"];

  [rootMenu setSubmenu:appMenu];
  [NSApp setMainMenu: menuBar];
}

- (void)applicationDidFinishLaunching:(NSNotification*)aNotification {
  (void)aNotification;
  NSWindow* window = self->window_;
  assert(window);
  [window makeKeyAndOrderFront:nil];
  [self initMenu];
}

- (BOOL)applicationShouldTerminateAfterLastWindowClosed:
  (NSApplication*)theApplication {
  (void)theApplication;
  return YES;
}

@end
