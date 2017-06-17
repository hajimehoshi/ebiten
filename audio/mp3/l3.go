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
	"math"
)

var (
	powtab34 = make([]float64, 8207)
	pretab   = []float64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 2, 2, 3, 3, 3, 2}
)

func init() {
	for i := range powtab34 {
		powtab34[i] = math.Pow(float64(i), 4.0/3.0)
	}
}

func (f *frame) requantizeProcessLong(gr, ch, is_pos, sfb int) {
	sf_mult := 0.5
	if f.sideInfo.scalefac_scale[gr][ch] != 0 {
		sf_mult = 1.0
	}
	tmp1 := 1.0
	// https://github.com/technosaurus/PDMP3/issues/4
	if sfb < 21 {
		pf_x_pt := float64(f.sideInfo.preflag[gr][ch]) * pretab[sfb]
		tmp1 = math.Pow(2.0, -(sf_mult * (float64(f.mainData.scalefac_l[gr][ch][sfb]) + pf_x_pt)))
	}
	tmp2 := math.Pow(2.0, 0.25*(float64(f.sideInfo.global_gain[gr][ch])-210))
	tmp3 := 0.0
	if f.mainData.is[gr][ch][is_pos] < 0.0 {
		tmp3 = -powtab34[int(-f.mainData.is[gr][ch][is_pos])]
	} else {
		tmp3 = powtab34[int(f.mainData.is[gr][ch][is_pos])]
	}
	f.mainData.is[gr][ch][is_pos] = float32(tmp1 * tmp2 * tmp3)
}

func (f *frame) requantizeProcessShort(gr, ch, is_pos, sfb, win int) {
	sf_mult := 0.5
	if f.sideInfo.scalefac_scale[gr][ch] != 0 {
		sf_mult = 1.0
	}
	tmp1 := 1.0
	// https://github.com/technosaurus/PDMP3/issues/4
	if sfb < 12 {
		tmp1 = math.Pow(2.0, -(sf_mult * float64(f.mainData.scalefac_s[gr][ch][sfb][win])))
	}
	tmp2 := math.Pow(2.0, 0.25*(float64(f.sideInfo.global_gain[gr][ch])-210.0-
		8.0*float64(f.sideInfo.subblock_gain[gr][ch][win])))
	tmp3 := 0.0
	if f.mainData.is[gr][ch][is_pos] < 0 {
		tmp3 = -powtab34[int(-f.mainData.is[gr][ch][is_pos])]
	} else {
		tmp3 = powtab34[int(f.mainData.is[gr][ch][is_pos])]
	}
	f.mainData.is[gr][ch][is_pos] = float32(tmp1 * tmp2 * tmp3)
}

type sfBandIndices struct {
	l []int
	s []int
}

var (
	sfBandIndicesSet = []sfBandIndices{
		{
			l: []int{0, 4, 8, 12, 16, 20, 24, 30, 36, 44, 52, 62, 74, 90, 110, 134, 162, 196, 238, 288, 342, 418, 576},
			s: []int{0, 4, 8, 12, 16, 22, 30, 40, 52, 66, 84, 106, 136, 192},
		},
		{
			l: []int{0, 4, 8, 12, 16, 20, 24, 30, 36, 42, 50, 60, 72, 88, 106, 128, 156, 190, 230, 276, 330, 384, 576},
			s: []int{0, 4, 8, 12, 16, 22, 28, 38, 50, 64, 80, 100, 126, 192},
		},
		{
			l: []int{0, 4, 8, 12, 16, 20, 24, 30, 36, 44, 54, 66, 82, 102, 126, 156, 194, 240, 296, 364, 448, 550, 576},
			s: []int{0, 4, 8, 12, 16, 22, 30, 42, 58, 78, 104, 138, 180, 192},
		},
	}
)

func (f *frame) l3Requantize(gr int, ch int) {
	/* Setup sampling frequency index */
	sfreq := f.header.sampling_frequency
	/* Determine type of block to process */
	if (f.sideInfo.win_switch_flag[gr][ch] == 1) && (f.sideInfo.block_type[gr][ch] == 2) { /* Short blocks */
		/* Check if the first two subbands
		 *(=2*18 samples = 8 long or 3 short sfb's) uses long blocks */
		if f.sideInfo.mixed_block_flag[gr][ch] != 0 { /* 2 longbl. sb  first */
			/* First process the 2 long block subbands at the start */
			sfb := 0
			next_sfb := sfBandIndicesSet[sfreq].l[sfb+1]
			for i := 0; i < 36; i++ {
				if i == next_sfb {
					sfb++
					next_sfb = sfBandIndicesSet[sfreq].l[sfb+1]
				}
				f.requantizeProcessLong(gr, ch, i, sfb)
			}
			/* And next the remaining,non-zero,bands which uses short blocks */
			sfb = 3
			next_sfb = sfBandIndicesSet[sfreq].s[sfb+1] * 3
			win_len := sfBandIndicesSet[sfreq].s[sfb+1] -
				sfBandIndicesSet[sfreq].s[sfb]

			for i := 36; i < int(f.sideInfo.count1[gr][ch]); /* i++ done below! */ {
				/* Check if we're into the next scalefac band */
				if i == next_sfb { /* Yes */
					sfb++
					next_sfb = sfBandIndicesSet[sfreq].s[sfb+1] * 3
					win_len = sfBandIndicesSet[sfreq].s[sfb+1] -
						sfBandIndicesSet[sfreq].s[sfb]
				}
				for win := 0; win < 3; win++ {
					for j := 0; j < win_len; j++ {
						f.requantizeProcessShort(gr, ch, i, sfb, win)
						i++
					}
				}

			}
		} else { /* Only short blocks */
			sfb := 0
			next_sfb := sfBandIndicesSet[sfreq].s[sfb+1] * 3
			win_len := sfBandIndicesSet[sfreq].s[sfb+1] -
				sfBandIndicesSet[sfreq].s[sfb]
			for i := 0; i < int(f.sideInfo.count1[gr][ch]); /* i++ done below! */ {
				/* Check if we're into the next scalefac band */
				if i == next_sfb {
					sfb++
					next_sfb = sfBandIndicesSet[sfreq].s[sfb+1] * 3
					win_len = sfBandIndicesSet[sfreq].s[sfb+1] -
						sfBandIndicesSet[sfreq].s[sfb]
				}
				for win := 0; win < 3; win++ {
					for j := 0; j < win_len; j++ {
						f.requantizeProcessShort(gr, ch, i, sfb, win)
						i++
					}
				}
			}
		}
	} else { /* Only long blocks */
		sfb := 0
		next_sfb := sfBandIndicesSet[sfreq].l[sfb+1]
		for i := 0; i < int(f.sideInfo.count1[gr][ch]); i++ {
			if i == next_sfb {
				sfb++
				next_sfb = sfBandIndicesSet[sfreq].l[sfb+1]
			}
			f.requantizeProcessLong(gr, ch, i, sfb)
		}
	}
}

func (f *frame) l3Reorder(gr int, ch int) {
	re := make([]float32, 576)

	sfreq := f.header.sampling_frequency /* Setup sampling freq index */
	/* Only reorder short blocks */
	if (f.sideInfo.win_switch_flag[gr][ch] == 1) && (f.sideInfo.block_type[gr][ch] == 2) { /* Short blocks */
		/* Check if the first two subbands
		 *(=2*18 samples = 8 long or 3 short sfb's) uses long blocks */
		sfb := 0
		/* 2 longbl. sb  first */
		if f.sideInfo.mixed_block_flag[gr][ch] != 0 {
			sfb = 3
		}
		next_sfb := sfBandIndicesSet[sfreq].s[sfb+1] * 3
		win_len := sfBandIndicesSet[sfreq].s[sfb+1] - sfBandIndicesSet[sfreq].s[sfb]
		i := 36
		if sfb == 0 {
			i = 0
		}
		for i < 576 {
			/* Check if we're into the next scalefac band */
			if i == next_sfb {
				/* Copy reordered data back to the original vector */
				for j := 0; j < 3*win_len; j++ {
					f.mainData.is[gr][ch][3*sfBandIndicesSet[sfreq].s[sfb]+j] = re[j]
				}
				/* Check if this band is above the rzero region,if so we're done */
				if i >= f.sideInfo.count1[gr][ch] {
					return
				}
				sfb++
				next_sfb = sfBandIndicesSet[sfreq].s[sfb+1] * 3
				win_len = sfBandIndicesSet[sfreq].s[sfb+1] - sfBandIndicesSet[sfreq].s[sfb]
			}
			for win := 0; win < 3; win++ { /* Do the actual reordering */
				for j := 0; j < win_len; j++ {
					re[j*3+win] = f.mainData.is[gr][ch][i]
					i++
				}
			}
		}
		/* Copy reordered data of last band back to original vector */
		for j := 0; j < 3*win_len; j++ {
			f.mainData.is[gr][ch][3*sfBandIndicesSet[sfreq].s[12]+j] = re[j]
		}
	}
}

var (
	is_ratios = [6]float32{0.000000, 0.267949, 0.577350, 1.000000, 1.732051, 3.732051}
)

func (f *frame) stereoProcessIntensityLong(gr int, sfb int) {
	is_ratio_l := float32(0)
	is_ratio_r := float32(0)
	/* Check that((is_pos[sfb]=scalefac) != 7) => no intensity stereo */
	is_pos := f.mainData.scalefac_l[gr][0][sfb]
	if is_pos != 7 {
		sfreq := f.header.sampling_frequency /* Setup sampling freq index */
		sfb_start := sfBandIndicesSet[sfreq].l[sfb]
		sfb_stop := sfBandIndicesSet[sfreq].l[sfb+1]
		if is_pos == 6 { /* tan((6*PI)/12 = PI/2) needs special treatment! */
			is_ratio_l = 1.0
			is_ratio_r = 0.0
		} else {
			is_ratio_l = is_ratios[is_pos] / (1.0 + is_ratios[is_pos])
			is_ratio_r = 1.0 / (1.0 + is_ratios[is_pos])
		}
		/* Now decode all samples in this scale factor band */
		for i := sfb_start; i < sfb_stop; i++ {
			f.mainData.is[gr][0][i] *= is_ratio_l
			f.mainData.is[gr][1][i] *= is_ratio_r
		}
	}
}

func (f *frame) stereoProcessIntensityShort(gr int, sfb int) {
	is_ratio_l := float32(0)
	is_ratio_r := float32(0)
	sfreq := f.header.sampling_frequency /* Setup sampling freq index */
	/* The window length */
	win_len := sfBandIndicesSet[sfreq].s[sfb+1] - sfBandIndicesSet[sfreq].s[sfb]
	/* The three windows within the band has different scalefactors */
	for win := 0; win < 3; win++ {
		/* Check that((is_pos[sfb]=scalefac) != 7) => no intensity stereo */
		is_pos := f.mainData.scalefac_s[gr][0][sfb][win]
		if is_pos != 7 {
			sfb_start := sfBandIndicesSet[sfreq].s[sfb]*3 + win_len*win
			sfb_stop := sfb_start + win_len
			if is_pos == 6 { /* tan((6*PI)/12 = PI/2) needs special treatment! */
				is_ratio_l = 1.0
				is_ratio_r = 0.0
			} else {
				is_ratio_l = is_ratios[is_pos] / (1.0 + is_ratios[is_pos])
				is_ratio_r = 1.0 / (1.0 + is_ratios[is_pos])
			}
			/* Now decode all samples in this scale factor band */
			for i := sfb_start; i < sfb_stop; i++ {
				// https://github.com/technosaurus/PDMP3/issues/3
				f.mainData.is[gr][0][i] *= is_ratio_l
				f.mainData.is[gr][1][i] *= is_ratio_r
			}
		}
	}
}

func (f *frame) l3Stereo(gr int) {
	/* Do nothing if joint stereo is not enabled */
	if (f.header.mode != 1) || (f.header.mode_extension == 0) {
		return
	}
	/* Do Middle/Side("normal") stereo processing */
	if (f.header.mode_extension & 0x2) != 0 {
		/* Determine how many frequency lines to transform */
		i := 0
		if f.sideInfo.count1[gr][0] > f.sideInfo.count1[gr][1] {
			i = 1
		}
		max_pos := int(f.sideInfo.count1[gr][i])
		/* Do the actual processing */
		const invSqrt2 = math.Sqrt2 / 2
		for i := 0; i < max_pos; i++ {
			left := (f.mainData.is[gr][0][i] + f.mainData.is[gr][1][i]) * invSqrt2
			right := (f.mainData.is[gr][0][i] - f.mainData.is[gr][1][i]) * invSqrt2
			f.mainData.is[gr][0][i] = left
			f.mainData.is[gr][1][i] = right
		}
	}
	/* Do intensity stereo processing */
	if (f.header.mode_extension & 0x1) != 0 {
		/* Setup sampling frequency index */
		sfreq := f.header.sampling_frequency
		/* First band that is intensity stereo encoded is first band scale factor
		 * band on or above count1 frequency line. N.B.: Intensity stereo coding is
		 * only done for higher subbands, but logic is here for lower subbands. */
		/* Determine type of block to process */
		if (f.sideInfo.win_switch_flag[gr][0] == 1) &&
			(f.sideInfo.block_type[gr][0] == 2) { /* Short blocks */
			/* Check if the first two subbands
			 *(=2*18 samples = 8 long or 3 short sfb's) uses long blocks */
			if f.sideInfo.mixed_block_flag[gr][0] != 0 { /* 2 longbl. sb  first */
				for sfb := 0; sfb < 8; sfb++ { /* First process 8 sfb's at start */
					/* Is this scale factor band above count1 for the right channel? */
					if sfBandIndicesSet[sfreq].l[sfb] >= f.sideInfo.count1[gr][1] {
						f.stereoProcessIntensityLong(gr, sfb)
					}
				}
				/* And next the remaining bands which uses short blocks */
				for sfb := 3; sfb < 12; sfb++ {
					/* Is this scale factor band above count1 for the right channel? */
					if sfBandIndicesSet[sfreq].s[sfb]*3 >= f.sideInfo.count1[gr][1] {
						f.stereoProcessIntensityShort(gr, sfb) /* intensity stereo processing */
					}
				}
			} else { /* Only short blocks */
				for sfb := 0; sfb < 12; sfb++ {
					/* Is this scale factor band above count1 for the right channel? */
					if sfBandIndicesSet[sfreq].s[sfb]*3 >= f.sideInfo.count1[gr][1] {
						f.stereoProcessIntensityShort(gr, sfb) /* intensity stereo processing */
					}
				}
			}
		} else { /* Only long blocks */
			for sfb := 0; sfb < 21; sfb++ {
				/* Is this scale factor band above count1 for the right channel? */
				if sfBandIndicesSet[sfreq].l[sfb] >= f.sideInfo.count1[gr][1] {
					/* Perform the intensity stereo processing */
					f.stereoProcessIntensityLong(gr, sfb)
				}
			}
		}
	}
}

var (
	cs = [8]float32{0.857493, 0.881742, 0.949629, 0.983315, 0.995518, 0.999161, 0.999899, 0.999993}
	ca = [8]float32{-0.514496, -0.471732, -0.313377, -0.181913, -0.094574, -0.040966, -0.014199, -0.003700}
)

func (f *frame) l3Antialias(gr int, ch int) {
	/* No antialiasing is done for short blocks */
	if (f.sideInfo.win_switch_flag[gr][ch] == 1) &&
		(f.sideInfo.block_type[gr][ch] == 2) &&
		(f.sideInfo.mixed_block_flag[gr][ch]) == 0 {
		return
	}
	/* Setup the limit for how many subbands to transform */
	sblim := 32
	if (f.sideInfo.win_switch_flag[gr][ch] == 1) &&
		(f.sideInfo.block_type[gr][ch] == 2) &&
		(f.sideInfo.mixed_block_flag[gr][ch] == 1) {
		sblim = 2
	}
	/* Do the actual antialiasing */
	for sb := 1; sb < sblim; sb++ {
		for i := 0; i < 8; i++ {
			li := 18*sb - 1 - i
			ui := 18*sb + i
			lb := f.mainData.is[gr][ch][li]*cs[i] - f.mainData.is[gr][ch][ui]*ca[i]
			ub := f.mainData.is[gr][ch][ui]*cs[i] + f.mainData.is[gr][ch][li]*ca[i]
			f.mainData.is[gr][ch][li] = lb
			f.mainData.is[gr][ch][ui] = ub
		}
	}
}

var store = [2][32][18]float32{}

func (f *frame) l3HybridSynthesis(gr int, ch int) {
	// Loop through all 32 subbands
	for sb := 0; sb < 32; sb++ {
		// Determine blocktype for this subband
		bt := int(f.sideInfo.block_type[gr][ch])
		if (f.sideInfo.win_switch_flag[gr][ch] == 1) &&
			(f.sideInfo.mixed_block_flag[gr][ch] == 1) && (sb < 2) {
			bt = 0
		}
		// Do the inverse modified DCT and windowing
		in := make([]float32, 18)
		for i := range in {
			in[i] = f.mainData.is[gr][ch][sb*18+i]
		}
		rawout := imdctWin(in, bt)
		// Overlapp add with stored vector into main_data vector
		for i := 0; i < 18; i++ {
			f.mainData.is[gr][ch][sb*18+i] = rawout[i] + store[ch][sb][i]
			store[ch][sb][i] = rawout[i+18]
		}
	}
}

func (f *frame) l3FrequencyInversion(gr int, ch int) {
	for sb := 1; sb < 32; sb += 2 { //OPT? : for(sb = 18; sb < 576; sb += 36)
		for i := 1; i < 18; i += 2 {
			f.mainData.is[gr][ch][sb*18+i] = -f.mainData.is[gr][ch][sb*18+i]
		}
	}
}

var g_synth_n_win = [64][32]float32{}

func init() {
	for i := 0; i < 64; i++ {
		for j := 0; j < 32; j++ {
			g_synth_n_win[i][j] =
				float32(math.Cos(float64((16+i)*(2*j+1)) * (math.Pi / 64.0)))
		}
	}
}

var g_synth_dtbl = [512]float32{
	0.000000000, -0.000015259, -0.000015259, -0.000015259,
	-0.000015259, -0.000015259, -0.000015259, -0.000030518,
	-0.000030518, -0.000030518, -0.000030518, -0.000045776,
	-0.000045776, -0.000061035, -0.000061035, -0.000076294,
	-0.000076294, -0.000091553, -0.000106812, -0.000106812,
	-0.000122070, -0.000137329, -0.000152588, -0.000167847,
	-0.000198364, -0.000213623, -0.000244141, -0.000259399,
	-0.000289917, -0.000320435, -0.000366211, -0.000396729,
	-0.000442505, -0.000473022, -0.000534058, -0.000579834,
	-0.000625610, -0.000686646, -0.000747681, -0.000808716,
	-0.000885010, -0.000961304, -0.001037598, -0.001113892,
	-0.001205444, -0.001296997, -0.001388550, -0.001480103,
	-0.001586914, -0.001693726, -0.001785278, -0.001907349,
	-0.002014160, -0.002120972, -0.002243042, -0.002349854,
	-0.002456665, -0.002578735, -0.002685547, -0.002792358,
	-0.002899170, -0.002990723, -0.003082275, -0.003173828,
	0.003250122, 0.003326416, 0.003387451, 0.003433228,
	0.003463745, 0.003479004, 0.003479004, 0.003463745,
	0.003417969, 0.003372192, 0.003280640, 0.003173828,
	0.003051758, 0.002883911, 0.002700806, 0.002487183,
	0.002227783, 0.001937866, 0.001617432, 0.001266479,
	0.000869751, 0.000442505, -0.000030518, -0.000549316,
	-0.001098633, -0.001693726, -0.002334595, -0.003005981,
	-0.003723145, -0.004486084, -0.005294800, -0.006118774,
	-0.007003784, -0.007919312, -0.008865356, -0.009841919,
	-0.010848999, -0.011886597, -0.012939453, -0.014022827,
	-0.015121460, -0.016235352, -0.017349243, -0.018463135,
	-0.019577026, -0.020690918, -0.021789551, -0.022857666,
	-0.023910522, -0.024932861, -0.025909424, -0.026840210,
	-0.027725220, -0.028533936, -0.029281616, -0.029937744,
	-0.030532837, -0.031005859, -0.031387329, -0.031661987,
	-0.031814575, -0.031845093, -0.031738281, -0.031478882,
	0.031082153, 0.030517578, 0.029785156, 0.028884888,
	0.027801514, 0.026535034, 0.025085449, 0.023422241,
	0.021575928, 0.019531250, 0.017257690, 0.014801025,
	0.012115479, 0.009231567, 0.006134033, 0.002822876,
	-0.000686646, -0.004394531, -0.008316040, -0.012420654,
	-0.016708374, -0.021179199, -0.025817871, -0.030609131,
	-0.035552979, -0.040634155, -0.045837402, -0.051132202,
	-0.056533813, -0.061996460, -0.067520142, -0.073059082,
	-0.078628540, -0.084182739, -0.089706421, -0.095169067,
	-0.100540161, -0.105819702, -0.110946655, -0.115921021,
	-0.120697021, -0.125259399, -0.129562378, -0.133590698,
	-0.137298584, -0.140670776, -0.143676758, -0.146255493,
	-0.148422241, -0.150115967, -0.151306152, -0.151962280,
	-0.152069092, -0.151596069, -0.150497437, -0.148773193,
	-0.146362305, -0.143264771, -0.139450073, -0.134887695,
	-0.129577637, -0.123474121, -0.116577148, -0.108856201,
	0.100311279, 0.090927124, 0.080688477, 0.069595337,
	0.057617188, 0.044784546, 0.031082153, 0.016510010,
	0.001068115, -0.015228271, -0.032379150, -0.050354004,
	-0.069168091, -0.088775635, -0.109161377, -0.130310059,
	-0.152206421, -0.174789429, -0.198059082, -0.221984863,
	-0.246505737, -0.271591187, -0.297210693, -0.323318481,
	-0.349868774, -0.376800537, -0.404083252, -0.431655884,
	-0.459472656, -0.487472534, -0.515609741, -0.543823242,
	-0.572036743, -0.600219727, -0.628295898, -0.656219482,
	-0.683914185, -0.711318970, -0.738372803, -0.765029907,
	-0.791213989, -0.816864014, -0.841949463, -0.866363525,
	-0.890090942, -0.913055420, -0.935195923, -0.956481934,
	-0.976852417, -0.996246338, -1.014617920, -1.031936646,
	-1.048156738, -1.063217163, -1.077117920, -1.089782715,
	-1.101211548, -1.111373901, -1.120223999, -1.127746582,
	-1.133926392, -1.138763428, -1.142211914, -1.144287109,
	1.144989014, 1.144287109, 1.142211914, 1.138763428,
	1.133926392, 1.127746582, 1.120223999, 1.111373901,
	1.101211548, 1.089782715, 1.077117920, 1.063217163,
	1.048156738, 1.031936646, 1.014617920, 0.996246338,
	0.976852417, 0.956481934, 0.935195923, 0.913055420,
	0.890090942, 0.866363525, 0.841949463, 0.816864014,
	0.791213989, 0.765029907, 0.738372803, 0.711318970,
	0.683914185, 0.656219482, 0.628295898, 0.600219727,
	0.572036743, 0.543823242, 0.515609741, 0.487472534,
	0.459472656, 0.431655884, 0.404083252, 0.376800537,
	0.349868774, 0.323318481, 0.297210693, 0.271591187,
	0.246505737, 0.221984863, 0.198059082, 0.174789429,
	0.152206421, 0.130310059, 0.109161377, 0.088775635,
	0.069168091, 0.050354004, 0.032379150, 0.015228271,
	-0.001068115, -0.016510010, -0.031082153, -0.044784546,
	-0.057617188, -0.069595337, -0.080688477, -0.090927124,
	0.100311279, 0.108856201, 0.116577148, 0.123474121,
	0.129577637, 0.134887695, 0.139450073, 0.143264771,
	0.146362305, 0.148773193, 0.150497437, 0.151596069,
	0.152069092, 0.151962280, 0.151306152, 0.150115967,
	0.148422241, 0.146255493, 0.143676758, 0.140670776,
	0.137298584, 0.133590698, 0.129562378, 0.125259399,
	0.120697021, 0.115921021, 0.110946655, 0.105819702,
	0.100540161, 0.095169067, 0.089706421, 0.084182739,
	0.078628540, 0.073059082, 0.067520142, 0.061996460,
	0.056533813, 0.051132202, 0.045837402, 0.040634155,
	0.035552979, 0.030609131, 0.025817871, 0.021179199,
	0.016708374, 0.012420654, 0.008316040, 0.004394531,
	0.000686646, -0.002822876, -0.006134033, -0.009231567,
	-0.012115479, -0.014801025, -0.017257690, -0.019531250,
	-0.021575928, -0.023422241, -0.025085449, -0.026535034,
	-0.027801514, -0.028884888, -0.029785156, -0.030517578,
	0.031082153, 0.031478882, 0.031738281, 0.031845093,
	0.031814575, 0.031661987, 0.031387329, 0.031005859,
	0.030532837, 0.029937744, 0.029281616, 0.028533936,
	0.027725220, 0.026840210, 0.025909424, 0.024932861,
	0.023910522, 0.022857666, 0.021789551, 0.020690918,
	0.019577026, 0.018463135, 0.017349243, 0.016235352,
	0.015121460, 0.014022827, 0.012939453, 0.011886597,
	0.010848999, 0.009841919, 0.008865356, 0.007919312,
	0.007003784, 0.006118774, 0.005294800, 0.004486084,
	0.003723145, 0.003005981, 0.002334595, 0.001693726,
	0.001098633, 0.000549316, 0.000030518, -0.000442505,
	-0.000869751, -0.001266479, -0.001617432, -0.001937866,
	-0.002227783, -0.002487183, -0.002700806, -0.002883911,
	-0.003051758, -0.003173828, -0.003280640, -0.003372192,
	-0.003417969, -0.003463745, -0.003479004, -0.003479004,
	-0.003463745, -0.003433228, -0.003387451, -0.003326416,
	0.003250122, 0.003173828, 0.003082275, 0.002990723,
	0.002899170, 0.002792358, 0.002685547, 0.002578735,
	0.002456665, 0.002349854, 0.002243042, 0.002120972,
	0.002014160, 0.001907349, 0.001785278, 0.001693726,
	0.001586914, 0.001480103, 0.001388550, 0.001296997,
	0.001205444, 0.001113892, 0.001037598, 0.000961304,
	0.000885010, 0.000808716, 0.000747681, 0.000686646,
	0.000625610, 0.000579834, 0.000534058, 0.000473022,
	0.000442505, 0.000396729, 0.000366211, 0.000320435,
	0.000289917, 0.000259399, 0.000244141, 0.000213623,
	0.000198364, 0.000167847, 0.000152588, 0.000137329,
	0.000122070, 0.000106812, 0.000106812, 0.000091553,
	0.000076294, 0.000076294, 0.000061035, 0.000061035,
	0.000045776, 0.000045776, 0.000030518, 0.000030518,
	0.000030518, 0.000030518, 0.000015259, 0.000015259,
	0.000015259, 0.000015259, 0.000015259, 0.000015259,
}

var v_vec = [2][1024]float32{}

func (f *frame) l3SubbandSynthesis(gr int, ch int, out []uint32) {
	u_vec := make([]float32, 512)
	s_vec := make([]float32, 32)

	/* Number of channels(1 for mono and 2 for stereo) */
	nch := 2
	if f.header.mode == mpeg1ModeSingleChannel {
		nch = 1
	}
	/* Setup the n_win windowing vector and the v_vec intermediate vector */
	for ss := 0; ss < 18; ss++ { /* Loop through 18 samples in 32 subbands */
		for i := 1023; i > 63; i-- { /* Shift up the V vector */
			v_vec[ch][i] = v_vec[ch][i-64]
		}
		for i := 0; i < 32; i++ { /* Copy next 32 time samples to a temp vector */
			s_vec[i] = f.mainData.is[gr][ch][i*18+ss]
		}
		for i := 0; i < 64; i++ { /* Matrix multiply input with n_win[][] matrix */
			sum := float32(0)
			for j := 0; j < 32; j++ {
				sum += g_synth_n_win[i][j] * s_vec[j]
			}
			v_vec[ch][i] = sum
		}
		for i := 0; i < 8; i++ { /* Build the U vector */
			for j := 0; j < 32; j++ { /* <<7 == *128 */
				u_vec[(i<<6)+j] = v_vec[ch][(i<<7)+j]
				u_vec[(i<<6)+j+32] = v_vec[ch][(i<<7)+j+96]
			}
		}
		for i := 0; i < 512; i++ { /* Window by u_vec[i] with g_synth_dtbl[i] */
			u_vec[i] *= g_synth_dtbl[i]
		}
		for i := 0; i < 32; i++ { /* Calc 32 samples,store in outdata vector */
			sum := float32(0)
			for j := 0; j < 16; j++ { /* sum += u_vec[j*32 + i]; */
				sum += u_vec[(j<<5)+i]
			}
			/* sum now contains time sample 32*ss+i. Convert to 16-bit signed int */
			samp := int(sum * 32767)
			if samp > 32767 {
				samp = 32767
			} else if samp < -32767 {
				samp = -32767
			}
			samp &= 0xffff
			s := uint32(samp)
			if ch == 0 { /* This function must be called for channel 0 first */
				/* We always run in stereo mode,& duplicate channels here for mono */
				if nch == 1 {
					out[32*ss+i] = (s << 16) | (s)
				} else {
					out[32*ss+i] = s << 16
				}
			} else {
				out[32*ss+i] |= s
			}
		}
	}
}
