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
