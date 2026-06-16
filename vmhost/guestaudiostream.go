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

package vmhost

import (
	"io"
	"sync"
	"time"
)

// GuestAudioStream is one guest audio player observed from the host: a single stream the host reads and
// inspects. Read pulls its samples from the guest on demand, so it can serve directly as a host audio
// player's source. Its methods may be called from any goroutine.
type GuestAudioStream struct {
	// session pulls the samples; id and rate are fixed at creation. None of these needs the lock.
	session *GuestSession
	id      int64
	rate    int

	// mu guards the fields below, so each stream's reads are independent of the other streams and of the
	// session's map operations.
	mu        sync.Mutex
	playing   bool
	volume    float64
	readBytes int64
}

// setControl records a control update for the player.
func (p *GuestAudioStream) setControl(playing bool, volume float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.playing = playing
	p.volume = volume
}

// markClosed marks the stream not playing, used when the guest reports its player closed and the stream
// is dropped.
func (p *GuestAudioStream) markClosed() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.playing = false
}

// Read reads the stream's samples: 32-bit little-endian floats with two interleaved channels, at the
// guest's sample rate (see [GuestSession.AudioSampleRate]), with the volume NOT applied (see
// [GuestAudioStream.Volume]). It pulls the samples from the guest on demand, so it may block briefly
// while the session is busy. It returns 0 bytes when the stream is paused or has produced none yet, and
// io.EOF when the source reaches its end. io.EOF does not close the stream: if the guest seeks the
// source back and plays again, a later Read yields its samples; the stream is removed only when the
// guest closes its player or the session closes.
func (p *GuestAudioStream) Read(b []byte) (int, error) {
	n, eof := p.session.readGuestAudio(p.id, b)

	p.mu.Lock()
	p.readBytes += int64(n)
	p.mu.Unlock()

	if n == 0 && eof {
		return 0, io.EOF
	}
	return n, nil
}

// IsPlaying reports whether the guest's player is currently playing.
func (p *GuestAudioStream) IsPlaying() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.playing
}

// Volume returns the guest player's volume, in [0,1]. It is not applied to the samples from
// [GuestAudioStream.Read]; the host applies it when playing.
func (p *GuestAudioStream) Volume() float64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.volume
}

// Position returns how much of the player the host has read so far. It is 0 when the guest reported no
// sample rate.
func (p *GuestAudioStream) Position() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.rate <= 0 {
		return 0
	}
	samples := p.readBytes / 8
	return time.Duration(samples) * time.Second / time.Duration(p.rate)
}
