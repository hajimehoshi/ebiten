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
	"errors"
	"io"
	"unsafe"
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

//export Get_Byte
func Get_Byte() C.unsigned {
	for len(readerCache) == 0 && !readerEOF {
		buf := make([]uint8, 4096)
		n, err := reader.Read(buf)
		if err != nil {
			if err == io.EOF {
				readerEOF = true
			} else {
				panic(err)
			}
		}
		readerCache = buf[:n]
	}
	if readerEOF {
		return eof
	}
	b := readerCache[0]
	readerCache = readerCache[1:]
	readerPos++
	return C.unsigned(b)
}

//export Get_Bytes
func Get_Bytes(num C.unsigned, data *C.unsigned) C.unsigned {
	s := C.unsigned(0)
	for i := 0; i < int(num); i++ {
		v := Get_Byte()
		if v == eof {
			return eof
		}
		*(*C.unsigned)(unsafe.Pointer(uintptr(unsafe.Pointer(data)) + uintptr(i)*unsafe.Sizeof(s))) = v
	}
	return C.OK
}

func getBytes(num int) ([]int, error) {
	r := make([]int, num)
	for i := 0; i < num; i++ {
		v := Get_Byte()
		if v == eof {
			return r, io.EOF
		}
		r[i] = int(v)
	}
	return r, nil
}

//export Get_Filepos
func Get_Filepos() C.unsigned {
	if len(readerCache) == 0 && readerEOF {
		return eof
	}
	return C.unsigned(readerPos)
}

//export writeToWriter
func writeToWriter(data unsafe.Pointer, size C.int) C.size_t {
	buf := C.GoBytes(data, size)
	n, err := writer.Write(buf)
	if err != nil {
		panic(err)
	}
	return C.size_t(n)
}

var g_error error

func decode(r io.Reader, w io.Writer) error {
	reader = r
	writer = w
	for Get_Filepos() != eof {
		if C.Read_Frame() == C.OK {
			C.Decode_L3()
			continue
		}
		if Get_Filepos() == eof {
			break
		}
		return errors.New("mp3: not enough maindata to decode frame")
	}
	return g_error
}
