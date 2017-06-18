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

var mpeg1ScalefacSizes = [16][2]int{
	{0, 0}, {0, 1}, {0, 2}, {0, 3}, {3, 0}, {1, 1}, {1, 2}, {1, 3},
	{2, 1}, {2, 2}, {2, 3}, {3, 1}, {3, 2}, {3, 3}, {4, 2}, {4, 3},
}

func (s *source) readMainL3(prev *mainDataBytes, header *mpeg1FrameHeader, sideInfo *mpeg1SideInfo) (*mpeg1MainData, *mainDataBytes, error) {
	nch := header.numberOfChannels()
	// Calculate header audio data size
	framesize := header.frameSize()
	if framesize > 2000 {
		return nil, nil, fmt.Errorf("mp3: framesize = %d", framesize)
	}
	// Sideinfo is 17 bytes for one channel and 32 bytes for two
	sideinfo_size := 32
	if nch == 1 {
		sideinfo_size = 17
	}
	// Main data size is the rest of the frame,including ancillary data
	main_data_size := framesize - sideinfo_size - 4 // sync+header
	// CRC is 2 bytes
	if header.protection_bit == 0 {
		main_data_size -= 2
	}
	// Assemble main data buffer with data from this frame and the previous
	// two frames. main_data_begin indicates how many bytes from previous
	// frames that should be used. This buffer is later accessed by the
	// getMainBits function in the same way as the side info is.
	m, err := s.getMainData(prev, main_data_size, sideInfo.main_data_begin)
	if err != nil {
		// This could be due to not enough data in reservoir
		return nil, nil, err
	}
	md := &mpeg1MainData{}
	for gr := 0; gr < 2; gr++ {
		for ch := 0; ch < nch; ch++ {
			part_2_start := m.getMainPos()
			// Number of bits in the bitstream for the bands
			slen1 := mpeg1ScalefacSizes[sideInfo.scalefac_compress[gr][ch]][0]
			slen2 := mpeg1ScalefacSizes[sideInfo.scalefac_compress[gr][ch]][1]
			if (sideInfo.win_switch_flag[gr][ch] != 0) && (sideInfo.block_type[gr][ch] == 2) {
				if sideInfo.mixed_block_flag[gr][ch] != 0 {
					for sfb := 0; sfb < 8; sfb++ {
						md.scalefac_l[gr][ch][sfb] = m.getMainBits(slen1)
					}
					for sfb := 3; sfb < 12; sfb++ {
						//slen1 for band 3-5,slen2 for 6-11
						nbits := slen2
						if sfb < 6 {
							nbits = slen1
						}
						for win := 0; win < 3; win++ {
							md.scalefac_s[gr][ch][sfb][win] = m.getMainBits(nbits)
						}
					}
				} else {
					for sfb := 0; sfb < 12; sfb++ {
						//slen1 for band 3-5,slen2 for 6-11
						nbits := slen2
						if sfb < 6 {
							nbits = slen1
						}
						for win := 0; win < 3; win++ {
							md.scalefac_s[gr][ch][sfb][win] = m.getMainBits(nbits)
						}
					}
				}
			} else { // block_type == 0 if winswitch == 0
				// Scale factor bands 0-5
				if (sideInfo.scfsi[ch][0] == 0) || (gr == 0) {
					for sfb := 0; sfb < 6; sfb++ {
						md.scalefac_l[gr][ch][sfb] = m.getMainBits(slen1)
					}
				} else if (sideInfo.scfsi[ch][0] == 1) && (gr == 1) {
					// Copy scalefactors from granule 0 to granule 1
					for sfb := 0; sfb < 6; sfb++ {
						md.scalefac_l[1][ch][sfb] = md.scalefac_l[0][ch][sfb]
					}
				}
				// Scale factor bands 6-10
				if (sideInfo.scfsi[ch][1] == 0) || (gr == 0) {
					for sfb := 6; sfb < 11; sfb++ {
						md.scalefac_l[gr][ch][sfb] = m.getMainBits(slen1)
					}
				} else if (sideInfo.scfsi[ch][1] == 1) && (gr == 1) {
					// Copy scalefactors from granule 0 to granule 1
					for sfb := 6; sfb < 11; sfb++ {
						md.scalefac_l[1][ch][sfb] = md.scalefac_l[0][ch][sfb]
					}
				}
				// Scale factor bands 11-15
				if (sideInfo.scfsi[ch][2] == 0) || (gr == 0) {
					for sfb := 11; sfb < 16; sfb++ {
						md.scalefac_l[gr][ch][sfb] = m.getMainBits(slen2)
					}
				} else if (sideInfo.scfsi[ch][2] == 1) && (gr == 1) {
					// Copy scalefactors from granule 0 to granule 1
					for sfb := 11; sfb < 16; sfb++ {
						md.scalefac_l[1][ch][sfb] = md.scalefac_l[0][ch][sfb]
					}
				}
				// Scale factor bands 16-20
				if (sideInfo.scfsi[ch][3] == 0) || (gr == 0) {
					for sfb := 16; sfb < 21; sfb++ {
						md.scalefac_l[gr][ch][sfb] = m.getMainBits(slen2)
					}
				} else if (sideInfo.scfsi[ch][3] == 1) && (gr == 1) {
					// Copy scalefactors from granule 0 to granule 1
					for sfb := 16; sfb < 21; sfb++ {
						md.scalefac_l[1][ch][sfb] = md.scalefac_l[0][ch][sfb]
					}
				}
			}
			// Read Huffman coded data. Skip stuffing bits.
			if err := m.readHuffman(header, sideInfo, md, part_2_start, gr, ch); err != nil {
				return nil, nil, err
			}
		}
	}
	// The ancillary data is stored here,but we ignore it.
	return md, m, nil
}

type mainDataBytes struct {
	// Large static data
	vec []int
	// Index into the current byte(0-7)
	idx int
	pos int
}

func (s *source) getMainData(prev *mainDataBytes, size int, offset int) (*mainDataBytes, error) {
	if size > 1500 {
		return nil, fmt.Errorf("mp3: size = %d", size)
	}
	// Check that there's data available from previous frames if needed
	if prev != nil && offset > len(prev.vec) {
		// No,there is not, so we skip decoding this frame, but we have to
		// read the main_data bits from the bitstream in case they are needed
		// for decoding the next frame.
		buf := make([]int, size)
		n := 0
		var err error
		for n < size && err == nil {
			nn, err2 := s.getBytes(buf)
			n += nn
			err = err2
		}
		if n < size {
			if err == io.EOF {
				return nil, fmt.Errorf("mp3: unexpected EOF at getMainData")
			}
			return nil, err
		}
		m := &mainDataBytes{
			vec: append(prev.vec, buf...),
		}
		// TODO: Define a special error and enable to continue the next frame.
		return m, fmt.Errorf("mp3: frame can't be decoded")
	}
	// Copy data from previous frames
	vec := []int{}
	if prev != nil {
		v := prev.vec
		vec = v[len(v)-offset:]
	}
	// Read the main_data from file
	buf := make([]int, size)
	n := 0
	var err error
	for n < size && err == nil {
		nn, err2 := s.getBytes(buf)
		n += nn
		err = err2
	}
	if n < size {
		if err == io.EOF {
			return nil, fmt.Errorf("mp3: unexpected EOF at getMainData")
		}
		return nil, err
	}
	m := &mainDataBytes{
		vec: append(vec, buf...),
	}
	return m, nil
}

func (m *mainDataBytes) getMainBit() int {
	tmp := uint(m.vec[m.pos]) >> (7 - uint(m.idx))
	tmp &= 0x01
	m.pos += (m.idx + 1) >> 3
	m.idx = (m.idx + 1) & 0x07
	return int(tmp)
}

func (m *mainDataBytes) getMainBits(num int) int {
	if num == 0 {
		return 0
	}
	// Form a word of the next four bytes
	b := make([]int, 4)
	copy(b, m.vec[m.pos:])
	tmp := (uint32(b[0]) << 24) | (uint32(b[1]) << 16) | (uint32(b[2]) << 8) | (uint32(b[3]) << 0)

	// Remove bits already used
	tmp = tmp << uint(m.idx)

	// Remove bits after the desired bits
	tmp = tmp >> (32 - uint(num))

	// Update pointers
	m.pos += (m.idx + num) >> 3
	m.idx = (m.idx + num) & 0x07
	return int(tmp)
}

func (m *mainDataBytes) getMainPos() int {
	pos := m.pos
	pos *= 8 // Multiply by 8 to get number of bits
	pos += m.idx
	return pos
}

func (m *mainDataBytes) setMainPos(bit_pos int) {
	m.pos = bit_pos >> 3
	m.idx = bit_pos & 0x7
}
