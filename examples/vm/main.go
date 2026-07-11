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
// socket, forwards the window's input (keyboard, mouse, touches, and gamepads) to it, composites its
// rendered frames into the window, plays its audio, and applies the gamepad vibrations it requests.
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
	"slices"
	"strings"
	"time"

	"github.com/ebitengine/debugui"
	"golang.org/x/mod/module"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
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

	// pkg is the package text field's buffer.
	pkg string
	// guestTPS is the ticks-per-second rate the guest is driven at, bound to the panel's slider; 0 pauses it.
	guestTPS int
	status   string

	launching bool
	results   chan launchResult

	gp          *guestProcess
	guestScreen *ebiten.Image

	// screenSet reports whether guestScreen has been handed to the current session via
	// SetOutsideScreen; it is cleared when the session or the screen changes.
	screenSet bool

	// guestTPSAdopted records whether the current guest's requested TPS has been adopted once its first
	// tick was processed. It resets when a guest is adopted: a freshly launched guest is driven at the
	// rate its own game requests.
	guestTPSAdopted bool

	// audioContext is the host's single audio context, created lazily at the guest's sample rate.
	audioContext *audio.Context
	// audioPlayers maps each guest stream to the host player that plays it; audioStreamsBuf is reused
	// by AppendAudioStreams.
	audioPlayers    map[*vmhost.GuestAudioStream]*audio.Player
	audioStreamsBuf []*vmhost.GuestAudioStream

	// gamepadIDsBuf and gamepadStatesBuf are reused each tick by forwardInput.
	gamepadIDsBuf    []ebiten.GamepadID
	gamepadStatesBuf []vmhost.GamepadState

	// keyBuf and runeBuf are reused each tick by forwardInput.
	keyBuf  []ebiten.Key
	runeBuf []rune

	// touchIDsBuf is reused each tick by forwardInput.
	touchIDsBuf []ebiten.TouchID

	// tickAccum carries the sub-tick remainder between host updates, in units where hostTPS equals one tick.
	tickAccum int

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
			g.guestTPSAdopted = false
			g.status = "Running " + g.pkg
		}
	default:
	}

	state, err := g.debugui.Update(func(ctx *debugui.Context) error {
		ctx.Window("Virtualization host", image.Rect(10, 10, 410, 170), func(layout debugui.ContainerLayout) {
			ctx.Text("Package to run as a guest:")
			ctx.SetGridLayout([]int{-1, 60}, nil)
			ctx.TextField(&g.pkg)
			ctx.Button("Launch").On(g.launchGuest)
			ctx.SetGridLayout([]int{-1, -1}, nil)
			ctx.Text("Guest TPS:")
			ctx.Slider(&g.guestTPS, 0, 300, 1)
			ctx.SetGridLayout([]int{-1}, nil)
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
	if err := g.advanceGuestTicks(state); err != nil {
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
		return nil
	}
	// Once the freshly launched guest has processed its first tick, it has reported its requested TPS,
	// so adopt that rate to drive it as its own game intends.
	if !g.guestTPSAdopted && g.gp.session.ProcessedTicks() > 0 {
		g.adoptRequestedTPS()
		g.guestTPSAdopted = true
	}
	return g.updateAudio()
}

// updateAudio plays each guest player on its own host player, so the guest's players stay unmixed.
func (g *Game) updateAudio() error {
	rate := g.gp.session.AudioSampleRate()
	if rate == 0 {
		// The guest has not produced audio yet, so its sample rate is unknown.
		return nil
	}
	if g.audioContext == nil {
		g.audioContext = audio.NewContext(rate)
	}
	if g.audioContext.SampleRate() != rate {
		// One audio context per process, so a later guest at a different rate than the first cannot be
		// played without resampling; skip its audio to keep the example simple.
		g.status = fmt.Sprintf("Running %s (audio off: sample rate %d != %d)", g.gp.pkg, rate, g.audioContext.SampleRate())
		return nil
	}

	g.audioStreamsBuf = slices.Delete(g.audioStreamsBuf, 0, len(g.audioStreamsBuf))
	g.audioStreamsBuf = g.gp.session.AppendAudioStreams(g.audioStreamsBuf)
	for _, stream := range g.audioStreamsBuf {
		hp := g.audioPlayers[stream]
		if hp == nil {
			// Start a host player only for a stream that is currently playing; a finished or paused stream
			// gets none, and a replayed one gets a fresh host player when it plays again.
			if !stream.IsPlaying() {
				continue
			}
			var err error
			hp, err = g.audioContext.NewPlayerF32(stream)
			if err != nil {
				return err
			}
			// oto reads ahead this far, pulling the samples from the guest; keep it small for low latency
			// but large enough to cover a momentarily busy session.
			hp.SetBufferSize(time.Second / 20)
			hp.Play()
			g.audioPlayers[stream] = hp
		}
		// The forwarded samples are raw, so apply the guest player's volume on the host side.
		hp.SetVolume(stream.Volume())
	}
	// Close finished host players: a host player stops once its guest stream reaches EOF and plays out
	// (or the stream is closed), so this waits for the tail instead of cutting it.
	for stream, hp := range g.audioPlayers {
		if !hp.IsPlaying() {
			if err := hp.Close(); err != nil {
				log.Printf("vm: closing an audio player: %v", err)
			}
			delete(g.audioPlayers, stream)
		}
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

// advanceGuestTicks gives the guest its screen, forwards the window's input to it, and advances it by the
// number of ticks due this host update at the guest's TPS.
func (g *Game) advanceGuestTicks(state debugui.InputCapturingState) error {
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
	g.gp.session.AdvanceTicks(g.guestTickCount())
	return nil
}

// guestTickCount returns how many ticks to advance the guest this host update so it runs at g.guestTPS
// ticks per second on average, regardless of the host's tick rate.
func (g *Game) guestTickCount() int {
	// A non-positive rate pauses the guest: no ticks advance.
	if g.guestTPS <= 0 {
		return 0
	}
	hostTPS := ebiten.TPS()
	if hostTPS <= 0 {
		// Guard against a non-positive rate (e.g. SyncWithFPS).
		hostTPS = ebiten.DefaultTPS
	}
	// Every hostTPS accumulated units make one tick, so hostTPS updates (one second) yield guestTPS ticks.
	g.tickAccum += g.guestTPS
	n := g.tickAccum / hostTPS
	g.tickAccum %= hostTPS
	return n
}

// adoptRequestedTPS sets the guest's drive rate to the rate the guest's own game requests (via
// ebiten.SetTPS), so the host paces it as the game intends instead of at the slider's manual value. It
// is a no-op while no guest is running.
func (g *Game) adoptRequestedTPS() {
	if g.gp == nil {
		return
	}
	tps := g.gp.session.RequestedTPS()
	if tps == ebiten.SyncWithFPS {
		// SyncWithFPS ties the guest's tick rate to the display's refresh rate. This host advances the
		// guest from its own Update, so approximate that intent with the host's tick rate.
		tps = ebiten.TPS()
		if tps <= 0 {
			tps = ebiten.DefaultTPS
		}
	}
	g.guestTPS = tps
}

// forwardInput sends the window's input to the guest, except input the debug UI is consuming (a hovered
// or focused widget), so the panel stays usable.
func (g *Game) forwardInput(state debugui.InputCapturingState) {
	s := g.gp.session

	if state&debugui.InputCapturingStateFocus == 0 {
		g.keyBuf = inpututil.AppendJustPressedKeys(g.keyBuf[:0])
		for _, k := range g.keyBuf {
			s.PressKey(k)
		}
		g.keyBuf = inpututil.AppendJustReleasedKeys(g.keyBuf[:0])
		for _, k := range g.keyBuf {
			s.ReleaseKey(k)
		}
		g.runeBuf = ebiten.AppendInputChars(g.runeBuf[:0])
		for _, r := range g.runeBuf {
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

	// Gamepads are mirrored unconditionally: unlike keyboard and mouse input they are not gated on the
	// debug UI, which does not consume gamepad input.
	g.gamepadIDsBuf = ebiten.AppendGamepadIDs(g.gamepadIDsBuf[:0])
	g.gamepadStatesBuf = g.gamepadStatesBuf[:0]
	for _, id := range g.gamepadIDsBuf {
		g.gamepadStatesBuf = append(g.gamepadStatesBuf, gamepadState(id))
	}
	s.UpdateGamepads(g.gamepadStatesBuf)

	// Touches are forwarded as press/move/release events, like the keyboard and mouse buttons, and
	// unconditionally: dropping a release while the panel is hovered would leave the guest with a stuck
	// touch. The guest fills the window, so the positions map directly.
	g.touchIDsBuf = inpututil.AppendJustPressedTouchIDs(g.touchIDsBuf[:0])
	for _, id := range g.touchIDsBuf {
		x, y := ebiten.TouchPosition(id)
		s.PressTouch(id, float64(x), float64(y))
	}
	g.touchIDsBuf = ebiten.AppendTouchIDs(g.touchIDsBuf[:0])
	for _, id := range g.touchIDsBuf {
		// A just-pressed touch was already positioned by PressTouch above; only a continuing touch moves.
		if inpututil.TouchPressDuration(id) == 1 {
			continue
		}
		x, y := ebiten.TouchPosition(id)
		s.MoveTouch(id, float64(x), float64(y))
	}
	g.touchIDsBuf = inpututil.AppendJustReleasedTouchIDs(g.touchIDsBuf[:0])
	for _, id := range g.touchIDsBuf {
		s.ReleaseTouch(id)
	}
}

// gamepadState reads the current state of one host gamepad through the public ebiten API into the
// snapshot the guest is fed.
func gamepadState(id ebiten.GamepadID) vmhost.GamepadState {
	state := vmhost.GamepadState{
		ID:    id,
		SDLID: ebiten.GamepadSDLID(id),
		Name:  ebiten.GamepadName(id),
	}

	for a := 0; a < ebiten.GamepadAxisCount(id); a++ {
		state.Axes = append(state.Axes, ebiten.GamepadAxisValue(id, a))
	}
	for b := 0; b < ebiten.GamepadButtonCount(id); b++ {
		state.Buttons = append(state.Buttons, ebiten.IsGamepadButtonPressed(id, ebiten.GamepadButton(b)))
	}

	if !ebiten.IsStandardGamepadLayoutAvailable(id) {
		return state
	}
	state.StandardAxes = map[ebiten.StandardGamepadAxis]float64{}
	for a := ebiten.StandardGamepadAxis(0); a <= ebiten.StandardGamepadAxisMax; a++ {
		if !ebiten.IsStandardGamepadAxisAvailable(id, a) {
			continue
		}
		state.StandardAxes[a] = ebiten.StandardGamepadAxisValue(id, a)
	}
	state.StandardButtons = map[ebiten.StandardGamepadButton]vmhost.GamepadStandardButtonState{}
	for b := ebiten.StandardGamepadButton(0); b <= ebiten.StandardGamepadButtonMax; b++ {
		if !ebiten.IsStandardGamepadButtonAvailable(id, b) {
			continue
		}
		state.StandardButtons[b] = vmhost.GamepadStandardButtonState{
			Pressed: ebiten.IsStandardGamepadButtonPressed(id, b),
			Value:   ebiten.StandardGamepadButtonValue(id, b),
		}
	}
	return state
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.gp != nil && g.screenSet {
		g.gp.session.AdvanceFrame()
		g.gp.session.CompositeFrame()
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
// via CompositeFrame, so it must run on the host frame, not concurrently with Draw; reaping the process
// is left to a goroutine so a slow exit cannot stall the frame.
func (g *Game) closeGuest() {
	if g.gp == nil {
		return
	}
	gp := g.gp
	g.gp = nil
	g.screenSet = false
	for stream, hp := range g.audioPlayers {
		if err := hp.Close(); err != nil {
			log.Printf("vm: closing an audio player: %v", err)
		}
		delete(g.audioPlayers, stream)
	}
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
// the returned session's screen-touching methods (SetOutsideScreen, CompositeFrame, Close) must run on
// the host frame.
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

	session, err := vmhost.NewGuestSession(conn, &vmhost.NewGuestSessionOptions{
		// Mirror each vibration the guest requests onto the host's own gamepad. The guest's gamepad IDs
		// match the host's, because the host forwards its own gamepads to the guest. VibrateGamepad is
		// concurrent-safe, so running this on the session goroutine is fine.
		OnGamepadVibration: func(v vmhost.GamepadVibration) {
			ebiten.VibrateGamepad(v.GamepadID, &ebiten.VibrateGamepadOptions{
				Duration:        v.Duration,
				StrongMagnitude: v.StrongMagnitude,
				WeakMagnitude:   v.WeakMagnitude,
			})
		},
	})
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
		ln:           ln,
		endpoint:     endpoint,
		dir:          dir,
		pin:          pin,
		results:      make(chan launchResult, 1),
		pkg:          pkg,
		guestTPS:     ebiten.DefaultTPS,
		audioPlayers: map[*vmhost.GuestAudioStream]*audio.Player{},
		status:       "Edit the package and press Enter or Launch",
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
