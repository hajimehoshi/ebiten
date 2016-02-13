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
	"sync"
	"time"

	"golang.org/x/mobile/exp/audio/al"
)

const (
	maxSourceNum = 32
	maxBufferNum = 8
)

var totalBufferNum = 0

var playerCache = []*player{}

type player struct {
	alSource   al.Source
	alBuffers  []al.Buffer
	source     io.ReadSeeker
	sampleRate int
	isClosed   bool
}

var m sync.Mutex

func newPlayerFromCache(src io.ReadSeeker, sampleRate int) (*player, error) {
	for _, p := range playerCache {
		if p.sampleRate != sampleRate {
			continue
		}
		if !p.isClosed {
			continue
		}
		p.source = src
		p.isClosed = false
		return p, nil
	}
	if maxSourceNum <= len(playerCache) {
		return nil, ErrTooManyPlayers
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
	playerCache = append(playerCache, p)
	return p, nil
}

func newPlayer(src io.ReadSeeker, sampleRate int) (*Player, error) {
	m.Lock()
	defer m.Unlock()

	if e := al.OpenDevice(); e != nil {
		m.Unlock()
		return nil, fmt.Errorf("audio: OpenAL initialization failed: %v", e)
	}
	p, err := newPlayerFromCache(src, sampleRate)
	if err != nil {
		return nil, err
	}
	return &Player{p}, nil
}

const bufferSize = 1024

func (p *player) proceed() error {
	m.Lock()
	if err := al.Error(); err != 0 {
		panic(fmt.Sprintf("audio: before proceed: %d", err))
	}
	processedNum := p.alSource.BuffersProcessed()
	if 0 < processedNum {
		bufs := make([]al.Buffer, processedNum)
		p.alSource.UnqueueBuffers(bufs...)
		if err := al.Error(); err != 0 {
			panic(fmt.Sprintf("audio: Unqueue in process: %d", err))
		}
		p.alBuffers = append(p.alBuffers, bufs...)
	}
	m.Unlock()

	for 0 < len(p.alBuffers) {
		b := make([]byte, bufferSize)
		n, err := p.source.Read(b)
		if 0 < n {
			m.Lock()
			buf := p.alBuffers[0]
			p.alBuffers = p.alBuffers[1:]
			buf.BufferData(al.FormatStereo16, b[:n], int32(p.sampleRate))
			p.alSource.QueueBuffers(buf)
			if err := al.Error(); err != 0 {
				panic(fmt.Sprintf("audio: Queue in process: %d", err))
			}
			m.Unlock()
		}
		if err != nil {
			return err
		}
	}

	m.Lock()
	if p.alSource.State() == al.Stopped {
		al.RewindSources(p.alSource)
		al.PlaySources(p.alSource)
		if err := al.Error(); err != 0 {
			panic(fmt.Sprintf("audio: PlaySource in process: %d", err))
		}
	}
	m.Unlock()

	return nil
}

func (p *player) play() error {
	// TODO: What if play is already called?
	m.Lock()
	n := maxBufferNum - int(p.alSource.BuffersQueued()) - len(p.alBuffers)
	if 0 < n {
		p.alBuffers = append(p.alBuffers, al.GenBuffers(n)...)
		totalBufferNum += n
		if maxSourceNum*maxBufferNum < totalBufferNum {
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
	m.Unlock()

	go func() {
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
			time.Sleep(1)
		}
	}()
	return nil
}

func (p *player) close() error {
	m.Lock()
	defer m.Unlock()

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
