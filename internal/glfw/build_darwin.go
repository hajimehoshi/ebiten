// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2012 The glfw3-go Authors
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

package glfw

// #cgo CFLAGS: -D_GLFW_COCOA -Wno-deprecated-declarations
// #cgo LDFLAGS: -framework Cocoa -framework IOKit -framework CoreVideo
import "C"
