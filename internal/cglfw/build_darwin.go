// SPDX-License-Identifier: BSD-3-Clause
// SPDX-FileCopyrightText: 2012 The glfw3-go Authors
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

package cglfw

// #cgo CFLAGS: -D_GLFW_COCOA -Wno-deprecated-declarations
// #cgo LDFLAGS: -framework Cocoa -framework IOKit -framework CoreVideo
import "C"
