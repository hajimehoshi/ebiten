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
// extern unsigned hsynth_init, synth_init;
// extern t_mpeg1_main_data g_main_data;
// extern t_mpeg1_side_info g_side_info;
import "C"

var store = [2][32][18]float32{}

//export L3_Hybrid_Synthesis
func L3_Hybrid_Synthesis(gr C.unsigned, ch C.unsigned) {
	for sb := 0; sb < 32; sb++ { /* Loop through all 32 subbands */
		/* Determine blocktype for this subband */
		bt := 0
		if (C.g_side_info.win_switch_flag[gr][ch] == 1) &&
			(C.g_side_info.mixed_block_flag[gr][ch] == 1) && (sb < 2) {
			bt = int(C.g_side_info.block_type[gr][ch])
		}
		/* Do the inverse modified DCT and windowing */
		in := make([]float32, 18)
		for i := range in {
			in[i] = float32(C.g_main_data.is[gr][ch][sb*18+i])
		}
		rawout := imdctWin(in, bt)
		for i := 0; i < 18; i++ { /* Overlapp add with stored vector into main_data vector */
			C.g_main_data.is[gr][ch][sb*18+i] = C.float(rawout[i] + store[ch][sb][i])
			store[ch][sb][i] = rawout[i+18]
		}
	}
}
