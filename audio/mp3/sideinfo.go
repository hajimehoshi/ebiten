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

func readAudioL3() error {
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
		return fmt.Errorf("mp3: framesize = %d\n", framesize)
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
	if err := getSideinfo(sideinfo_size); err != nil {
		return err
	}
	/* Parse audio data */
	/* Pointer to where we should start reading main data */
	theMPEG1SideInfo.main_data_begin = getSideBits(9)
	/* Get private bits. Not used for anything. */
	if C.g_frame_header.mode == C.mpeg1_mode_single_channel {
		theMPEG1SideInfo.private_bits = getSideBits(5)
	} else {
		theMPEG1SideInfo.private_bits = getSideBits(3)
	}
	/* Get scale factor selection information */
	for ch := 0; ch < nch; ch++ {
		for scfsi_band := 0; scfsi_band < 4; scfsi_band++ {
			theMPEG1SideInfo.scfsi[ch][scfsi_band] = getSideBits(1)
		}
	}
	/* Get the rest of the side information */
	for gr := 0; gr < 2; gr++ {
		for ch := 0; ch < nch; ch++ {
			theMPEG1SideInfo.part2_3_length[gr][ch] = getSideBits(12)
			theMPEG1SideInfo.big_values[gr][ch] = getSideBits(9)
			theMPEG1SideInfo.global_gain[gr][ch] = getSideBits(8)
			theMPEG1SideInfo.scalefac_compress[gr][ch] = getSideBits(4)
			theMPEG1SideInfo.win_switch_flag[gr][ch] = getSideBits(1)
			if theMPEG1SideInfo.win_switch_flag[gr][ch] == 1 {
				theMPEG1SideInfo.block_type[gr][ch] = getSideBits(2)
				theMPEG1SideInfo.mixed_block_flag[gr][ch] = getSideBits(1)
				for region := 0; region < 2; region++ {
					theMPEG1SideInfo.table_select[gr][ch][region] = getSideBits(5)
				}
				for window := 0; window < 3; window++ {
					theMPEG1SideInfo.subblock_gain[gr][ch][window] = getSideBits(3)
				}
				if (theMPEG1SideInfo.block_type[gr][ch] == 2) && (theMPEG1SideInfo.mixed_block_flag[gr][ch] == 0) {
					theMPEG1SideInfo.region0_count[gr][ch] = 8 /* Implicit */
				} else {
					theMPEG1SideInfo.region0_count[gr][ch] = 7 /* Implicit */
				}
				/* The standard is wrong on this!!! */ /* Implicit */
				theMPEG1SideInfo.region1_count[gr][ch] = 20 - theMPEG1SideInfo.region0_count[gr][ch]
			} else {
				for region := 0; region < 3; region++ {
					theMPEG1SideInfo.table_select[gr][ch][region] = getSideBits(5)
				}
				theMPEG1SideInfo.region0_count[gr][ch] = getSideBits(4)
				theMPEG1SideInfo.region1_count[gr][ch] = getSideBits(3)
				theMPEG1SideInfo.block_type[gr][ch] = 0 /* Implicit */
			}
			theMPEG1SideInfo.preflag[gr][ch] = getSideBits(1)
			theMPEG1SideInfo.scalefac_scale[gr][ch] = getSideBits(1)
			theMPEG1SideInfo.count1table_select[gr][ch] = getSideBits(1)
		}
	}
	return nil
}

// A sideInfo is a bit reservoir for side info
type sideInfo struct {
	vec []int
	idx int // Index into the current byte(0-7)
}

var theSideInfo sideInfo

func getSideinfo(size int) error {
	buf := make([]int, size)
	n := 0
	var err error
	for n < size && err == nil {
		nn, err2 := getBytes(buf[n:])
		n += nn
		err = err2
	}
	if n < size {
		if err == io.EOF {
			return fmt.Errorf("mp3: unexpected EOF at getSideinfo")
		}
		return fmt.Errorf("mp3: couldn't read sideinfo %d bytes at pos %d: %v",
			size, getFilepos(), err)
	}
	theSideInfo.vec = buf[:n]
	theSideInfo.idx = 0
	return nil
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
