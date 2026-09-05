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

package audio

import (
	"io"
	"sync"
	"time"
)

type (
	dummyContext struct{}
	dummyPlayer  struct {
		r       io.Reader
		playing bool
		volume  float64

		// eof is whether the source has been exhausted by playing through. Like the real players,
		// a player which has finished its source refuses to play until Seek resets this state.
		eof bool

		// drained is whether the simulated device has output the data it had buffered when the
		// source was exhausted. See BufferedSize.
		drained bool

		// readGen is incremented by PauseAndStopReading to stop the goroutine reading r.
		readGen int

		mu sync.Mutex
	}
)

func (c *dummyContext) NewPlayer(r io.Reader) player {
	return &dummyPlayer{
		r:      r,
		volume: 1,
	}
}

func (c *dummyContext) MaxBufferSize() int {
	return 48000 * channelCount * bitDepthInBytesInt16 / 4
}

func (c *dummyContext) Suspend() error {
	return nil
}

func (c *dummyContext) Resume() error {
	return nil
}

func (c *dummyContext) Err() error {
	return nil
}

func (p *dummyPlayer) Pause() {
	p.mu.Lock()
	p.playing = false
	p.mu.Unlock()
}

func (p *dummyPlayer) Play() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.eof {
		return
	}
	p.playing = true
	gen := p.readGen
	go func() {
		var buf [4096]byte
		for {
			stopped, err := p.readOnce(gen, buf[:])
			if stopped {
				return
			}
			if err != nil {
				if err != io.EOF {
					panic(err)
				}
				break
			}
			time.Sleep(time.Millisecond)
		}
		p.mu.Lock()
		defer p.mu.Unlock()
		// The source is exhausted only when it was played through. A paused player would still
		// have unplayed data in its buffer in a real player.
		if p.playing {
			p.eof = true
		}
	}()
}

// readOnce performs one read from the source with the mutex held, so that PauseAndStopReading waits
// for an in-flight read. stopped reports that PauseAndStopReading was called and reading must not
// continue.
func (p *dummyPlayer) readOnce(gen int, buf []byte) (stopped bool, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.readGen != gen {
		return true, nil
	}
	_, err = p.r.Read(buf)
	return false, err
}

func (p *dummyPlayer) PauseAndStopReading() {
	p.mu.Lock()
	p.playing = false
	p.readGen++
	p.mu.Unlock()
}

func (p *dummyPlayer) IsPlaying() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.playing
}

func (p *dummyPlayer) Volume() float64 {
	// The volume is not shared with the goroutine Play spawns, but the mutex is held so that the
	// test double is uniformly safe like the real player it stands in for.
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.volume
}

func (p *dummyPlayer) SetVolume(volume float64) {
	// See Volume for why the mutex is held.
	p.mu.Lock()
	defer p.mu.Unlock()
	p.volume = volume
}

// BufferedSize always reports an empty buffer, as the simulated device consumes the data as soon as
// it is read.
//
// The context calls this exactly once per tick, before it reads the position and checks IsPlaying.
// The player keeps playing for one more tick after its source is exhausted, so that the position
// reaches its final value while the player is still playing, as a real player which still has
// buffered data to output.
func (p *dummyPlayer) BufferedSize() int {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.eof {
		if p.drained {
			p.playing = false
		}
		p.drained = true
	}
	return 0
}

func (p *dummyPlayer) Err() error {
	return nil
}

func (p *dummyPlayer) SetBufferSize(bufferSize int) {
}

func (p *dummyPlayer) Seek(offset int64, whence int) (int64, error) {
	// Seeking discards the buffered data and resets the finished state as real players do, so
	// that the source can be played again.
	p.mu.Lock()
	defer p.mu.Unlock()
	p.eof = false
	p.drained = false
	return 0, nil
}

func init() {
	driverForTesting = &dummyContext{}
}

type dummyHook struct {
	updates []func(vmGuest bool) error
}

func (h *dummyHook) OnSuspendAudio(f func() error) {
}

func (h *dummyHook) OnResumeAudio(f func() error) {
}

func (h *dummyHook) AppendHookOnBeforeUpdateWithVMGuestInfo(f func(vmGuest bool) error) {
	h.updates = append(h.updates, f)
}

func init() {
	hookerForTesting = &dummyHook{}
}

func UpdateForTesting() error {
	for _, f := range hookerForTesting.(*dummyHook).updates {
		if err := f(false); err != nil {
			return err
		}
	}
	return nil
}

func PlayersCountForTesting() int {
	c := CurrentContext()
	c.m.Lock()
	n := len(c.playingPlayers)
	c.m.Unlock()
	return n
}

func BufferSizeForTesting(p *Player) int {
	p.p.m.Lock()
	defer p.p.m.Unlock()
	return p.p.initBufferSize
}

// PlayingButUntrackedForTesting reports whether the player is playing but is not tracked by its
// context as a playing player. This must never be true: an untracked playing player is not
// updated by the context and is not guarded from being garbage-collected.
//
// The player's lock is held across both checks, so that a player which is being started or
// stopped concurrently is not reported.
func PlayingButUntrackedForTesting(p *Player) bool {
	pi := p.p

	pi.m.Lock()
	defer pi.m.Unlock()

	if pi.closed || !pi.isPlaying() {
		return false
	}

	c := pi.context
	c.m.Lock()
	defer c.m.Unlock()
	_, ok := c.playingPlayers[pi]
	return !ok
}

// ContextCreatedForTesting reports whether the underlying audio device has been created.
func ContextCreatedForTesting() bool {
	c := CurrentContext()
	if c == nil {
		return false
	}
	return c.playerFactory.currentContext() != nil
}

func ResetContextForTesting() {
	theContext = nil
}

func (i *InfiniteLoop) SetNoBlendForTesting(value bool) {
	i.noBlendForTesting = value
}
