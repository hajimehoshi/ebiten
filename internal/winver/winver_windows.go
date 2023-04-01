// Copyright 2023 The Ebitengine Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package winver

import (
	"unsafe"
)

func isWindowsVersionOrGreater(major, minor, sp uint16) bool {
	osvi := _OSVERSIONINFOEXW{
		dwMajorVersion:    uint32(major),
		dwMinorVersion:    uint32(minor),
		wServicePackMajor: sp,
	}
	osvi.dwOSVersionInfoSize = uint32(unsafe.Sizeof(osvi))
	var mask uint32 = _VER_MAJORVERSION | _VER_MINORVERSION | _VER_SERVICEPACKMAJOR
	cond := _VerSetConditionMask(0, _VER_MAJORVERSION, _VER_GREATER_EQUAL)
	cond = _VerSetConditionMask(cond, _VER_MINORVERSION, _VER_GREATER_EQUAL)
	cond = _VerSetConditionMask(cond, _VER_SERVICEPACKMAJOR, _VER_GREATER_EQUAL)

	// HACK: Use RtlVerifyVersionInfo instead of VerifyVersionInfoW as the
	//       latter lies unless the user knew to embed a non-default manifest
	//       announcing support for Windows 10 via supportedOS GUID
	return _RtlVerifyVersionInfo(&osvi, mask, cond) == 0
}

func isWindows10BuildOrGreater(build uint16) bool {
	osvi := _OSVERSIONINFOEXW{
		dwMajorVersion: 10,
		dwMinorVersion: 0,
		dwBuildNumber:  uint32(build),
	}
	osvi.dwOSVersionInfoSize = uint32(unsafe.Sizeof(osvi))
	var mask uint32 = _VER_MAJORVERSION | _VER_MINORVERSION | _VER_BUILDNUMBER
	cond := _VerSetConditionMask(0, _VER_MAJORVERSION, _VER_GREATER_EQUAL)
	cond = _VerSetConditionMask(cond, _VER_MINORVERSION, _VER_GREATER_EQUAL)
	cond = _VerSetConditionMask(cond, _VER_BUILDNUMBER, _VER_GREATER_EQUAL)

	// HACK: Use RtlVerifyVersionInfo instead of VerifyVersionInfoW as the
	//       latter lies unless the user knew to embed a non-default manifest
	//       announcing support for Windows 10 via supportedOS GUID
	return _RtlVerifyVersionInfo(&osvi, mask, cond) == 0
}

func IsWindowsXPOrGreater() bool {
	return isWindowsVersionOrGreater(uint16(_HIBYTE(_WIN32_WINNT_WINXP)), uint16(_LOBYTE(_WIN32_WINNT_WINXP)), 0)
}

func IsWindowsVistaOrGreater() bool {
	return isWindowsVersionOrGreater(uint16(_HIBYTE(_WIN32_WINNT_VISTA)), uint16(_LOBYTE(_WIN32_WINNT_VISTA)), 0)
}

func IsWindows7OrGreater() bool {
	return isWindowsVersionOrGreater(uint16(_HIBYTE(_WIN32_WINNT_WIN7)), uint16(_LOBYTE(_WIN32_WINNT_WIN7)), 0)
}

func IsWindows8OrGreater() bool {
	return isWindowsVersionOrGreater(uint16(_HIBYTE(_WIN32_WINNT_WIN8)), uint16(_LOBYTE(_WIN32_WINNT_WIN8)), 0)
}

func IsWindows8Point1OrGreater() bool {
	return isWindowsVersionOrGreater(uint16(_HIBYTE(_WIN32_WINNT_WINBLUE)), uint16(_LOBYTE(_WIN32_WINNT_WINBLUE)), 0)
}

func IsWindows10OrGreater() bool {
	return isWindows10BuildOrGreater(0)
}

func IsWindows10AnniversaryUpdateOrGreater() bool {
	return isWindows10BuildOrGreater(14393)
}

func IsWindows10CreatorsUpdateOrGreater() bool {
	return isWindows10BuildOrGreater(15063)
}
