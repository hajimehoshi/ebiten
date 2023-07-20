// SPDX-License-Identifier: BSD-3-Clause
// SPDX-FileCopyrightText: 2012 The glfw3-go Authors
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

//go:build freebsd || linux || netbsd || openbsd

package glfw

/*
#ifdef _GLFW_WAYLAND
	#include "glfw/src/wl_init.c"
	#include "glfw/src/wl_monitor.c"
	#include "glfw/src/wl_window.c"
	#include "glfw/src/wayland-idle-inhibit-unstable-v1-client-protocol.c"
	#include "glfw/src/wayland-pointer-constraints-unstable-v1-client-protocol.c"
	#include "glfw/src/wayland-relative-pointer-unstable-v1-client-protocol.c"
	#include "glfw/src/wayland-viewporter-client-protocol.c"
	#include "glfw/src/wayland-xdg-decoration-unstable-v1-client-protocol.c"
	#include "glfw/src/wayland-xdg-shell-client-protocol.c"
#endif
*/
import "C"
