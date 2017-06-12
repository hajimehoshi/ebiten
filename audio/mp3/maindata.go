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
import "C"

import (
	"fmt"
)

/* Bit reservoir for main data */
type mainData struct {
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

var theMainData mainData

//export Get_Main_Data
func Get_Main_Data(size C.unsigned, begin C.unsigned) C.int {
	if size > 1500 {
		g_error = fmt.Errorf("size = %d", size)
	}
	/* Check that there's data available from previous frames if needed */
	if int(begin) > theMainData.top {
		// No,there is not,so we skip decoding this frame,but we have to
		// read the main_data bits from the bitstream in case they are needed
		// for decoding the next frame.
		b, err := getBytes(int(size))
		if err != nil {
			g_error = err
			return C.ERROR
		}
		copy(theMainData.vec[theMainData.top:], b)
		/* Set up pointers */
		theMainData.ptr = theMainData.vec[0:]
		theMainData.pos = 0
		theMainData.idx = 0
		theMainData.top += int(size)
		return C.ERROR
	}
	/* Copy data from previous frames */
	for i := 0; i < int(begin); i++ {
		theMainData.vec[i] = theMainData.vec[theMainData.top-int(begin)+i]
	}
	/* Read the main_data from file */
	b, err := getBytes(int(size))
	if err != nil {
		g_error = err
		return C.ERROR
	}
	copy(theMainData.vec[begin:], b)
	/* Set up pointers */
	theMainData.ptr = theMainData.vec[0:]
	theMainData.pos = 0
	theMainData.idx = 0
	theMainData.top = int(begin) + int(size)
	return C.OK
}

//export Get_Main_Bit
func Get_Main_Bit() C.unsigned {
	tmp := uint(theMainData.ptr[0]) >> (7 - uint(theMainData.idx))
	tmp &= 0x01
	theMainData.ptr = theMainData.ptr[(theMainData.idx+1)>>3:]
	theMainData.pos += (theMainData.idx + 1) >> 3
	theMainData.idx = (theMainData.idx + 1) & 0x07
	return C.unsigned(tmp)
}

//export Get_Main_Bits
func Get_Main_Bits(num C.unsigned) C.unsigned {
	if num == 0 {
		return 0
	}
	/* Form a word of the next four bytes */
	b := make([]int, 4)
	for i := range b {
		if len(theMainData.ptr) > i {
			b[i] = theMainData.ptr[i]
		}
	}
	tmp := (uint32(b[0]) << 24) | (uint32(b[1]) << 16) | (uint32(b[2]) << 8) | (uint32(b[3]) << 0)

	/* Remove bits already used */
	tmp = tmp << uint(theMainData.idx)

	/* Remove bits after the desired bits */
	tmp = tmp >> (32 - num)

	/* Update pointers */
	theMainData.ptr = theMainData.ptr[(theMainData.idx+int(num))>>3:]
	theMainData.pos += (theMainData.idx + int(num)) >> 3
	theMainData.idx = (theMainData.idx + int(num)) & 0x07

	/* Done */
	return C.unsigned(tmp)
}

//export Get_Main_Pos
func Get_Main_Pos() C.unsigned {
	pos := theMainData.pos
	pos *= 8               /* Multiply by 8 to get number of bits */
	pos += theMainData.idx /* Add current bit index */
	return C.unsigned(pos)
}

//export Set_Main_Pos
func Set_Main_Pos(bit_pos C.unsigned) C.int {
	theMainData.ptr = theMainData.vec[bit_pos>>3:]
	theMainData.pos = int(bit_pos) >> 3
	theMainData.idx = int(bit_pos) & 0x7
	return C.OK
}
