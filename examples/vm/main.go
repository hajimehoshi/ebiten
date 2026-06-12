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

//go:build !ebitenginevm

// This is a virtualization host that embeds any Ebitengine program as a guest process inside its own
// window — roughly `go run <package>`, but the program runs as a guest driven by this host. Run it from
// the repo root:
//
//	go run ./examples/vm [package]
//
// Enter a package (an import path or a path like ./examples/paint) in the panel and click Launch, or
// press Enter. The host builds it with -tags ebitenginevm, runs it pointed at a private socket,
// forwards the window's input to it, and composites its rendered frames into the window. Audio and
// gamepads are not forwarded yet.
package main

import (
	"errors"
	"fmt"
	"image"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ebitengine/debugui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vmhost"
)

// guestProcess bundles a running guest with the resources needed to tear it down.
type guestProcess struct {
	session *vmhost.GuestSession
	cmd     *exec.Cmd
	bin     string
	pkg     string // the package this guest was built from
}

// launchResult is the outcome of an asynchronous build-and-launch.
type launchResult struct {
	gp  *guestProcess
	err error
}

type Game struct {
	debugui debugui.DebugUI

	ln       net.Listener
	endpoint string
	dir      string

	pkg    string // the package text field's buffer
	status string

	launching bool
	results   chan launchResult

	gp          *guestProcess
	guestScreen *ebiten.Image

	// screenSet reports whether guestScreen has been handed to the current session via
	// SetOutsideScreen; it is cleared when the session or the screen changes.
	screenSet bool

	width  int
	height int
}

func (g *Game) Update() error {
	// Adopt an asynchronously-built guest once it is ready.
	select {
	case r := <-g.results:
		g.launching = false
		if r.err != nil {
			g.status = r.err.Error()
		} else {
			g.closeGuest()
			g.gp = r.gp
			g.screenSet = false
			g.status = "Running " + g.pkg
		}
	default:
	}

	state, err := g.debugui.Update(func(ctx *debugui.Context) error {
		ctx.Window("Virtualization host", image.Rect(10, 10, 360, 150), func(layout debugui.ContainerLayout) {
			ctx.Text("Package to run as a guest:")
			ctx.TextField(&g.pkg)
			ctx.Button("Launch").On(g.launchGuest)
			ctx.Text(g.status)
		})
		return nil
	})
	if err != nil {
		return err
	}

	// Launch on Enter while the text field is focused. The field's own On event also fires on blur, which
	// would relaunch whenever focus leaves it, so launching is driven only by the button and this explicit
	// Enter check.
	if state&debugui.InputCapturingStateFocus != 0 &&
		(inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyNumpadEnter)) {
		g.launchGuest()
	}

	if g.gp == nil {
		return nil
	}
	if err := g.advanceGuestTick(state); err != nil {
		if errors.Is(err, ebiten.Termination) {
			g.status = g.pkg + " exited"
		} else {
			g.status = "Guest error: " + err.Error()
		}
		g.closeGuest()
	}
	return nil
}

// launchGuest kicks off an asynchronous build-and-launch of g.pkg, unless one is already in flight. The
// build runs in a goroutine so the window stays responsive; the result is adopted in Update.
func (g *Game) launchGuest() {
	if g.launching || g.pkg == "" {
		return
	}
	// Don't rebuild the package that is already running; only a change (or a guest that has stopped)
	// warrants a relaunch.
	if g.gp != nil && g.gp.pkg == g.pkg {
		return
	}
	g.launching = true
	g.status = "Building " + g.pkg + " ..."
	pkg := g.pkg
	// The launch tick names the binary. Launches are serialized by g.launching, so at most one launch
	// starts per tick, and an old guest's binary may still be running (and locked, on Windows) while
	// the next one builds, so every launch needs its own path.
	bin := filepath.Join(g.dir, fmt.Sprintf("guest-%d", ebiten.Tick()))
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	go func() {
		gp, err := buildAndStartGuest(g.ln, bin, g.endpoint, pkg)
		g.results <- launchResult{gp: gp, err: err}
	}()
}

// advanceGuestTick gives the guest its screen, forwards the window's input to it, and advances it one tick.
func (g *Game) advanceGuestTick(state debugui.InputCapturingState) error {
	if g.width == 0 || g.height == 0 {
		return nil
	}
	// The guest renders at the host's device scale factor, so its screen is physical-sized: the
	// window's size in device-independent pixels times the scale.
	scale := ebiten.Monitor().DeviceScaleFactor()
	pw, ph := int(float64(g.width)*scale), int(float64(g.height)*scale)
	if g.guestScreen == nil || g.guestScreen.Bounds().Dx() != pw || g.guestScreen.Bounds().Dy() != ph {
		g.guestScreen = ebiten.NewImage(pw, ph)
		g.screenSet = false
	}
	if !g.screenSet {
		if err := g.gp.session.SetOutsideScreen(g.guestScreen); err != nil {
			return err
		}
		g.screenSet = true
	}
	if err := g.forwardInput(state); err != nil {
		return err
	}
	return g.gp.session.AdvanceTick()
}

// forwardInput sends the window's input to the guest, except input the debug UI is consuming (a hovered
// or focused widget), so the panel stays usable.
func (g *Game) forwardInput(state debugui.InputCapturingState) error {
	s := g.gp.session

	if state&debugui.InputCapturingStateFocus == 0 {
		for _, k := range inpututil.AppendJustPressedKeys(nil) {
			if err := s.PressKey(k); err != nil {
				return err
			}
		}
		for _, k := range inpututil.AppendJustReleasedKeys(nil) {
			if err := s.ReleaseKey(k); err != nil {
				return err
			}
		}
		for _, r := range ebiten.AppendInputChars(nil) {
			if err := s.TypeRune(r); err != nil {
				return err
			}
		}
	}

	if state&debugui.InputCapturingStateHover == 0 {
		// The guest fills the whole window, so cursor coordinates map directly.
		x, y := ebiten.CursorPosition()
		if err := s.MoveCursor(float64(x), float64(y)); err != nil {
			return err
		}
		for _, b := range []ebiten.MouseButton{ebiten.MouseButtonLeft, ebiten.MouseButtonRight, ebiten.MouseButtonMiddle} {
			if inpututil.IsMouseButtonJustPressed(b) {
				if err := s.PressMouseButton(b); err != nil {
					return err
				}
			}
			if inpututil.IsMouseButtonJustReleased(b) {
				if err := s.ReleaseMouseButton(b); err != nil {
					return err
				}
			}
		}
		if wx, wy := ebiten.Wheel(); wx != 0 || wy != 0 {
			if err := s.ScrollWheel(wx, wy); err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.gp != nil && g.screenSet {
		g.gp.session.AdvanceFrame()
		// guestScreen is physical-sized; scale it back down to fill the logical screen.
		scale := ebiten.Monitor().DeviceScaleFactor()
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(1/scale, 1/scale)
		screen.DrawImage(g.guestScreen, op)
	}
	g.debugui.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	g.width, g.height = outsideWidth, outsideHeight
	return outsideWidth, outsideHeight
}

// closeGuest stops the current guest, if any. The graceful Close releases the host's mirrored images, so
// it must run on the host frame; reaping the process is left to a goroutine so a slow exit cannot stall
// the frame.
func (g *Game) closeGuest() {
	if g.gp == nil {
		return
	}
	gp := g.gp
	g.gp = nil
	g.screenSet = false
	if err := gp.session.Close(); err != nil {
		log.Printf("vm: closing the guest: %v", err)
	}
	go func() {
		// Reaping happens off the frame and has no caller to return to, so log rather than discard.
		if err := gp.cmd.Wait(); err != nil {
			log.Printf("vm: waiting for the guest: %v", err)
		}
		if err := os.Remove(gp.bin); err != nil {
			log.Printf("vm: removing the guest binary: %v", err)
		}
	}()
}

// buildAndStartGuest builds pkg as a guest at the given binary path, launches it pointed at the host's
// endpoint, and returns a handle once it has connected. It is safe to call off the main goroutine; only
// the returned session's SetOutsideScreen/AdvanceTick/AdvanceFrame/Close must run on the host frame.
func buildAndStartGuest(ln net.Listener, bin, endpoint, pkg string) (gp *guestProcess, err error) {
	build := exec.Command("go", "build", "-tags", "ebitenginevm", "-o", bin, pkg)
	build.Stdout = os.Stderr
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return nil, fmt.Errorf("building %s failed (see console): %w", pkg, err)
	}
	defer func() {
		// The binary outlives this function only on success.
		if err != nil {
			err = errors.Join(err, os.Remove(bin))
		}
	}()

	cmd := exec.Command(bin)
	cmd.Env = append(os.Environ(), "EBITENGINE_VM_ENDPOINT="+endpoint)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	defer func() {
		// The process outlives this function only on success. This runs before the binary removal
		// above (deferred calls run in reverse order): a running executable cannot be removed on
		// Windows.
		if err != nil {
			err = errors.Join(err, cmd.Process.Kill(), cmd.Wait())
		}
	}()

	// Both *net.UnixListener and *net.TCPListener provide SetDeadline.
	if err := ln.(interface{ SetDeadline(time.Time) error }).SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return nil, err
	}
	conn, err := ln.Accept()
	if err != nil {
		return nil, fmt.Errorf("%s did not connect as a guest (is it an Ebitengine app?): %w", pkg, err)
	}
	defer func() {
		// The connection outlives this function only on success (the session takes ownership).
		if err != nil {
			err = errors.Join(err, conn.Close())
		}
	}()

	session, err := vmhost.NewGuestSession(conn, nil)
	if err != nil {
		return nil, err
	}
	return &guestProcess{session: session, cmd: cmd, bin: bin, pkg: pkg}, nil
}

func run() (err error) {
	dir, err := os.MkdirTemp("", "ebiten-vm")
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, os.RemoveAll(dir))
	}()

	ln, err := net.Listen("unix", filepath.Join(dir, "vm.sock"))
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, ln.Close())
	}()

	endpoint, err := vmhost.EndpointURLFromAddr(ln.Addr())
	if err != nil {
		return err
	}

	pkg := "github.com/hajimehoshi/ebiten/v2/examples/rotate"
	if len(os.Args) > 1 {
		pkg = os.Args[1]
	}
	g := &Game{
		ln:       ln,
		endpoint: endpoint,
		dir:      dir,
		results:  make(chan launchResult, 1),
		pkg:      pkg,
		status:   "Edit the package and press Enter or Launch",
	}
	g.launchGuest()

	ebiten.SetWindowSize(640, 480)
	ebiten.SetWindowTitle("Ebitengine virtualization host")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	return ebiten.RunGame(g)
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
