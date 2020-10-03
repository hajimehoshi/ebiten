// Copyright 2018 The Ebiten Authors
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

package audio_test

import (
	"io"
	"math"
	"testing"

	. "github.com/hajimehoshi/ebiten/v2/audio"
)

func TestInfiniteLoop(t *testing.T) {
	indexToByte := func(index int) byte {
		return byte(math.Sin(float64(index)) * 256)
	}

	src := make([]byte, 256)
	for i := range src {
		src[i] = indexToByte(i)
	}
	l := NewInfiniteLoop(BytesReadSeekCloser(src), int64(len(src)))

	buf := make([]byte, len(src)*4)
	if _, err := io.ReadFull(l, buf); err != nil {
		t.Error(err)
	}
	for i, b := range buf {
		got := b
		want := indexToByte(i % len(src))
		if got != want {
			t.Errorf("index: %d, got: %v, want: %v", i, got, want)
		}
	}

	n, err := l.Seek(int64(len(src))*5+128, io.SeekStart)
	if err != nil {
		t.Error(err)
	}
	if want := int64(128); n != want {
		t.Errorf("got: %v, want: %v", n, want)
	}

	n2, err := l.Seek(int64(len(src))*6+64, io.SeekCurrent)
	if err != nil {
		t.Error(err)
	}
	if want := int64(192); n2 != want {
		t.Errorf("got: %v, want: %v", n, want)
	}

	buf2 := make([]byte, len(src)*7)
	if _, err := io.ReadFull(l, buf2); err != nil {
		t.Error(err)
	}
	for i, b := range buf2 {
		got := b
		want := indexToByte((i + 192) % len(src))
		if got != want {
			t.Errorf("index: %d, got: %v, want: %v", i, got, want)
		}
	}

	// Seek to negative position is an error.
	if _, err := l.Seek(-1, io.SeekStart); err == nil {
		t.Errorf("got: %v, want: %v", err, nil)
	}
}

func TestInfiniteLoopWithIntro(t *testing.T) {
	const (
		srcLength   = 17 * 4
		introLength = 19 * 4
		loopLength  = 23 * 4
	)

	indexToByte := func(index int) byte {
		return byte(math.Sin(float64(index)) * 256)
	}
	src := make([]byte, srcLength)
	for i := range src {
		src[i] = indexToByte(i)
	}
	srcInf := NewInfiniteLoop(BytesReadSeekCloser(src), srcLength)
	l := NewInfiniteLoopWithIntro(srcInf, introLength, loopLength)

	buf := make([]byte, srcLength*4)
	if _, err := io.ReadFull(l, buf); err != nil {
		t.Error(err)
	}
	for i, b := range buf {
		got := b
		want := byte(0)
		if i < introLength {
			want = indexToByte(i % srcLength)
		} else {
			want = indexToByte(((i-introLength)%loopLength + introLength) % srcLength)
		}
		if got != want {
			t.Errorf("index: %d, got: %v, want: %v", i, got, want)
		}
	}

	n, err := l.Seek(srcLength*5+128, io.SeekStart)
	if err != nil {
		t.Error(err)
	}
	if want := int64((srcLength*5+128-introLength)%loopLength + introLength); n != want {
		t.Errorf("got: %v, want: %v", n, want)
	}

	n2, err := l.Seek(srcLength*6+64, io.SeekCurrent)
	if err != nil {
		t.Error(err)
	}
	if want := int64(((srcLength*11+192)-introLength)%loopLength + introLength); n2 != want {
		t.Errorf("got: %v, want: %v", n, want)
	}

	buf2 := make([]byte, srcLength*7)
	if _, err := io.ReadFull(l, buf2); err != nil {
		t.Error(err)
	}
	for i, b := range buf2 {
		got := b
		idx := ((int(n2+int64(i))-introLength)%loopLength + introLength) % srcLength
		want := indexToByte(idx)
		if got != want {
			t.Errorf("index: %d, got: %v, want: %v", i, got, want)
		}
	}

	// Seek to negative position is an error.
	if _, err := l.Seek(-1, io.SeekStart); err == nil {
		t.Errorf("got: %v, want: %v", err, nil)
	}
}
