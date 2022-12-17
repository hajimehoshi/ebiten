package glfwwin

import (
	"fmt"
	cf "github.com/hajimehoshi/ebiten/v2/internal/corefoundation"
)

// Initialize OpenGL support
func initNSGL() error {
	if _glfw.context.framework != 0 {
		return nil
	}

	_glfw.context.framework = cf.CFBundleGetBundleWithIdentifier(cf.CFStringCreateWithCString(cf.KCFAllocatorDefault, []byte("com.apple.opengl\x00"), cf.KCFStringEncodingUTF8))

	if _glfw.context.framework == 0 {
		return fmt.Errorf("cocoa: failed to create application delegate")
	}
	return nil
}
