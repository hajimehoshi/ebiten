// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2012 The glfw3-go Authors
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

//go:build freebsd || netbsd || openbsd

package cglfw

import "C"

// #cgo !wayland openbsd pkg-config: x11 xau xcb xdmcp
// #cgo wayland,!openbsd pkg-config: wayland-client wayland-cursor wayland-egl epoll-shim
// #cgo CFLAGS: -D_GLFW_HAS_DLOPEN
// #cgo !wayland openbsd CFLAGS: -D_GLFW_X11 -D_GLFW_HAS_GLXGETPROCADDRESSARB
// #cgo wayland,!openbsd CFLAGS: -D_GLFW_WAYLAND
// #cgo LDFLAGS: -lm
import "C"
