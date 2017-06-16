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
// //extern t_mpeg1_main_data g_main_data;
// extern t_mpeg1_header    g_frame_header;
// //extern t_mpeg1_side_info g_side_info;
import "C"

import (
	"io"
)

const (
	eof = 0xffffffff
)

var (
	reader      io.Reader
	readerCache []uint8
	readerPos   int
	readerEOF   bool
	writer      io.Writer
)

func decodeL3() error {
	out := make([]int, 576)
	// Number of channels(1 for mono and 2 for stereo)
	nch := 2
	if C.g_frame_header.mode == C.mpeg1_mode_single_channel {
		nch = 1
	}
	for gr := 0; gr < 2; gr++ {
		for ch := 0; ch < nch; ch++ {
			l3Requantize(gr, ch)
			// Reorder short blocks
			l3Reorder(gr, ch)
		}
		l3Stereo(gr)
		for ch := 0; ch < nch; ch++ {
			l3Antialias(gr, ch)
			// (IMDCT,windowing,overlapp add)
			l3HybridSynthesis(gr, ch)
			l3FrequencyInversion(gr, ch)
			// Polyphase subband synthesis
			l3SubbandSynthesis(gr, ch, out)
		}
		if err := audioWriteRaw(out); err != nil {
			return err
		}
	}
	return nil
}

func audioWriteRaw(samples []int) error {
	nch := 2
	if C.g_frame_header.mode == C.mpeg1_mode_single_channel {
		nch = 1
	}
	s := make([]uint8, len(samples)*2*nch)
	for i, v := range samples {
		if nch == 1 {
			s[2*i] = uint8(v)
			s[2*i+1] = uint8(v >> 8)
		} else {
			s[4*i] = uint8(v)
			s[4*i+1] = uint8(v >> 8)
			s[4*i+2] = uint8(v >> 16)
			s[4*i+3] = uint8(v >> 24)
		}
	}
	if _, err := writer.Write(s); err != nil {
		return err
	}
	return nil
}

func getByte() (uint8, error) {
	for len(readerCache) == 0 && !readerEOF {
		buf := make([]uint8, 4096)
		n, err := reader.Read(buf)
		readerCache = append(readerCache, buf[:n]...)
		if err != nil {
			if err == io.EOF {
				readerEOF = true
			} else {
				return 0, err
			}
		}
	}
	if len(readerCache) == 0 {
		return 0, io.EOF
	}
	b := readerCache[0]
	readerCache = readerCache[1:]
	readerPos++
	return b, nil
}

func getBytes(buf []int) (int, error) {
	for i := range buf {
		v, err := getByte()
		buf[i] = int(v)
		if err == io.EOF {
			return i, io.EOF
		}
	}
	return len(buf), nil
}

//export Get_Filepos
func Get_Filepos() C.unsigned {
	if len(readerCache) == 0 && readerEOF {
		return eof
	}
	return C.unsigned(readerPos)
}

func decode(r io.Reader, w io.Writer) error {
	// TODO: Decoder should know number of channels
	reader = r
	writer = w
	for Get_Filepos() != eof {
		err := readFrame()
		if err == nil {
			if err := decodeL3(); err != nil {
				return err
			}
			continue
		}
		if Get_Filepos() == eof {
			break
		}
		return err
	}
	return nil
}
