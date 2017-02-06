package ui

import "syscall"

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	user32           = syscall.NewLazyDLL("user32.dll")
	getConsoleWindow = kernel32.NewProc("GetConsoleWindow")
	showWindowAsync  = user32.NewProc("ShowWindowAsync")
)

func hideConsoleWindowOnWindows() {
	windowHandle, _, _ := getConsoleWindow.Call()
	const SW_HIDE = 0
	showWindowAsync.Call(windowHandle, SW_HIDE)
}
