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

//export Read_CRC
func Read_CRC() C.int {
	buf := make([]int, 2)
	n := 0
	var err error
	for n < 2 && err == nil {
		nn, err2 := getBytes(buf[n:])
		n += nn
		err = err2
	}
	if err == io.EOF {
		if n < 2 {
			return C.ERROR
		}
		return C.OK
	}
	if err != nil {
		g_error = err
		return C.ERROR
	}
	return C.OK
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
