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
	"io"
	"net"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
)

// TestNewGuestSessionHandshakeError asserts that session establishment fails when the peer is not a
// valid vmguest (here, a wrong handshake magic), rather than proceeding with a broken session.
func TestNewGuestSessionHandshakeError(t *testing.T) {
	skipIfVMUnsupported(t)

	hostConn, guestConn := net.Pipe()
	defer func() {
		_ = hostConn.Close()
	}()
	defer func() {
		_ = guestConn.Close()
	}()

	// A peer that is not a vmguest: it answers the handshake with the wrong magic.
	go func() {
		var buf [8]byte
		if _, err := io.ReadFull(guestConn, buf[:]); err != nil {
			return
		}
		_, _ = guestConn.Write([]byte("nope\x00\x00\x00\x01"))
	}()

	if _, err := vmhost.NewGuestSession(hostConn, nil); err == nil {
		t.Fatal("NewGuestSession with a non-vmguest peer succeeded; want an error")
	}
}
