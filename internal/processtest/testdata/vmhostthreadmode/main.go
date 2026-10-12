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

package main

import (
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
	"github.com/hajimehoshi/ebiten/v2/internal/vmprotocol"
)

type game struct {
	updated bool
}

func (g *game) Update() error {
	g.updated = true
	// A nil connection proves rejection happens before the handshake.
	if _, err := vmhost.NewGuestSession(nil, nil); err == nil {
		return errors.New("session during single-thread RunGame succeeded")
	}
	return ebiten.Termination
}

func (*game) Draw(*ebiten.Image) {
}

func (*game) Layout(int, int) (int, int) {
	return 16, 16
}

func run() error {
	mode := os.Args[1]
	if mode == "tag-before" {
		if _, err := vmhost.NewGuestSession(nil, nil); err == nil {
			return errors.New("session with single-thread build tag succeeded")
		}
		return nil
	}
	if mode == "before" {
		host, peer := net.Pipe()
		defer host.Close()
		defer peer.Close()
		go func() {
			_ = vmprotocol.PerformHandshake(peer, false)
		}()
		session, err := vmhost.NewGuestSession(host, nil)
		if err != nil {
			return err
		}
		defer session.Close()
	}
	ebiten.SetWindowVisible(false)
	var g game
	err := ebiten.RunGameWithOptions(&g, &ebiten.RunGameOptions{
		SingleThread: true,
	})
	if mode == "before" {
		if err == nil {
			return errors.New("single-thread RunGame with an existing session succeeded")
		}
		if g.updated {
			return errors.New("unsupported game reached Update")
		}
		return nil
	}
	if err != nil {
		return err
	}
	if !g.updated {
		return errors.New("game did not reach Update")
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
