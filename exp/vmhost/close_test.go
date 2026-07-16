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
	"os"
	"os/exec"
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
