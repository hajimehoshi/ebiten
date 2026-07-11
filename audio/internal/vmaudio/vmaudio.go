// Copyright 2026 The Ebitengine Authors
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

// Package vmaudio is the virtual audio device for virtualization guests.
//
// A guest process plays no audio of its own: the host owns the real devices. The guest's players are
// decoders the host reads on demand — it pulls each player's samples at whatever rate its device
// needs, so the players are never mixed and each is its own stream. The host owns the playback pace;
// the guest only decodes when asked. Separately, the guest pushes control changes (creation,
// play/pause, volume) to the host each tick. Samples are 32-bit little-endian floats with two
// interleaved channels, matching the audio package's device format.
package vmaudio

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"weak"

	"github.com/hajimehoshi/ebiten/v2/internal/vmguest"
	"github.com/hajimehoshi/ebiten/v2/internal/vmprotocol"
)

var theContext atomic.Pointer[Context]

// init wires the virtual device to the guest's connection without the UI backend depending on this
// package: the device pushes control changes through a post-tick hook and answers the host's sample
// reads through the audio-read handler.
func init() {
	vmguest.AppendPostTickHook(forwardControl)
	vmguest.RegisterAudioReadHandler(readAudio)
}

// NewContext returns a new virtual audio context at the given sample rate. The audio package creates it
// only for a virtualization guest, and at most one per process. The guest keeps the rate its game
// chose; the host observes it (it need not match or even play the audio).
func NewContext(sampleRate int) *Context {
	c := &Context{
		sampleRate: sampleRate,
		players:    map[int64]weak.Pointer[Player]{},
	}
	theContext.Store(c)
	return c
}

// CurrentContext returns the context created by NewContext, or nil if there is none.
func CurrentContext() *Context {
	return theContext.Load()
}

// forwardControl is the post-tick hook: it pushes the control changes accumulated since the last tick,
// tagged with tick (the guest's ebiten.Tick() during it) so the host can stamp a newly-started stream.
func forwardControl(enc vmprotocol.GuestMessageEncoder, tick int) error {
	c := CurrentContext()
	if c == nil {
		return nil
	}
	c.controlsBuf = c.takeControlChanges(c.controlsBuf[:0])
	if len(c.controlsBuf) == 0 {
		return nil
	}
	return enc.EncodeGuestMessage(&vmprotocol.GuestMessage{
		Kind:            vmprotocol.GuestMessageKindAudioControl,
		StartTick:       tick,
		AudioSampleRate: c.sampleRate,
		AudioControls:   c.controlsBuf,
	})
}

// readAudio is the audio-read handler: it reads player id's samples into buf on demand.
func readAudio(id int64, buf []byte) (n int, eof bool) {
	c := CurrentContext()
	if c == nil {
		return 0, true
	}
	return c.read(id, buf)
}

// Context is a virtual audio device: it does not consume the players' sources on a clock of its own.
// Instead the host pulls each player's samples on demand, and the device pushes control changes. It
// does not mix.
type Context struct {
	sampleRate int

	mu        sync.Mutex
	players   map[int64]weak.Pointer[Player]
	nextID    int64
	suspended bool

	// closedIDs holds the players closed since the last control push, so the next push reports their
	// removal to the host.
	closedIDs []int64

	// controlsBuf is the buffer forwardControl reuses across ticks for the control changes it sends.
	controlsBuf []vmprotocol.AudioControl
}

// SampleRate returns the context's sample rate, in per-channel samples per second.
func (c *Context) SampleRate() int {
	return c.sampleRate
}

// NewPlayer creates a new virtual player reading src.
func (c *Context) NewPlayer(src io.Reader) *Player {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nextID++
	p := &Player{
		c:      c,
		id:     c.nextID,
		src:    src,
		volume: 1,
	}
	c.players[p.id] = weak.Make(p)
	// Reclaim a player the game abandons: the map holds it only weakly, so once it is unreferenced and no
	// longer playing GC collects it, and this cleanup removes the entry and reports the removal (the same
	// path as an explicit Close).
	runtime.AddCleanup(p, c.closePlayer, p.id)
	return p
}

// Suspend pauses the device: it produces no samples and consumes no sources until Resume.
func (c *Context) Suspend() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.suspended = true
	return nil
}

// Resume resumes the device after Suspend.
func (c *Context) Resume() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.suspended = false
	return nil
}

// Err returns the device error. A virtual device has none: Err always returns nil.
func (c *Context) Err() error {
	return nil
}

// takeControlChanges drains the control changes accumulated since the last call — appending them to controls
// and returning the extended slice — and marks them taken: each player's reported state is advanced and
// the closed-ID buffer is emptied, so a change is forwarded only once. The appended entries are sorted
// by player ID, so the forwarded order does not depend on map iteration.
func (c *Context) takeControlChanges(controls []vmprotocol.AudioControl) []vmprotocol.AudioControl {
	c.mu.Lock()
	defer c.mu.Unlock()
	start := len(controls)
	for _, wp := range c.players {
		p := wp.Value()
		if p == nil {
			// The player has been collected; its cleanup will remove the entry and report the removal.
			continue
		}
		if ctrl, ok := p.takeControlChange(); ok {
			controls = append(controls, ctrl)
		}
	}
	for _, id := range c.closedIDs {
		controls = append(controls, vmprotocol.AudioControl{ID: id, Closed: true})
	}
	c.closedIDs = c.closedIDs[:0]
	slices.SortFunc(controls[start:], func(a, b vmprotocol.AudioControl) int {
		return cmp.Compare(a.ID, b.ID)
	})
	return controls
}

// read reads player id's samples into buf and reports whether its source has ended. The player is kept
// until Close, so a finished source can be sought back and replayed.
func (c *Context) read(id int64, buf []byte) (n int, eof bool) {
	p, suspended := c.playerForRead(id)
	if p == nil {
		// The player is unknown — never created, or already closed. Nothing more to read.
		return 0, true
	}

	// The source read runs without c.mu held, so a blocking read does not stall the map or the control
	// push.
	return p.read(buf, suspended)
}

// playerForRead returns the player with the given ID (nil if unknown) and whether the device is
// suspended.
func (c *Context) playerForRead(id int64) (*Player, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.players[id].Value(), c.suspended
}

// closePlayer drops a player — closed by the game, or reclaimed by GC once abandoned — and records its
// ID so the next control push reports the removal to the host. It is idempotent: dropping an
// already-removed player does nothing.
func (c *Context) closePlayer(id int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.players[id]; !ok {
		return
	}
	delete(c.players, id)
	c.closedIDs = append(c.closedIDs, id)
}

// Player is a virtual audio player: the audio package controls it through the device player methods
// (Play, Pause, and the rest), while the host pulls its source on demand.
type Player struct {
	// c is the owning context, used by Close to remove the player; it is fixed at creation.
	c  *Context
	id int64

	mu      sync.Mutex
	src     io.Reader
	playing bool
	volume  float64
	eof     bool
	err     error

	// buf holds bytes read from the source but not yet returned (a short or unaligned source read can
	// leave a partial frame behind).
	buf []byte

	// lastSent is the control state last reported to the host. Its zero value has ID 0, which no player
	// has, so the first comparison always differs and the player's creation is reported.
	lastSent vmprotocol.AudioControl
}

func (p *Player) Play() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.playing = true
}

func (p *Player) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.playing = false
}

func (p *Player) IsPlaying() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.playing && !p.finishedLocked()
}

func (p *Player) Volume() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.volume
}

func (p *Player) SetVolume(volume float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.volume = volume
}

// BufferedSize returns the number of bytes read from the source but not yet consumed.
func (p *Player) BufferedSize() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.buf)
}

func (p *Player) Err() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.err
}

// SetBufferSize does nothing: the virtual device has no real-time deadline to buffer ahead for.
func (p *Player) SetBufferSize(bufferSize int) {
}

// Close removes the player from the device, so it is no longer forwarded and the host is told of its
// removal on the next control push. It always returns nil.
func (p *Player) Close() error {
	p.c.closePlayer(p.id)
	return nil
}

// Seek seeks the source, which must be an io.Seeker, and discards the bytes buffered from the old
// position.
func (p *Player) Seek(offset int64, whence int) (int64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	s, ok := p.src.(io.Seeker)
	if !ok {
		return 0, fmt.Errorf("vmaudio: the source must be an io.Seeker")
	}
	n, err := s.Seek(offset, whence)
	if err != nil {
		return 0, err
	}
	p.buf = p.buf[:0]
	p.eof = false
	return n, nil
}

// takeControlChange returns the player's current control state and whether it differs from the state
// last reported to the host, recording the new state when it does. ok is false when nothing changed
// since the last call, so no redundant update is sent.
func (p *Player) takeControlChange() (vmprotocol.AudioControl, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	c := vmprotocol.AudioControl{
		ID:      p.id,
		Playing: p.playing && !p.finishedLocked(),
		Volume:  p.volume,
	}
	if c == p.lastSent {
		return vmprotocol.AudioControl{}, false
	}
	p.lastSent = c
	return c, true
}

// read reads up to len(buf) of the player's source into buf as raw PCM (volume not applied), and
// reports whether the source has ended. A paused or suspended player consumes no source and produces
// nothing, but is not finished. A source past its end ends the player once its buffer can no longer
// form a frame; a source with no data available now (e.g. a real-time stream) yields a short read
// rather than blocking. p.mu is held for the duration.
func (p *Player) read(buf []byte, suspended bool) (n int, eof bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.playing || suspended {
		return 0, false
	}

	// Keep whole stereo frames (8 bytes); a trailing partial frame stays buffered for the next read.
	need := len(buf) - len(buf)%8
	for len(p.buf) < need && !p.eof && p.err == nil {
		m := len(p.buf)
		p.buf = slices.Grow(p.buf, need-m)[:need]
		read, err := p.src.Read(p.buf[m:])
		p.buf = p.buf[:m+read]
		if err != nil {
			if errors.Is(err, io.EOF) {
				p.eof = true
			} else {
				p.err = err
			}
		}
		if read == 0 && err == nil {
			break
		}
	}

	consume := min(len(p.buf), need)
	consume -= consume % 8
	n = copy(buf, p.buf[:consume])
	p.buf = p.buf[:copy(p.buf, p.buf[consume:])]

	return n, p.finishedLocked()
}

// finishedLocked reports whether the player can produce no more samples until a Seek: its source failed,
// or ended with too little buffered to form a frame. p.mu must be held.
func (p *Player) finishedLocked() bool {
	return p.err != nil || (p.eof && len(p.buf) < 8)
}
