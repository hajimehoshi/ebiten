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

type mpeg1Layer int

const (
	mpeg1LayerReserved mpeg1Layer = 0
	mpeg1Layer3        mpeg1Layer = 1
	mpeg1Layer2        mpeg1Layer = 2
	mpeg1Layer1        mpeg1Layer = 3
)

type mpeg1Mode int

const (
	mpeg1ModeStereo mpeg1Mode = iota
	mpeg1ModeJointStereo
	mpeg1ModeDualChannel
	mpeg1ModeSingleChannel
)

// A mepg1FrameHeader is MPEG1 Layer 1-3 frame header
type mpeg1FrameHeader struct {
	id                 int        // 1 bit
	layer              mpeg1Layer // 2 bits
	protection_bit     int        // 1 bit
	bitrate_index      int        // 4 bits
	sampling_frequency int        // 2 bits
	padding_bit        int        // 1 bit
	private_bit        int        // 1 bit
	mode               mpeg1Mode  // 2 bits
	mode_extension     int        // 2 bits
	copyright          int        // 1 bit
	original_or_copy   int        // 1 bit
	emphasis           int        // 2 bits
}

// A mpeg1SideInfo is  MPEG1 Layer 3 Side Information.
// [2][2] means [gr][ch].
type mpeg1SideInfo struct {
	main_data_begin   int       // 9 bits
	private_bits      int       // 3 bits in mono, 5 in stereo
	scfsi             [2][4]int // 1 bit
	part2_3_length    [2][2]int // 12 bits
	big_values        [2][2]int // 9 bits
	global_gain       [2][2]int // 8 bits
	scalefac_compress [2][2]int // 4 bits
	win_switch_flag   [2][2]int // 1 bit

	block_type       [2][2]int    // 2 bits
	mixed_block_flag [2][2]int    // 1 bit
	table_select     [2][2][3]int // 5 bits
	subblock_gain    [2][2][3]int // 3 bits

	region0_count [2][2]int // 4 bits
	region1_count [2][2]int // 3 bits

	preflag            [2][2]int // 1 bit
	scalefac_scale     [2][2]int // 1 bit
	count1table_select [2][2]int // 1 bit
	count1             [2][2]int // Not in file,calc. by huff.dec.!
}

// A mpeg1MainData is MPEG1 Layer 3 Main Data.
type mpeg1MainData struct {
	scalefac_l [2][2][21]int      // 0-4 bits
	scalefac_s [2][2][12][3]int   // 0-4 bits
	is         [2][2][576]float32 // Huffman coded freq. lines
}

type frame struct {
	prev *frame

	header   *mpeg1FrameHeader
	sideInfo *mpeg1SideInfo
	mainData mpeg1MainData

	mainDataBytes *mainDataBytes
	store         [2][32][18]float32
	v_vec         [2][1024]float32
}

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

func (h *mpeg1FrameHeader) frameSize() int {
	return (144*mpeg1Bitrates[h.layer][h.bitrate_index])/
		samplingFrequency[h.sampling_frequency] +
		int(h.padding_bit)
}

func (h *mpeg1FrameHeader) numberOfChannels() int {
	if h.mode == mpeg1ModeSingleChannel {
		return 1
	}
	return 2
}
