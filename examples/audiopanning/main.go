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

package main

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"io"
	"log"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	raudio "github.com/hajimehoshi/ebiten/v2/examples/resources/audio"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/images"
)

const (
	screenWidth  = 640
	screenHeight = 480
	sampleRate   = 48000
)

var ebitenImage *ebiten.Image

type Game struct {
	player    *audio.Player
	panstream *StereoPanStream

	// panning goes from -1 to 1
	// -1: 100% left channel, 0% right channel
	// 0: 100% both channels
	// 1: 0% left channel, 100% right channel
	panning float64

	count int
	xpos  float64

	audioContext *audio.Context
}

func (g *Game) initAudioIfNeeded() {
	if g.player != nil {
		return
	}

	if g.audioContext == nil {
		g.audioContext = audio.NewContext(sampleRate)
	}

	// Decode an Ogg file.
	// oggS is a decoded io.ReadCloser and io.Seeker.
	oggS, err := vorbis.DecodeF32(bytes.NewReader(raudio.Ragtime_ogg))
	if err != nil {
		log.Fatal(err)
	}

	// Wrap the raw audio with the StereoPanStream
	g.panstream = NewStereoPanStream(audio.NewInfiniteLoop(oggS, oggS.Length()))
	g.panstream.SetPan(g.panning)

	g.player, err = g.audioContext.NewPlayerF32(g.panstream)
	if err != nil {
		log.Fatal(err)
	}

	// Play the infinite-length stream. This never ends.
	g.player.Play()
}

// time is within the 0 ... 1 range
func lerp(a, b, t float64) float64 {
	return a*(1-t) + b*t
}

func (g *Game) Update() error {
	g.count++
	r := float64(g.count) * ((1 / float64(ebiten.TPS())) * 2 * math.Pi) * 0.1 // full cycle every 10 seconds
	g.xpos = (float64(screenWidth) / 2) + math.Cos(r)*(float64(screenWidth)/2)
	g.panning = lerp(-1, 1, g.xpos/float64(screenWidth))

	// Initialize the audio after the panning is determined.
	g.initAudioIfNeeded()
	g.panstream.SetPan(g.panning)

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	pos := g.player.Position()
	msg := fmt.Sprintf(`TPS: %0.2f
This is an example using
stereo audio panning.
Current: %0.2f[s]
Panning: %.2f`, ebiten.ActualTPS(), float64(pos)/float64(time.Second), g.panning)
	ebitenutil.DebugPrint(screen, msg)

	// draw image to show where the sound is at related to the screen
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(g.xpos-float64(ebitenImage.Bounds().Dx()/2), screenHeight/2)
	screen.DrawImage(ebitenImage, op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	// Decode an image from the image file's byte slice.
	img, _, err := image.Decode(bytes.NewReader(images.Ebiten_png))
	if err != nil {
		log.Fatal(err)
	}
	ebitenImage = ebiten.NewImageFromImage(img)

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Audio Panning Loop (Ebitengine Demo)")
	g := &Game{}
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

// StereoPanStream is an audio buffer that changes the stereo channel's signal
// based on the Panning.
type StereoPanStream struct {
	io.ReadSeeker
	pan float64 // -1: left; 0: center; 1: right
	buf []byte
}

func (s *StereoPanStream) Read(p []byte) (int, error) {
	// If the stream has a buffer that was read in the previous time, use this first.
	var bufN int
	if len(s.buf) > 0 {
		bufN = copy(p, s.buf)
		s.buf = s.buf[bufN:]
	}

	readN, err := s.ReadSeeker.Read(p[bufN:])
	if err != nil && err != io.EOF {
		return 0, err
	}

	// Align the buffer size in multiples of 4. The extra part is pushed to the buffer for the
	// next time.
	totalN := bufN + readN
	extra := totalN - totalN/8*8
	s.buf = append(s.buf, p[totalN-extra:totalN]...)
	alignedN := totalN - extra

	// This implementation uses a linear scale, ranging from -1 to 1, for stereo or mono sounds.
	// If pan = 0.0, the balance for the sound in each speaker is at 100% left and 100% right.
	// When pan is -1.0, only the left channel of the stereo sound is audible, when pan is 1.0,
	// only the right channel of the stereo sound is audible.
	// https://docs.unity3d.com/ScriptReference/AudioSource-panStereo.html
	ls := float32(math.Min(s.pan*-1+1, 1))
	rs := float32(math.Min(s.pan+1, 1))
	for i := 0; i < alignedN; i += 8 {
		lc := math.Float32frombits(uint32(p[i])|(uint32(p[i+1])<<8)|(uint32(p[i+2])<<16)|(uint32(p[i+3])<<24)) * ls
		rc := math.Float32frombits(uint32(p[i+4])|(uint32(p[i+5])<<8)|(uint32(p[i+6])<<16)|(uint32(p[i+7])<<24)) * rs
		lcBits := math.Float32bits(lc)
		rcBits := math.Float32bits(rc)

		p[i] = byte(lcBits)
		p[i+1] = byte(lcBits >> 8)
		p[i+2] = byte(lcBits >> 16)
		p[i+3] = byte(lcBits >> 24)
		p[i+4] = byte(rcBits)
		p[i+5] = byte(rcBits >> 8)
		p[i+6] = byte(rcBits >> 16)
		p[i+7] = byte(rcBits >> 24)
	}
	return alignedN, err
}

func (s *StereoPanStream) SetPan(pan float64) {
	s.pan = math.Min(math.Max(-1, pan), 1)
}

func (s *StereoPanStream) Pan() float64 {
	return s.pan
}

// NewStereoPanStream returns a new StereoPanStream with a buffered src.
//
// The src's format must be linear PCM (16bits little endian, 2 channel stereo)
// without a header (e.g. RIFF header). The sample rate must be same as that
// of the audio context.
func NewStereoPanStream(src io.ReadSeeker) *StereoPanStream {
	return &StereoPanStream{
		ReadSeeker: src,
	}
}
