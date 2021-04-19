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

// +build example jsgo

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

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/audio"
	"github.com/hajimehoshi/ebiten/audio/vorbis"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	raudio "github.com/hajimehoshi/ebiten/examples/resources/audio"
	"github.com/hajimehoshi/ebiten/examples/resources/images"
)

const (
	screenWidth  = 320
	screenHeight = 240
	sampleRate   = 22050
)

var img *ebiten.Image

var audioContext *audio.Context

func init() {
	var err error
	audioContext, err = audio.NewContext(sampleRate)
	if err != nil {
		log.Fatal(err)
	}
}

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
}

func (g *Game) initAudio() {
	if g.player != nil {
		return
	}

	// Decode an Ogg file.
	// oggS is a decoded io.ReadCloser and io.Seeker.
	oggS, err := vorbis.Decode(audioContext, audio.BytesReadSeekCloser(raudio.Ragtime_ogg))
	if err != nil {
		log.Fatal(err)
	}

	// Wrap the raw audio with the StereoPanStream
	g.panstream = NewStereoPanStreamFromReader(audio.NewInfiniteLoop(oggS, oggS.Length()))

	g.player, err = audio.NewPlayer(audioContext, g.panstream)
	if err != nil {
		log.Fatal(err)
	}

	// Play the infinite-length stream. This never ends.
	g.player.Play()
}

// time is whithin the 0 ... 1 range
func lerp(a, b, t float64) float64 {
	return a*(1-t) + b*t
}

func (g *Game) Update(screen *ebiten.Image) error {
	g.initAudio()
	g.count++
	r := float64(g.count) * ((1.0 / 60.0) * 2 * math.Pi) * 0.1 // full cycle every 10 seconds
	g.xpos = (float64(screenWidth) / 2) + math.Cos(r)*(float64(screenWidth)/2)
	g.panning = lerp(-1, 1, g.xpos/float64(screenWidth))
	g.panstream.SetPan(g.panning)
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	pos := g.player.Current()
	msg := fmt.Sprintf(`TPS: %0.2f
This is an example using
stereo audio panning.
Current: %0.2f[s]
Panning: %.2f`, ebiten.CurrentTPS(), float64(pos)/float64(time.Second), g.panning)
	ebitenutil.DebugPrint(screen, msg)

	// draw image to show where the sound is at related to the screen
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(g.xpos-float64(img.Bounds().Dx()/2), screenHeight/2)
	screen.DrawImage(img, op)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	// Decode image from a byte slice instead of a file so that
	// this example works in any working directory.
	// If you want to use a file, there are some options:
	// 1) Use os.Open and pass the file to the image decoder.
	//    This is a very regular way, but doesn't work on browsers.
	// 2) Use ebitenutil.OpenFile and pass the file to the image decoder.
	//    This works even on browsers.
	// 3) Use ebitenutil.NewImageFromFile to create an ebiten.Image directly from a file.
	//    This also works on browsers.
	rawimg, _, err := image.Decode(bytes.NewReader(images.Ebiten_png))
	if err != nil {
		log.Fatal(err)
	}
	img, _ = ebiten.NewImageFromImage(rawimg, ebiten.FilterDefault)

	ebiten.SetWindowSize(screenWidth*2, screenHeight*2)
	ebiten.SetWindowTitle("Audio Panning Loop (Ebiten Demo)")
	g := &Game{}
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

// StereoPanStream is an audio buffer that changes the stereo channel's signal
// based on the Panning.
type StereoPanStream struct {
	audio.ReadSeekCloser
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

	readN, err := s.ReadSeekCloser.Read(p[bufN:])
	if err != nil && err != io.EOF {
		return 0, err
	}

	// Align the buffer size in multiples of 4. The extra part is pushed to the buffer for the
	// next time.
	totalN := bufN + readN
	extra := totalN - totalN/4*4
	s.buf = append(s.buf, p[totalN-extra:totalN]...)
	alignedN := totalN - extra

	// This implementation uses a linear scale, ranging from -1 to 1, for stereo or mono sounds.
	// If pan = 0.0, the balance for the sound in each speaker is at 100% left and 100% right.
	// When pan is -1.0, only the left channel of the stereo sound is audible, when pan is 1.0,
	// only the right channel of the stereo sound is audible.
	// https://docs.unity3d.com/ScriptReference/AudioSource-panStereo.html
	ls := math.Min(s.pan*-1+1, 1)
	rs := math.Min(s.pan+1, 1)
	for i := 0; i < alignedN; i += 4 {
		lc := int16(float64(int16(p[i])|int16(p[i+1])<<8) * ls)
		rc := int16(float64(int16(p[i+2])|int16(p[i+3])<<8) * rs)

		p[i] = byte(lc)
		p[i+1] = byte(lc >> 8)
		p[i+2] = byte(rc)
		p[i+3] = byte(rc >> 8)
	}
	return alignedN, err
}

func (s *StereoPanStream) SetPan(pan float64) {
	s.pan = math.Min(math.Max(-1, pan), 1)
}

func (s *StereoPanStream) Pan() float64 {
	return s.pan
}

// NewStereoPanStreamFromReader returns a new StereoPanStream with a buffered src.
//
// The src's format must be linear PCM (16bits little endian, 2 channel stereo)
// without a header (e.g. RIFF header). The sample rate must be same as that
// of the audio context.
func NewStereoPanStreamFromReader(src audio.ReadSeekCloser) *StereoPanStream {
	return &StereoPanStream{
		ReadSeekCloser: src,
	}
}
