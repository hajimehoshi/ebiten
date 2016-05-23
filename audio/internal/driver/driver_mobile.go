// Copyright 2016 Hajime Hoshi
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

// +build android

package driver

import (
	"fmt"
	"runtime"

	"golang.org/x/mobile/exp/audio/al"
)

const (
	maxBufferNum = 8
)

type Player struct {
	alSource   al.Source
	alBuffers  []al.Buffer
	sampleRate int
	isClosed   bool
	alFormat   uint32
}

func alFormat(channelNum, bytesPerSample int) uint32 {
	switch {
	case channelNum == 1 && bytesPerSample == 1:
		return al.FormatMono8
	case channelNum == 1 && bytesPerSample == 2:
		return al.FormatMono16
	case channelNum == 2 && bytesPerSample == 1:
		return al.FormatStereo8
	case channelNum == 2 && bytesPerSample == 2:
		return al.FormatStereo16
	}
	panic(fmt.Sprintf("driver: invalid channel num (%d) or bytes per sample (%d)", channelNum, bytesPerSample))
}

func NewPlayer(sampleRate, channelNum, bytesPerSample int) (*Player, error) {
	var p *Player
	if err := al.OpenDevice(); err != nil {
		return nil, fmt.Errorf("driver: OpenAL initialization failed: %v", err)
	}
	s := al.GenSources(1)
	if e := al.Error(); e != 0 {
		return nil, fmt.Errorf("driver: al.GenSources error: %d", e)
	}
	p = &Player{
		alSource:   s[0],
		alBuffers:  []al.Buffer{},
		sampleRate: sampleRate,
		alFormat:   alFormat(channelNum, bytesPerSample),
	}
	runtime.SetFinalizer(p, (*Player).Close)

	bs := al.GenBuffers(maxBufferNum)
	const bufferSize = 1024
	emptyBytes := make([]byte, bufferSize)
	for _, b := range bs {
		// Note that the third argument of only the first buffer is used.
		b.BufferData(p.alFormat, emptyBytes, int32(p.sampleRate))
		p.alSource.QueueBuffers(b)
	}
	al.PlaySources(p.alSource)
	return p, nil
}

func (p *Player) Proceed(data []byte) error {
	if err := al.Error(); err != 0 {
		return fmt.Errorf("driver: before proceed: %d", err)
	}
	processedNum := p.alSource.BuffersProcessed()
	if 0 < processedNum {
		bufs := make([]al.Buffer, processedNum)
		p.alSource.UnqueueBuffers(bufs...)
		if err := al.Error(); err != 0 {
			return fmt.Errorf("driver: Unqueue in process: %d", err)
		}
		p.alBuffers = append(p.alBuffers, bufs...)
	}

	if len(p.alBuffers) == 0 {
		// This can happen (#207)
		return nil
	}
	buf := p.alBuffers[0]
	p.alBuffers = p.alBuffers[1:]
	buf.BufferData(p.alFormat, data, int32(p.sampleRate))
	p.alSource.QueueBuffers(buf)
	if err := al.Error(); err != 0 {
		return fmt.Errorf("driver: Queue in process: %d", err)
	}

	if p.alSource.State() == al.Stopped || p.alSource.State() == al.Initial {
		al.RewindSources(p.alSource)
		al.PlaySources(p.alSource)
		if err := al.Error(); err != 0 {
			return fmt.Errorf("driver: PlaySource in process: %d", err)
		}
	}

	return nil
}

func (p *Player) Close() error {
	if err := al.Error(); err != 0 {
		return fmt.Errorf("driver: error before closing: %d", err)
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
		return fmt.Errorf("driver: error after closing: %d", err)
	}
	runtime.SetFinalizer(p, nil)
	return nil
}
