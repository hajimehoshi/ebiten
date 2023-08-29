// Copyright 2023 The Ebitengine Authors
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

//go:build ignore

package main

import (
	"errors"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

type emptyStream struct {
	length int64
	n      int64
}

func (e *emptyStream) Read(buf []byte) (int, error) {
	n := int64(len(buf))
	if e.n+n >= e.length {
		n := e.length - e.n
		e.n = e.length
		return int(n), io.EOF
	}
	e.n += n
	return int(n), nil
}

func (e *emptyStream) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		e.n = offset
	case io.SeekCurrent:
		e.n += offset
	case io.SeekEnd:
		e.n = e.length + offset
	}
	if e.n > e.length || e.n < 0 {
		return 0, errors.New("out of range")
	}
	return e.n, nil
}

type Game struct {
	playerCount         int
	finishedPlayerCount int
	tickCount           int

	m sync.Mutex
}

func (g *Game) countUpFinishedPlayer() {
	g.m.Lock()
	defer g.m.Unlock()
	g.finishedPlayerCount++
}

func (g *Game) Update() error {
	g.tickCount++
	if g.tickCount > 600 {
		return errors.New("time out")
	}

	g.m.Lock()
	c := g.finishedPlayerCount
	g.m.Unlock()
	if g.playerCount == c {
		return ebiten.Termination
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
}

func (g *Game) Layout(width, height int) (int, int) {
	return width, height
}

func main() {
	// Drivers might not be available, especially on Linux on GitHub Actions.
	// TODO: Enable this by install a dummy driver.
	if strings.TrimSpace(os.Getenv("GITHUB_ACTIONS")) == "true" && runtime.GOOS == "linux" {
		return
	}

	game := &Game{
		playerCount: 1000,
	}

	ctx := audio.NewContext(48000)
	var players []*audio.Player
	for i := 0; i < game.playerCount; i++ {
		p, err := ctx.NewPlayer(&emptyStream{length: 48000 * 2 * 2})
		if err != nil {
			panic(err)
		}
		players = append(players, p)

		// Play players in different goroutines from the game's goroutine in order to call the context's gcPlayers and addPlayer
		// at the same time.
		go func() {
			p.Play()
			for i := 0; i < 3; i++ {
				for {
					if !p.IsPlaying() {
						p.Rewind()
						p.Play()
						break
					}
					time.Sleep(100 * time.Millisecond)
				}
			}
			game.countUpFinishedPlayer()
		}()
	}

	if err := ebiten.RunGame(game); err != nil {
		panic(err)
	}
}
