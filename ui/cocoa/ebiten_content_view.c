// -*- objc -*-

#include "ebiten_content_view.h"
#include "input.h"

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
  [self interpretKeyEvents:[NSArray arrayWithObject:theEvent]];
}

- (void)insertText:(id)aString {
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

- (void)moveDown:(id)sender {
  
}

- (void)moveLeft:(id)sender {
}

- (void)moveRight:(id)sender {
}

- (void)moveUp:(id)sender {
}

@end
