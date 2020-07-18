// SPDX-License-Identifier: MIT

// +build !js

// Copyright (c) 2010 Khronos Group.
// This material may be distributed subject to the terms and conditions
// set forth in the Open Publication License, v 1.0, 8 June 1999.
// http://opencontent.org/openpub/.
//
// Copyright (c) 1991-2006 Silicon Graphics, Inc.
// This document is licensed under the SGI Free Software B License.
// For details, see http://oss.sgi.com/projects/FreeB.

// Package gl implements Go bindings to OpenGL.
package gl

const (
	VERTEX_SHADER        = 0x8B31
	FRAGMENT_SHADER      = 0x8B30
	ARRAY_BUFFER         = 0x8892
	ELEMENT_ARRAY_BUFFER = 0x8893
	DYNAMIC_DRAW         = 0x88E8
	STREAM_DRAW          = 0x88E0
	PIXEL_UNPACK_BUFFER  = 0x88EC
	SHORT                = 0x1402
	FLOAT                = 0x1406

	ZERO                = 0
	ONE                 = 1
	SRC_ALPHA           = 0x0302
	DST_ALPHA           = 0x0304
	ONE_MINUS_SRC_ALPHA = 0x0303
	ONE_MINUS_DST_ALPHA = 0x0305
	DST_COLOR           = 0x0306

	FALSE = 0
	TRUE  = 1

	BLEND                = 0x0BE2
	CLAMP_TO_EDGE        = 0x812F
	COLOR_ATTACHMENT0    = 0x8CE0
	COMPILE_STATUS       = 0x8B81
	FRAMEBUFFER          = 0x8D40
	FRAMEBUFFER_BINDING  = 0x8CA6
	FRAMEBUFFER_COMPLETE = 0x8CD5
	INFO_LOG_LENGTH      = 0x8B84
	LINK_STATUS          = 0x8B82
	MAX_TEXTURE_SIZE     = 0x0D33
	NEAREST              = 0x2600
	NO_ERROR             = 0
	READ_WRITE           = 0x88BA
	RGBA                 = 0x1908
	TEXTURE0             = 0x84C0
	TEXTURE_2D           = 0x0DE1
	TEXTURE_MAG_FILTER   = 0x2800
	TEXTURE_MIN_FILTER   = 0x2801
	TEXTURE_WRAP_S       = 0x2802
	TEXTURE_WRAP_T       = 0x2803
	TRIANGLES            = 0x0004
	UNPACK_ALIGNMENT     = 0x0CF5
	UNSIGNED_BYTE        = 0x1401
	UNSIGNED_SHORT       = 0x1403
	WRITE_ONLY           = 0x88B9
)

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
