// Copyright 2024 The Ebitengine Authors
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

package main

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/gen2brain/mpeg"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

type mpegPlayer struct {
	mpg *mpeg.MPEG

	currentFrame *ebiten.Image
	audioPlayer  *audio.Player

	// These members are used when the video doesn't have an audio stream.
	refTime time.Time

	m sync.Mutex
}

func newMPEGPlayer(src io.Reader) (*mpegPlayer, error) {
	mpg, err := mpeg.New(src)
	if err != nil {
		return nil, err
	}
	if mpg.NumVideoStreams() == 0 {
		return nil, fmt.Errorf("video: no video streams")
	}
	if !mpg.HasHeaders() {
		return nil, fmt.Errorf("video: missing headers")
	}

	p := &mpegPlayer{
		mpg:          mpg,
		currentFrame: ebiten.NewImage(mpg.Width(), mpg.Height()),
	}

	// If the video doesn't have an audio stream, initialization is done.
	if mpg.NumAudioStreams() == 0 {
		return p, nil
	}

	// If the video has an audio stream, initialize an audio player.
	ctx := audio.CurrentContext()
	if ctx == nil {
		return nil, fmt.Errorf("video: audio.Context is not initialized")
	}
	if mpg.Channels() != 2 {
		return nil, fmt.Errorf("video: mpeg audio stream must be 2 but was %d", mpg.Channels())
	}
	if ctx.SampleRate() != mpg.Samplerate() {
		return nil, fmt.Errorf("video: mpeg audio stream sample rate %d doesn't match with audio context sample rate %d", mpg.Samplerate(), ctx.SampleRate())
	}

	mpg.SetAudioFormat(mpeg.AudioS16)

	audioPlayer, err := ctx.NewPlayer(&mpegAudio{
		audio: mpg.Audio(),
		m:     &p.m,
	})
	if err != nil {
		return nil, err
	}
	p.audioPlayer = audioPlayer

	return p, nil
}

func (p *mpegPlayer) Update() {
	p.m.Lock()
	defer p.m.Unlock()

	var pos float64
	if p.audioPlayer != nil {
		pos = p.audioPlayer.Position().Seconds()
	} else {
		if p.refTime != (time.Time{}) {
			pos = time.Since(p.refTime).Seconds()
		}
	}

	video := p.mpg.Video()
	if video.HasEnded() {
		p.currentFrame.Clear()
		return
	}

	d := 1 / p.mpg.Framerate()
	var mpegFrame *mpeg.Frame
	for video.Time()+d <= pos && !video.HasEnded() {
		mpegFrame = video.Decode()
	}

	if mpegFrame == nil {
		return
	}
	p.currentFrame.WritePixels(mpegFrame.RGBA().Pix)
}

// Draw draws the current frame onto the given screen.
func (p *mpegPlayer) Draw(screen *ebiten.Image) {
	frame := p.currentFrame
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	fw, fh := frame.Bounds().Dx(), frame.Bounds().Dy()

	op := ebiten.DrawImageOptions{}
	wf, hf := float64(sw)/float64(fw), float64(sh)/float64(fh)
	s := wf
	if hf < wf {
		s = hf
	}
	op.GeoM.Scale(s, s)

	offsetX, offsetY := float64(screen.Bounds().Min.X), float64(screen.Bounds().Min.Y)
	op.GeoM.Translate(offsetX+(float64(sw)-float64(fw)*s)/2, offsetY+(float64(sh)-float64(fh)*s)/2)
	op.Filter = ebiten.FilterLinear

	screen.DrawImage(frame, &op)
}

// Play starts playing the video.
func (p *mpegPlayer) Play() {
	p.m.Lock()
	defer p.m.Unlock()

	if p.mpg.HasEnded() {
		return
	}

	if p.audioPlayer != nil {
		if p.audioPlayer.IsPlaying() {
			return
		}
		// Play refers (*mpegAudio).Read function, where the same mutex is used.
		// In order to avoid dead lock, use a different goroutine to start playing.
		// This issue happens especially on Windows where goroutines at Play are avoided in Oto (#1768).
		// TODO: Remove this hack in the future (ebitengine/oto#235).
		go p.audioPlayer.Play()
		return
	}

	if p.refTime != (time.Time{}) {
		return
	}
	p.refTime = time.Now()
}

type mpegAudio struct {
	audio *mpeg.Audio

	// leftovers is the remaining audio samples of the previous Read call.
	leftovers []byte

	// m is the mutex shared with the mpegPlayer.
	// As *mpeg.MPEG is not concurrent safe, this mutex is necessary.
	m *sync.Mutex
}

func (a *mpegAudio) Read(buf []byte) (int, error) {
	a.m.Lock()
	defer a.m.Unlock()

	var readBytes int
	if len(a.leftovers) > 0 {
		n := copy(buf, a.leftovers)
		readBytes += n
		buf = buf[n:]

		copy(a.leftovers, a.leftovers[n:])
		a.leftovers = a.leftovers[:len(a.leftovers)-n]
	}

	for len(buf) > 0 && !a.audio.HasEnded() {
		mpegSamples := a.audio.Decode()
		if mpegSamples == nil {
			break
		}

		bs := make([]byte, len(mpegSamples.S16)*2)
		for i, s := range mpegSamples.S16 {
			bs[i*2] = byte(s)
			bs[i*2+1] = byte(s >> 8)
		}

		n := copy(buf, bs)
		readBytes += n
		buf = buf[n:]

		if n < len(bs) {
			a.leftovers = append(a.leftovers, bs[n:]...)
			break
		}
	}

	if a.audio.HasEnded() {
		return readBytes, io.EOF
	}
	return readBytes, nil
}
