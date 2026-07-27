// Copyright 2026 The Ebitengine Authors
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

//go:build ((darwin && !ios) || freebsd || (linux && !android) || netbsd || windows) && !nintendosdk && !playstation5

// Package windowsystem reports whether the system has a window system.
package windowsystem

import (
	"os"
	"runtime"
)

// Available reports whether a window system is available.
func Available() bool {
	switch runtime.GOOS {
	case "freebsd", "linux", "netbsd":
		// A device without a display server, like a handheld running its
		// applications on a framebuffer device, sets neither variable.
		return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
	}
	return true
}
