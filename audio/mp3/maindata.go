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

type mainDataBytes struct {
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

var theMainDataBytes mainDataBytes

//export Get_Main_Data
func Get_Main_Data(size C.unsigned, begin C.unsigned) C.int {
	if size > 1500 {
		g_error = fmt.Errorf("size = %d", size)
	}
	/* Check that there's data available from previous frames if needed */
	if int(begin) > theMainDataBytes.top {
		// No,there is not,so we skip decoding this frame,but we have to
		// read the main_data bits from the bitstream in case they are needed
		// for decoding the next frame.
		b, err := getBytes(int(size))
		if err != nil {
			g_error = err
			return C.ERROR
		}
		copy(theMainDataBytes.vec[theMainDataBytes.top:], b)
		/* Set up pointers */
		theMainDataBytes.ptr = theMainDataBytes.vec[0:]
		theMainDataBytes.pos = 0
		theMainDataBytes.idx = 0
		theMainDataBytes.top += int(size)
		return C.ERROR
	}
	/* Copy data from previous frames */
	for i := 0; i < int(begin); i++ {
		theMainDataBytes.vec[i] = theMainDataBytes.vec[theMainDataBytes.top-int(begin)+i]
	}
	/* Read the main_data from file */
	b, err := getBytes(int(size))
	if err != nil {
		g_error = err
		return C.ERROR
	}
	copy(theMainDataBytes.vec[begin:], b)
	/* Set up pointers */
	theMainDataBytes.ptr = theMainDataBytes.vec[0:]
	theMainDataBytes.pos = 0
	theMainDataBytes.idx = 0
	theMainDataBytes.top = int(begin) + int(size)
	return C.OK
}

//export Get_Main_Bit
func Get_Main_Bit() C.unsigned {
	tmp := uint(theMainDataBytes.ptr[0]) >> (7 - uint(theMainDataBytes.idx))
	tmp &= 0x01
	theMainDataBytes.ptr = theMainDataBytes.ptr[(theMainDataBytes.idx+1)>>3:]
	theMainDataBytes.pos += (theMainDataBytes.idx + 1) >> 3
	theMainDataBytes.idx = (theMainDataBytes.idx + 1) & 0x07
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
		if len(theMainDataBytes.ptr) > i {
			b[i] = theMainDataBytes.ptr[i]
		}
	}
	tmp := (uint32(b[0]) << 24) | (uint32(b[1]) << 16) | (uint32(b[2]) << 8) | (uint32(b[3]) << 0)

	/* Remove bits already used */
	tmp = tmp << uint(theMainDataBytes.idx)

	/* Remove bits after the desired bits */
	tmp = tmp >> (32 - num)

	/* Update pointers */
	theMainDataBytes.ptr = theMainDataBytes.ptr[(theMainDataBytes.idx+int(num))>>3:]
	theMainDataBytes.pos += (theMainDataBytes.idx + int(num)) >> 3
	theMainDataBytes.idx = (theMainDataBytes.idx + int(num)) & 0x07

	/* Done */
	return C.unsigned(tmp)
}

//export Get_Main_Pos
func Get_Main_Pos() C.unsigned {
	pos := theMainDataBytes.pos
	pos *= 8                    /* Multiply by 8 to get number of bits */
	pos += theMainDataBytes.idx /* Add current bit index */
	return C.unsigned(pos)
}

//export Set_Main_Pos
func Set_Main_Pos(bit_pos C.unsigned) C.int {
	theMainDataBytes.ptr = theMainDataBytes.vec[bit_pos>>3:]
	theMainDataBytes.pos = int(bit_pos) >> 3
	theMainDataBytes.idx = int(bit_pos) & 0x7
	return C.OK
}
