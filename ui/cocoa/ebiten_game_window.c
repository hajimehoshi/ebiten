// -*- objc -*-

#import "ebiten_game_window.h"

#import "ebiten_game_content_view.h"

@class NSOpenGLContext;

void ebiten_WindowClosed(void* nativeWindow);

@implementation EbitenGameWindow {
@private
  NSOpenGLContext* glContext_;
}

- (id)initWithSize:(NSSize)size
         glContext:(NSOpenGLContext*)glContext {
  self->glContext_ = glContext;
  [self->glContext_ retain];

  NSUInteger style = (NSTitledWindowMask | NSClosableWindowMask |
                      NSMiniaturizableWindowMask);
  NSRect windowRect =
    [NSWindow frameRectForContentRect:NSMakeRect(0, 0, size.width, size.height)
                            styleMask:style];
  NSScreen* screen = [[NSScreen screens] objectAtIndex:0];
  NSSize screenSize = [screen visibleFrame].size;
  NSRect contentRect = NSMakeRect(0, 0, size.width, size.height);
  self = [super initWithContentRect:contentRect
                          styleMask:style
                            backing:NSBackingStoreBuffered
                              defer:YES];
  if (self != nil) {
    [self center];
    [self setReleasedWhenClosed:YES];
    [self setDelegate:self];
    [self setDocumentEdited:YES];

    NSRect rect = NSMakeRect(0, 0, size.width, size.height);
    NSView* contentView = [[EbitenGameContentView alloc] initWithFrame:rect];
    [self setContentView:contentView];
    [contentView release];
  }

  return self;
}

- (void)dealloc {
  [self->glContext_ release];
  [super dealloc];
}

- (NSOpenGLContext*)glContext {
  return self->glContext_;
}

- (BOOL)windowShouldClose:(id)sender {
  if ([sender isDocumentEdited]) {
    // TODO: add the application's name
    NSAlert* alert = [NSAlert alertWithMessageText:@"Quit the game?"
                                     defaultButton:@"Quit"
                                   alternateButton:nil
                                       otherButton:@"Cancel"
                         informativeTextWithFormat:@""];
    SEL selector = @selector(alertDidEnd:returnCode:contextInfo:);
    [alert beginSheetModalForWindow:sender
                      modalDelegate:self
                     didEndSelector:selector
                        contextInfo:nil];
    [alert release];
  }
  return NO;
}

- (void)alertDidEnd:(NSAlert*)alert
         returnCode:(NSInteger)returnCode
        contextInfo:(void*)contextInfo {
  (void)alert;
  (void)contextInfo;
  if (returnCode == NSAlertDefaultReturn) {
    [self close];
    ebiten_WindowClosed(self);
  }
}

- (BOOL)canBecomeMainWindow {
  return YES;
}

@end
