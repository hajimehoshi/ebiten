// Copyright 2020 The Ebiten Authors
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

package audio

import (
	"io"
	"sync"

	"github.com/hajimehoshi/oto"
)

func newContextImpl(sampleRate int, initCh chan struct{}) context {
	return &otoContext{
		sampleRate: sampleRate,
		initCh:     initCh,
	}
}

type otoContext struct {
	sampleRate int
	initCh     <-chan struct{}

	c    *oto.Context
	once sync.Once
}

func (c *otoContext) NewPlayer() io.WriteCloser {
	return &otoPlayer{c: c}
}

func (c *otoContext) Close() error {
	if c.c == nil {
		return nil
	}
	return c.c.Close()
}

func (c *otoContext) ensureContext() error {
	var err error
	c.once.Do(func() {
		<-c.initCh
		c.c, err = oto.NewContext(c.sampleRate, channelNum, bytesPerSample/channelNum, bufferSize())
	})
	return err
}

type otoPlayer struct {
	c    *otoContext
	p    *oto.Player
	once sync.Once
}

func (p *otoPlayer) Write(buf []byte) (int, error) {
	// Initialize oto.Player lazily to enable calling NewContext in an 'init' function.
	// Accessing oto.Player functions requires the environment to be already initialized,
	// but if Ebiten is used for a shared library, the timing when init functions are called
	// is unexpectable.
	// e.g. a variable for JVM on Android might not be set.
	if err := p.ensurePlayer(); err != nil {
		return 0, err
	}
	return p.p.Write(buf)
}

func (p *otoPlayer) Close() error {
	if p.p == nil {
		return nil
	}
	return p.p.Close()
}

func (p *otoPlayer) ensurePlayer() error {
	if err := p.c.ensureContext(); err != nil {
		return err
	}
	p.once.Do(func() {
		p.p = p.c.c.NewPlayer()
	})
	return nil
}
