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
	for ; (is_pos <= 572) && (int(Get_Main_Pos()) <= bit_pos_end); is_pos++ {
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
	if int(C.Get_Main_Pos()) > (bit_pos_end + 1) {
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
	C.Set_Main_Pos(C.unsigned(bit_pos_end) + 1)
	return nil
}
