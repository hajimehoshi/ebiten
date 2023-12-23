// Copyright 2022 The Ebitengine Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !playstation5

package gl

const (
	ALWAYS                = 0x0207
	ARRAY_BUFFER          = 0x8892
	BACK                  = 0x0405
	BLEND                 = 0x0BE2
	CLAMP_TO_EDGE         = 0x812F
	COLOR_ATTACHMENT0     = 0x8CE0
	COMPILE_STATUS        = 0x8B81
	DECR_WRAP             = 0x8508
	DEPTH24_STENCIL8      = 0x88F0
	DST_ALPHA             = 0x0304
	DST_COLOR             = 0x0306
	DYNAMIC_DRAW          = 0x88E8
	ELEMENT_ARRAY_BUFFER  = 0x8893
	FALSE                 = 0
	FLOAT                 = 0x1406
	FRAGMENT_SHADER       = 0x8B30
	FRAMEBUFFER           = 0x8D40
	FRAMEBUFFER_BINDING   = 0x8CA6
	FRAMEBUFFER_COMPLETE  = 0x8CD5
	FRONT                 = 0x0404
	FRONT_AND_BACK        = 0x0408
	FUNC_ADD              = 0x8006
	FUNC_REVERSE_SUBTRACT = 0x800b
	FUNC_SUBTRACT         = 0x800a
	HIGH_FLOAT            = 0x8DF2
	INCR_WRAP             = 0x8507
	INFO_LOG_LENGTH       = 0x8B84
	INVERT                = 0x150A
	KEEP                  = 0x1E00
	LINK_STATUS           = 0x8B82
	MAX                   = 0x8008
	MAX_TEXTURE_SIZE      = 0x0D33
	MIN                   = 0x8007
	NEAREST               = 0x2600
	NO_ERROR              = 0
	NOTEQUAL              = 0x0205
	ONE                   = 1
	ONE_MINUS_DST_ALPHA   = 0x0305
	ONE_MINUS_DST_COLOR   = 0x0307
	ONE_MINUS_SRC_ALPHA   = 0x0303
	ONE_MINUS_SRC_COLOR   = 0x0301
	PIXEL_PACK_BUFFER     = 0x88EB
	PIXEL_UNPACK_BUFFER   = 0x88EC
	READ_WRITE            = 0x88BA
	RENDERBUFFER          = 0x8D41
	RGBA                  = 0x1908
	SCISSOR_TEST          = 0x0C11
	SHORT                 = 0x1402
	SRC_ALPHA             = 0x0302
	SRC_ALPHA_SATURATE    = 0x0308
	SRC_COLOR             = 0x0300
	STENCIL_ATTACHMENT    = 0x8D20
	STENCIL_BUFFER_BIT    = 0x0400
	STENCIL_INDEX8        = 0x8D48
	STENCIL_TEST          = 0x0B90
	STREAM_DRAW           = 0x88E0
	TEXTURE0              = 0x84C0
	TEXTURE_2D            = 0x0DE1
	TEXTURE_MAG_FILTER    = 0x2800
	TEXTURE_MIN_FILTER    = 0x2801
	TEXTURE_WRAP_S        = 0x2802
	TEXTURE_WRAP_T        = 0x2803
	TRIANGLES             = 0x0004
	TRUE                  = 1
	UNPACK_ALIGNMENT      = 0x0CF5
	UNSIGNED_BYTE         = 0x1401
	UNSIGNED_INT          = 0x1405
	VERTEX_SHADER         = 0x8B31
	WRITE_ONLY            = 0x88B9
	ZERO                  = 0
)
