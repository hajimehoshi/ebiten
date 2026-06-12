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
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/vmhost"
)

// guestActivation selects how a test guest is activated.
type guestActivation int

const (
	// activateByEnv builds the guest with the ebitenginevm build tag and passes the endpoint in
	// EBITENGINE_VM_ENDPOINT.
	activateByEnv guestActivation = iota
	// activateByOptions builds the guest without the build tag and passes the endpoint as a
	// command-line argument that the fixture feeds into RunGameOptions.VMGuestEndpoint.
	activateByOptions
)

// startGuest builds the guest program at pkgPath (an import path or a path relative to this package's
// directory, e.g. ./testdata/atlas), launches it pointed at a fresh listener on the given network
// ("unix" or "tcp"), accepts the connection, and returns a GuestSession ready to drive. All resources
// are released via t.Cleanup.
func startGuest(t *testing.T, pkgPath string, activation guestActivation, network string) *vmhost.GuestSession {
	t.Helper()
	skipIfVMUnsupported(t)

	guestBin := filepath.Join(t.TempDir(), "guest")
	if runtime.GOOS == "windows" {
		guestBin += ".exe"
	}
	// VCS stamping requires git, which fails on a repository mounted into a container under a
	// different owner (the Steam CI job).
	buildArgs := []string{"build", "-buildvcs=false"}
	if activation == activateByEnv {
		buildArgs = append(buildArgs, "-tags", "ebitenginevm")
	}
	buildArgs = append(buildArgs, "-o", guestBin, pkgPath)
	if out, err := exec.Command("go", buildArgs...).CombinedOutput(); err != nil {
		t.Fatalf("building the guest failed: %v\n%s", err, out)
	}

	ln, endpoint := newGuestListener(t, network)

	var cmd *exec.Cmd
	switch activation {
	case activateByEnv:
		cmd = exec.Command(guestBin)
		cmd.Env = append(os.Environ(), "EBITENGINE_VM_ENDPOINT="+endpoint)
	case activateByOptions:
		cmd = exec.Command(guestBin, endpoint)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting the guest failed: %v", err)
	}
	t.Cleanup(func() {
		if err := cmd.Wait(); err != nil {
			t.Errorf("waiting for the guest failed: %v", err)
		}
	})

	conn, err := ln.Accept()
	if err != nil {
		t.Fatalf("accepting the guest failed: %v", err)
	}

	guest, err := vmhost.NewGuestSession(conn, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := guest.Close(); err != nil {
			t.Errorf("closing the guest failed: %v", err)
		}
	})

	return guest
}

// newGuestListener opens a listener for the guest to dial on the given network: a fresh temporary
// socket for "unix", or an ephemeral loopback port for "tcp". It returns the listener and its
// endpoint URL, with an accept deadline set so that a guest that never dials fails the test instead
// of blocking it. All resources are released via t.Cleanup.
func newGuestListener(t *testing.T, network string) (net.Listener, string) {
	t.Helper()

	var ln net.Listener
	switch network {
	case "unix":
		// Not t.TempDir, whose path embeds the test name: the socket path must fit in sun_path
		// (~104 bytes on macOS), so the directory name is kept short.
		sockDir, err := os.MkdirTemp("", "vg")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if err := os.RemoveAll(sockDir); err != nil {
				t.Errorf("removing the socket dir failed: %v", err)
			}
		})
		ln, err = net.Listen("unix", filepath.Join(sockDir, "g.sock"))
		if err != nil {
			t.Fatal(err)
		}
	case "tcp":
		// Loopback only: the guest is a local process, and a non-loopback listener could trigger
		// firewall prompts.
		var err error
		ln, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatalf("unknown network: %s", network)
	}
	t.Cleanup(func() {
		if err := ln.Close(); err != nil {
			t.Errorf("closing the listener failed: %v", err)
		}
	})

	// Both *net.UnixListener and *net.TCPListener provide SetDeadline.
	if err := ln.(interface{ SetDeadline(time.Time) error }).SetDeadline(time.Now().Add(60 * time.Second)); err != nil {
		t.Fatalf("setting the accept deadline failed: %v", err)
	}

	endpoint, err := vmhost.EndpointURLFromAddr(ln.Addr())
	if err != nil {
		t.Fatal(err)
	}
	return ln, endpoint
}

// skipIfVMUnsupported skips tests on platforms where the vm host/guest model does not run.
func skipIfVMUnsupported(t *testing.T) {
	t.Helper()
	switch runtime.GOOS {
	case "android", "ios", "js":
		t.Skipf("the vm is not supported on %s", runtime.GOOS)
	}
}
