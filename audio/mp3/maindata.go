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
// extern t_mpeg1_main_data g_main_data;
// extern t_mpeg1_header    g_frame_header;
// extern t_mpeg1_side_info g_side_info;
import "C"

import (
	"fmt"
)

var mpeg1_scalefac_sizes = [16][2]int{
	{0, 0}, {0, 1}, {0, 2}, {0, 3}, {3, 0}, {1, 1}, {1, 2}, {1, 3},
	{2, 1}, {2, 2}, {2, 3}, {3, 1}, {3, 2}, {3, 3}, {4, 2}, {4, 3},
}

//export Read_Main_L3
func Read_Main_L3() C.int {
	/* Number of channels(1 for mono and 2 for stereo) */
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
		g_error = fmt.Errorf("mp3: framesize = %d", framesize)
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
	/* Assemble main data buffer with data from this frame and the previous
	 * two frames. main_data_begin indicates how many bytes from previous
	 * frames that should be used. This buffer is later accessed by the
	 * getMainBits function in the same way as the side info is.
	 */
	if getMainData(main_data_size, int(C.g_side_info.main_data_begin)) != C.OK {
		return C.ERROR /* This could be due to not enough data in reservoir */
	}
	for gr := 0; gr < 2; gr++ {
		for ch := 0; ch < nch; ch++ {
			part_2_start := int(Get_Main_Pos())
			/* Number of bits in the bitstream for the bands */
			slen1 := mpeg1_scalefac_sizes[C.g_side_info.scalefac_compress[gr][ch]][0]
			slen2 := mpeg1_scalefac_sizes[C.g_side_info.scalefac_compress[gr][ch]][1]
			if (C.g_side_info.win_switch_flag[gr][ch] != 0) && (C.g_side_info.block_type[gr][ch] == 2) {
				if C.g_side_info.mixed_block_flag[gr][ch] != 0 {
					for sfb := 0; sfb < 8; sfb++ {
						C.g_main_data.scalefac_l[gr][ch][sfb] = C.uint(getMainBits(slen1))
					}
					for sfb := 3; sfb < 12; sfb++ {
						/*slen1 for band 3-5,slen2 for 6-11*/
						nbits := slen2
						if sfb < 6 {
							nbits = slen1
						}
						for win := 0; win < 3; win++ {
							C.g_main_data.scalefac_s[gr][ch][sfb][win] = C.uint(getMainBits(nbits))
						}
					}
				} else {
					for sfb := 0; sfb < 12; sfb++ {
						/*slen1 for band 3-5,slen2 for 6-11*/
						nbits := slen2
						if sfb < 6 {
							nbits = slen1
						}
						for win := 0; win < 3; win++ {
							C.g_main_data.scalefac_s[gr][ch][sfb][win] = C.uint(getMainBits(nbits))
						}
					}
				}
			} else { /* block_type == 0 if winswitch == 0 */
				/* Scale factor bands 0-5 */
				if (C.g_side_info.scfsi[ch][0] == 0) || (gr == 0) {
					for sfb := 0; sfb < 6; sfb++ {
						C.g_main_data.scalefac_l[gr][ch][sfb] = C.uint(getMainBits(slen1))
					}
				} else if (C.g_side_info.scfsi[ch][0] == 1) && (gr == 1) {
					/* Copy scalefactors from granule 0 to granule 1 */
					for sfb := 0; sfb < 6; sfb++ {
						C.g_main_data.scalefac_l[1][ch][sfb] = C.g_main_data.scalefac_l[0][ch][sfb]
					}
				}
				/* Scale factor bands 6-10 */
				if (C.g_side_info.scfsi[ch][1] == 0) || (gr == 0) {
					for sfb := 6; sfb < 11; sfb++ {
						C.g_main_data.scalefac_l[gr][ch][sfb] = C.uint(getMainBits(slen1))
					}
				} else if (C.g_side_info.scfsi[ch][1] == 1) && (gr == 1) {
					/* Copy scalefactors from granule 0 to granule 1 */
					for sfb := 6; sfb < 11; sfb++ {
						C.g_main_data.scalefac_l[1][ch][sfb] = C.g_main_data.scalefac_l[0][ch][sfb]
					}
				}
				/* Scale factor bands 11-15 */
				if (C.g_side_info.scfsi[ch][2] == 0) || (gr == 0) {
					for sfb := 11; sfb < 16; sfb++ {
						C.g_main_data.scalefac_l[gr][ch][sfb] = C.uint(getMainBits(slen2))
					}
				} else if (C.g_side_info.scfsi[ch][2] == 1) && (gr == 1) {
					/* Copy scalefactors from granule 0 to granule 1 */
					for sfb := 11; sfb < 16; sfb++ {
						C.g_main_data.scalefac_l[1][ch][sfb] = C.uint(C.g_main_data.scalefac_l[0][ch][sfb])
					}
				}
				/* Scale factor bands 16-20 */
				if (C.g_side_info.scfsi[ch][3] == 0) || (gr == 0) {
					for sfb := 16; sfb < 21; sfb++ {
						C.g_main_data.scalefac_l[gr][ch][sfb] = C.uint(getMainBits(slen2))
					}
				} else if (C.g_side_info.scfsi[ch][3] == 1) && (gr == 1) {
					/* Copy scalefactors from granule 0 to granule 1 */
					for sfb := 16; sfb < 21; sfb++ {
						C.g_main_data.scalefac_l[1][ch][sfb] = C.g_main_data.scalefac_l[0][ch][sfb]
					}
				}
			}
			/* Read Huffman coded data. Skip stuffing bits. */
			if err := readHuffman(part_2_start, gr, ch); err != nil {
				g_error = err
				return C.ERROR
			}
		}
	}
	/* The ancillary data is stored here,but we ignore it. */
	return C.OK
}

type mainDataBytes struct {
	// Large static data
	vec [2 * 1024]int
	// Pointer into the reservoir
	ptr []int
	// Index into the current byte(0-7)
	idx int
	// Number of bytes in reservoir(0-1024)
	top int

	pos int
}

var theMainDataBytes mainDataBytes

func getMainData(size int, begin int) int {
	if size > 1500 {
		g_error = fmt.Errorf("size = %d", size)
	}
	/* Check that there's data available from previous frames if needed */
	if int(begin) > theMainDataBytes.top {
		// No,there is not,so we skip decoding this frame,but we have to
		// read the main_data bits from the bitstream in case they are needed
		// for decoding the next frame.
		b, err := getBytes(size)
		if err != nil {
			g_error = err
			return C.ERROR
		}
		copy(theMainDataBytes.vec[theMainDataBytes.top:], b)
		/* Set up pointers */
		theMainDataBytes.ptr = theMainDataBytes.vec[0:]
		theMainDataBytes.pos = 0
		theMainDataBytes.idx = 0
		theMainDataBytes.top += size
		return C.ERROR
	}
	/* Copy data from previous frames */
	for i := 0; i < begin; i++ {
		theMainDataBytes.vec[i] = theMainDataBytes.vec[theMainDataBytes.top-begin+i]
	}
	/* Read the main_data from file */
	b, err := getBytes(int(size))
	if err != nil {
		g_error = err
		return C.ERROR
	}
	copy(theMainDataBytes.vec[begin:], b)
	/* Set up pointers */
	theMainDataBytes.ptr = theMainDataBytes.vec[0:]
	theMainDataBytes.pos = 0
	theMainDataBytes.idx = 0
	theMainDataBytes.top = begin + size
	return C.OK
}

func getMainBit() int {
	tmp := uint(theMainDataBytes.ptr[0]) >> (7 - uint(theMainDataBytes.idx))
	tmp &= 0x01
	theMainDataBytes.ptr = theMainDataBytes.ptr[(theMainDataBytes.idx+1)>>3:]
	theMainDataBytes.pos += (theMainDataBytes.idx + 1) >> 3
	theMainDataBytes.idx = (theMainDataBytes.idx + 1) & 0x07
	return int(tmp)
}

func getMainBits(num int) int {
	if num == 0 {
		return 0
	}
	/* Form a word of the next four bytes */
	b := make([]int, 4)
	for i := range b {
		if len(theMainDataBytes.ptr) > i {
			b[i] = theMainDataBytes.ptr[i]
		}
	}
	tmp := (uint32(b[0]) << 24) | (uint32(b[1]) << 16) | (uint32(b[2]) << 8) | (uint32(b[3]) << 0)

	/* Remove bits already used */
	tmp = tmp << uint(theMainDataBytes.idx)

	/* Remove bits after the desired bits */
	tmp = tmp >> (32 - uint(num))

	/* Update pointers */
	theMainDataBytes.ptr = theMainDataBytes.ptr[(theMainDataBytes.idx+int(num))>>3:]
	theMainDataBytes.pos += (theMainDataBytes.idx + num) >> 3
	theMainDataBytes.idx = (theMainDataBytes.idx + num) & 0x07

	/* Done */
	return int(tmp)
}

//export Get_Main_Pos
func Get_Main_Pos() C.unsigned {
	pos := theMainDataBytes.pos
	pos *= 8                    /* Multiply by 8 to get number of bits */
	pos += theMainDataBytes.idx /* Add current bit index */
	return C.unsigned(pos)
}

//export Set_Main_Pos
func Set_Main_Pos(bit_pos C.unsigned) C.int {
	theMainDataBytes.ptr = theMainDataBytes.vec[bit_pos>>3:]
	theMainDataBytes.pos = int(bit_pos) >> 3
	theMainDataBytes.idx = int(bit_pos) & 0x7
	return C.OK
}
