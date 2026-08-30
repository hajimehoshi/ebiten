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
	"sync/atomic"
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

		// buffered is the byte count read from r but not consumed by the device yet.
		buffered int

		// bufferedDrain is the byte count the simulated device consumes per BufferedSize call. If
		// zero, the buffered size is always zero and the player stops as soon as its source is
		// exhausted, as if it had no buffer.
		bufferedDrain int

		// drained is whether the device has consumed the last buffered byte after the source was
		// exhausted. The playing state is cleared on the next BufferedSize call, as a real device
		// stops playing one tick after it has output its last buffered byte.
		drained bool

		// readGen is incremented by PauseAndStopReading to stop the goroutine reading r.
		readGen int

		mu sync.Mutex
	}
)

// dummyBufferedDrain is the bufferedDrain value of the players created after it is set.
// This is accessed atomically as a goroutine of an earlier test's context can create a player
// even after ResetContextForTesting. See SetBufferedDrainForTesting.
var dummyBufferedDrain atomic.Int64

// SetBufferedDrainForTesting makes the dummy players report a buffered size which the simulated
// device consumes by drainSize bytes per BufferedSize call, so that they keep playing their
// buffered data after the source is exhausted, as the real players do.
// If drainSize is zero, the buffered size is always zero.
func SetBufferedDrainForTesting(drainSize int) {
	dummyBufferedDrain.Store(int64(drainSize))
}

func (c *dummyContext) NewPlayer(r io.Reader) player {
	return &dummyPlayer{
		r:             r,
		volume:        1,
		bufferedDrain: int(dummyBufferedDrain.Load()),
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
			_, stopped, err := p.readOnce(gen, buf[:])
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
		if p.bufferedDrain == 0 {
			p.playing = false
		}
	}()
}

// readOnce performs one read from the source with the mutex held, so that PauseAndStopReading waits
// for an in-flight read. The read bytes are added to the buffered size in the same critical section.
// stopped reports that PauseAndStopReading was called and reading must not continue.
func (p *dummyPlayer) readOnce(gen int, buf []byte) (n int, stopped bool, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.readGen != gen {
		return 0, true, nil
	}
	n, err = p.r.Read(buf)
	p.buffered += n
	return n, false, err
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
	return p.volume
}

func (p *dummyPlayer) SetVolume(volume float64) {
	p.volume = volume
}

func (p *dummyPlayer) BufferedSize() int {
	// This clears the playing state when the device has drained everything, and the context calls
	// this exactly once per tick before it checks IsPlaying, so that the player stops playing one
	// tick after the position reaches its final value, as real players do.
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.bufferedDrain == 0 {
		return 0
	}
	// The device stops playing on this call when it consumed the last buffered byte on the
	// previous call.
	if p.playing && p.eof && p.drained {
		p.playing = false
		return 0
	}
	if p.buffered > p.bufferedDrain {
		p.buffered -= p.bufferedDrain
		return p.buffered
	}
	p.buffered = 0
	if p.eof {
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
	p.buffered = 0
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
