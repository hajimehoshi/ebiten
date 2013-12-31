// -*- objc -*-

#ifndef GO_EBITEN_UI_COCOA_EBITEN_GAME_WINDOW_H_
#define GO_EBITEN_UI_COCOA_EBITEN_GAME_WINDOW_H_

#import <Cocoa/Cocoa.h>

@interface EbitenGameWindow : NSWindow<NSWindowDelegate>

- (id)initWithSize:(NSSize)size
         glContext:(NSOpenGLContext*)glContext;
- (NSOpenGLContext*)glContext;

@end

#endif
