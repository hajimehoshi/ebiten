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
	"errors"
	"fmt"
	"io"
	"math"
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
			name:   "SeekCurrentNegative",
			offset: -100,
			whence: io.SeekCurrent,
		},
		{
			name:   "SeekEndBeforeStart",
			offset: -(size + 100),
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

func TestDecodeWithSampleRateInvalidSampleRate(t *testing.T) {
	data := wavFile(testSampleRate, nil, nil, []byte{1, 2, 3, 4})
	for _, sampleRate := range []int{0, -1} {
		if _, err := wav.DecodeWithSampleRate(sampleRate, bytes.NewReader(data)); err == nil {
			t.Errorf("wav.DecodeWithSampleRate(%d): got no error, want an error", sampleRate)
		}
	}
}

// failingReader delivers the first limit bytes of its source and then fails with err.
type failingReader struct {
	r     *bytes.Reader
	limit int64
	err   error
}

func (f *failingReader) Read(buf []byte) (int, error) {
	pos := f.r.Size() - int64(f.r.Len())
	if pos >= f.limit {
		return 0, f.err
	}
	if int64(len(buf)) > f.limit-pos {
		buf = buf[:f.limit-pos]
	}
	return f.r.Read(buf)
}

func TestDecodeHeaderReadError(t *testing.T) {
	data := wavFile(testSampleRate, nil, nil, []byte{1, 2, 3, 4})
	for _, tc := range []struct {
		name  string
		limit int64
	}{
		{
			name:  "header",
			limit: 0,
		},
		{
			name:  "inside the header",
			limit: 5,
		},
		{
			name:  "chunk header",
			limit: 12,
		},
		{
			name:  "inside the chunk header",
			limit: 15,
		},
		{
			name:  "'fmt ' chunk",
			limit: 20,
		},
		{
			name:  "inside the 'fmt ' chunk",
			limit: 30,
		},
		{
			name:  "'data' chunk header",
			limit: 36,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			newSource := func() io.Reader {
				return &failingReader{
					r:     bytes.NewReader(data),
					limit: tc.limit,
					err:   errSourceRead,
				}
			}
			if _, err := wav.DecodeWithoutResampling(newSource()); !errors.Is(err, errSourceRead) {
				t.Errorf("wav.DecodeWithoutResampling: got %v, want %v", err, errSourceRead)
			}
			if _, err := wav.DecodeWithSampleRate(testSampleRate, newSource()); !errors.Is(err, errSourceRead) {
				t.Errorf("wav.DecodeWithSampleRate: got %v, want %v", err, errSourceRead)
			}
			if _, err := wav.DecodeF32(newSource()); !errors.Is(err, errSourceRead) {
				t.Errorf("wav.DecodeF32: got %v, want %v", err, errSourceRead)
			}

			truncated := bytes.NewReader(data[:tc.limit])
			if _, err := wav.DecodeWithoutResampling(truncated); err == nil {
				t.Errorf("wav.DecodeWithoutResampling with a truncated source: got no error, want an error")
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

func TestDecodeSeekSmallNegativePosition(t *testing.T) {
	data := make([]byte, 8000)
	for i := range data {
		data[i] = byte(i)
	}

	for _, tc := range []struct {
		name   string
		decode func(src io.Reader) (*wav.Stream, error)
		src    []byte
	}{
		{
			name:   "Stereo",
			decode: wav.DecodeWithoutResampling,
			src:    pcmWavFile(2, 16, data),
		},
		{
			name:   "Mono",
			decode: wav.DecodeWithoutResampling,
			src:    pcmWavFile(1, 16, data),
		},
		{
			name:   "StereoF32",
			decode: wav.DecodeF32,
			src:    pcmWavFile(2, 16, data),
		},
		{
			name:   "MonoF32",
			decode: wav.DecodeF32,
			src:    pcmWavFile(1, 16, data),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := tc.decode(bytes.NewReader(tc.src))
			if err != nil {
				t.Fatal(err)
			}

			// An offset in (-4, 0) is rounded toward the frame boundary before it reaches the
			// decoder, so the requested position must be resolved and checked before the
			// rounding, whichever whence it comes from.
			for _, offset := range []int64{-1, -2, -3, -4} {
				if _, err := s.Seek(offset, io.SeekStart); err == nil {
					t.Errorf("Seek(%d, io.SeekStart): got no error, want an error", offset)
				}
				if _, err := s.Seek(offset, io.SeekCurrent); err == nil {
					t.Errorf("Seek(%d, io.SeekCurrent) at 0: got no error, want an error", offset)
				}
				if _, err := s.Seek(-s.Length()+offset, io.SeekEnd); err == nil {
					t.Errorf("Seek(-Length()%+d, io.SeekEnd): got no error, want an error", offset)
				}
			}

			// The stream must not be broken by the rejected seeks.
			b := make([]byte, 8)
			if _, err := io.ReadFull(s, b); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func setDataChunkSize(wavFile []byte, size uint32) {
	_, after, _ := bytes.Cut(wavFile, []byte("data"))
	binary.LittleEndian.PutUint32(after, size)
}

func TestDecodePlaceholderDataChunkSize(t *testing.T) {
	data := make([]byte, 8002)
	for i := range data {
		data[i] = byte(i)
	}

	decoders := []struct {
		name   string
		decode func(src io.Reader) (*wav.Stream, error)
	}{
		{
			name:   "DecodeWithoutResampling",
			decode: wav.DecodeWithoutResampling,
		},
		{
			name:   "DecodeF32",
			decode: wav.DecodeF32,
		},
		{
			name: "DecodeWithSampleRate",
			decode: func(src io.Reader) (*wav.Stream, error) {
				return wav.DecodeWithSampleRate(testSampleRate*2, src)
			},
		},
	}

	for _, tc := range []struct {
		name          string
		declaredSize  uint32
		channelCount  int
		bitsPerSample int
	}{
		{
			name:          "ZeroStereoS16",
			declaredSize:  0,
			channelCount:  2,
			bitsPerSample: 16,
		},
		{
			name:          "ZeroMonoU8",
			declaredSize:  0,
			channelCount:  1,
			bitsPerSample: 8,
		},
		{
			name:          "MaxStereoS16",
			declaredSize:  0xffffffff,
			channelCount:  2,
			bitsPerSample: 16,
		},
		{
			name:          "MaxMonoU8",
			declaredSize:  0xffffffff,
			channelCount:  1,
			bitsPerSample: 8,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			intact := pcmWavFile(tc.channelCount, tc.bitsPerSample, data)
			patched := bytes.Clone(intact)
			setDataChunkSize(patched, tc.declaredSize)

			for _, d := range decoders {
				t.Run(d.name, func(t *testing.T) {
					want, err := d.decode(bytes.NewReader(intact))
					if err != nil {
						t.Fatal(err)
					}
					wantBytes, err := io.ReadAll(want)
					if err != nil {
						t.Fatal(err)
					}
					if len(wantBytes) == 0 {
						t.Fatal("the intact file decoded to no bytes")
					}

					t.Run("Seekable", func(t *testing.T) {
						s, err := d.decode(bytes.NewReader(patched))
						if err != nil {
							t.Fatal(err)
						}
						if got, want := s.Length(), want.Length(); got != want {
							t.Errorf("Length(): got: %d, want: %d", got, want)
						}
						got, err := io.ReadAll(s)
						if err != nil {
							t.Fatal(err)
						}
						if !bytes.Equal(got, wantBytes) {
							t.Errorf("decoded %d bytes, want %d bytes", len(got), len(wantBytes))
						}

						if _, err := s.Seek(0, io.SeekStart); err != nil {
							t.Fatal(err)
						}
						got, err = io.ReadAll(s)
						if err != nil {
							t.Fatal(err)
						}
						if !bytes.Equal(got, wantBytes) {
							t.Errorf("after Seek(0, io.SeekStart): decoded %d bytes, want %d bytes", len(got), len(wantBytes))
						}

						mid := want.Length() / 2
						if _, err := s.Seek(mid, io.SeekStart); err != nil {
							t.Fatal(err)
						}
						got, err = io.ReadAll(s)
						if err != nil {
							t.Fatal(err)
						}
						if !bytes.Equal(got, wantBytes[mid:]) {
							t.Errorf("after Seek(%d, io.SeekStart): decoded %d bytes, want %d bytes", mid, len(got), len(wantBytes[mid:]))
						}
					})

					t.Run("NonSeekable", func(t *testing.T) {
						s, err := d.decode(struct{ io.Reader }{bytes.NewReader(patched)})
						if err != nil {
							t.Fatal(err)
						}
						if got := s.Length(); got != -1 {
							t.Errorf("Length(): got: %d, want: -1", got)
						}
						got, err := io.ReadAll(s)
						if err != nil {
							t.Fatal(err)
						}
						if !bytes.Equal(got, wantBytes) {
							t.Errorf("decoded %d bytes, want %d bytes", len(got), len(wantBytes))
						}
					})
				})
			}
		})
	}
}

func TestDecodePlaceholderDataChunkSizeShortReads(t *testing.T) {
	data := make([]byte, 1001)
	for i := range data {
		data[i] = byte(i + 1)
	}
	patched := pcmWavFile(2, 16, data)
	setDataChunkSize(patched, 0)

	s, err := wav.DecodeWithoutResampling(&shortReader{r: bytes.NewReader(patched), maxN: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Seek(0, io.SeekStart); !errors.Is(err, errors.ErrUnsupported) {
		t.Errorf("Seek: got %v, want an error wrapping errors.ErrUnsupported", err)
	}

	var got []byte
	for {
		var buf [4]byte
		n, err := s.Read(buf[:])
		if n == 0 && err == nil {
			t.Fatal("Read: got (0, <nil>), want a non-zero byte count or an error")
		}
		got = append(got, buf[:n]...)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if want := data[:len(data)/4*4]; !bytes.Equal(got, want) {
		t.Errorf("decoded %d bytes, want %d bytes", len(got), len(want))
	}
}

var errSourceRead = errors.New("wav_test: source read failed")

// failOnceReader returns the bytes before failAt together with errSourceRead the first time a Read
// reaches failAt.
type failOnceReader struct {
	r      *bytes.Reader
	failAt int64
}

func (f *failOnceReader) Read(buf []byte) (int, error) {
	pos := f.r.Size() - int64(f.r.Len())
	if f.failAt <= pos || f.failAt > pos+int64(len(buf)) {
		return f.r.Read(buf)
	}
	n, err := f.r.Read(buf[:f.failAt-pos])
	f.failAt = -1
	if err != nil {
		return n, err
	}
	return n, errSourceRead
}

func TestDecodePlaceholderDataChunkSizeSourceErrorWithData(t *testing.T) {
	data := make([]byte, 1002)
	for i := range data {
		data[i] = byte(i + 1)
	}
	patched := pcmWavFile(2, 16, data)
	setDataChunkSize(patched, 0)
	dataStart := bytes.Index(patched, []byte("data")) + 8

	s, err := wav.DecodeWithoutResampling(&failOnceReader{
		r:      bytes.NewReader(patched),
		failAt: int64(dataStart + 3*4 + 1),
	})
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 64)
	n, err := s.Read(buf)
	if !errors.Is(err, errSourceRead) {
		t.Errorf("Read: got error %v, want %v", err, errSourceRead)
	}
	if got, want := n, 3*4; got != want {
		t.Errorf("Read: got %d bytes, want %d", got, want)
	}
	if got, want := buf[:n], data[:n]; !bytes.Equal(got, want) {
		t.Errorf("Read: got %v, want %v", got, want)
	}

	rest, err := io.ReadAll(s)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := append(buf[:n:n], rest...), data[:len(data)/4*4]; !bytes.Equal(got, want) {
		t.Errorf("decoded %d bytes, want %d bytes", len(got), len(want))
	}
}

func TestDecodePlaceholderDataChunkSizeSeekUnsupported(t *testing.T) {
	file := pcmWavFile(2, 16, make([]byte, 1000))
	setDataChunkSize(file, 0)

	s, err := wav.DecodeWithoutResampling(&failingSeekSource{
		Reader: bytes.NewReader(file),
		err:    errors.New("wav_test: source seek failed"),
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, whence := range []int{io.SeekStart, io.SeekCurrent, io.SeekEnd} {
		if _, err := s.Seek(0, whence); !errors.Is(err, errors.ErrUnsupported) {
			t.Errorf("Seek(0, %d): got %v, want an error wrapping errors.ErrUnsupported", whence, err)
		}
	}
}

func TestSeekOverflow(t *testing.T) {
	for _, channels := range []int{1, 2} {
		for _, bits := range []int{8, 16} {
			for _, decode := range []struct {
				name string
				f    func(io.Reader) (*wav.Stream, error)
			}{
				{
					name: "Int16",
					f:    wav.DecodeWithoutResampling,
				},
				{
					name: "Float32",
					f:    wav.DecodeF32,
				},
				{
					name: "Resampled",
					f: func(src io.Reader) (*wav.Stream, error) {
						return wav.DecodeWithSampleRate(testSampleRate*2, src)
					},
				},
			} {
				t.Run(fmt.Sprintf("%s/channels=%d/bits=%d", decode.name, channels, bits), func(t *testing.T) {
					data := make([]byte, 8000)
					for i := range data {
						data[i] = byte(i)
					}
					s, err := decode.f(bytes.NewReader(pcmWavFile(channels, bits, data)))
					if err != nil {
						t.Fatal(err)
					}
					checkSeekOverflow(t, s)
				})
			}
		}
	}
}

func checkSeekOverflow(t *testing.T, r io.ReadSeeker) {
	t.Helper()
	want := make([]byte, 128)
	if _, err := io.ReadFull(r, want); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Seek(64, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	for _, whence := range []int{io.SeekCurrent, io.SeekEnd} {
		if _, err := r.Seek(math.MaxInt64, whence); err == nil {
			t.Errorf("Seek(MaxInt64, %d) succeeded", whence)
		}
		if pos, err := r.Seek(0, io.SeekCurrent); err != nil || pos != 64 {
			t.Errorf("position after rejected seek = (%d, %v), want 64", pos, err)
		}
	}
	got := make([]byte, 64)
	if _, err := io.ReadFull(r, got); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want[64:]) {
		t.Error("audio changed after rejected seeks")
	}
}

func TestSeekEOF(t *testing.T) {
	for _, channels := range []int{1, 2} {
		for _, bits := range []int{8, 16} {
			for _, decode := range []struct {
				name string
				f    func(io.Reader) (*wav.Stream, error)
			}{
				{
					name: "Int16",
					f:    wav.DecodeWithoutResampling,
				},
				{
					name: "Float32",
					f:    wav.DecodeF32,
				},
				{
					name: "Resampled",
					f: func(src io.Reader) (*wav.Stream, error) {
						return wav.DecodeWithSampleRate(testSampleRate*2, src)
					},
				},
			} {
				t.Run(fmt.Sprintf("%s/channels=%d/bits=%d", decode.name, channels, bits), func(t *testing.T) {
					data := make([]byte, 8000)
					for i := range data {
						data[i] = byte(i)
					}
					s, err := decode.f(bytes.NewReader(pcmWavFile(channels, bits, data)))
					if err != nil {
						t.Fatal(err)
					}
					checkEOFSeeks(t, s)
				})
			}
		}
	}
}

var wavDecoders = []struct {
	name           string
	f              func(io.Reader) (*wav.Stream, error)
	bytesPerSample int64
}{
	{
		name:           "Int16",
		f:              wav.DecodeWithoutResampling,
		bytesPerSample: 4,
	},
	{
		name:           "Float32",
		f:              wav.DecodeF32,
		bytesPerSample: 8,
	},
	{
		name: "Resampled",
		f: func(src io.Reader) (*wav.Stream, error) {
			return wav.DecodeWithSampleRate(testSampleRate*2, src)
		},
		bytesPerSample: 4,
	},
}

func TestSeekRounding(t *testing.T) {
	for _, channels := range []int{1, 2} {
		for _, bits := range []int{8, 16} {
			for _, decode := range wavDecoders {
				t.Run(fmt.Sprintf("%s/channels=%d/bits=%d", decode.name, channels, bits), func(t *testing.T) {
					data := make([]byte, 8000)
					for i := range data {
						data[i] = byte(i)
					}
					s, err := decode.f(bytes.NewReader(pcmWavFile(channels, bits, data)))
					if err != nil {
						t.Fatal(err)
					}
					size := decode.bytesPerSample
					base := s.Length() / 2 / size * size
					want := make([]byte, 4*size)
					if _, err := s.Seek(base, io.SeekStart); err != nil {
						t.Fatal(err)
					}
					if _, err := io.ReadFull(s, want); err != nil {
						t.Fatal(err)
					}

					for r := int64(1); r < size; r++ {
						for _, tc := range []struct {
							name   string
							from   int64
							offset int64
							whence int
						}{
							{
								name:   "SeekStart",
								from:   0,
								offset: base + r,
								whence: io.SeekStart,
							},
							{
								name:   "SeekCurrent",
								from:   0,
								offset: base + r,
								whence: io.SeekCurrent,
							},
							{
								name:   "SeekCurrentBackward",
								from:   base + size,
								offset: r - size,
								whence: io.SeekCurrent,
							},
							{
								name:   "SeekEnd",
								from:   0,
								offset: base + r - s.Length(),
								whence: io.SeekEnd,
							},
						} {
							if _, err := s.Seek(tc.from, io.SeekStart); err != nil {
								t.Fatal(err)
							}
							pos, err := s.Seek(tc.offset, tc.whence)
							if err != nil {
								t.Fatal(err)
							}
							if got, want := pos, base; got != want {
								t.Errorf("%s to %d: got %d, want %d", tc.name, base+r, got, want)
							}
							got := make([]byte, len(want))
							if _, err := io.ReadFull(s, got); err != nil {
								t.Fatal(err)
							}
							if !bytes.Equal(got, want) {
								t.Errorf("%s to %d: the data read differs from the data at %d", tc.name, base+r, base)
							}
						}

						pos, err := s.Seek(s.Length()+r, io.SeekStart)
						if err != nil {
							t.Fatal(err)
						}
						if got, want := pos, s.Length(); got != want {
							t.Errorf("Seek(Length()+%d, io.SeekStart): got %d, want %d", r, got, want)
						}
					}
				})
			}
		}
	}
}

func TestSeekCurrentAfterUnalignedRead(t *testing.T) {
	for _, channels := range []int{1, 2} {
		for _, bits := range []int{8, 16} {
			for _, decode := range wavDecoders {
				t.Run(fmt.Sprintf("%s/channels=%d/bits=%d", decode.name, channels, bits), func(t *testing.T) {
					data := make([]byte, 8000)
					for i := range data {
						data[i] = byte(i)
					}
					src := pcmWavFile(channels, bits, data)

					ref, err := decode.f(bytes.NewReader(src))
					if err != nil {
						t.Fatal(err)
					}
					want := make([]byte, 64)
					if _, err := io.ReadFull(ref, want); err != nil {
						t.Fatal(err)
					}

					s, err := decode.f(bytes.NewReader(src))
					if err != nil {
						t.Fatal(err)
					}
					n, err := s.Read(make([]byte, decode.bytesPerSample+1))
					if err != nil {
						t.Fatal(err)
					}
					pos, err := s.Seek(0, io.SeekCurrent)
					if err != nil {
						t.Fatal(err)
					}
					if got, want := pos, int64(n); got != want {
						t.Errorf("Seek(0, io.SeekCurrent) after reading %d bytes: got %d, want %d", n, got, want)
					}
					got := make([]byte, 16)
					if _, err := io.ReadFull(s, got); err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(got, want[n:n+len(got)]) {
						t.Errorf("reading after Seek(0, io.SeekCurrent) at %d: got %v, want %v", n, got, want[n:n+len(got)])
					}
				})
			}
		}
	}
}

// seekCountingReader is an io.ReadSeeker that counts its seeks.
type seekCountingReader struct {
	r     io.ReadSeeker
	seeks int
}

func (s *seekCountingReader) Read(buf []byte) (int, error) {
	return s.r.Read(buf)
}

func (s *seekCountingReader) Seek(offset int64, whence int) (int64, error) {
	s.seeks++
	return s.r.Seek(offset, whence)
}

func TestSeekCurrentDoesNotSeekSource(t *testing.T) {
	for _, channels := range []int{1, 2} {
		for _, bits := range []int{8, 16} {
			for _, decode := range wavDecoders {
				t.Run(fmt.Sprintf("%s/channels=%d/bits=%d", decode.name, channels, bits), func(t *testing.T) {
					data := make([]byte, 8000)
					for i := range data {
						data[i] = byte(i)
					}
					src := pcmWavFile(channels, bits, data)

					ref, err := decode.f(bytes.NewReader(src))
					if err != nil {
						t.Fatal(err)
					}
					want := readAligned(t, ref)

					r := &seekCountingReader{
						r: bytes.NewReader(src),
					}
					s, err := decode.f(r)
					if err != nil {
						t.Fatal(err)
					}
					got := make([]byte, 4*decode.bytesPerSample)
					if _, err := io.ReadFull(s, got); err != nil {
						t.Fatal(err)
					}
					if _, err := s.Read(make([]byte, 1)); !errors.Is(err, io.ErrShortBuffer) {
						t.Fatalf("Read(a buffer of 1 byte): got %v, want %v", err, io.ErrShortBuffer)
					}

					seeks := r.seeks
					pos, err := s.Seek(0, io.SeekCurrent)
					if err != nil {
						t.Fatal(err)
					}
					if pos != int64(len(got)) {
						t.Errorf("Seek(0, io.SeekCurrent): got %d, want %d", pos, len(got))
					}
					if r.seeks != seeks {
						t.Errorf("Seek(0, io.SeekCurrent) sought the source %d times, want 0", r.seeks-seeks)
					}
					got = append(got, readAligned(t, s)...)
					if !bytes.Equal(got, want) {
						t.Error("reading after Seek(0, io.SeekCurrent): the data differs from reading without the query")
					}

					seeks = r.seeks
					pos, err = s.Seek(0, io.SeekCurrent)
					if err != nil {
						t.Fatal(err)
					}
					if pos != int64(len(want)) {
						t.Errorf("Seek(0, io.SeekCurrent) at the end: got %d, want %d", pos, len(want))
					}
					if n, err := s.Read(make([]byte, 64)); n != 0 || !errors.Is(err, io.EOF) {
						t.Errorf("Read after Seek(0, io.SeekCurrent) at the end: got (%d, %v), want (0, %v)", n, err, io.EOF)
					}
					if r.seeks != seeks {
						t.Errorf("Seek(0, io.SeekCurrent) at the end sought the source %d times, want 0", r.seeks-seeks)
					}
				})
			}
		}
	}
}

func checkEOFSeeks(t *testing.T, s interface {
	io.ReadSeeker
	Length() int64
}) {
	t.Helper()
	want := make([]byte, 64)
	if _, err := io.ReadFull(s, want); err != nil {
		t.Fatal(err)
	}
	for _, target := range []int64{s.Length(), s.Length() + 32, 1 << 40, math.MaxInt64/8*8 - 1024} {
		for _, whence := range []int{io.SeekStart, io.SeekCurrent, io.SeekEnd} {
			if _, err := s.Seek(0, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			offset := target
			if whence == io.SeekEnd {
				offset -= s.Length()
			}
			pos, err := s.Seek(offset, whence)
			if err != nil {
				t.Fatalf("Seek(%d, %d): %v", offset, whence, err)
			}
			if pos != target {
				t.Errorf("Seek(%d, %d) = %d, want %d", offset, whence, pos, target)
			}
			buf := make([]byte, 64)
			for _, size := range []int{1, 3, len(buf), len(buf)} {
				if n, err := s.Read(buf[:size]); n != 0 || !errors.Is(err, io.EOF) {
					t.Errorf("Read at %d = (%d, %v), want (0, EOF)", target, n, err)
				}
			}
			if pos, err := s.Seek(0, io.SeekCurrent); err != nil || pos != target {
				t.Errorf("current = (%d, %v), want %d", pos, err, target)
			}
			if pos, err := s.Seek(8, io.SeekCurrent); err != nil || pos != target+8 {
				t.Errorf("advance = (%d, %v), want %d", pos, err, target+8)
			}
			if pos, err := s.Seek(-target-8, io.SeekCurrent); err != nil || pos != 0 {
				t.Fatalf("seek back = (%d, %v)", pos, err)
			}
			if _, err := io.ReadFull(s, buf); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(buf, want) {
				t.Error("reading after seeking back returned different bytes")
			}
		}
	}
}

type failingSeekSource struct {
	*bytes.Reader
	err error
}

func (s *failingSeekSource) Seek(offset int64, whence int) (int64, error) {
	if s.err != nil {
		return 0, s.err
	}
	return s.Reader.Seek(offset, whence)
}

func TestSeekEOFSourceError(t *testing.T) {
	for _, decode := range []struct {
		name string
		f    func(io.Reader) (*wav.Stream, error)
	}{
		{
			name: "Int16",
			f:    wav.DecodeWithoutResampling,
		},
		{
			name: "Float32",
			f:    wav.DecodeF32,
		},
	} {
		t.Run(decode.name, func(t *testing.T) {
			src := &failingSeekSource{
				Reader: bytes.NewReader(pcmWavFile(2, 16, make([]byte, 8000))),
			}
			s, err := decode.f(src)
			if err != nil {
				t.Fatal(err)
			}
			sentinel := errors.New("source seek failed")
			src.err = sentinel
			for _, pos := range []int64{s.Length(), s.Length() + 32, 1 << 40} {
				if _, err := s.Seek(pos, io.SeekStart); !errors.Is(err, sentinel) {
					t.Errorf("Seek(%d, SeekStart): got %v, want source error", pos, err)
				}
			}
			src.err = nil
			if _, err := s.Seek(0, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			if _, err := io.ReadFull(s, make([]byte, 64)); err != nil {
				t.Error(err)
			}
		})
	}
}

func readWithSizes(t *testing.T, r io.Reader, sizes []int, bytesPerSample int) []byte {
	t.Helper()
	var got []byte
	for i := 0; ; i++ {
		size := sizes[i%len(sizes)]
		buf := make([]byte, size)
		n, err := r.Read(buf)
		if n%bytesPerSample != 0 {
			t.Errorf("Read(a buffer of %d bytes) at %d: got %d bytes, want a multiple of %d", size, len(got), n, bytesPerSample)
		}
		got = append(got, buf[:n]...)
		if errors.Is(err, io.EOF) {
			return got
		}
		if size < bytesPerSample && errors.Is(err, io.ErrShortBuffer) {
			continue
		}
		if err != nil {
			t.Fatalf("Read(a buffer of %d bytes) at %d: %v", size, len(got), err)
		}
		if n == 0 {
			t.Fatalf("Read(a buffer of %d bytes) at %d: got (0, <nil>)", size, len(got))
		}
	}
}

func readAligned(t *testing.T, r io.Reader) []byte {
	t.Helper()
	var got []byte
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		got = append(got, buf[:n]...)
		if errors.Is(err, io.EOF) {
			return got
		}
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestReadWholeSamples(t *testing.T) {
	decoders := append(wavDecoders, struct {
		name           string
		f              func(io.Reader) (*wav.Stream, error)
		bytesPerSample int64
	}{
		name: "SameSampleRate",
		f: func(src io.Reader) (*wav.Stream, error) {
			return wav.DecodeWithSampleRate(testSampleRate, src)
		},
		bytesPerSample: 4,
	})
	data := make([]byte, 1003)
	for i := range data {
		data[i] = byte(i*7 + 1)
	}
	for _, channels := range []int{1, 2} {
		for _, bits := range []int{8, 16} {
			for _, placeholderSize := range []bool{false, true} {
				for _, decode := range decoders {
					t.Run(fmt.Sprintf("%s/channels=%d/bits=%d/placeholderSize=%t", decode.name, channels, bits, placeholderSize), func(t *testing.T) {
						file := pcmWavFile(channels, bits, data)
						if placeholderSize {
							setDataChunkSize(file, 0)
						}
						ref, err := decode.f(bytes.NewReader(file))
						if err != nil {
							t.Fatal(err)
						}
						want := readAligned(t, ref)

						s, err := decode.f(&shortReader{
							r:    bytes.NewReader(file),
							maxN: 1,
						})
						if err != nil {
							t.Fatal(err)
						}
						got := readWithSizes(t, s, []int{1, 5, 3, 6, 2, 7, 9, 13, 4, 10, 8, 1027}, int(decode.bytesPerSample))
						if !bytes.Equal(got, want) {
							t.Errorf("got %d bytes, want %d bytes equal to reading with aligned buffers", len(got), len(want))
						}
					})
				}
			}
		}
	}
}
