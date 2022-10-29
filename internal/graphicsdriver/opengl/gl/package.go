// SPDX-License-Identifier: MIT

// Copyright (c) 2010 Khronos Group.
// This material may be distributed subject to the terms and conditions
// set forth in the Open Publication License, v 1.0, 8 June 1999.
// http://opencontent.org/openpub/.
//
// Copyright (c) 1991-2006 Silicon Graphics, Inc.
// This document is licensed under the SGI Free Software B License.
// For details, see http://oss.sgi.com/projects/FreeB.

//go:build !js
// +build !js

// Package gl implements Go bindings to OpenGL.
package gl

// Init initializes the OpenGL bindings by loading the function pointers (for
// each OpenGL function) from the active OpenGL context.
//
// It must be called under the presence of an active OpenGL context, e.g.,
// always after calling window.MakeContextCurrent() and always before calling
// any OpenGL functions exported by this package.
//
// On Windows, Init loads pointers that are context-specific (and hence you
// must re-init if switching between OpenGL contexts, although not calling Init
// again after switching between OpenGL contexts may work if the contexts belong
// to the same graphics driver/device).
//
// On macOS and the other POSIX systems, the behavior is different, but code
// written compatible with the Windows behavior is compatible with macOS and the
// other POSIX systems. That is, always Init under an active OpenGL context, and
// always re-init after switching graphics contexts.
//
// For information about caveats of Init, you should read the "Platform Specific
// Function Retrieval" section of https://www.opengl.org/wiki/Load_OpenGL_Functions.
func Init() error {
	return InitWithProcAddrFunc(getProcAddress)
}
