// Copyright 2025 The Ebitengine Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package wav_test

import (
	"bytes"
	"io"
	"testing"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

func makeWav(channelCount int, bitsPerSample int, sampleRate int, data []byte) []byte {
	var sb bytes.Buffer
	sb.WriteString("RIFF")
	sb.Write([]byte{0, 0, 0, 0}) // `decode` does not use file size
	sb.WriteString("WAVE")

	sb.WriteString("fmt ")

	format := 1
	fmtData := []byte{
		byte(format),
		byte(format >> 8),
		byte(channelCount),
		byte(channelCount >> 8),
		byte(sampleRate),
		byte(sampleRate >> 8),
		byte(sampleRate >> 16),
		byte(sampleRate >> 24),
		0, 0, 0, 0, // `decode` does not use dwAvgBytesPerSec
		0, 0, // `decdode` does not use wBlockAlign
		byte(bitsPerSample),
		byte(bitsPerSample >> 8),
	}
	lenFmtData := len(fmtData)
	sb.Write([]byte{
		byte(lenFmtData),
		byte(lenFmtData >> 8),
		byte(lenFmtData >> 16),
		byte(lenFmtData >> 24),
	})
	sb.Write(fmtData)

	sb.WriteString("data")
	lenData := len(data)
	sb.Write([]byte{
		byte(lenData),
		byte(lenData >> 8),
		byte(lenData >> 16),
		byte(lenData >> 24),
	})
	sb.Write(data)
	return sb.Bytes()
}

func int16ToBytes(a []int16) []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(a))), len(a)*2)
}

func int16ToFloat32(a []byte) []float32 {
	outF32 := make([]float32, len(a)/2)
	for i := 0; i < len(a); i += 2 {
		vi16l := a[i]
		vi16h := a[i+1]
		outF32[i/2] = float32(int16(vi16l)|int16(vi16h)<<8) / (1 << 15)
	}
	return outF32
}

func byteToFloat32(a []byte) []float32 {
	outF32 := make([]float32, len(a))
	for i := 0; i < len(a); i += 1 {
		// This converts the byte into a int16, and then into a float32
		// This gives slightly different floats than converting from byte to float32 directly
		// but is kept this way as that is what the package used to do
		v := int16(int(a[i])*0x101 - (1 << 15))
		outF32[i] = float32(int32(v)) / (1 << 15)
	}
	return outF32
}

func int24ToBytes(a []int32) []byte {
	out := make([]byte, len(a)*3)
	for i := range a {
		v := a[i] * 1 << 8
		out[i*3] = byte(v >> 8)
		out[i*3+1] = byte(v >> 16)
		out[i*3+2] = byte(v >> 24)
	}
	return out
}

func int24ToFloat32(a []byte) []float32 {
	outF32 := make([]float32, len(a)/3)
	for i := 0; i < len(a); i += 3 {
		var r int32
		r |= int32(a[i]) << 8
		r |= int32(a[i+1]) << 16
		r |= int32(a[i+2]) << 24
		r = r / 1 >> 8
		outF32[i/3] = float32(r) / (1 << 23)
	}
	return outF32
}

func toStereo(conv func(a []byte) []float32) func(a []byte) []float32 {
	return func(a []byte) []float32 {
		mono := conv(a)
		out := make([]float32, len(mono)*2)
		for i := range mono {
			out[i*2] = mono[i]
			out[i*2+1] = mono[i]
		}
		return out
	}
}
func TestDecodeF32(t *testing.T) {
	type testCase struct {
		Name          string
		Data          []byte
		ChannelCount  int
		BitsPerSample int
		SampleRate    int
		Converter     func([]byte) []float32
	}
	cases := []testCase{
		{
			Name:          "Stereo, 24 Bit",
			Data:          int24ToBytes([]int32{-8388608, -8388608, 0, 0, 8388607, 8388607}),
			ChannelCount:  2,
			BitsPerSample: 24,
			SampleRate:    44100,
			Converter:     int24ToFloat32,
		},
		{
			Name:          "Mono, 24 Bit",
			Data:          int24ToBytes([]int32{-8388608, 0, 8388607}),
			ChannelCount:  1,
			BitsPerSample: 24,
			SampleRate:    44100,
			Converter:     toStereo(int24ToFloat32),
		},
		{
			Name:          "Stereo, 16 Bit",
			Data:          int16ToBytes([]int16{-32768, -32768, 0, 0, 32767, 32767}),
			ChannelCount:  2,
			BitsPerSample: 16,
			SampleRate:    44100,
			Converter:     int16ToFloat32,
		},
		{
			Name:          "Mono, 16 Bit",
			Data:          int16ToBytes([]int16{-32768, 0, 32767}),
			ChannelCount:  1,
			BitsPerSample: 16,
			SampleRate:    44100,
			Converter:     toStereo(int16ToFloat32),
		},
		{
			Name:          "Stereo, 8 Bit",
			Data:          []byte{0, 0, 128, 128, 255, 255},
			ChannelCount:  2,
			BitsPerSample: 8,
			SampleRate:    44100,
			Converter:     byteToFloat32,
		},
		{
			Name:          "Mono, 8 Bit",
			Data:          []byte{0, 128, 255},
			ChannelCount:  1,
			BitsPerSample: 8,
			SampleRate:    44100,
			Converter:     toStereo(byteToFloat32),
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			var out []byte
			if len(c.Data) > 0 {
				outF32 := c.Converter(c.Data)
				out = unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(outF32))), len(outF32)*4)
			}

			in := makeWav(c.ChannelCount, c.BitsPerSample, c.SampleRate, c.Data)

			stream, err := wav.DecodeF32(bytes.NewReader(in))
			if err != nil {
				t.Fatal(err)
			}
			got, err := io.ReadAll(stream)
			if err != nil {
				t.Fatal(err)
			}
			want := out
			if !bytes.Equal(got, want) {
				t.Errorf("got: %v, want: %v", got, want)
			}
		})
	}
}
