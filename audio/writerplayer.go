// Copyright 2019 The Ebiten Authors
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
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/hooks"
)

// writerDriver represents a driver using io.WriteClosers.
// The actual implementation is otoDriver.
type writerDriver interface {
	NewPlayer() io.WriteCloser
	io.Closer
}

var writerDriverForTesting writerDriver

func newWriterDriver(sampleRate int) writerDriver {
	if writerDriverForTesting != nil {
		return writerDriverForTesting
	}

	ch := make(chan struct{})
	var once sync.Once
	hooks.AppendHookOnBeforeUpdate(func() error {
		once.Do(func() {
			close(ch)
		})
		return nil
	})
	return newOtoDriver(sampleRate, ch)
}

type writerPlayerFactory struct {
	driver writerDriver
}

func newWriterPlayerFactory(sampleRate int) *writerPlayerFactory {
	return &writerPlayerFactory{
		driver: newWriterDriver(sampleRate),
	}
}

type writerPlayer struct {
	context          *Context
	driver           writerDriver
	src              io.Reader
	playing          bool
	closedExplicitly bool
	isLoopActive     bool

	buf     []byte
	readbuf []byte
	pos     int64
	volume  float64

	m sync.Mutex
}

func (c *writerPlayerFactory) newPlayerImpl(context *Context, src io.Reader) (playerImpl, error) {
	p := &writerPlayer{
		context: context,
		driver:  c.driver,
		src:     src,
		volume:  1,
	}
	if seeker, ok := p.src.(io.Seeker); ok {
		// Get the current position of the source.
		pos, err := seeker.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, err
		}
		p.pos = pos
	}
	return p, nil
}

func (p *writerPlayer) Close() error {
	p.m.Lock()
	defer p.m.Unlock()

	p.playing = false
	if p.closedExplicitly {
		return fmt.Errorf("audio: the player is already closed")
	}
	p.closedExplicitly = true
	return nil
}

func (p *writerPlayer) Play() {
	p.m.Lock()
	defer p.m.Unlock()

	if p.closedExplicitly {
		p.context.setError(fmt.Errorf("audio: the player is already closed"))
		return
	}

	p.playing = true
	if p.isLoopActive {
		return
	}

	// Set p.isLoopActive to true here, not in the loop. This prevents duplicated active loops.
	p.isLoopActive = true
	p.context.addPlayer(p)

	go p.loop()
	return
}

func (p *writerPlayer) loop() {
	p.context.waitUntilInited()

	w := p.driver.NewPlayer()
	wclosed := make(chan struct{})
	defer func() {
		<-wclosed
		w.Close()
	}()

	defer func() {
		p.m.Lock()
		p.playing = false
		p.context.removePlayer(p)
		p.isLoopActive = false
		p.m.Unlock()
	}()

	ch := make(chan []byte)
	defer close(ch)

	go func() {
		for buf := range ch {
			if _, err := w.Write(buf); err != nil {
				p.context.setError(err)
				break
			}
			p.context.setReady()
		}
		close(wclosed)
	}()

	for {
		buf, ok := p.read()
		if !ok {
			return
		}
		ch <- buf
	}
}

func (p *writerPlayer) read() ([]byte, bool) {
	if p.context.hasError() {
		return nil, false
	}

	if p.closedExplicitly {
		return nil, false
	}

	p.context.acquireSemaphore()
	defer func() {
		p.context.releaseSemaphore()
	}()

	p.m.Lock()
	defer p.m.Unlock()

	// playing can be false when pausing.
	if !p.playing {
		return nil, false
	}

	const bufSize = 2048
	if p.readbuf == nil {
		p.readbuf = make([]byte, bufSize)
	}
	n, err := p.src.Read(p.readbuf[:bufSize-len(p.buf)])
	if err != nil {
		if err != io.EOF {
			p.context.setError(err)
			return nil, false
		}
		if n == 0 {
			return nil, false
		}
	}
	buf := append(p.buf, p.readbuf[:n]...)

	n2 := len(buf) - len(buf)%bytesPerSample
	buf, p.buf = buf[:n2], buf[n2:]

	for i := 0; i < len(buf)/2; i++ {
		v16 := int16(buf[2*i]) | (int16(buf[2*i+1]) << 8)
		v16 = int16(float64(v16) * p.volume)
		buf[2*i] = byte(v16)
		buf[2*i+1] = byte(v16 >> 8)
	}
	p.pos += int64(len(buf))

	return buf, true
}

func (p *writerPlayer) IsPlaying() bool {
	p.m.Lock()
	r := p.playing
	p.m.Unlock()
	return r
}

func (p *writerPlayer) Rewind() error {
	if _, ok := p.src.(io.Seeker); !ok {
		panic("audio: player to be rewound must be io.Seeker")
	}
	return p.Seek(0)
}

func (p *writerPlayer) Seek(offset time.Duration) error {
	p.m.Lock()
	defer p.m.Unlock()

	o := int64(offset) * bytesPerSample * int64(p.context.SampleRate()) / int64(time.Second)
	o = o - (o % bytesPerSample)

	seeker, ok := p.src.(io.Seeker)
	if !ok {
		panic("audio: the source must be io.Seeker when seeking")
	}
	pos, err := seeker.Seek(o, io.SeekStart)
	if err != nil {
		return err
	}

	p.buf = nil
	p.pos = pos
	return nil
}

func (p *writerPlayer) Pause() {
	p.m.Lock()
	p.playing = false
	p.m.Unlock()
}

func (p *writerPlayer) Current() time.Duration {
	p.m.Lock()
	sample := p.pos / bytesPerSample
	p.m.Unlock()
	return time.Duration(sample) * time.Second / time.Duration(p.context.SampleRate())
}

func (p *writerPlayer) Volume() float64 {
	p.m.Lock()
	v := p.volume
	p.m.Unlock()
	return v
}

func (p *writerPlayer) SetVolume(volume float64) {
	// The condition must be true when volume is NaN.
	if !(0 <= volume && volume <= 1) {
		panic("audio: volume must be in between 0 and 1")
	}

	p.m.Lock()
	p.volume = volume
	p.m.Unlock()
}

func (p *writerPlayer) source() io.Reader {
	return p.src
}
