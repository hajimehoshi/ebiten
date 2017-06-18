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

var mpeg1Bitrates = map[mpeg1Layer][15]int{
	mpeg1Layer1: {
		0, 32000, 64000, 96000, 128000, 160000, 192000, 224000,
		256000, 288000, 320000, 352000, 384000, 416000, 448000,
	},
	mpeg1Layer2: {
		0, 32000, 48000, 56000, 64000, 80000, 96000, 112000,
		128000, 160000, 192000, 224000, 256000, 320000, 384000,
	},
	mpeg1Layer3: {
		0, 32000, 40000, 48000, 56000, 64000, 80000, 96000,
		112000, 128000, 160000, 192000, 224000, 256000, 320000,
	},
}

var samplingFrequency = [3]int{44100, 48000, 32000}

func (f *frame) size() int {
	return (144*mpeg1Bitrates[f.header.layer][f.header.bitrate_index])/
		samplingFrequency[f.header.sampling_frequency] +
		int(f.header.padding_bit)
}

func (f *frame) readAudioL3() error {
	nch := f.numberOfChannels()
	/* Calculate header audio data size */
	framesize := f.size()
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
	if f.header.protection_bit == 0 {
		main_data_size -= 2
	}
	/* Read sideinfo from bitstream into buffer used by getSideBits() */
	s, err := getSideinfo(sideinfo_size)
	if err != nil {
		return err
	}
	/* Parse audio data */
	/* Pointer to where we should start reading main data */
	f.sideInfo.main_data_begin = s.getSideBits(9)
	/* Get private bits. Not used for anything. */
	if f.header.mode == mpeg1ModeSingleChannel {
		f.sideInfo.private_bits = s.getSideBits(5)
	} else {
		f.sideInfo.private_bits = s.getSideBits(3)
	}
	/* Get scale factor selection information */
	for ch := 0; ch < nch; ch++ {
		for scfsi_band := 0; scfsi_band < 4; scfsi_band++ {
			f.sideInfo.scfsi[ch][scfsi_band] = s.getSideBits(1)
		}
	}
	/* Get the rest of the side information */
	for gr := 0; gr < 2; gr++ {
		for ch := 0; ch < nch; ch++ {
			f.sideInfo.part2_3_length[gr][ch] = s.getSideBits(12)
			f.sideInfo.big_values[gr][ch] = s.getSideBits(9)
			f.sideInfo.global_gain[gr][ch] = s.getSideBits(8)
			f.sideInfo.scalefac_compress[gr][ch] = s.getSideBits(4)
			f.sideInfo.win_switch_flag[gr][ch] = s.getSideBits(1)
			if f.sideInfo.win_switch_flag[gr][ch] == 1 {
				f.sideInfo.block_type[gr][ch] = s.getSideBits(2)
				f.sideInfo.mixed_block_flag[gr][ch] = s.getSideBits(1)
				for region := 0; region < 2; region++ {
					f.sideInfo.table_select[gr][ch][region] = s.getSideBits(5)
				}
				for window := 0; window < 3; window++ {
					f.sideInfo.subblock_gain[gr][ch][window] = s.getSideBits(3)
				}
				if (f.sideInfo.block_type[gr][ch] == 2) && (f.sideInfo.mixed_block_flag[gr][ch] == 0) {
					f.sideInfo.region0_count[gr][ch] = 8 /* Implicit */
				} else {
					f.sideInfo.region0_count[gr][ch] = 7 /* Implicit */
				}
				/* The standard is wrong on this!!! */ /* Implicit */
				f.sideInfo.region1_count[gr][ch] = 20 - f.sideInfo.region0_count[gr][ch]
			} else {
				for region := 0; region < 3; region++ {
					f.sideInfo.table_select[gr][ch][region] = s.getSideBits(5)
				}
				f.sideInfo.region0_count[gr][ch] = s.getSideBits(4)
				f.sideInfo.region1_count[gr][ch] = s.getSideBits(3)
				f.sideInfo.block_type[gr][ch] = 0 /* Implicit */
			}
			f.sideInfo.preflag[gr][ch] = s.getSideBits(1)
			f.sideInfo.scalefac_scale[gr][ch] = s.getSideBits(1)
			f.sideInfo.count1table_select[gr][ch] = s.getSideBits(1)
		}
	}
	return nil
}

// A sideInfoBytes is a bit reservoir for side info
type sideInfoBytes struct {
	vec []int
	idx int // Index into the current byte(0-7)
}

func getSideinfo(size int) (*sideInfoBytes, error) {
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
			return nil, fmt.Errorf("mp3: unexpected EOF at getSideinfo")
		}
		return nil, fmt.Errorf("mp3: couldn't read sideinfo %d bytes at pos %d: %v",
			size, getFilepos(), err)
	}
	s := &sideInfoBytes{
		vec: buf[:n],
	}
	return s, nil
}

func (s *sideInfoBytes) getSideBits(num int) int {
	// Form a word of the next four bytes
	// TODO: endianness?
	b := make([]int, 4)
	for i := range b {
		if len(s.vec) > i {
			b[i] = s.vec[i]
		}
	}
	tmp := (uint32(b[0]) << 24) | (uint32(b[1]) << 16) | (uint32(b[2]) << 8) | (uint32(b[3]) << 0)
	// Remove bits already used
	tmp = tmp << uint(s.idx)
	// Remove bits after the desired bits
	tmp = tmp >> (32 - uint(num))
	// Update pointers
	s.vec = s.vec[(s.idx+int(num))>>3:]
	s.idx = (s.idx + int(num)) & 0x07
	return int(tmp)
}
