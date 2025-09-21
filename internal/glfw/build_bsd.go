// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2012 The glfw3-go Authors
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

//go:build freebsd || netbsd || openbsd

package glfw

// #cgo pkg-config: x11 xau xcb xdmcp
// #cgo CFLAGS: -D_GLFW_HAS_DLOPEN -D_GLFW_X11 -D_GLFW_HAS_GLXGETPROCADDRESSARB
// #cgo LDFLAGS: -lm
import "C"
