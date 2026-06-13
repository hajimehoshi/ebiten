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
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vmhost"
)

// TestEbitenPackageAsGuest runs the public ebiten package's own test suite as a virtualization guest,
// as an end-to-end test of the host/guest protocol: every draw and read-back in those tests is
// forwarded over the protocol to this host, which renders on the real GPU.
func TestEbitenPackageAsGuest(t *testing.T) {
	skipIfVMUnsupported(t)

	const guestPkg = "github.com/hajimehoshi/ebiten/v2"

	// Compile the guest's test binary in guest mode.
	guestBin := filepath.Join(t.TempDir(), "ebiten.test")
	build := exec.Command("go", "test", "-c", "-buildvcs=false", "-tags", "ebitenginevm", "-o", guestBin, guestPkg)
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		t.Fatalf("compiling the guest test binary failed: %v", err)
	}

	ln, endpoint := newGuestListener(t, "unix")
	if err := runGuest(ln, guestBin, endpoint); err != nil {
		t.Error(err)
	}
}

// runGuest launches the guest test binary as a process, drives its single tick (which runs the tests via
// MainWithRunLoop), and reports the result through the process exit code.
func runGuest(ln net.Listener, binary, endpoint string) error {
	args := []string{"-test.shuffle=on"}
	// The guest is itself a test binary, so let it honor -short like any other.
	if testing.Short() {
		args = append(args, "-test.short")
	}
	cmd := exec.Command(binary, args...)
	cmd.Env = append(os.Environ(), "EBITENGINE_VM_ENDPOINT="+endpoint)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting the guest failed: %w", err)
	}

	conn, err := ln.Accept()
	if err != nil {
		return fmt.Errorf("accepting the guest failed: %w", err)
	}

	guest, err := vmhost.NewGuestSession(conn, nil)
	if err != nil {
		return err
	}
	// The screen only sizes the guest here (320x240 device-independent pixels): the tests run inside
	// the guest, and the host never composites a frame.
	scale := ebiten.Monitor().DeviceScaleFactor()
	if err := guest.SetOutsideScreen(ebiten.NewImage(int(320*scale), int(240*scale))); err != nil {
		return err
	}

	// One tick runs the tests, as in an ordinary go test of the package (MainWithRunLoop runs m.Run
	// inside Update); it ends by returning Termination. WaitTick blocks until the session goroutine has
	// processed the tick or ended; the outcome is read from Err. Any error other than Termination means
	// driving failed mid-tick - most often the guest process died (e.g. a panicking test), which
	// surfaces here (and as a close error below) as a connection EOF. The guest's own output, including
	// any panic stack trace, has already gone to stderr, and its exit code below is authoritative.
	guest.AdvanceTick()
	guest.WaitTick()
	driveErr := guest.Err()
	if errors.Is(driveErr, ebiten.Termination) {
		driveErr = nil
	} else if driveErr != nil {
		driveErr = fmt.Errorf("driving the guest tests failed: %w", driveErr)
	}

	// Closing lets a still-running guest's RunGame return so its MainWithRunLoop exits with the test result
	// code; it also releases the host images mirrored for the guest.
	closeErr := guest.Close()
	if closeErr != nil {
		closeErr = fmt.Errorf("closing the guest failed: %w", closeErr)
	}

	// Reap the guest; a non-zero exit code is the authoritative failure (a failed or crashed test), with
	// the details already on stderr above.
	waitErr := cmd.Wait()
	if waitErr != nil {
		var exit *exec.ExitError
		if errors.As(waitErr, &exit) {
			waitErr = fmt.Errorf("the guest tests failed (exit code %d); see its output above", exit.ExitCode())
		} else {
			waitErr = fmt.Errorf("waiting for the guest failed: %w", waitErr)
		}
	}

	return errors.Join(waitErr, driveErr, closeErr)
}
