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

//go:build ebitenginevm

// This guest exercises audio forwarding with two players: a finite ramp source (every sample
// component is its 1-based index) starts at tick 3, and an infinite constant 0.25 source at half
// volume starts at tick 4 and is closed at tick 8. The host pulls each player's samples on demand and
// asserts the two streams separately, byte-exactly — the ramp drains to its end, the flat source stays
// raw 0.25 (the reported volume is not applied) until the close removes it.
//
// It is launched by a host; see vmhost's audio test.
package main

import (
	"bytes"
	"encoding/binary"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

const sampleRate = 48000

// rampFrames is the length of the ramp source in stereo frames.
const rampFrames = 2000

func rampBytes() []byte {
	b := make([]byte, 8*rampFrames)
	for i := range 2 * rampFrames {
		binary.LittleEndian.PutUint32(b[4*i:], math.Float32bits(float32(i+1)))
	}
	return b
}

// flatReader endlessly yields sample components of value 0.25.
type flatReader struct{}

func (flatReader) Read(p []byte) (int, error) {
	n := len(p) - len(p)%4
	for i := 0; i < n; i += 4 {
		binary.LittleEndian.PutUint32(p[i:], math.Float32bits(0.25))
	}
	return n, nil
}

type game struct {
	ticks int
	ramp  *audio.Player
	flat  *audio.Player
}

func (g *game) Update() error {
	g.ticks++
	switch g.ticks {
	case 3:
		g.ramp.Play()
	case 4:
		// SetVolume initializes the audio device, so it must not run before RunGame: the device would be
		// a real one instead of the virtual forwarding one.
		g.flat.SetVolume(0.5)
		g.flat.Play()
	case 8:
		// Close a player before its (infinite) source ends, exercising the explicit close signal: the
		// host must drop the player even though the source never reaches EOF.
		if err := g.flat.Close(); err != nil {
			return err
		}
	}
	// The guest never terminates on its own: the host reads the streams and then closes the session.
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func main() {
	ctx := audio.NewContext(sampleRate)
	ramp, err := ctx.NewPlayerF32(bytes.NewReader(rampBytes()))
	if err != nil {
		log.Fatal(err)
	}
	flat, err := ctx.NewPlayerF32(flatReader{})
	if err != nil {
		log.Fatal(err)
	}
	if err := ebiten.RunGame(&game{ramp: ramp, flat: flat}); err != nil {
		log.Fatal(err)
	}
}
