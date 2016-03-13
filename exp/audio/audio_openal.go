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
	"time"

	"golang.org/x/mobile/exp/audio/al"
)

const (
	maxBufferNum = 8
)

// TODO: This should be in player
var totalBufferNum = 0

type player struct {
	alSource   al.Source
	alBuffers  []al.Buffer
	source     io.Reader
	sampleRate int
	isClosed   bool
}

func startPlaying(src io.Reader, sampleRate int) (*player, error) {
	if e := al.OpenDevice(); e != nil {
		return nil, fmt.Errorf("audio: OpenAL initialization failed: %v", e)
	}
	s := al.GenSources(1)
	if err := al.Error(); err != 0 {
		panic(fmt.Sprintf("audio: al.GenSources error: %d", err))
	}
	p := &player{
		alSource:   s[0],
		alBuffers:  []al.Buffer{},
		source:     src,
		sampleRate: sampleRate,
	}
	runtime.SetFinalizer(p, (*player).close)
	if err := p.start(); err != nil {
		return nil, err
	}
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
		panic(fmt.Sprintf("audio: before proceed: %d", err))
	}
	processedNum := p.alSource.BuffersProcessed()
	if 0 < processedNum {
		bufs := tmpAlBuffers[:processedNum]
		p.alSource.UnqueueBuffers(bufs...)
		if err := al.Error(); err != 0 {
			panic(fmt.Sprintf("audio: Unqueue in process: %d", err))
		}
		p.alBuffers = append(p.alBuffers, bufs...)
	}

	for 0 < len(p.alBuffers) {
		n, err := p.source.Read(tmpBuffer)
		if 0 < n {
			buf := p.alBuffers[0]
			p.alBuffers = p.alBuffers[1:]
			buf.BufferData(al.FormatStereo16, tmpBuffer[:n], int32(p.sampleRate))
			p.alSource.QueueBuffers(buf)
			if err := al.Error(); err != 0 {
				panic(fmt.Sprintf("audio: Queue in process: %d", err))
			}
		}
		if err != nil {
			return err
		}
		if n == 0 {
			time.Sleep(1)
		}
	}

	if p.alSource.State() == al.Stopped {
		al.RewindSources(p.alSource)
		al.PlaySources(p.alSource)
		if err := al.Error(); err != 0 {
			panic(fmt.Sprintf("audio: PlaySource in process: %d", err))
		}
	}

	return nil
}

func (p *player) start() error {
	n := maxBufferNum - int(p.alSource.BuffersQueued()) - len(p.alBuffers)
	if 0 < n {
		p.alBuffers = append(p.alBuffers, al.GenBuffers(n)...)
		totalBufferNum += n
		if maxBufferNum < totalBufferNum {
			panic("audio: too many buffers are created")
		}
	}
	if 0 < len(p.alBuffers) {
		emptyBytes := make([]byte, bufferSize)
		for _, buf := range p.alBuffers {
			// Note that the third argument of only the first buffer is used.
			buf.BufferData(al.FormatStereo16, emptyBytes, int32(p.sampleRate))
			p.alSource.QueueBuffers(buf)
		}
		p.alBuffers = []al.Buffer{}
	}
	al.PlaySources(p.alSource)

	go func() {
		// TODO: Is it OK to close asap?
		defer p.close()
		for {
			err := p.proceed()
			if err == io.EOF {
				break
			}
			if err != nil {
				// TODO: Record the last error
				panic(err)
			}
			runtime.Gosched()
		}
	}()
	return nil
}

// TODO: When is this called? Can we remove this?
func (p *player) close() error {
	if err := al.Error(); err != 0 {
		panic(fmt.Sprintf("audio: error before closing: %d", err))
	}
	if p.isClosed {
		return nil
	}
	var bs []al.Buffer
	al.RewindSources(p.alSource)
	al.StopSources(p.alSource)
	n := p.alSource.BuffersQueued()
	if 0 < n {
		bs = make([]al.Buffer, n)
		p.alSource.UnqueueBuffers(bs...)
		p.alBuffers = append(p.alBuffers, bs...)
	}
	p.isClosed = true
	if err := al.Error(); err != 0 {
		panic(fmt.Sprintf("audio: error after closing: %d", err))
	}
	runtime.SetFinalizer(p, nil)
	return nil
}
