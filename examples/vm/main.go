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
// Enter a package in the panel and click Launch, or press Enter. The package may be an import path,
// optionally with an @version query (e.g. example.com/game@latest), or a local path like
// ./examples/paint. Because the host and guest speak a version-locked protocol, an import path is
// built in a generated module that pins Ebitengine to the host's own version; a local path is built
// in its own module. The host builds the guest with -tags ebitenginevm, runs it pointed at a private
// socket, forwards the window's input to it, and composites its rendered frames into the window.
// Audio and gamepads are not forwarded yet.
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
	"runtime/debug"
	"strings"
	"time"

	"github.com/ebitengine/debugui"
	"golang.org/x/mod/module"

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
	pin      ebitenginePin

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
		g.status = "Guest error: " + err.Error()
		g.closeGuest()
		return nil
	}
	// The session runs the guest on its own goroutine; a termination or error surfaces here.
	if err := g.gp.session.Err(); err != nil {
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
		gp, err := buildAndStartGuest(g.ln, g.dir, bin, g.endpoint, pkg, g.pin)
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
	g.forwardInput(state)
	g.gp.session.AdvanceTick()
	return nil
}

// forwardInput sends the window's input to the guest, except input the debug UI is consuming (a hovered
// or focused widget), so the panel stays usable.
func (g *Game) forwardInput(state debugui.InputCapturingState) {
	s := g.gp.session

	if state&debugui.InputCapturingStateFocus == 0 {
		for _, k := range inpututil.AppendJustPressedKeys(nil) {
			s.PressKey(k)
		}
		for _, k := range inpututil.AppendJustReleasedKeys(nil) {
			s.ReleaseKey(k)
		}
		for _, r := range ebiten.AppendInputChars(nil) {
			s.TypeRune(r)
		}
	}

	if state&debugui.InputCapturingStateHover == 0 {
		// The guest fills the whole window, so cursor coordinates map directly.
		x, y := ebiten.CursorPosition()
		s.MoveCursor(float64(x), float64(y))
		for _, b := range []ebiten.MouseButton{ebiten.MouseButtonLeft, ebiten.MouseButtonRight, ebiten.MouseButtonMiddle} {
			if inpututil.IsMouseButtonJustPressed(b) {
				s.PressMouseButton(b)
			}
			if inpututil.IsMouseButtonJustReleased(b) {
				s.ReleaseMouseButton(b)
			}
		}
		if wx, wy := ebiten.Wheel(); wx != 0 || wy != 0 {
			s.ScrollWheel(wx, wy)
		}
	}
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

// closeGuest stops the current guest, if any. Close releases the mirror images that Draw composites
// via AdvanceFrame, so it must run on the host frame, not concurrently with Draw; reaping the process
// is left to a goroutine so a slow exit cannot stall the frame.
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

// ebitengineModule is the import path of the Ebitengine module the host is built against. The guest
// must be built against the same version, since the host and guest speak a version-locked protocol.
const ebitengineModule = "github.com/hajimehoshi/ebiten/v2"

// ebitenginePin says how to force a guest build onto the host's Ebitengine version. require is the
// version for the generated module's require directive; replace is the right-hand side of a replace
// directive that overrides every version of the module — either "<module>@<version>" or a local
// directory.
type ebitenginePin struct {
	require string
	replace string
}

// moduleReplacementVersion returns a placeholder require version compatible with the module path's
// major-version suffix: a "/vN" suffix requires a "vN.x.x" version, and an unversioned path takes
// "v0.0.0".
func moduleReplacementVersion(modulePath string) string {
	_, pathMajor, ok := module.SplitPathVersion(modulePath)
	if !ok || pathMajor == "" {
		return "v0.0.0"
	}
	// pathMajor is a separator followed by the major version, e.g. "/v2" or ".v2".
	return pathMajor[1:] + ".0.0"
}

// resolveEbitenginePin reads the host's own build information to determine which Ebitengine version a
// guest must be built against.
func resolveEbitenginePin() (ebitenginePin, error) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return ebitenginePin{}, errors.New("the host has no build information; cannot determine its Ebitengine version")
	}

	// A directory replacement ignores the require version, but the require line still needs one
	// matching the module path's major version.
	dirVersion := moduleReplacementVersion(ebitengineModule)

	// Ebitengine as a dependency of the host.
	for _, dep := range bi.Deps {
		if dep.Path != ebitengineModule {
			continue
		}
		m := dep
		if dep.Replace != nil {
			m = dep.Replace
		}
		if m.Version != "" {
			return ebitenginePin{require: m.Version, replace: m.Path + "@" + m.Version}, nil
		}
		// A directory replacement recorded in the host's own build. Only an absolute path can be
		// reproduced for the guest; a relative one is resolved against the host's source tree, whose
		// location is unknown at run time.
		if filepath.IsAbs(m.Path) {
			return ebitenginePin{require: dirVersion, replace: m.Path}, nil
		}
		return ebitenginePin{}, fmt.Errorf("the host pins %s to a non-absolute path %q, which cannot be reproduced for the guest", ebitengineModule, m.Path)
	}

	// Ebitengine is the host's main module: the host was built from the Ebitengine repository itself.
	if bi.Main.Path == ebitengineModule {
		if v := bi.Main.Version; v != "" && v != "(devel)" {
			return ebitenginePin{require: v, replace: ebitengineModule + "@" + v}, nil
		}
		dir, err := ebitengineModuleDir()
		if err != nil {
			return ebitenginePin{}, err
		}
		return ebitenginePin{require: dirVersion, replace: dir}, nil
	}

	return ebitenginePin{}, fmt.Errorf("%s is not a dependency of the host", ebitengineModule)
}

// ebitengineModuleDir returns the local source directory of the Ebitengine module, resolved from the
// host's working directory.
func ebitengineModuleDir() (string, error) {
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", ebitengineModule).Output()
	if err != nil {
		return "", fmt.Errorf("locating the %s source: %w", ebitengineModule, err)
	}
	dir := strings.TrimSpace(string(out))
	if dir == "" {
		return "", fmt.Errorf("the %s source directory is unknown", ebitengineModule)
	}
	return dir, nil
}

// buildGuest builds spec into a binary at bin with the ebitenginevm build tag, forcing the guest onto
// the host's Ebitengine version. spec is either a local path, built in its own module, or an import
// path with an optional @version query, built in a module generated under workDir.
func buildGuest(workDir, bin, spec string, pin ebitenginePin) error {
	if isFileSystemPath(spec) {
		// A local package is built in its own module, which already pins its Ebitengine version.
		build := exec.Command("go", "build", "-tags", "ebitenginevm", "-o", bin, spec)
		build.Stdout = os.Stderr
		build.Stderr = os.Stderr
		return build.Run()
	}

	pkg, version, _ := strings.Cut(spec, "@")

	// 'go build' rejects a version query, and neither 'go install pkg@v' nor 'go run pkg@v' permits the
	// dependency override needed to pin Ebitengine. So the package is built inside a generated module
	// that requires it and replaces Ebitengine with the host's version.
	md, err := os.MkdirTemp(workDir, "mod")
	if err != nil {
		return err
	}
	defer func() {
		if err := os.RemoveAll(md); err != nil {
			log.Printf("vm: removing the temporary module: %v", err)
		}
	}()

	if err := goModuleCmd(md, "mod", "init", "ebitenginevmguest"); err != nil {
		return err
	}
	if err := goModuleCmd(md, "mod", "edit",
		"-require="+ebitengineModule+"@"+pin.require,
		"-replace="+ebitengineModule+"="+pin.replace); err != nil {
		return err
	}

	if isWithinModule(pkg, ebitengineModule) {
		// A package inside the Ebitengine module is already provided by the pinned require above, so it
		// must not be fetched separately (and cannot be independently versioned).
		if version != "" {
			return fmt.Errorf("a version query is not allowed on %s, which is part of %s", pkg, ebitengineModule)
		}
	} else {
		query := pkg + "@latest"
		if version != "" {
			query = pkg + "@" + version
		}
		if err := goModuleCmd(md, "get", query); err != nil {
			return err
		}
	}

	return goModuleCmd(md, "build", "-mod=mod", "-tags", "ebitenginevm", "-o", bin, pkg)
}

// goModuleCmd runs a go command in dir with the workspace disabled, so an enclosing go.work cannot
// override the generated module's pins.
func goModuleCmd(dir string, args ...string) error {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// isFileSystemPath reports whether spec refers to a package by file system path rather than import path.
func isFileSystemPath(spec string) bool {
	if filepath.IsAbs(spec) {
		return true
	}
	if spec == "." || spec == ".." {
		return true
	}
	for _, prefix := range []string{"./", "../", `.\`, `..\`} {
		if strings.HasPrefix(spec, prefix) {
			return true
		}
	}
	return false
}

// isWithinModule reports whether the import path pkg is provided by the module.
func isWithinModule(pkg, module string) bool {
	return pkg == module || strings.HasPrefix(pkg, module+"/")
}

// buildAndStartGuest builds pkg as a guest at the given binary path, launches it pointed at the host's
// endpoint, and returns a handle once it has connected. It is safe to call off the main goroutine; only
// the returned session's SetOutsideScreen/AdvanceTick/AdvanceFrame/Close must run on the host frame.
func buildAndStartGuest(ln net.Listener, workDir, bin, endpoint, pkg string, pin ebitenginePin) (gp *guestProcess, err error) {
	if err := buildGuest(workDir, bin, pkg, pin); err != nil {
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

	// Resolve the host's Ebitengine version once, while the working directory is still the one the host
	// was launched from; guests are pinned to it so they speak the same version-locked protocol.
	pin, err := resolveEbitenginePin()
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
		pin:      pin,
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
