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

func (f *frame) readNextFrame() (*frame, error) {
	nf := &frame{
		prev: f,
	}
	if nf.prev != nil {
		nf.store = nf.prev.store
		nf.v_vec = nf.prev.v_vec
	}
	h, err := readHeader()
	if err != nil {
		return nil, err
	}
	nf.header = h
	// Get CRC word if present
	if nf.header.protection_bit == 0 {
		if err := readCRC(); err != nil {
			return nil, err
		}
	}
	if nf.header.layer != mpeg1Layer3 {
		return nil, fmt.Errorf("mp3: only layer3 (want %d; got %d) is supported!", mpeg1Layer3, nf.header.layer)
	}
	// Get side info
	s, err := readSideInfo(nf.header)
	if err != nil {
		return nil, err
	}
	nf.sideInfo = s
	// If there's not enough main data in the bit reservoir,
	// signal to calling function so that decoding isn't done!
	// Get main data(scalefactors and Huffman coded frequency data)
	if err := nf.readMainL3(); err != nil {
		return nil, err
	}
	return nf, nil
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

func readHeader() (*mpeg1FrameHeader, error) {
	// Get the next four bytes from the bitstream
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
			if n == 0 {
				return nil, eof
			}
			return nil, fmt.Errorf("mp3: unexpected EOF at readHeader")
		}
		return nil, err
	}
	b1 := uint32(buf[0])
	b2 := uint32(buf[1])
	b3 := uint32(buf[2])
	b4 := uint32(buf[3])
	header := (b1 << 24) | (b2 << 16) | (b3 << 8) | (b4 << 0)
	for !isHeader(uint32(header)) {
		// No,so scan the bitstream one byte at a time until we find it or EOF
		// Shift the values one byte to the left
		b1 = b2
		b2 = b3
		b3 = b4
		// Get one new byte from the bitstream
		b, err := getByte()
		if err != nil {
			if err == io.EOF {
				return nil, fmt.Errorf("mp3: unexpected EOF at readHeader")
			}
			return nil, err
		}
		b4 = uint32(b)
		header = (b1 << 24) | (b2 << 16) | (b3 << 8) | (b4 << 0)
	}
	// If we get here we've found the sync word,and can decode the header
	// which is in the low 20 bits of the 32-bit sync+header word.
	// Decode the header
	h := &mpeg1FrameHeader{}
	h.id = int((header & 0x00180000) >> 19)
	h.layer = mpeg1Layer((header & 0x00060000) >> 17)
	h.protection_bit = int((header & 0x00010000) >> 16)
	h.bitrate_index = int((header & 0x0000f000) >> 12)
	h.sampling_frequency = int((header & 0x00000c00) >> 10)
	h.padding_bit = int((header & 0x00000200) >> 9)
	h.private_bit = int((header & 0x00000100) >> 8)
	h.mode = mpeg1Mode((header & 0x000000c0) >> 6)
	h.mode_extension = int((header & 0x00000030) >> 4)
	h.copyright = int((header & 0x00000008) >> 3)
	h.original_or_copy = int((header & 0x00000004) >> 2)
	h.emphasis = int((header & 0x00000003) >> 0)
	// Check for invalid values and impossible combinations
	if h.id != 3 {
		return nil, fmt.Errorf("mp3: ID must be 3. Header word is 0x%08x at file pos %d",
			header, getFilepos())
	}
	if h.bitrate_index == 0 {
		return nil, fmt.Errorf("mp3: Free bitrate format NIY! Header word is 0x%08x at file pos %d",
			header, getFilepos())
	}
	if h.bitrate_index == 15 {
		return nil, fmt.Errorf("mp3: bitrate_index = 15 is invalid! Header word is 0x%08x at file pos %d",
			header, getFilepos())
	}
	if h.sampling_frequency == 3 {
		return nil, fmt.Errorf("mp3: sampling_frequency = 3 is invalid! Header word is 0x%08x at file pos %d",
			header, getFilepos())
	}
	if h.layer == mpeg1LayerReserved {
		return nil, fmt.Errorf("mp3: layer = %d is invalid! Header word is 0x%08x at file pos %d",
			mpeg1LayerReserved, header, getFilepos())
	}
	return h, nil
}

func (f *frame) readHuffman(part_2_start, gr, ch int) error {
	// Check that there is any data to decode. If not,zero the array.
	if f.sideInfo.part2_3_length[gr][ch] == 0 {
		for is_pos := 0; is_pos < 576; is_pos++ {
			f.mainData.is[gr][ch][is_pos] = 0.0
		}
		return nil
	}
	// Calculate bit_pos_end which is the index of the last bit for this part.
	bit_pos_end := part_2_start + f.sideInfo.part2_3_length[gr][ch] - 1
	// Determine region boundaries
	region_1_start := 0
	region_2_start := 0
	if (f.sideInfo.win_switch_flag[gr][ch] == 1) && (f.sideInfo.block_type[gr][ch] == 2) {
		region_1_start = 36  // sfb[9/3]*3=36
		region_2_start = 576 // No Region2 for short block case.
	} else {
		sfreq := f.header.sampling_frequency
		region_1_start =
			sfBandIndicesSet[sfreq].l[f.sideInfo.region0_count[gr][ch]+1]
		region_2_start =
			sfBandIndicesSet[sfreq].l[f.sideInfo.region0_count[gr][ch]+
				f.sideInfo.region1_count[gr][ch]+2]
	}
	// Read big_values using tables according to region_x_start
	for is_pos := 0; is_pos < f.sideInfo.big_values[gr][ch]*2; is_pos++ {
		table_num := 0
		if is_pos < region_1_start {
			table_num = f.sideInfo.table_select[gr][ch][0]
		} else if is_pos < region_2_start {
			table_num = f.sideInfo.table_select[gr][ch][1]
		} else {
			table_num = f.sideInfo.table_select[gr][ch][2]
		}
		// Get next Huffman coded words
		x, y, _, _, err := huffmanDecode(f.mainDataBytes, table_num)
		if err != nil {
			return err
		}
		// In the big_values area there are two freq lines per Huffman word
		f.mainData.is[gr][ch][is_pos] = float32(x)
		is_pos++
		f.mainData.is[gr][ch][is_pos] = float32(y)
	}
	// Read small values until is_pos = 576 or we run out of huffman data
	table_num := f.sideInfo.count1table_select[gr][ch] + 32
	is_pos := f.sideInfo.big_values[gr][ch] * 2
	for (is_pos <= 572) && (f.mainDataBytes.getMainPos() <= bit_pos_end) {
		// Get next Huffman coded words
		x, y, v, w, err := huffmanDecode(f.mainDataBytes, table_num)
		if err != nil {
			return err
		}
		f.mainData.is[gr][ch][is_pos] = float32(v)
		is_pos++
		if is_pos >= 576 {
			break
		}
		f.mainData.is[gr][ch][is_pos] = float32(w)
		is_pos++
		if is_pos >= 576 {
			break
		}
		f.mainData.is[gr][ch][is_pos] = float32(x)
		is_pos++
		if is_pos >= 576 {
			break
		}
		f.mainData.is[gr][ch][is_pos] = float32(y)
		is_pos++
	}
	// Check that we didn't read past the end of this section
	if f.mainDataBytes.getMainPos() > (bit_pos_end + 1) {
		// Remove last words read
		is_pos -= 4
	}
	// Setup count1 which is the index of the first sample in the rzero reg.
	f.sideInfo.count1[gr][ch] = is_pos
	// Zero out the last part if necessary
	for is_pos < 576 {
		f.mainData.is[gr][ch][is_pos] = 0.0
		is_pos++
	}
	// Set the bitpos to point to the next part to read
	f.mainDataBytes.setMainPos(bit_pos_end + 1)
	return nil
}
