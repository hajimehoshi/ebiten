// -*- objc -*-

#include <stdlib.h>
#include <OpenGL/gl.h>

#import "ebiten_game_window.h"

void initMenu(void) {
  NSString* processName = [[NSProcessInfo processInfo] processName];

  NSMenu* menuBar = [NSMenu new];
  [NSApp setMainMenu: menuBar];
  [menuBar release];

  NSMenuItem* rootMenuItem = [NSMenuItem new];
  [menuBar addItem:rootMenuItem];
  [rootMenuItem release];

  NSMenu* appMenu = [NSMenu new];
  [rootMenuItem setSubmenu:appMenu];
  [appMenu release];

  [appMenu addItemWithTitle:[@"Quit " stringByAppendingString:processName]
                     action:@selector(performClose:)
              keyEquivalent:@"q"];
}

void StartApplication(void) {
  NSApplication* app = [NSApplication sharedApplication];
  [app setActivationPolicy:NSApplicationActivationPolicyRegular];

  initMenu();

  [app finishLaunching];
}

NSOpenGLContext* CreateGLContext(NSOpenGLContext* sharedGLContext) {
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
                               shareContext:sharedGLContext];
  [format release];
  return glContext;
}

EbitenGameWindow* CreateGameWindow(size_t width, size_t height, const char* title, NSOpenGLContext* glContext) {
  NSSize size = NSMakeSize(width, height);
  EbitenGameWindow* window = [[EbitenGameWindow alloc]
                               initWithSize:size
                                  glContext:glContext];
  [glContext release];

  NSString* nsTitle = [[NSString alloc]
                      initWithUTF8String:title];
  [window setTitle: nsTitle];
  [nsTitle release];

  [window makeKeyAndOrderFront:nil];
  [glContext setView:[window contentView]];
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
  static BOOL initialBoot = YES;
  if (initialBoot) {
    [NSApp activateIgnoringOtherApps:YES];
    initialBoot = NO;
  }
}

void UseGLContext(NSOpenGLContext* glContext) {
  CGLContextObj cglContext = [glContext CGLContextObj];
  CGLLockContext(cglContext);
  [glContext makeCurrentContext];
}

void UnuseGLContext(void) {
  NSOpenGLContext* glContext = [NSOpenGLContext currentContext];
  [glContext flushBuffer];
  [NSOpenGLContext clearCurrentContext];
  CGLContextObj cglContext = [glContext CGLContextObj];
  CGLUnlockContext(cglContext);
}

NSOpenGLContext* GetGLContext(EbitenGameWindow* window) {
  return [window glContext];
}
