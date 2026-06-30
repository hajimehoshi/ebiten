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

// This is a headless vmhost driver: it launches an Ebitengine app as a virtualization guest, runs it
// through its ticks faster than real time while injecting input, and reads the rendered frame back as
// pixels — all from a hidden host window, with no changes to the app's source. Edit the INPUT SCRIPT
// block in Update to script the keys, clicks, touches, and gamepads the guest observes. It is a
// starting-point template to copy and adapt, not a stable command. See the test-ebitengine-app skill.
//
// Written against ebiten commit c8db8fd6d (2026-06-30); exp/vmhost is experimental, so update this
// driver if its API has moved since.
package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/png"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
)

var (
	pkg      = flag.String("pkg", "", "package path of the guest app to test, e.g. ./examples/paint")
	ticks    = flag.Int("ticks", 60, "number of ticks to run before dumping the frame and exiting")
	out      = flag.String("out", "frame.png", "PNG output path for the final frame")
	logicalW = flag.Int("w", 320, "logical screen width in device-independent pixels")
	logicalH = flag.Int("h", 240, "logical screen height in device-independent pixels")
)

// driver is the host: an ebiten.Game that drives one guest and composites its final frame into its own
// (hidden) screen image. It does all its work in a single Update, then terminates.
type driver struct {
	guest  *vmhost.GuestSession
	screen *ebiten.Image
}

func (d *driver) Update() error {
	// Create the host-owned screen and size the guest to it. SetOutsideScreen must precede AdvanceTicks.
	// The guest renders at the host's device scale factor, so the image is sized in physical pixels.
	scale := ebiten.Monitor().DeviceScaleFactor()
	d.screen = ebiten.NewImage(int(float64(*logicalW)*scale), int(float64(*logicalH)*scale))
	if err := d.guest.SetOutsideScreen(d.screen); err != nil {
		return err
	}

	// ---- INPUT SCRIPT ---------------------------------------------------------------------------------
	// Drive the guest through its ticks. Advancing many ticks in one call compresses wall-clock time: the
	// guest runs them back-to-back as fast as it can rather than at the host's frame rate, and a tick
	// forwards no rendering (only a frame does). To feed input, split the run into segments and inject
	// between them; injected state is seen from the next tick on and persists until changed. Examples:
	//
	//	d.guest.PressKey(ebiten.KeyArrowRight) // held down for the whole run
	//	d.guest.AdvanceTicks(*ticks)
	//
	//	d.guest.AdvanceTicks(30)               // let it settle for 30 ticks
	//	d.guest.MoveCursor(160, 120)
	//	d.guest.PressMouseButton(ebiten.MouseButtonLeft)
	//	d.guest.AdvanceTicks(1)                // one tick with the button held
	//	d.guest.ReleaseMouseButton(ebiten.MouseButtonLeft)
	//	d.guest.AdvanceTicks(*ticks)
	//
	// Default — no input, just run *ticks ticks:
	d.guest.AdvanceTicks(*ticks)
	// ---------------------------------------------------------------------------------------------------

	// Render the final frame. WaitFrame blocks until every queued tick has run and the frame is rendered,
	// so the screen reflects the end of the run.
	d.guest.AdvanceFrame()
	if !d.guest.WaitFrame() {
		return ebiten.Termination // the session ended (guest terminated, crashed, or timed out); see Err()
	}
	d.guest.CompositeFrame() // composite the completed frame into d.screen

	if err := d.dump(); err != nil {
		return err
	}
	return ebiten.Termination
}

// dump writes the current screen to a PNG. ReadPixels returns premultiplied-alpha RGBA, which is
// identical to non-premultiplied bytes for an opaque screen (the common case).
func (d *driver) dump() error {
	b := d.screen.Bounds()
	img := image.NewRGBA(b)
	d.screen.ReadPixels(img.Pix)
	f, err := os.Create(*out)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func (d *driver) Draw(screen *ebiten.Image) {
	// Nothing to present: the host window is hidden, and the guest's frame is composited into d.screen
	// in Update. Draw d.screen here instead if a visible host window is wanted.
}

func (d *driver) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}

func main() {
	if err := xmain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func xmain() error {
	flag.Parse()
	if *pkg == "" {
		return errors.New("specify the guest package with -pkg, e.g. -pkg ./examples/paint")
	}

	// A short temp dir keeps the unix socket path within the OS limit (~104 bytes on macOS).
	dir, err := os.MkdirTemp("", "vg")
	if err != nil {
		return err
	}
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	ln, err := net.Listen("unix", filepath.Join(dir, "g.sock"))
	if err != nil {
		return err
	}
	defer func() {
		_ = ln.Close()
	}()

	endpoint, err := vmhost.EndpointURLFromAddr(ln.Addr())
	if err != nil {
		return err
	}

	// Build the guest with the ebitenginevm tag so its RunGame connects to this host instead of opening
	// a window. The app's own source is unchanged.
	guestBin := filepath.Join(dir, "guest")
	build := exec.Command("go", "build", "-tags", "ebitenginevm", "-o", guestBin, *pkg)
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("building the guest failed: %w", err)
	}

	cmd := exec.Command(guestBin)
	cmd.Env = append(os.Environ(), "EBITENGINE_VM_ENDPOINT="+endpoint)
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting the guest failed: %w", err)
	}

	conn, err := ln.Accept()
	if err != nil {
		return fmt.Errorf("accepting the guest failed: %w", err)
	}

	// NewGuestSession performs the protocol handshake over the connection; it needs no graphics context,
	// so it can run before RunGame. IdleTimeout fails fast if the guest wedges.
	guest, err := vmhost.NewGuestSession(conn, &vmhost.NewGuestSessionOptions{IdleTimeout: 10 * time.Second})
	if err != nil {
		return err
	}

	// Drive the guest from a hidden host window.
	ebiten.SetWindowVisible(false)
	runErr := ebiten.RunGame(&driver{guest: guest})

	// Close ends the session so the guest's RunGame returns and its process exits; cmd.Wait then waits
	// for that exit and frees the process's resources (without the Close it would block). The frame is
	// already captured, so teardown trouble is a warning, not a failed run.
	if err := guest.Close(); err != nil {
		slog.Warn("closing the guest session failed", "err", err)
	}
	if err := cmd.Wait(); err != nil {
		slog.Warn("waiting for the guest process failed", "err", err)
	}

	if runErr != nil {
		return fmt.Errorf("host run failed: %w", runErr)
	}
	if err := guest.Err(); err != nil && !errors.Is(err, ebiten.Termination) {
		return fmt.Errorf("guest session failed: %w", err)
	}
	slog.Info("wrote frame", "path", *out, "ticks", *ticks)
	return nil
}
