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
func stereoI16FmtChunkData(sampleRate uint32) []byte {
	var buf []byte
	buf = binary.LittleEndian.AppendUint16(buf, 1)              // Format tag (linear PCM)
	buf = binary.LittleEndian.AppendUint16(buf, 2)              // Channel count
	buf = binary.LittleEndian.AppendUint32(buf, sampleRate)     // Sample rate
	buf = binary.LittleEndian.AppendUint32(buf, sampleRate*2*2) // Byte rate
	buf = binary.LittleEndian.AppendUint16(buf, 4)              // Block align
	buf = binary.LittleEndian.AppendUint16(buf, 16)             // Bits per sample
	return buf
}

// wavFile returns a WAV file whose 'data' chunk holds data.
// beforeFmt and afterFmt are put before and after the 'fmt ' chunk respectively.
func wavFile(sampleRate uint32, beforeFmt []chunk, afterFmt []chunk, data []byte) []byte {
	var body []byte
	for _, c := range beforeFmt {
		body = appendChunk(body, c.id, []byte(c.data))
	}
	body = appendChunk(body, "fmt ", stereoI16FmtChunkData(sampleRate))
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
			src := wavFile(testSampleRate, tc.beforeFmt, tc.afterFmt, data)
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

// permissiveSeeker is an io.ReadSeeker that silently succeeds without moving for an unrecognized
// whence, so that a rejected seek can only come from the reader under test.
type permissiveSeeker struct {
	r *bytes.Reader
}

func (p *permissiveSeeker) Read(buf []byte) (int, error) {
	return p.r.Read(buf)
}

func (p *permissiveSeeker) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart, io.SeekCurrent, io.SeekEnd:
		return p.r.Seek(offset, whence)
	}
	return p.r.Seek(0, io.SeekCurrent)
}

func TestDecodeSeekInvalidWhence(t *testing.T) {
	for _, tc := range []struct {
		name     string
		channels int
	}{
		{"Mono", 1},
		{"Stereo", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]byte, 8000)
			buf := []byte("RIFF")
			buf = binary.LittleEndian.AppendUint32(buf, uint32(len(data)+36))
			buf = append(buf, "WAVE"...)
			buf = append(buf, "fmt "...)
			buf = binary.LittleEndian.AppendUint32(buf, 16)
			buf = binary.LittleEndian.AppendUint16(buf, 1)
			buf = binary.LittleEndian.AppendUint16(buf, uint16(tc.channels))
			buf = binary.LittleEndian.AppendUint32(buf, testSampleRate)
			buf = binary.LittleEndian.AppendUint32(buf, uint32(testSampleRate*2*tc.channels))
			buf = binary.LittleEndian.AppendUint16(buf, uint16(2*tc.channels))
			buf = binary.LittleEndian.AppendUint16(buf, 16)
			buf = append(buf, "data"...)
			buf = binary.LittleEndian.AppendUint32(buf, uint32(len(data)))
			buf = append(buf, data...)

			s, err := wav.DecodeWithoutResampling(&permissiveSeeker{r: bytes.NewReader(buf)})
			if err != nil {
				t.Fatal(err)
			}
			for _, whence := range []int{-1, 3, 100} {
				if _, err := s.Seek(0, whence); err == nil {
					t.Errorf("Seek(0, %d): got no error, want an error", whence)
				}
			}
		})
	}
}

func TestDecodeSeekOutOfRangeLeavesStreamIntact(t *testing.T) {
	const size = 8000
	const headerFill = 0x5a
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i)
	}
	// A large chunk before 'fmt ' pushes the data chunk far from the file
	// start, so that a small negative offset resolves inside the header.
	header := bytes.Repeat([]byte{headerFill}, 200)
	buf := wavFile(testSampleRate, []chunk{{id: "LIST", data: string(header)}}, nil, data)

	const pos = 4
	for _, tc := range []struct {
		name   string
		offset int64
		whence int
	}{
		{
			name:   "SeekStartNegative",
			offset: -100,
			whence: io.SeekStart,
		},
		{
			name:   "SeekStartPastEnd",
			offset: size + 4,
			whence: io.SeekStart,
		},
		{
			name:   "SeekCurrentNegative",
			offset: -100,
			whence: io.SeekCurrent,
		},
		{
			name:   "SeekEndBeforeStart",
			offset: -(size + 100),
			whence: io.SeekEnd,
		},
		{
			name:   "SeekEndPastEnd",
			offset: 100,
			whence: io.SeekEnd,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := wav.DecodeWithoutResampling(bytes.NewReader(buf))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.Seek(pos, io.SeekStart); err != nil {
				t.Fatal(err)
			}

			if _, err := s.Seek(tc.offset, tc.whence); err == nil {
				t.Errorf("Seek(%d, %d): got no error, want an error", tc.offset, tc.whence)
			}

			b := make([]byte, 8)
			n, err := s.Read(b)
			if err != nil {
				t.Fatal(err)
			}
			if n != len(b) {
				t.Errorf("Read: got %d bytes, want %d", n, len(b))
			}
			if want := data[pos : pos+len(b)]; !bytes.Equal(b, want) {
				t.Errorf("Read after a failed Seek: got %#x, want %#x", b, want)
			}
		})
	}
}

func TestDecodeInvalidSampleRate(t *testing.T) {
	testCases := []struct {
		name       string
		sampleRate uint32
		wantErr    bool
	}{
		{
			name:       "zero",
			sampleRate: 0,
			wantErr:    true,
		},
		{
			name:       "valid",
			sampleRate: testSampleRate,
			wantErr:    false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := wavFile(tc.sampleRate, nil, nil, []byte{1, 2, 3, 4})

			if _, err := wav.DecodeWithoutResampling(bytes.NewReader(data)); (err != nil) != tc.wantErr {
				t.Errorf("wav.DecodeWithoutResampling: err %v, wantErr %v", err, tc.wantErr)
			}
			if _, err := wav.DecodeWithSampleRate(testSampleRate, bytes.NewReader(data)); (err != nil) != tc.wantErr {
				t.Errorf("wav.DecodeWithSampleRate: err %v, wantErr %v", err, tc.wantErr)
			}
			if _, err := wav.DecodeF32(bytes.NewReader(data)); (err != nil) != tc.wantErr {
				t.Errorf("wav.DecodeF32: err %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// pcmWavFile returns a linear PCM WAV file with the given channel count and bit depth
// whose 'data' chunk holds data.
func pcmWavFile(channelCount, bitsPerSample int, data []byte) []byte {
	blockAlign := channelCount * bitsPerSample / 8

	var fmtData []byte
	fmtData = binary.LittleEndian.AppendUint16(fmtData, 1) // Format tag (linear PCM)
	fmtData = binary.LittleEndian.AppendUint16(fmtData, uint16(channelCount))
	fmtData = binary.LittleEndian.AppendUint32(fmtData, testSampleRate)
	fmtData = binary.LittleEndian.AppendUint32(fmtData, testSampleRate*uint32(blockAlign))
	fmtData = binary.LittleEndian.AppendUint16(fmtData, uint16(blockAlign))
	fmtData = binary.LittleEndian.AppendUint16(fmtData, uint16(bitsPerSample))

	var body []byte
	body = appendChunk(body, "fmt ", fmtData)
	body = appendChunk(body, "data", data)

	var buf []byte
	buf = append(buf, "RIFF"...)
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(body)+4))
	buf = append(buf, "WAVE"...)
	return append(buf, body...)
}

func TestDecodePartialFrame(t *testing.T) {
	const dataSize = 65

	testCases := []struct {
		name          string
		channelCount  int
		bitsPerSample int
		wantFrames    int64
	}{
		{
			name:          "MonoU8",
			channelCount:  1,
			bitsPerSample: 8,
			wantFrames:    65,
		},
		{
			name:          "MonoS16",
			channelCount:  1,
			bitsPerSample: 16,
			wantFrames:    32,
		},
		{
			name:          "StereoU8",
			channelCount:  2,
			bitsPerSample: 8,
			wantFrames:    32,
		},
		{
			name:          "StereoS16",
			channelCount:  2,
			bitsPerSample: 16,
			wantFrames:    16,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]byte, dataSize)
			for i := range data {
				data[i] = byte(i + 1)
			}
			src := pcmWavFile(tc.channelCount, tc.bitsPerSample, data)

			for _, d := range []struct {
				name          string
				decode        func(src io.Reader) (*wav.Stream, error)
				bytesPerFrame int64
			}{
				{
					name:          "DecodeWithoutResampling",
					decode:        wav.DecodeWithoutResampling,
					bytesPerFrame: 4,
				},
				{
					name:          "DecodeF32",
					decode:        wav.DecodeF32,
					bytesPerFrame: 8,
				},
			} {
				t.Run(d.name, func(t *testing.T) {
					s, err := d.decode(bytes.NewReader(src))
					if err != nil {
						t.Fatal(err)
					}
					want := tc.wantFrames * d.bytesPerFrame
					if got := s.Length(); got != want {
						t.Errorf("Length(): got: %d, want: %d", got, want)
					}
					bs, err := io.ReadAll(s)
					if err != nil {
						t.Fatal(err)
					}
					if got := int64(len(bs)); got != want {
						t.Errorf("len(io.ReadAll(s)): got: %d, want: %d", got, want)
					}
				})
			}
		})
	}
}

func TestDecodeDataChunkBeforeFmtChunk(t *testing.T) {
	var body []byte
	body = appendChunk(body, "data", []byte{1, 2, 3, 4})
	body = appendChunk(body, "fmt ", stereoI16FmtChunkData(testSampleRate))

	buf := []byte("RIFF")
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(body)+4))
	buf = append(buf, "WAVE"...)
	buf = append(buf, body...)

	if _, err := wav.DecodeWithoutResampling(bytes.NewReader(buf)); err == nil {
		t.Errorf("DecodeWithoutResampling: got no error, want an error")
	}
}

// shortReader is an io.Reader that returns at most maxN bytes for each Read.
type shortReader struct {
	r    *bytes.Reader
	maxN int
}

func (s *shortReader) Read(buf []byte) (int, error) {
	if len(buf) > s.maxN {
		buf = buf[:s.maxN]
	}
	return s.r.Read(buf)
}

// TestDecodeF32ShortReads tests a source returning fewer bytes than one sample at a time.
func TestDecodeF32ShortReads(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	s, err := wav.DecodeF32(&shortReader{r: bytes.NewReader(wavFile(testSampleRate, nil, nil, data)), maxN: 1})
	if err != nil {
		t.Fatal(err)
	}

	var got int
	for {
		var buf [64]byte
		n, err := s.Read(buf[:])
		if n == 0 && err == nil {
			t.Fatal("Read: got (0, <nil>), want a non-zero byte count or an error")
		}
		got += n
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if want := len(data) * 2; got != want {
		t.Errorf("read %d bytes, want %d", got, want)
	}
}
