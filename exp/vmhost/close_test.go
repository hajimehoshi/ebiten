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
	"bytes"
	"net"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
)

// TestCloseShutsGuestDownCleanly asserts that a closed session lets the guest process exit by itself
// with code 0: losing the host ends the guest's [ebiten.RunGame] without an error. A host that spawned
// the guest can therefore wait for the process and read a clean run from its exit code, rather than
// killing it and disregarding the result.
func TestCloseShutsGuestDownCleanly(t *testing.T) {
	skipIfVMUnsupported(t)

	guestBin := buildGuest(t, "./testdata/guest", activateByEnv)
	ln, endpoint := newGuestListener(t, "unix")

	var stderr bytes.Buffer
	cmd := exec.Command(guestBin)
	cmd.Env = append(os.Environ(), "EBITENGINE_VM_ENDPOINT="+endpoint)
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting the guest failed: %v", err)
	}

	conn, err := ln.Accept()
	if err != nil {
		t.Fatalf("accepting the guest failed: %v", err)
	}
	guest, err := vmhost.NewGuestSession(conn, nil)
	if err != nil {
		t.Fatal(err)
	}

	// The guest renders at the host's device scale factor, so the screen is physical-sized.
	scale := ebiten.Monitor().DeviceScaleFactor()
	if err := guest.SetOutsideScreen(ebiten.NewImage(int(320*scale), int(240*scale))); err != nil {
		t.Fatal(err)
	}
	guest.AdvanceTicks(3)
	if !guest.WaitTicks() {
		t.Fatalf("advancing the guest failed: %v", guest.Err())
	}

	if err := guest.Close(); err != nil {
		t.Fatalf("closing the guest session failed: %v", err)
	}

	// The guest exits on its own: no Kill, and Wait reports exit code 0.
	waitErr := make(chan error, 1)
	go func() {
		waitErr <- cmd.Wait()
	}()
	select {
	case err := <-waitErr:
		if err != nil {
			t.Fatalf("the guest did not exit cleanly after Close: %v\nguest stderr:\n%s", err, &stderr)
		}
	case <-time.After(30 * time.Second):
		_ = cmd.Process.Kill()
		<-waitErr
		t.Fatalf("the guest did not exit after Close\nguest stderr:\n%s", &stderr)
	}
	if code := cmd.ProcessState.ExitCode(); code != 0 {
		t.Errorf("the guest's exit code = %d; want 0", code)
	}
}

func TestCloseIsNotDelayedByTheIdleTimeout(t *testing.T) {
	skipIfVMUnsupported(t)

	const idleTimeout = 10 * time.Second
	const budget = time.Second

	conn := newSilentConn()
	guest, err := vmhost.NewGuestSession(conn, &vmhost.NewGuestSessionOptions{
		IdleTimeout: idleTimeout,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := guest.SetOutsideScreen(ebiten.NewImage(320, 240)); err != nil {
		t.Fatal(err)
	}
	<-conn.writing

	closed := make(chan error, 1)
	go func() {
		closed <- guest.Close()
	}()
	<-conn.poked
	conn.release()

	select {
	case err := <-closed:
		if err != nil {
			t.Errorf("closing the guest session failed: %v", err)
		}
	case <-time.After(budget):
		t.Fatalf("Close did not return within %v; the deadline poke was overwritten by an idle-timeout refresh", budget)
	}
}

// silentConn is a connection whose peer answers the handshake and then never speaks again, so a read on
// it ends only when its read deadline expires. Its first write after the handshake blocks until release
// is called, and its writes ignore deadlines.
type silentConn struct {
	// writing is closed once the first write after the handshake is entered, and poked once SetDeadline
	// is called. released lets that write finish, and closed ends every wait.
	writing  chan struct{}
	poked    chan struct{}
	released chan struct{}
	closed   chan struct{}

	writingOnce  sync.Once
	pokedOnce    sync.Once
	releasedOnce sync.Once
	closedOnce   sync.Once

	mu sync.Mutex
	// handshake holds the preamble the session wrote, served back by Read as the peer's answer.
	handshake     []byte
	handshakeSent bool
	// readDeadline is the deadline reads end at; changed is closed and replaced whenever it is set, so a
	// blocked read observes the new one.
	readDeadline time.Time
	changed      chan struct{}
}

func newSilentConn() *silentConn {
	return &silentConn{
		writing:  make(chan struct{}),
		poked:    make(chan struct{}),
		released: make(chan struct{}),
		closed:   make(chan struct{}),
		changed:  make(chan struct{}),
	}
}

// release lets the blocked write finish.
func (c *silentConn) release() {
	c.releasedOnce.Do(func() {
		close(c.released)
	})
}

func (c *silentConn) Write(p []byte) (int, error) {
	if c.recordHandshake(p) {
		return len(p), nil
	}
	c.writingOnce.Do(func() {
		close(c.writing)
	})
	select {
	case <-c.released:
	case <-c.closed:
		return 0, net.ErrClosed
	}
	return len(p), nil
}

// recordHandshake keeps the first write, the handshake preamble, for Read to serve back. It reports
// whether p was that write.
func (c *silentConn) recordHandshake(p []byte) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.handshakeSent {
		return false
	}
	c.handshakeSent = true
	c.handshake = append(c.handshake, p...)
	return true
}

func (c *silentConn) Read(p []byte) (int, error) {
	if n := c.takeHandshake(p); n > 0 {
		return n, nil
	}
	for {
		deadline, changed := c.readDeadlineState()
		if !deadline.IsZero() && !time.Now().Before(deadline) {
			return 0, os.ErrDeadlineExceeded
		}
		if err := waitForDeadline(deadline, changed, c.closed); err != nil {
			return 0, err
		}
	}
}

// waitForDeadline blocks until deadline (when one is set), changed is closed, or closed is closed,
// which it reports as an error.
func waitForDeadline(deadline time.Time, changed, closed <-chan struct{}) error {
	var expired <-chan time.Time
	if !deadline.IsZero() {
		timer := time.NewTimer(time.Until(deadline))
		defer timer.Stop()
		expired = timer.C
	}
	select {
	case <-expired:
	case <-changed:
	case <-closed:
		return net.ErrClosed
	}
	return nil
}

// takeHandshake copies whatever is left of the peer's handshake answer into p.
func (c *silentConn) takeHandshake(p []byte) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := copy(p, c.handshake)
	c.handshake = c.handshake[n:]
	return n
}

func (c *silentConn) SetDeadline(t time.Time) error {
	c.pokedOnce.Do(func() {
		close(c.poked)
	})
	c.setReadDeadline(t)
	return nil
}

func (c *silentConn) SetReadDeadline(t time.Time) error {
	c.setReadDeadline(t)
	return nil
}

func (c *silentConn) SetWriteDeadline(time.Time) error {
	return nil
}

// setReadDeadline records t and wakes a blocked read so that it observes the new deadline.
func (c *silentConn) setReadDeadline(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readDeadline = t
	close(c.changed)
	c.changed = make(chan struct{})
}

// readDeadlineState returns the current read deadline and the channel closed when it is replaced.
func (c *silentConn) readDeadlineState() (time.Time, <-chan struct{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.readDeadline, c.changed
}

func (c *silentConn) Close() error {
	c.closedOnce.Do(func() {
		close(c.closed)
	})
	return nil
}

func (c *silentConn) LocalAddr() net.Addr {
	return silentAddr{}
}

func (c *silentConn) RemoteAddr() net.Addr {
	return silentAddr{}
}

// silentAddr is the address of both ends of a silentConn, which has no transport.
type silentAddr struct{}

func (silentAddr) Network() string {
	return "silent"
}

func (silentAddr) String() string {
	return "silent"
}
