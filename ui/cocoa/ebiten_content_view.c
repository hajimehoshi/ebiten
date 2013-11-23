// -*- objc -*-

#include "ebiten_content_view.h"
#include "input.h"

void ebiten_EbitenOpenGLView_InputUpdated(InputType inputType, int x, int y);

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
  ebiten_EbitenOpenGLView_InputUpdated(InputTypeMouseDown, x, y);
}

- (void)mouseUp:(NSEvent*)theEvent {
  (void)theEvent;
  NSPoint location = [self convertPoint:[theEvent locationInWindow]
                               fromView:nil];
  int x = location.x;
  int y = location.y;
  ebiten_EbitenOpenGLView_InputUpdated(InputTypeMouseUp, x, y);
}

- (void)mouseDragged:(NSEvent*)theEvent {
  NSPoint location = [self convertPoint:[theEvent locationInWindow]
                               fromView:nil];
  int x = location.x;
  int y = location.y;
  ebiten_EbitenOpenGLView_InputUpdated(InputTypeMouseDragged, x, y);
}

@end
