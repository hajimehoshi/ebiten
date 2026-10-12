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

//go:build ignore

package main

import (
	"fmt"
	"image"
	"net"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
	"github.com/hajimehoshi/ebiten/v2/internal/vmprotocol"
)

type game struct {
	guest  *vmhost.GuestSession
	resume chan struct{}
	answer chan error
}

func (g *game) runGuest(conn net.Conn) {
	defer conn.Close()
	if err := vmprotocol.PerformHandshake(conn, false); err != nil {
		g.answer <- err
		return
	}
	enc := vmprotocol.NewEncoder(conn)
	dec := vmprotocol.NewDecoder(conn)
	var id uint64
	for {
		var msg vmprotocol.HostMessage
		if err := dec.DecodeHostMessage(&msg); err != nil {
			return
		}
		if msg.Kind == vmprotocol.HostMessageKindAdvanceTick {
			if id == 0 {
				if err := enc.EncodeGuestMessage(&vmprotocol.GuestMessage{
					Kind: vmprotocol.GuestMessageKindGraphicsCommands,
					GraphicsCommands: []vmprotocol.GraphicsCommand{
						{
							Kind:    vmprotocol.GraphicsCommandKindNewImage,
							ImageID: 1,
							Width:   1,
							Height:  1,
						},
						{
							Kind:    vmprotocol.GraphicsCommandKindWritePixels,
							ImageID: 1,
							Regions: []image.Rectangle{image.Rect(0, 0, 1, 1)},
							Pixels:  [][]byte{{1, 2, 3, 255}},
						},
					},
				}); err != nil {
					g.answer <- err
					return
				}
				if err := enc.EncodeGuestMessage(&vmprotocol.GuestMessage{
					Kind:        vmprotocol.GuestMessageKindQueryReadPixelsAsync,
					ReadImageID: 1,
					ReadRegions: []image.Rectangle{image.Rect(0, 0, 1, 1)},
				}); err != nil {
					g.answer <- err
					return
				}
				var answer vmprotocol.HostMessage
				if err := dec.DecodeHostMessage(&answer); err != nil {
					g.answer <- err
					return
				}
				if answer.Err != "" || answer.ReadbackID == 0 {
					g.answer <- fmt.Errorf("starting read-back failed: %+v", answer)
					return
				}
				id = answer.ReadbackID
			} else {
				<-g.resume
				if err := enc.EncodeGuestMessage(&vmprotocol.GuestMessage{
					Kind:       vmprotocol.GuestMessageKindQueryPollReadPixels,
					ReadbackID: id,
				}); err != nil {
					g.answer <- err
					return
				}
				var answer vmprotocol.HostMessage
				if err := dec.DecodeHostMessage(&answer); err != nil {
					g.answer <- err
					return
				}
				if answer.Err == "" {
					g.answer <- fmt.Errorf("read-back was not aborted: %+v", answer)
					return
				}
				g.answer <- nil
			}
		}
		if err := enc.EncodeGuestMessage(&vmprotocol.GuestMessage{
			Kind: vmprotocol.GuestMessageKindDone,
		}); err != nil {
			return
		}
	}
}

func (g *game) Update() error {
	hostConn, guestConn := net.Pipe()
	go g.runGuest(guestConn)
	guest, err := vmhost.NewGuestSession(hostConn, nil)
	if err != nil {
		return err
	}
	g.guest = guest
	if err := guest.SetOutsideScreen(ebiten.NewImage(16, 16)); err != nil {
		return err
	}
	guest.AdvanceTicks(1)
	if !guest.WaitTicks() {
		return fmt.Errorf("starting tick failed: %w", guest.Err())
	}
	guest.AdvanceTicks(1)
	return ebiten.Termination
}

func (*game) Draw(*ebiten.Image) {}

func (*game) Layout(int, int) (int, int) { return 16, 16 }

func main() {
	ebiten.SetWindowVisible(false)
	g := &game{
		resume: make(chan struct{}),
		answer: make(chan error, 1),
	}
	if err := ebiten.RunGame(g); err != nil {
		panic(err)
	}
	close(g.resume)
	select {
	case err := <-g.answer:
		if err != nil {
			panic(err)
		}
	case <-time.After(5 * time.Second):
		panic("read-back poll blocked after RunGame returned")
	}
	closed := make(chan error, 1)
	go func() { closed <- g.guest.Close() }()
	select {
	case err := <-closed:
		if err != nil {
			panic(err)
		}
	case <-time.After(5 * time.Second):
		panic("Close blocked after RunGame returned")
	}
}
