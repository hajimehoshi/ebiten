// +build !windows

package ui

import "github.com/go-gl/glfw/v3.2/glfw"

func hideConsoleWindowOnWindows(*glfw.Window) {}
