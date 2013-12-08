// -*- objc -*-

#include <stdlib.h>
#include <OpenGL/gl.h>

#import "ebiten_controller.h"
#import "ebiten_window.h"

void initMenu(void) {
  NSString* processName = [[NSProcessInfo processInfo] processName];

  NSMenu* menuBar = [NSMenu new];
  [NSApp setMainMenu: menuBar];

  NSMenuItem* rootMenuItem = [NSMenuItem new];
  [menuBar addItem:rootMenuItem];

  NSMenu* appMenu = [NSMenu new];
  [rootMenuItem setSubmenu:appMenu];
  [appMenu addItemWithTitle:[@"Quit " stringByAppendingString:processName]
                     action:@selector(performClose:)
              keyEquivalent:@"q"];
}

void StartApplication(void) {
  EbitenController* controller = [[EbitenController alloc] init];
  NSApplication* app = [NSApplication sharedApplication];
  [app setActivationPolicy:NSApplicationActivationPolicyRegular];

  initMenu();

  [app setDelegate:controller];
  [app finishLaunching];
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

void* CreateWindow(size_t width, size_t height, const char* title, void* glContext_) {
  NSOpenGLContext* glContext = (NSOpenGLContext*)glContext_;

  NSSize size = NSMakeSize(width, height);
  EbitenWindow* window = [[EbitenWindow alloc]
                            initWithSize:size
                               glContext:glContext];
  [window setTitle: [[NSString alloc]
                      initWithUTF8String:title]];
  [window makeKeyAndOrderFront:nil];

  [(NSOpenGLContext*)glContext setView:[window contentView]];

  return window;
}

static BOOL initialBoot = YES;

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
  if (initialBoot) {
    [NSApp activateIgnoringOtherApps:YES];
    initialBoot = NO;
  }
}

void UseGLContext(void* glContextPtr) {
  NSOpenGLContext* glContext = (NSOpenGLContext*)glContextPtr;
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

void* GetGLContext(void* window) {
  return [(EbitenWindow*)window glContext];
}

void BeginDrawing(void* window) {
  // TODO: CGLLock
  [[(EbitenWindow*)window glContext] makeCurrentContext];
  glClear(GL_COLOR_BUFFER_BIT);
}

void EndDrawing(void* window) {
  [[(EbitenWindow*)window glContext] flushBuffer];
}
