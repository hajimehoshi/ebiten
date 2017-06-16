// Copyright 2017 The Ebiten Authors
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

// +build !js

package mp3

// #include "pdmp3.h"
//
// extern t_mpeg1_header    g_frame_header;
// extern t_mpeg1_side_info g_side_info;
import "C"

import (
	"fmt"
	"io"
)

var g_mpeg1_bitrates = [3][15]int{
	{ /* Layer 1 */
		0, 32000, 64000, 96000, 128000, 160000, 192000, 224000,
		256000, 288000, 320000, 352000, 384000, 416000, 448000,
	},
	{ /* Layer 2 */
		0, 32000, 48000, 56000, 64000, 80000, 96000, 112000,
		128000, 160000, 192000, 224000, 256000, 320000, 384000,
	},
	{ /* Layer 3 */
		0, 32000, 40000, 48000, 56000, 64000, 80000, 96000,
		112000, 128000, 160000, 192000, 224000, 256000, 320000,
	},
}

var g_sampling_frequency = [3]int{44100, 48000, 32000}

//export Read_Audio_L3
func Read_Audio_L3() C.int {
	nch := 2
	if C.g_frame_header.mode == C.mpeg1_mode_single_channel {
		nch = 1
	}
	/* Calculate header audio data size */
	framesize := (144*
		g_mpeg1_bitrates[C.g_frame_header.layer-1][C.g_frame_header.bitrate_index])/
		g_sampling_frequency[C.g_frame_header.sampling_frequency] +
		int(C.g_frame_header.padding_bit)
	if framesize > 2000 {
		g_error = fmt.Errorf("mp3: framesize = %d\n", framesize)
		return C.ERROR
	}
	/* Sideinfo is 17 bytes for one channel and 32 bytes for two */
	sideinfo_size := 32
	if nch == 1 {
		sideinfo_size = 17
	}
	/* Main data size is the rest of the frame,including ancillary data */
	main_data_size := framesize - sideinfo_size - 4 /* sync+header */
	/* CRC is 2 bytes */
	if C.g_frame_header.protection_bit == 0 {
		main_data_size -= 2
	}
	/* Read sideinfo from bitstream into buffer used by getSideBits() */
	getSideinfo(sideinfo_size)
	if Get_Filepos() == eof {
		return C.ERROR
	}
	/* Parse audio data */
	/* Pointer to where we should start reading main data */
	C.g_side_info.main_data_begin = C.uint(getSideBits(9))
	/* Get private bits. Not used for anything. */
	if C.g_frame_header.mode == C.mpeg1_mode_single_channel {
		C.g_side_info.private_bits = C.uint(getSideBits(5))
	} else {
		C.g_side_info.private_bits = C.uint(getSideBits(3))
	}
	/* Get scale factor selection information */
	for ch := 0; ch < nch; ch++ {
		for scfsi_band := 0; scfsi_band < 4; scfsi_band++ {
			C.g_side_info.scfsi[ch][scfsi_band] = C.uint(getSideBits(1))
		}
	}
	/* Get the rest of the side information */
	for gr := 0; gr < 2; gr++ {
		for ch := 0; ch < nch; ch++ {
			C.g_side_info.part2_3_length[gr][ch] = C.uint(getSideBits(12))
			C.g_side_info.big_values[gr][ch] = C.uint(getSideBits(9))
			C.g_side_info.global_gain[gr][ch] = C.uint(getSideBits(8))
			C.g_side_info.scalefac_compress[gr][ch] = C.uint(getSideBits(4))
			C.g_side_info.win_switch_flag[gr][ch] = C.uint(getSideBits(1))
			if C.g_side_info.win_switch_flag[gr][ch] == 1 {
				C.g_side_info.block_type[gr][ch] = C.uint(getSideBits(2))
				C.g_side_info.mixed_block_flag[gr][ch] = C.uint(getSideBits(1))
				for region := 0; region < 2; region++ {
					C.g_side_info.table_select[gr][ch][region] = C.uint(getSideBits(5))
				}
				for window := 0; window < 3; window++ {
					C.g_side_info.subblock_gain[gr][ch][window] = C.uint(getSideBits(3))
				}
				if (C.g_side_info.block_type[gr][ch] == 2) && (C.g_side_info.mixed_block_flag[gr][ch] == 0) {
					C.g_side_info.region0_count[gr][ch] = 8 /* Implicit */
				} else {
					C.g_side_info.region0_count[gr][ch] = 7 /* Implicit */
				}
				/* The standard is wrong on this!!! */ /* Implicit */
				C.g_side_info.region1_count[gr][ch] = 20 - C.g_side_info.region0_count[gr][ch]
			} else {
				for region := 0; region < 3; region++ {
					C.g_side_info.table_select[gr][ch][region] = C.uint(getSideBits(5))
				}
				C.g_side_info.region0_count[gr][ch] = C.uint(getSideBits(4))
				C.g_side_info.region1_count[gr][ch] = C.uint(getSideBits(3))
				C.g_side_info.block_type[gr][ch] = 0 /* Implicit */
			}
			C.g_side_info.preflag[gr][ch] = C.uint(getSideBits(1))
			C.g_side_info.scalefac_scale[gr][ch] = C.uint(getSideBits(1))
			C.g_side_info.count1table_select[gr][ch] = C.uint(getSideBits(1))
		}
	}
	return C.OK
}

// A sideInfo is a bit reservoir for side info
type sideInfo struct {
	vec []int
	idx int // Index into the current byte(0-7)
}

var theSideInfo sideInfo

func getSideinfo(size int) {
	buf := make([]int, size)
	n, err := getBytes(buf)
	if err != nil && err != io.EOF {
		g_error = fmt.Errorf("mp3: couldn't read sideinfo %d bytes at pos %d: %v",
			size, Get_Filepos(), err)
		return
	}
	theSideInfo.vec = buf[:n]
	theSideInfo.idx = 0
}

func getSideBits(num int) int {
	// Form a word of the next four bytes
	// TODO: endianness?
	b := make([]int, 4)
	for i := range b {
		if len(theSideInfo.vec) > i {
			b[i] = theSideInfo.vec[i]
		}
	}
	tmp := (uint32(b[0]) << 24) | (uint32(b[1]) << 16) | (uint32(b[2]) << 8) | (uint32(b[3]) << 0)
	// Remove bits already used
	tmp = tmp << uint(theSideInfo.idx)
	// Remove bits after the desired bits
	tmp = tmp >> (32 - uint(num))
	// Update pointers
	theSideInfo.vec = theSideInfo.vec[(theSideInfo.idx+int(num))>>3:]
	theSideInfo.idx = (theSideInfo.idx + int(num)) & 0x07
	return int(tmp)
}
