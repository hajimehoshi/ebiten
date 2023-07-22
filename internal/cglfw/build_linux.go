// SPDX-License-Identifier: BSD-3-Clause
// SPDX-FileCopyrightText: 2012 The glfw3-go Authors
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

package cglfw

// #cgo !wayland CFLAGS: -D_GLFW_X11
// #cgo wayland CFLAGS: -D_GLFW_WAYLAND
// #cgo !wayland LDFLAGS: -lX11 -lXrandr -lXxf86vm -lXi -lXcursor -lm -lXinerama -ldl -lrt
// #cgo wayland LDFLAGS: -lwayland-client -lwayland-cursor -lwayland-egl -lxkbcommon -lm -ldl -lrt
import "C"
