// -*- objc -*-

#ifndef GO_EBITEN_UI_COCOA_EBITEN_OPENGL_VIEW_H_
#define GO_EBITEN_UI_COCOA_EBITEN_OPENGL_VIEW_H_

#import <Cocoa/Cocoa.h>
#import <QuartzCore/QuartzCore.h>

@interface EbitenOpenGLView : NSOpenGLView

- (CVReturn)getFrameForTime:(CVTimeStamp const*)outputTime;
- (void)setScreenWidth:(size_t)screenWidth
          screenHeight:(size_t)screenHeight
           screenScale:(size_t)screenScale;

@end

#endif
