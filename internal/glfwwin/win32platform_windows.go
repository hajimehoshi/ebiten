// SPDX-License-Identifier: Zlib
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

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
