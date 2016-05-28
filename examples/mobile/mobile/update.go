// Copyright 2016 Hajime Hoshi
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

package mobile

import (
	"errors"
	"fmt"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/audio"
	"github.com/hajimehoshi/ebiten/ebitenutil"
	"github.com/hajimehoshi/ebiten/examples/common"
)

const (
	ScreenWidth  = 320
	ScreenHeight = 240
)

var (
	count        int
	gophersImage *ebiten.Image
)

func init() {
	var err error
	gophersImage, _, err = common.AssetImage("gophers.jpg", ebiten.FilterNearest)
	if err != nil {
		log.Fatal(err)
	}
}

const (
	screenWidth  = 320
	screenHeight = 240
	sampleRate   = 44100
	frequency    = 440
)

var audioContext *audio.Context

func init() {
	var err error
	// Let's lazy context creation!
	audioContext, err = audio.NewContext(sampleRate)
	if err != nil {
		log.Fatal(err)
	}
}

type stream struct {
	position int64
}

func (s *stream) Read(data []byte) (int, error) {
	if len(data)%4 != 0 {
		return 0, errors.New("len(data) % 4 must be 0")
	}
	const length = sampleRate / frequency // TODO: This should be integer?
	p := s.position / 4
	for i := 0; i < len(data)/4; i++ {
		const max = (1<<15 - 1) / 2
		b := int16(math.Sin(2*math.Pi*float64(p)/length) * max)
		data[4*i] = byte(b)
		data[4*i+1] = byte(b >> 8)
		data[4*i+2] = byte(b)
		data[4*i+3] = byte(b >> 8)
		p++
	}
	s.position += int64(len(data))
	s.position %= length * 4
	return len(data), nil
}

func (s *stream) Seek(offset int64, whence int) (int64, error) {
	const length = sampleRate / frequency
	switch whence {
	case 0:
		s.position = offset
	case 1:
		s.position += offset
	case 2:
		return 0, errors.New("whence must be 0 or 1")
	}
	s.position %= length * 4
	return s.position, nil
}

func (s *stream) Close() error {
	return nil
}

var player *audio.Player

func Update(screen *ebiten.Image) error {
	if player == nil {
		var err error
		player, err = audio.NewPlayer(audioContext, &stream{})
		if err != nil {
			return err
		}
		if err := player.Play(); err != nil {
			return err
		}
	}
	if err := audioContext.Update(); err != nil {
		return err
	}

	count++
	w, h := gophersImage.Size()
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(w)/2, -float64(h)/2)
	op.GeoM.Rotate(float64(count%360) * 2 * math.Pi / 360)
	op.GeoM.Translate(ScreenWidth/2, ScreenHeight/2)
	if err := screen.DrawImage(gophersImage, op); err != nil {
		return err
	}
	msg := ""
	for _, t := range ebiten.Touches() {
		x, y := t.Position()
		msg += fmt.Sprintf("ID: %d, (%d, %d)\n", t.ID(), x, y)
	}
	ebitenutil.DebugPrint(screen, msg)
	return nil
}
