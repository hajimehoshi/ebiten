// Copyright 2026 The Ebitengine Authors
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

package wav_test

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

func TestDecodeInvalidHugeChunkSize(t *testing.T) {
	buf := []byte("RIFF")
	buf = binary.LittleEndian.AppendUint32(buf, 0)
	buf = append(buf, "WAVE"...)
	buf = append(buf, "fmt "...)
	buf = binary.LittleEndian.AppendUint32(buf, 0x7fffffff)
	buf = binary.LittleEndian.AppendUint16(buf, 1)
	buf = binary.LittleEndian.AppendUint16(buf, 1)
	buf = binary.LittleEndian.AppendUint32(buf, 8000)
	buf = binary.LittleEndian.AppendUint32(buf, 8000)
	buf = binary.LittleEndian.AppendUint16(buf, 1)
	buf = binary.LittleEndian.AppendUint16(buf, 16)

	if _, err := wav.DecodeWithoutResampling(bytes.NewReader(buf)); err == nil {
		t.Errorf("DecodeWithoutResampling: got no error, want an error")
	}
}

func TestDecodeInvalidHugeUnknownChunkSize(t *testing.T) {
	buf := []byte("RIFF")
	buf = binary.LittleEndian.AppendUint32(buf, 0)
	buf = append(buf, "WAVE"...)
	buf = append(buf, "JUNK"...)
	buf = binary.LittleEndian.AppendUint32(buf, 0x7fffffff)
	buf = append(buf, 0, 0)

	if _, err := wav.DecodeWithoutResampling(bytes.NewReader(buf)); err == nil {
		t.Errorf("DecodeWithoutResampling: got no error, want an error")
	}
}

// TestDecodeFmtChunkLargerThan16 tests a 'fmt ' chunk larger than 16 bytes.
func TestDecodeFmtChunkLargerThan16(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	buf := []byte("RIFF")
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(data)+38))
	buf = append(buf, "WAVE"...)
	buf = append(buf, "fmt "...)
	buf = binary.LittleEndian.AppendUint32(buf, 18)
	buf = binary.LittleEndian.AppendUint16(buf, 1)
	buf = binary.LittleEndian.AppendUint16(buf, 2)
	buf = binary.LittleEndian.AppendUint32(buf, 8000)
	buf = binary.LittleEndian.AppendUint32(buf, 32000)
	buf = binary.LittleEndian.AppendUint16(buf, 4)
	buf = binary.LittleEndian.AppendUint16(buf, 16)
	buf = binary.LittleEndian.AppendUint16(buf, 0)
	buf = append(buf, "data"...)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(data)))
	buf = append(buf, data...)

	s, err := wav.DecodeWithoutResampling(bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := s.Length(), int64(len(data)); got != want {
		t.Errorf("Length(): got: %d, want: %d", got, want)
	}
	if _, err := s.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(s)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("data: got: %v, want: %v", got, data)
	}
}

func TestDecodeValid(t *testing.T) {
	data := make([]byte, 8000)
	buf := []byte("RIFF")
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(data)+36))
	buf = append(buf, "WAVE"...)
	buf = append(buf, "fmt "...)
	buf = binary.LittleEndian.AppendUint32(buf, 16)
	buf = binary.LittleEndian.AppendUint16(buf, 1)
	buf = binary.LittleEndian.AppendUint16(buf, 1)
	buf = binary.LittleEndian.AppendUint32(buf, 8000)
	buf = binary.LittleEndian.AppendUint32(buf, 8000)
	buf = binary.LittleEndian.AppendUint16(buf, 1)
	buf = binary.LittleEndian.AppendUint16(buf, 16)
	buf = append(buf, "data"...)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(data)))
	buf = append(buf, data...)

	s, err := wav.DecodeWithoutResampling(bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := s.Length(), int64(len(data))*2; got != want {
		t.Errorf("Length(): got: %d, want: %d", got, want)
	}
}

const testSampleRate = 44100

type chunk struct {
	id   string
	data string
}

// appendChunk appends a RIFF chunk, with a pad byte if the data size is odd.
func appendChunk(dst []byte, id string, data []byte) []byte {
	dst = append(dst, id...)
	dst = binary.LittleEndian.AppendUint32(dst, uint32(len(data)))
	dst = append(dst, data...)
	if len(data)%2 != 0 {
		dst = append(dst, 0)
	}
	return dst
}

// stereoI16FmtChunkData returns the content of a 'fmt ' chunk for 2 channels signed 16bit little endian PCM.
func stereoI16FmtChunkData() []byte {
	var buf []byte
	buf = binary.LittleEndian.AppendUint16(buf, 1)                  // Format tag (linear PCM)
	buf = binary.LittleEndian.AppendUint16(buf, 2)                  // Channel count
	buf = binary.LittleEndian.AppendUint32(buf, testSampleRate)     // Sample rate
	buf = binary.LittleEndian.AppendUint32(buf, testSampleRate*2*2) // Byte rate
	buf = binary.LittleEndian.AppendUint16(buf, 4)                  // Block align
	buf = binary.LittleEndian.AppendUint16(buf, 16)                 // Bits per sample
	return buf
}

// wavFile returns a WAV file whose 'data' chunk holds data.
// beforeFmt and afterFmt are put before and after the 'fmt ' chunk respectively.
func wavFile(beforeFmt []chunk, afterFmt []chunk, data []byte) []byte {
	var body []byte
	for _, c := range beforeFmt {
		body = appendChunk(body, c.id, []byte(c.data))
	}
	body = appendChunk(body, "fmt ", stereoI16FmtChunkData())
	for _, c := range afterFmt {
		body = appendChunk(body, c.id, []byte(c.data))
	}
	body = appendChunk(body, "data", data)

	var buf []byte
	buf = append(buf, "RIFF"...)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(body)+4))
	buf = append(buf, "WAVE"...)
	return append(buf, body...)
}

func TestDecodeChunkPadding(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	testCases := []struct {
		name      string
		beforeFmt []chunk
		afterFmt  []chunk
	}{
		{
			name: "no extra chunk",
		},
		{
			name:      "even-sized chunk before 'fmt '",
			beforeFmt: []chunk{{id: "LIST", data: "abcd"}},
		},
		{
			name:      "odd-sized chunk before 'fmt '",
			beforeFmt: []chunk{{id: "LIST", data: "abc"}},
		},
		{
			name:     "odd-sized chunk after 'fmt '",
			afterFmt: []chunk{{id: "LIST", data: "abc"}},
		},
		{
			name:      "odd-sized chunks around 'fmt '",
			beforeFmt: []chunk{{id: "LIST", data: "abc"}},
			afterFmt:  []chunk{{id: "JUNK", data: "e"}},
		},
		{
			name:      "consecutive odd-sized chunks",
			beforeFmt: []chunk{{id: "LIST", data: "abc"}, {id: "JUNK", data: "e"}},
		},
		{
			name:      "empty chunk",
			beforeFmt: []chunk{{id: "LIST", data: ""}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			src := wavFile(tc.beforeFmt, tc.afterFmt, data)
			s, err := wav.DecodeWithoutResampling(bytes.NewReader(src))
			if err != nil {
				t.Fatal(err)
			}
			if got, want := s.SampleRate(), testSampleRate; got != want {
				t.Errorf("s.SampleRate(): got: %d, want: %d", got, want)
			}
			if got, want := s.Length(), int64(len(data)); got != want {
				t.Errorf("s.Length(): got: %d, want: %d", got, want)
			}

			// Seek to the beginning before reading: sectionReader.Read doesn't refer to its offset,
			// so a wrong header size is observable only via Seek.
			if _, err := s.Seek(0, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			got, err := io.ReadAll(s)
			if err != nil {
				t.Fatal(err)
			}
			if want := data; !bytes.Equal(got, want) {
				t.Errorf("data: got: %v, want: %v", got, want)
			}
		})
	}
}
