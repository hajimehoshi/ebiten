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

type alSourceCacheEntry struct {
	source     al.Source
	sampleRate int
}

const maxSourceNum = 32

var alSourceCache = []alSourceCacheEntry{}

type player struct {
	alSource   al.Source
	alBuffers  []al.Buffer
	source     io.ReadSeeker
	sampleRate int
}

var m sync.Mutex

func newAlSource(sampleRate int) (al.Source, error) {
	for _, e := range alSourceCache {
		if e.sampleRate != sampleRate {
			continue
		}
		s := e.source.State()
		if s != al.Initial && s != al.Stopped {
			continue
		}
		return e.source, nil
	}
	if maxSourceNum <= len(alSourceCache) {
		return 0, ErrTooManyPlayers
	}
	s := al.GenSources(1)
	if err := al.Error(); err != 0 {
		panic(fmt.Sprintf("audio: al.GenSources error: %d", err))
	}
	e := alSourceCacheEntry{
		source:     s[0],
		sampleRate: sampleRate,
	}
	alSourceCache = append(alSourceCache, e)
	return s[0], nil
}

func newPlayer(src io.ReadSeeker, sampleRate int) (*Player, error) {
	m.Lock()
	if e := al.OpenDevice(); e != nil {
		m.Unlock()
		return nil, fmt.Errorf("audio: OpenAL initialization failed: %v", e)
	}
	s, err := newAlSource(sampleRate)
	if err != nil {
		m.Unlock()
		return nil, err
	}
	m.Unlock()
	p := &player{
		alSource:   s,
		alBuffers:  []al.Buffer{},
		source:     src,
		sampleRate: sampleRate,
	}
	runtime.SetFinalizer(p, (*player).close)
	return &Player{p}, nil
}

const bufferSize = 1024

func (p *player) proceed() error {
	m.Lock()
	processedNum := p.alSource.BuffersProcessed()
	if 0 < processedNum {
		bufs := make([]al.Buffer, processedNum)
		p.alSource.UnqueueBuffers(bufs...)
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
	}
	m.Unlock()

	return nil
}

func (p *player) play() error {
	// TODO: What if play is already called?
	emptyBytes := make([]byte, bufferSize)
	m.Lock()
	queued := p.alSource.BuffersQueued()
	bufs := al.GenBuffers(8 - int(queued))
	for _, buf := range bufs {
		// Note that the third argument of only the first buffer is used.
		buf.BufferData(al.FormatStereo16, emptyBytes, int32(p.sampleRate))
		p.alSource.QueueBuffers(buf)
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
	if p.alSource != 0 {
		al.RewindSources(p.alSource)
		al.StopSources(p.alSource)
		p.alSource = 0
	}
	if 0 < len(p.alBuffers) {
		al.DeleteBuffers(p.alBuffers...)
		p.alBuffers = []al.Buffer{}
	}
	if err := al.Error(); err != 0 {
		panic(fmt.Sprintf("audio: closing error: %d", err))
	}
	m.Unlock()
	runtime.SetFinalizer(p, nil)
	return nil
}

// TODO: Implement Close method
