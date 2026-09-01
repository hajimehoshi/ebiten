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

// This program is a host that drives a guest session before [ebiten.RunGame] starts, so its graphics
// driver does not exist yet. A guest query then cannot be served, and the session must report an
// error rather than panicking. It exits 0 when it does.
package main

import (
	"errors"
	"fmt"
	"image"
	"net"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
	"github.com/hajimehoshi/ebiten/v2/internal/vmprotocol"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	hostConn, guestConn := net.Pipe()
	defer func() {
		_ = guestConn.Close()
	}()

	// A guest that answers the host's first message with a read-back query.
	go func() {
		if err := vmprotocol.PerformHandshake(guestConn, false); err != nil {
			return
		}
		enc := vmprotocol.NewEncoder(guestConn)
		dec := vmprotocol.NewDecoder(guestConn)
		for {
			var msg vmprotocol.HostMessage
			if err := dec.DecodeHostMessage(&msg); err != nil {
				return
			}
			_ = enc.EncodeGuestMessage(&vmprotocol.GuestMessage{
				Kind:        vmprotocol.GuestMessageKindQueryReadPixels,
				ReadImageID: 1,
				ReadRegions: []image.Rectangle{image.Rect(0, 0, 4, 4)},
			})
			_ = enc.EncodeGuestMessage(&vmprotocol.GuestMessage{Kind: vmprotocol.GuestMessageKindDone})
		}
	}()

	guest, err := vmhost.NewGuestSession(hostConn, nil)
	if err != nil {
		return err
	}
	if err := guest.SetOutsideScreen(ebiten.NewImage(320, 240)); err != nil {
		return err
	}
	guest.AdvanceTicks(1)
	if guest.WaitTicks() {
		return errors.New("the guest's tick was processed; want the session to fail before the game starts")
	}
	if guest.Err() == nil {
		return errors.New("the session ended without an error")
	}
	return guest.Close()
}
