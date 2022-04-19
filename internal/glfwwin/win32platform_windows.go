// Copyright 2002-2006 Marcus Geelnard
// Copyright 2006-2019 Camilla Löwy
// Copyright 2022 The Ebiten Authors
//
// This software is provided 'as-is', without any express or implied
// warranty. In no event will the authors be held liable for any damages
// arising from the use of this software.
//
// Permission is granted to anyone to use this software for any purpose,
// including commercial applications, and to alter it and redistribute it
// freely, subject to the following restrictions:
//
// 1. The origin of this software must not be misrepresented; you must not
//    claim that you wrote the original software. If you use this software
//    in a product, an acknowledgment in the product documentation would
//    be appreciated but is not required.
//
// 2. Altered source versions must be plainly marked as such, and must not
//    be misrepresented as being the original software.
//
// 3. This notice may not be removed or altered from any source
//    distribution.

package glfwwin

const (
	_GLFW_WNDCLASSNAME = "GLFW30"
)

func _IsWindowsXPOrGreater() bool {
	return isWindowsVersionOrGreaterWin32(uint16(_HIBYTE(_WIN32_WINNT_WINXP)), uint16(_LOBYTE(_WIN32_WINNT_WINXP)), 0)
}

func _IsWindowsVistaOrGreater() bool {
	return isWindowsVersionOrGreaterWin32(uint16(_HIBYTE(_WIN32_WINNT_VISTA)), uint16(_LOBYTE(_WIN32_WINNT_VISTA)), 0)
}

func _IsWindows7OrGreater() bool {
	return isWindowsVersionOrGreaterWin32(uint16(_HIBYTE(_WIN32_WINNT_WIN7)), uint16(_LOBYTE(_WIN32_WINNT_WIN7)), 0)
}

func _IsWindows8OrGreater() bool {
	return isWindowsVersionOrGreaterWin32(uint16(_HIBYTE(_WIN32_WINNT_WIN8)), uint16(_LOBYTE(_WIN32_WINNT_WIN8)), 0)
}

func _IsWindows8Point1OrGreater() bool {
	return isWindowsVersionOrGreaterWin32(uint16(_HIBYTE(_WIN32_WINNT_WINBLUE)), uint16(_LOBYTE(_WIN32_WINNT_WINBLUE)), 0)
}

func isWindows10AnniversaryUpdateOrGreaterWin32() bool {
	return isWindows10BuildOrGreaterWin32(14393)
}

func isWindows10CreatorsUpdateOrGreaterWin32() bool {
	return isWindows10BuildOrGreaterWin32(15063)
}
