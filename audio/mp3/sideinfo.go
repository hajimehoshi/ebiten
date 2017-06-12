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

// A sideInfo is a bit reservoir for side info
type sideInfo struct {
	vec []int
	idx int // Index into the current byte(0-7)
}

var theSideInfo sideInfo

//export Get_Sideinfo
func Get_Sideinfo(size C.unsigned) {
	var err error
	theSideInfo.vec, err = getBytes(int(size))
	if err != nil {
		g_error = fmt.Errorf("mp3: couldn't read sideinfo %d bytes at pos %d: %v",
			size, Get_Filepos(), err)
		return
	}
	theSideInfo.idx = 0
}

//export Get_Side_Bits
func Get_Side_Bits(num C.unsigned) C.unsigned {
	// Form a word of the next four bytes
	// TODO: endianness?
	b := make([]int, 4)
	for i := range b {
		if len(theSideInfo.vec) > i {
			b[i] = theSideInfo.vec[i]
		}
	}
	tmp := (uint32(b[0]) << 24) | (uint32(b[1]) << 16) | (uint32(b[2]) << 8) | (uint32(b[3]) << 0)
	// Remove bits already used
	tmp = tmp << uint(theSideInfo.idx)
	// Remove bits after the desired bits
	tmp = tmp >> (32 - num)
	// Update pointers
	theSideInfo.vec = theSideInfo.vec[(theSideInfo.idx+int(num))>>3:]
	theSideInfo.idx = (theSideInfo.idx + int(num)) & 0x07
	return C.unsigned(tmp)
}
