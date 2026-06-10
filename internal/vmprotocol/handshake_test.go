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

package vmprotocol_test

import (
	"bytes"
	"io"
	"net"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/vmprotocol"
)

// newPipe returns the two ends of an in-memory connection, closing both via t.Cleanup.
func newPipe(t *testing.T) (net.Conn, net.Conn) {
	t.Helper()
	c1, c2 := net.Pipe()
	t.Cleanup(func() {
		if err := c1.Close(); err != nil {
			t.Errorf("closing the initiator pipe end failed: %v", err)
		}
		if err := c2.Close(); err != nil {
			t.Errorf("closing the responder pipe end failed: %v", err)
		}
	})
	return c1, c2
}

func TestPerformHandshakeMatch(t *testing.T) {
	c1, c2 := newPipe(t)

	errc := make(chan error, 1)
	go func() {
		// Guest side: the responder receives first.
		errc <- vmprotocol.PerformHandshake(c2, false)
	}()

	// Host side: the initiator sends first.
	if err := vmprotocol.PerformHandshake(c1, true); err != nil {
		t.Errorf("initiator handshake failed: %v", err)
	}
	if err := <-errc; err != nil {
		t.Errorf("responder handshake failed: %v", err)
	}
}

func TestPerformHandshakeVersionMismatchInitiator(t *testing.T) {
	c1, c2 := newPipe(t)

	// A peer that announces an incompatible protocol version. As the responder it receives the
	// initiator's preamble first, then sends its (wrong) one.
	peerErr := make(chan error, 1)
	go func() {
		var buf [8]byte
		if _, err := io.ReadFull(c2, buf[:]); err != nil {
			peerErr <- err
			return
		}
		peerErr <- vmprotocol.WriteHandshakeForTesting(c2, vmprotocol.ProtocolVersion+1)
	}()

	if err := vmprotocol.PerformHandshake(c1, true); err == nil {
		t.Fatal("expected a protocol version mismatch error, got nil")
	}
	if err := <-peerErr; err != nil {
		t.Errorf("the peer goroutine failed: %v", err)
	}
}

func TestPerformHandshakeVersionMismatchResponder(t *testing.T) {
	c1, c2 := newPipe(t)

	// A peer that announces an incompatible protocol version as the initiator. The responder must
	// still answer with its own preamble before rejecting, so the initiator too can name the mismatch
	// rather than seeing a bare connection close.
	type peerResult struct {
		reply [8]byte
		err   error
	}
	resc := make(chan peerResult, 1)
	go func() {
		var r peerResult
		r.err = vmprotocol.WriteHandshakeForTesting(c1, vmprotocol.ProtocolVersion+1)
		if r.err == nil {
			_, r.err = io.ReadFull(c1, r.reply[:])
		}
		resc <- r
	}()

	if err := vmprotocol.PerformHandshake(c2, false); err == nil {
		t.Fatal("expected a protocol version mismatch error, got nil")
	}
	r := <-resc
	if r.err != nil {
		t.Fatalf("the peer goroutine failed: %v", r.err)
	}
	var want bytes.Buffer
	if err := vmprotocol.WriteHandshakeForTesting(&want, vmprotocol.ProtocolVersion); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(r.reply[:], want.Bytes()) {
		t.Errorf("the responder's preamble = %q; want %q", r.reply[:], want.Bytes())
	}
}

func TestPerformHandshakeBadMagic(t *testing.T) {
	c1, c2 := newPipe(t)

	// A peer that isn't a vmguest: it sends a preamble with the wrong magic.
	peerErr := make(chan error, 1)
	go func() {
		var buf [8]byte
		if _, err := io.ReadFull(c2, buf[:]); err != nil {
			peerErr <- err
			return
		}
		_, err := c2.Write([]byte("nope\x00\x00\x00\x01"))
		peerErr <- err
	}()

	if err := vmprotocol.PerformHandshake(c1, true); err == nil {
		t.Fatal("expected a bad-magic error, got nil")
	}
	if err := <-peerErr; err != nil {
		t.Errorf("the peer goroutine failed: %v", err)
	}
}
