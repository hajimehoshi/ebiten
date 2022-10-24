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

// Package glconst provides constants to the implementations of Go bindings to
// OpenGL and OpenGL ES, in packages gl and gles respectively.
package glconst

// Constants shared between gl AND gles

const (
	ALWAYS               = 0x0207
	ARRAY_BUFFER         = 0x8892
	BLEND                = 0x0BE2
	CLAMP_TO_EDGE        = 0x812F
	COLOR_ATTACHMENT0    = 0x8CE0
	COMPILE_STATUS       = 0x8B81
	DYNAMIC_DRAW         = 0x88E8
	ELEMENT_ARRAY_BUFFER = 0x8893
	FALSE                = 0
	FLOAT                = 0x1406
	FRAGMENT_SHADER      = 0x8B30
	FRAMEBUFFER          = 0x8D40
	FRAMEBUFFER_BINDING  = 0x8CA6
	FRAMEBUFFER_COMPLETE = 0x8CD5
	INFO_LOG_LENGTH      = 0x8B84
	INVERT               = 0x150A
	KEEP                 = 0x1E00
	LINK_STATUS          = 0x8B82
	MAX_TEXTURE_SIZE     = 0x0D33
	NEAREST              = 0x2600
	NO_ERROR             = 0
	NOTEQUAL             = 0x0205
	PIXEL_PACK_BUFFER    = 0x88EB
	PIXEL_UNPACK_BUFFER  = 0x88EC
	READ_WRITE           = 0x88BA
	RENDERBUFFER         = 0x8D41
	RGBA                 = 0x1908
	SCISSOR_TEST         = 0x0C11
	SHORT                = 0x1402
	STENCIL_ATTACHMENT   = 0x8D20
	STENCIL_BUFFER_BIT   = 0x0400
	STENCIL_TEST         = 0x0B90
	STREAM_DRAW          = 0x88E0
	TEXTURE0             = 0x84C0
	TEXTURE_2D           = 0x0DE1
	TEXTURE_MAG_FILTER   = 0x2800
	TEXTURE_MIN_FILTER   = 0x2801
	TEXTURE_WRAP_S       = 0x2802
	TEXTURE_WRAP_T       = 0x2803
	TRIANGLES            = 0x0004
	TRUE                 = 1
	UNPACK_ALIGNMENT     = 0x0CF5
	UNSIGNED_BYTE        = 0x1401
	UNSIGNED_SHORT       = 0x1403
	VERTEX_SHADER        = 0x8B31
	WRITE_ONLY           = 0x88B9
)

// Constants unique to gl

const (
	DEPTH24_STENCIL8 = 0x88F0
)

// Constants unique to gles

const (
	HIGH_FLOAT     = 0x8DF2
	STENCIL_INDEX8 = 0x8D48
)
