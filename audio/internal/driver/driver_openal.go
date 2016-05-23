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

// +build !js,!windows,!android

package driver

import (
	"fmt"
	"runtime"

	"github.com/hajimehoshi/go-openal/openal"
)

// As x/mobile/exp/audio/al is broken on Mac OS X (https://github.com/golang/go/issues/15075),
// let's use github.com/hajimehoshi/go-openal instead.

const (
	maxBufferNum = 8
)

type Player struct {
	alDevice   *openal.Device
	alSource   openal.Source
	alBuffers  []openal.Buffer
	sampleRate int
	isClosed   bool
	alFormat   openal.Format
}

func alFormat(channelNum, bytesPerSample int) openal.Format {
	switch {
	case channelNum == 1 && bytesPerSample == 1:
		return openal.FormatMono8
	case channelNum == 1 && bytesPerSample == 2:
		return openal.FormatMono16
	case channelNum == 2 && bytesPerSample == 1:
		return openal.FormatStereo8
	case channelNum == 2 && bytesPerSample == 2:
		return openal.FormatStereo16
	}
	panic(fmt.Sprintf("driver: invalid channel num (%d) or bytes per sample (%d)", channelNum, bytesPerSample))
}

func NewPlayer(sampleRate, channelNum, bytesPerSample int) (*Player, error) {
	d := openal.OpenDevice("")
	if d == nil {
		return nil, fmt.Errorf("driver: OpenDevice must not return nil")
	}
	c := d.CreateContext()
	if c == nil {
		return nil, fmt.Errorf("driver: CreateContext must not return nil")
	}
	// Don't check openal.Err until making the current context is done.
	// Linux might fail this check even though it succeeds (#204).
	c.Activate()
	if err := openal.Err(); err != nil {
		return nil, fmt.Errorf("driver: Activate: %v", err)
	}
	s := openal.NewSource()
	if err := openal.Err(); err != nil {
		return nil, fmt.Errorf("driver: NewSource: %v", err)
	}
	p := &Player{
		alDevice:   d,
		alSource:   s,
		alBuffers:  []openal.Buffer{},
		sampleRate: sampleRate,
		alFormat:   alFormat(channelNum, bytesPerSample),
	}
	runtime.SetFinalizer(p, (*Player).Close)

	bs := openal.NewBuffers(maxBufferNum)
	const bufferSize = 1024
	emptyBytes := make([]byte, bufferSize)
	for _, b := range bs {
		// Note that the third argument of only the first buffer is used.
		b.SetData(p.alFormat, emptyBytes, int32(p.sampleRate))
		p.alBuffers = append(p.alBuffers, b)
	}
	p.alSource.Play()
	return p, nil
}

func (p *Player) Proceed(data []byte) error {
	if err := openal.Err(); err != nil {
		return fmt.Errorf("driver: starting Proceed: %v", err)
	}
	processedNum := p.alSource.BuffersProcessed()
	if 0 < processedNum {
		bufs := make([]openal.Buffer, processedNum)
		p.alSource.UnqueueBuffers(bufs)
		if err := openal.Err(); err != nil {
			return fmt.Errorf("driver: UnqueueBuffers: %v", err)
		}
		p.alBuffers = append(p.alBuffers, bufs...)
	}

	if len(p.alBuffers) == 0 {
		// This can happen (#207)
		return nil
	}
	buf := p.alBuffers[0]
	p.alBuffers = p.alBuffers[1:]
	buf.SetData(p.alFormat, data, int32(p.sampleRate))
	p.alSource.QueueBuffer(buf)
	if err := openal.Err(); err != nil {
		return fmt.Errorf("driver: QueueBuffer: %v", err)
	}

	if p.alSource.State() == openal.Stopped || p.alSource.State() == openal.Initial {
		p.alSource.Rewind()
		p.alSource.Play()
		if err := openal.Err(); err != nil {
			return fmt.Errorf("driver: Rewind or Play: %v", err)
		}
	}

	return nil
}

func (p *Player) Close() error {
	if err := openal.Err(); err != nil {
		return fmt.Errorf("driver: starting Close: %v", err)
	}
	if p.isClosed {
		return nil
	}
	var bs []openal.Buffer
	p.alSource.Rewind()
	p.alSource.Play()
	if n := p.alSource.BuffersQueued(); 0 < n {
		bs = make([]openal.Buffer, n)
		p.alSource.UnqueueBuffers(bs)
		p.alBuffers = append(p.alBuffers, bs...)
	}
	p.alDevice.CloseDevice()
	p.isClosed = true
	if err := openal.Err(); err != nil {
		return fmt.Errorf("driver: CloseDevice: %v", err)
	}
	runtime.SetFinalizer(p, nil)
	return nil
}
