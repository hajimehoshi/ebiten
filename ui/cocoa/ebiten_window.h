// -*- objc -*-

#ifndef GO_EBITEN_UI_COCOA_EBITEN_WINDOW_H_
#define GO_EBITEN_UI_COCOA_EBITEN_WINDOW_H_

#import <Cocoa/Cocoa.h>

@interface EbitenWindow : NSWindow<NSWindowDelegate>

- (id)initWithSize:(NSSize)size;
- (void)alertDidEnd:(NSAlert*)alert
         returnCode:(NSInteger)returnCode
        contextInfo:(void*)contextInfo;
- (BOOL)windowShouldClose:(id)sender;
- (void)beginDrawing;
- (void)endDrawing;

@end

#endif
