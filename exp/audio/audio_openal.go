// Copyright 2015 Hajime Hoshi
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

// +build !js,!windows

package audio

import (
	"fmt"
	"io"
	"runtime"

	"golang.org/x/mobile/exp/audio/al"
)

const (
	maxBufferNum = 8
)

type player struct {
	alSource   al.Source
	alBuffers  []al.Buffer
	source     io.Reader
	sampleRate int
	isClosed   bool
}

func newPlayer(src io.Reader, sampleRate int) (*player, error) {
	if e := al.OpenDevice(); e != nil {
		return nil, fmt.Errorf("audio: OpenAL initialization failed: %v", e)
	}
	s := al.GenSources(1)
	if err := al.Error(); err != 0 {
		return nil, fmt.Errorf("audio: al.GenSources error: %d", err)
	}
	p := &player{
		alSource:   s[0],
		alBuffers:  []al.Buffer{},
		source:     src,
		sampleRate: sampleRate,
	}
	runtime.SetFinalizer(p, (*player).close)

	bs := al.GenBuffers(maxBufferNum)
	emptyBytes := make([]byte, bufferSize)
	for _, b := range bs {
		// Note that the third argument of only the first buffer is used.
		b.BufferData(al.FormatStereo16, emptyBytes, int32(p.sampleRate))
		p.alSource.QueueBuffers(b)
	}
	al.PlaySources(p.alSource)
	return p, nil
}

const (
	bufferSize = 1024
)

var (
	tmpBuffer    = make([]byte, bufferSize)
	tmpAlBuffers = make([]al.Buffer, maxBufferNum)
)

func (p *player) proceed() error {
	if err := al.Error(); err != 0 {
		return fmt.Errorf("audio: before proceed: %d", err)
	}
	processedNum := p.alSource.BuffersProcessed()
	if 0 < processedNum {
		bufs := tmpAlBuffers[:processedNum]
		p.alSource.UnqueueBuffers(bufs...)
		if err := al.Error(); err != 0 {
			return fmt.Errorf("audio: Unqueue in process: %d", err)
		}
		p.alBuffers = append(p.alBuffers, bufs...)
	}

	if 0 < len(p.alBuffers) {
		n, err := p.source.Read(tmpBuffer)
		if 0 < n {
			buf := p.alBuffers[0]
			p.alBuffers = p.alBuffers[1:]
			buf.BufferData(al.FormatStereo16, tmpBuffer[:n], int32(p.sampleRate))
			p.alSource.QueueBuffers(buf)
			if err := al.Error(); err != 0 {
				return fmt.Errorf("audio: Queue in process: %d", err)
			}
		}
		if err != nil {
			return err
		}
	}

	if p.alSource.State() == al.Stopped || p.alSource.State() == al.Initial {
		al.RewindSources(p.alSource)
		al.PlaySources(p.alSource)
		if err := al.Error(); err != 0 {
			return fmt.Errorf("audio: PlaySource in process: %d", err)
		}
	}

	return nil
}

func (p *player) close() error {
	if err := al.Error(); err != 0 {
		return fmt.Errorf("audio: error before closing: %d", err)
	}
	if p.isClosed {
		return nil
	}
	var bs []al.Buffer
	al.RewindSources(p.alSource)
	al.StopSources(p.alSource)
	if n := p.alSource.BuffersQueued(); 0 < n {
		bs = make([]al.Buffer, n)
		p.alSource.UnqueueBuffers(bs...)
		p.alBuffers = append(p.alBuffers, bs...)
	}
	p.isClosed = true
	if err := al.Error(); err != 0 {
		return fmt.Errorf("audio: error after closing: %d", err)
	}
	runtime.SetFinalizer(p, nil)
	return nil
}
