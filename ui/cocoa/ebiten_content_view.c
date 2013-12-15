// -*- objc -*-

#include "ebiten_content_view.h"
#include "input.h"

void ebiten_KeyDown(void* nativeWindow, int keyCode);
void ebiten_KeyUp(void* nativeWindow, int keyCode);
void ebiten_MouseStateUpdated(void* nativeWindow, InputType inputType, int x, int y);

@implementation EbitenContentView {
}

- (BOOL)acceptsFirstResponder {
  return YES;
}

- (BOOL)isFlipped {
  return YES;
}

- (void)keyDown:(NSEvent*)theEvent {
  ebiten_KeyDown([self window], [theEvent keyCode]);
}

- (void)keyUp:(NSEvent*)theEvent {
  ebiten_KeyUp([self window], [theEvent keyCode]);
}

- (void)mouseDown:(NSEvent*)theEvent {
  NSPoint location = [self convertPoint:[theEvent locationInWindow]
                               fromView:nil];
  int x = location.x;
  int y = location.y;
  ebiten_MouseStateUpdated([self window], InputTypeMouseDown, x, y);
}

- (void)mouseUp:(NSEvent*)theEvent {
  (void)theEvent;
  NSPoint location = [self convertPoint:[theEvent locationInWindow]
                               fromView:nil];
  int x = location.x;
  int y = location.y;
  ebiten_MouseStateUpdated([self window], InputTypeMouseUp, x, y);
}

- (void)mouseDragged:(NSEvent*)theEvent {
  NSPoint location = [self convertPoint:[theEvent locationInWindow]
                               fromView:nil];
  int x = location.x;
  int y = location.y;
  ebiten_MouseStateUpdated([self window], InputTypeMouseDragged, x, y);
}

@end
