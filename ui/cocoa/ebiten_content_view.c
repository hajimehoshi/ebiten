// -*- objc -*-

#include "ebiten_content_view.h"
#include "input.h"

void ebiten_InputUpdated(void* nativeWindow, InputType inputType, int x, int y);

@implementation EbitenContentView {
}

- (BOOL)isFlipped {
  return YES;
}

- (void)mouseDown:(NSEvent*)theEvent {
  NSPoint location = [self convertPoint:[theEvent locationInWindow]
                               fromView:nil];
  int x = location.x;
  int y = location.y;
  ebiten_InputUpdated([self window], InputTypeMouseDown, x, y);
}

- (void)mouseUp:(NSEvent*)theEvent {
  (void)theEvent;
  NSPoint location = [self convertPoint:[theEvent locationInWindow]
                               fromView:nil];
  int x = location.x;
  int y = location.y;
  ebiten_InputUpdated([self window], InputTypeMouseUp, x, y);
}

- (void)mouseDragged:(NSEvent*)theEvent {
  NSPoint location = [self convertPoint:[theEvent locationInWindow]
                               fromView:nil];
  int x = location.x;
  int y = location.y;
  ebiten_InputUpdated([self window], InputTypeMouseDragged, x, y);
}

@end
