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
// starting-point template to copy and adapt, not a stable command. See the run-ebitengine-app-headless skill.
//
// Written against ebiten commit de81775da (2026-07-02); exp/vmhost is experimental, so update this
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
	"slices"
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
	guest    *vmhost.GuestSession
	screen   *ebiten.Image
	dumped   bool
	closed   bool
	closeErr error

	// audioStreams holds the guest's currently-open streams. onAudioStream appends to it during
	// AdvanceTicks and WaitTicks in Update; appendAudioStreams reads it (dropping any the guest has
	// closed) on the same goroutine, so no lock is needed.
	audioStreams []*vmhost.GuestAudioStream
}

// onAudioStream records a new guest audio stream. It is the session's OnAudioStream handler, invoked
// during AdvanceTicks and WaitTicks; it stashes the handle for appendAudioStreams to hand back.
func (d *driver) onAudioStream(s *vmhost.GuestAudioStream) {
	d.audioStreams = append(d.audioStreams, s)
}

// appendAudioStreams appends the guest's currently-open audio streams to dst and returns the extended
// slice, dropping any the guest has closed (IsClosed) — a closed stream never plays again and its Read is
// at io.EOF — so the tracked set stays live instead of growing without bound. Safe to call from Update.
func (d *driver) appendAudioStreams(dst []*vmhost.GuestAudioStream) []*vmhost.GuestAudioStream {
	d.audioStreams = slices.DeleteFunc(d.audioStreams, func(s *vmhost.GuestAudioStream) bool {
		return s.IsClosed()
	})
	return append(dst, d.audioStreams...)
}

func (d *driver) Update() error {
	defer func() {
		if err := d.guest.Close(); err != nil {
			d.closeErr = err
		}
		d.closed = true
	}()

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
		// The session ended (guest terminated, crashed, or timed out; see Err()), or no frame was ever
		// requested (a zero-tick run). Either way d.dumped stays false and xmain reports it.
		return ebiten.Termination
	}
	if !d.guest.CompositeFrame() {
		return errors.New("compositing the guest frame failed")
	}

	if err := d.dump(); err != nil {
		return err
	}
	d.dumped = true
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
	var processDone bool
	defer func() {
		if processDone {
			return
		}
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	if dl, ok := ln.(interface{ SetDeadline(time.Time) error }); ok {
		if err := dl.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
			return err
		}
	}
	conn, err := ln.Accept()
	if err != nil {
		return fmt.Errorf("accepting the guest failed: %w", err)
	}

	// The OnAudioStream handler below is a method on d, so build d before the session; its guest field is
	// filled in once NewGuestSession returns.
	d := &driver{}
	// NewGuestSession performs the protocol handshake over the connection; it needs no graphics context,
	// so it can run before RunGame. IdleTimeout fails fast if the guest wedges. OnAudioStream collects the
	// guest's audio streams as they start (see the "Observing audio" section of SKILL.md).
	guest, err := vmhost.NewGuestSession(conn, &vmhost.NewGuestSessionOptions{
		IdleTimeout:   10 * time.Second,
		OnAudioStream: d.onAudioStream,
	})
	if err != nil {
		return err
	}
	d.guest = guest

	// Drive the guest from a hidden host window.
	ebiten.SetWindowVisible(false)
	runErr := ebiten.RunGame(d)

	// Close is called from Update, which is inside the host's frame. After that the guest's RunGame
	// returns and cmd.Wait frees the process's resources.
	if d.closeErr != nil {
		slog.Warn("closing the guest session failed", "err", d.closeErr)
	}
	if !d.closed {
		_ = cmd.Process.Kill()
	}
	if err := cmd.Wait(); err != nil {
		slog.Warn("waiting for the guest process failed", "err", err)
	}
	processDone = true

	if runErr != nil {
		return fmt.Errorf("host run failed: %w", runErr)
	}
	if err := guest.Err(); err != nil && !errors.Is(err, ebiten.Termination) {
		return fmt.Errorf("guest session failed: %w", err)
	}
	if !d.dumped {
		// Reachable when the guest terminated itself (its Update returned ebiten.Termination) before the
		// final frame rendered, or when the run advanced no ticks so no frame was requested.
		if err := guest.Err(); err != nil {
			return fmt.Errorf("the guest ended before the frame was captured: %w", err)
		}
		return errors.New("no frame was captured; the run must advance at least one tick")
	}
	slog.Info("wrote frame", "path", *out, "ticks", *ticks)
	return nil
}
