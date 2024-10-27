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
	"image"
	"io"
	"math"
	"sync"
	"time"

	"github.com/gen2brain/mpeg"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

type mpegPlayer struct {
	mpg *mpeg.MPEG

	// yCbCrImage is the current frame image in YCbCr format.
	// An MPEG frame is stored in this image first.
	// Then, this image data is converted to RGB to frameImage.
	yCbCrImage *ebiten.Image

	// yCbCrBytes is the byte slice to store YCbCr data.
	// This includes Y, Cb, Cr, and alpha (always 0xff) data for each pixel.
	yCbCrBytes []byte

	// yCbCrShader is the shader to convert YCbCr to RGB.
	yCbCrShader *ebiten.Shader

	// frameImage is the current frame image in RGB format.
	frameImage *ebiten.Image

	audioPlayer *audio.Player

	// These members are used when the video doesn't have an audio stream.
	refTime time.Time

	src io.ReadCloser

	closeOnce sync.Once

	m sync.Mutex
}

func newMPEGPlayer(src io.ReadCloser) (*mpegPlayer, error) {
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
		mpg:        mpg,
		yCbCrImage: ebiten.NewImage(mpg.Width(), mpg.Height()),
		yCbCrBytes: make([]byte, 4*mpg.Width()*mpg.Height()),
		frameImage: ebiten.NewImage(mpg.Width(), mpg.Height()),
		src:        src,
	}

	s, err := ebiten.NewShader([]byte(`package main

//kage:unit pixels

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	// For this calculation, see the comment in the standard library color.YCbCrToRGB function.
	c := imageSrc0UnsafeAt(srcPos)
	return vec4(
		c.x + 1.40200 * (c.z-0.5),
		c.x - 0.34414 * (c.y-0.5) - 0.71414 * (c.z-0.5),
		c.x + 1.77200 * (c.y-0.5),
		1,
	)
}
`))
	if err != nil {
		return nil, err
	}
	p.yCbCrShader = s

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

	mpg.SetAudioFormat(mpeg.AudioF32N)

	audioPlayer, err := ctx.NewPlayerF32(&mpegAudio{
		audio: mpg.Audio(),
		m:     &p.m,
	})
	if err != nil {
		return nil, err
	}
	p.audioPlayer = audioPlayer

	return p, nil
}

// updateFrame upadtes the current video frame.
func (p *mpegPlayer) updateFrame() error {
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
		p.frameImage.Clear()
		var err error
		p.closeOnce.Do(func() {
			fmt.Println("The video has ended.")
			if err1 := p.src.Close(); err1 != nil {
				err = err1
			}
		})
		return err
	}

	d := 1 / p.mpg.Framerate()
	var mpegFrame *mpeg.Frame
	for video.Time()+d <= pos && !video.HasEnded() {
		mpegFrame = video.Decode()
	}

	if mpegFrame == nil {
		return nil
	}

	img := mpegFrame.YCbCr()
	if img.SubsampleRatio != image.YCbCrSubsampleRatio420 {
		return fmt.Errorf("video: subsample ratio must be 4:2:0")
	}
	w, h := p.mpg.Width(), p.mpg.Height()
	for j := 0; j < h; j++ {
		yi := j * img.YStride
		ci := (j / 2) * img.CStride
		// Create temporary slices to encourage BCE (boundary-checking elimination).
		ys := img.Y[yi : yi+w]
		cbs := img.Cb[ci : ci+w/2]
		crs := img.Cr[ci : ci+w/2]
		for i := 0; i < w; i++ {
			idx := 4 * (j*w + i)
			buf := p.yCbCrBytes[idx : idx+3]
			buf[0] = ys[i]
			buf[1] = cbs[i/2]
			buf[2] = crs[i/2]
			// p.yCbCrBytes[3] = 0xff is not needed as the shader ignores this part.
		}
	}

	p.yCbCrImage.WritePixels(p.yCbCrBytes)

	// Converting YCbCr to RGB on CPU is slow. Use a shader instead.
	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = p.yCbCrImage
	op.Blend = ebiten.BlendCopy
	p.frameImage.DrawRectShader(w, h, p.yCbCrShader, op)

	return nil
}

// Draw draws the current frame onto the given screen.
func (p *mpegPlayer) Draw(screen *ebiten.Image) error {
	if err := p.updateFrame(); err != nil {
		return err
	}

	frame := p.frameImage
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
	return nil
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

		bs := make([]byte, len(mpegSamples.Interleaved)*4)
		for i, s := range mpegSamples.Interleaved {
			v := math.Float32bits(s)
			bs[4*i] = byte(v)
			bs[4*i+1] = byte(v >> 8)
			bs[4*i+2] = byte(v >> 16)
			bs[4*i+3] = byte(v >> 24)
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
