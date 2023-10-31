// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2012 The glfw3-go Authors
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

//go:build darwin || freebsd || linux || netbsd || openbsd

package glfw

// #include <stdlib.h>
// #define GLFW_INCLUDE_NONE
// #include "glfw3_unix.h"
import "C"

import (
	"image"
	"image/draw"
)

func imageToGLFWImage(img image.Image) (glfwImg C.GLFWimage, free func()) {
	b := img.Bounds()
	if b.Dx() == 0 || b.Dy() == 0 {
		return C.GLFWimage{
			width:  C.int(b.Dx()),
			height: C.int(b.Dy()),
		}, func() {}
	}

	m := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(m, m.Bounds(), img, b.Min, draw.Src)
	pixels := m.Pix

	cpixels := C.CBytes(pixels)
	free = func() {
		C.free(cpixels)
	}

	return C.GLFWimage{
		width:  C.int(b.Dx()),
		height: C.int(b.Dy()),
		pixels: (*C.uchar)(cpixels),
	}, free
}
