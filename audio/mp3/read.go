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
// extern t_mpeg1_side_info g_side_info;
// extern t_mpeg1_header    g_frame_header;
import "C"

import (
	"fmt"
	"io"
)

func readCRC() error {
	buf := make([]int, 2)
	n := 0
	var err error
	for n < 2 && err == nil {
		nn, err2 := getBytes(buf[n:])
		n += nn
		err = err2
	}
	if err == io.EOF {
		if n < 2 {
			return fmt.Errorf("mp3: unexpected EOF at readCRC")
		}
		return nil
	}
	if err != nil {
		return err
	}
	return nil
}

func readFrame() error {
	if err := readHeader(); err != nil {
		return err
	}
	// Get CRC word if present
	if C.g_frame_header.protection_bit == 0 {
		if err := readCRC(); err != nil {
			return err
		}
	}

	if C.g_frame_header.layer != 3 {
		return fmt.Errorf("mp3: Only layer 3(!= %d) is supported!", C.g_frame_header.layer)
	}
	// Get side info
	if err := readAudioL3(); err != nil {
		return err
	}
	// If there's not enough main data in the bit reservoir,
	// signal to calling function so that decoding isn't done!
	// Get main data(scalefactors and Huffman coded frequency data)
	if err := readMainL3(); err != nil {
		return err
	}
	return nil
}

func isHeader(header uint32) bool {
	const C_SYNC = 0xffe00000
	if (header & C_SYNC) != C_SYNC {
		return false
	}
	// Bitrate must not be 15.
	if (header & (0xf << 12)) == 0xf<<12 {
		return false
	}
	// Sample Frequency must not be 3.
	if (header & (3 << 10)) == 3<<10 {
		return false
	}
	return true
}

func readHeader() error {
	/* Get the next four bytes from the bitstream */
	buf := make([]int, 4)
	n := 0
	var err error
	for n < 4 && err == nil {
		nn, err2 := getBytes(buf[n:])
		n += nn
		err = err2
	}
	if n < 4 {
		if err == io.EOF {
			return fmt.Errorf("mp3: unexpected EOF at readHeader")
		}
		return err
	}
	b1 := uint32(buf[0])
	b2 := uint32(buf[1])
	b3 := uint32(buf[2])
	b4 := uint32(buf[3])
	header := (b1 << 24) | (b2 << 16) | (b3 << 8) | (b4 << 0)
	for !isHeader(uint32(header)) {
		/* No,so scan the bitstream one byte at a time until we find it or EOF */
		/* Shift the values one byte to the left */
		b1 = b2
		b2 = b3
		b3 = b4
		/* Get one new byte from the bitstream */
		b, err := getByte()
		if err != nil {
			if err == io.EOF {
				return fmt.Errorf("mp3: unexpected EOF at readHeader")
			}
			return err
		}
		b4 = uint32(b)
		header = (b1 << 24) | (b2 << 16) | (b3 << 8) | (b4 << 0)
	} /* while... */
	/* If we get here we've found the sync word,and can decode the header
	 * which is in the low 20 bits of the 32-bit sync+header word. */
	/* Decode the header */
	C.g_frame_header.id = C.uint((header & 0x00180000) >> 19)
	C.g_frame_header.layer = C.t_mpeg1_layer((header & 0x00060000) >> 17)
	C.g_frame_header.protection_bit = C.uint((header & 0x00010000) >> 16)
	C.g_frame_header.bitrate_index = C.uint((header & 0x0000f000) >> 12)
	C.g_frame_header.sampling_frequency = C.uint((header & 0x00000c00) >> 10)
	C.g_frame_header.padding_bit = C.uint((header & 0x00000200) >> 9)
	C.g_frame_header.private_bit = C.uint((header & 0x00000100) >> 8)
	C.g_frame_header.mode = C.t_mpeg1_mode((header & 0x000000c0) >> 6)
	C.g_frame_header.mode_extension = C.uint((header & 0x00000030) >> 4)
	C.g_frame_header.copyright = C.uint((header & 0x00000008) >> 3)
	C.g_frame_header.original_or_copy = C.uint((header & 0x00000004) >> 2)
	C.g_frame_header.emphasis = C.uint((header & 0x00000003) >> 0)
	/* Check for invalid values and impossible combinations */
	if C.g_frame_header.id != 3 {
		return fmt.Errorf("mp3: ID must be 3\nHeader word is 0x%08x at file pos %d",
			header, getFilepos())
	}
	if C.g_frame_header.bitrate_index == 0 {
		return fmt.Errorf("mp3: Free bitrate format NIY!\nHeader word is 0x%08x at file pos %d",
			header, getFilepos())
	}
	if C.g_frame_header.bitrate_index == 15 {
		return fmt.Errorf("mp3: bitrate_index = 15 is invalid!\nHeader word is 0x%08x at file pos %d",
			header, getFilepos())
	}
	if C.g_frame_header.sampling_frequency == 3 {
		return fmt.Errorf("mp3: sampling_frequency = 3 is invalid! Header word is 0x%08x at file pos %d",
			header, getFilepos())
	}
	if C.g_frame_header.layer == 0 {
		return fmt.Errorf("mp3: layer = 0 is invalid! Header word is 0x%08x at file pos %d",
			header, getFilepos())
	}
	C.g_frame_header.layer = 4 - C.g_frame_header.layer
	return nil
}

func readHuffman(part_2_start, gr, ch int) error {
	/* Check that there is any data to decode. If not,zero the array. */
	if C.g_side_info.part2_3_length[gr][ch] == 0 {
		for is_pos := 0; is_pos < 576; is_pos++ {
			C.g_main_data.is[gr][ch][is_pos] = 0.0
		}
		return nil
	}
	/* Calculate bit_pos_end which is the index of the last bit for this part. */
	bit_pos_end := part_2_start + int(C.g_side_info.part2_3_length[gr][ch]) - 1
	/* Determine region boundaries */
	region_1_start := 0
	region_2_start := 0
	if (C.g_side_info.win_switch_flag[gr][ch] == 1) && (C.g_side_info.block_type[gr][ch] == 2) {
		region_1_start = 36  /* sfb[9/3]*3=36 */
		region_2_start = 576 /* No Region2 for short block case. */
	} else {
		sfreq := C.g_frame_header.sampling_frequency
		region_1_start =
			sfBandIndicesSet[sfreq].l[C.g_side_info.region0_count[gr][ch]+1]
		region_2_start =
			sfBandIndicesSet[sfreq].l[C.g_side_info.region0_count[gr][ch]+
				C.g_side_info.region1_count[gr][ch]+2]
	}
	/* Read big_values using tables according to region_x_start */
	for is_pos := 0; is_pos < int(C.g_side_info.big_values[gr][ch])*2; is_pos++ {
		table_num := 0
		if is_pos < region_1_start {
			table_num = int(C.g_side_info.table_select[gr][ch][0])
		} else if is_pos < region_2_start {
			table_num = int(C.g_side_info.table_select[gr][ch][1])
		} else {
			table_num = int(C.g_side_info.table_select[gr][ch][2])
		}
		/* Get next Huffman coded words */
		x, y, _, _, err := huffmanDecode(table_num)
		if err != nil {
			return err
		}
		/* In the big_values area there are two freq lines per Huffman word */
		C.g_main_data.is[gr][ch][is_pos] = C.float(x)
		is_pos++
		C.g_main_data.is[gr][ch][is_pos] = C.float(y)
	}
	/* Read small values until is_pos = 576 or we run out of huffman data */
	table_num := int(C.g_side_info.count1table_select[gr][ch]) + 32
	is_pos := int(C.g_side_info.big_values[gr][ch]) * 2
	for ; (is_pos <= 572) && (getMainPos() <= bit_pos_end); is_pos++ {
		/* Get next Huffman coded words */
		x, y, v, w, err := huffmanDecode(table_num)
		if err != nil {
			return err
		}
		C.g_main_data.is[gr][ch][is_pos] = C.float(v)
		is_pos++
		if is_pos >= 576 {
			break
		}
		C.g_main_data.is[gr][ch][is_pos] = C.float(w)
		is_pos++
		if is_pos >= 576 {
			break
		}
		C.g_main_data.is[gr][ch][is_pos] = C.float(x)
		is_pos++
		if is_pos >= 576 {
			break
		}
		C.g_main_data.is[gr][ch][is_pos] = C.float(y)
	}
	/* Check that we didn't read past the end of this section */
	if getMainPos() > (bit_pos_end + 1) {
		/* Remove last words read */
		is_pos -= 4
	}
	/* Setup count1 which is the index of the first sample in the rzero reg. */
	C.g_side_info.count1[gr][ch] = C.unsigned(is_pos)
	/* Zero out the last part if necessary */
	for ; /* is_pos comes from last for-loop */ is_pos < 576; is_pos++ {
		C.g_main_data.is[gr][ch][is_pos] = 0.0
	}
	/* Set the bitpos to point to the next part to read */
	setMainPos(bit_pos_end + 1)
	return nil
}
