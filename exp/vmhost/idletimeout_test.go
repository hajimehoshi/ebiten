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

package vmhost_test

import (
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
	"github.com/hajimehoshi/ebiten/v2/internal/vmprotocol"
)

const testIdleTimeout = 100 * time.Millisecond

func TestIdleTimeoutAtHandshake(t *testing.T) {
	skipIfVMUnsupported(t)

	hostConn, guestConn := net.Pipe()
	defer func() {
		_ = hostConn.Close()
	}()
	defer func() {
		_ = guestConn.Close()
	}()

	// The peer connects but never speaks, so the handshake must time out.
	_, err := vmhost.NewGuestSession(hostConn, &vmhost.NewGuestSessionOptions{
		IdleTimeout: testIdleTimeout,
	})
	if !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Errorf("NewGuestSession with a silent peer: got %v, want %v", err, os.ErrDeadlineExceeded)
	}
}

func TestIdleTimeoutMidOperation(t *testing.T) {
	skipIfVMUnsupported(t)

	hostConn, guestConn := net.Pipe()
	defer func() {
		_ = hostConn.Close()
	}()
	defer func() {
		_ = guestConn.Close()
	}()

	// The fake guest handshakes and concludes the outside-size operation, then receives the
	// AdvanceTick message and goes silent, as if wedged mid-Update.
	go func() {
		if err := vmprotocol.PerformHandshake(guestConn, false); err != nil {
			return
		}
		dec := vmprotocol.NewDecoder(guestConn)
		enc := vmprotocol.NewEncoder(guestConn)
		var msg vmprotocol.HostMessage
		if err := dec.DecodeHostMessage(&msg); err != nil {
			return
		}
		if err := enc.EncodeGuestMessage(&vmprotocol.GuestMessage{Kind: vmprotocol.GuestMessageKindDone}); err != nil {
			return
		}
		if err := dec.DecodeHostMessage(&msg); err != nil {
			return
		}
		// Block until the deferred close above unblocks the read.
		_ = dec.DecodeHostMessage(&msg)
	}()

	guest, err := vmhost.NewGuestSession(hostConn, &vmhost.NewGuestSessionOptions{
		IdleTimeout: testIdleTimeout,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := guest.SetOutsideScreen(ebiten.NewImage(320, 240)); err != nil {
		t.Fatal(err)
	}
	// AdvanceTicks is asynchronous; the tick blocks the session goroutine on the silent guest until the
	// idle timeout fires. WaitTicks blocks until that happens, and the error surfaces from Err.
	guest.AdvanceTicks(1)
	guest.WaitTicks()
	if err := guest.Err(); !errors.Is(err, os.ErrDeadlineExceeded) {
		t.Errorf("a silent guest mid-tick: got %v, want %v", err, os.ErrDeadlineExceeded)
	}
}
