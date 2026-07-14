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

//go:build !android && !ios && !js && !nintendosdk && !playstation5

// remoteBackend is the headless guest UI backend for virtualization. There is no window, no GPU, and
// no internal game loop: the game is driven step by step by the host process over a connection. The
// guest dials the host, then serves a lockstep request/response loop where the host sends one
// operation and the guest drives the game one step and streams back the graphics command batch.

package ui

import (
	"errors"
	"fmt"
	"image"
	"io"
	"math"
	"net"
	"slices"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/clock"
	"github.com/hajimehoshi/ebiten/v2/internal/color"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/remote"
	"github.com/hajimehoshi/ebiten/v2/internal/hook"
	"github.com/hajimehoshi/ebiten/v2/internal/thread"
	"github.com/hajimehoshi/ebiten/v2/internal/vibrate"
	"github.com/hajimehoshi/ebiten/v2/internal/vmguest"
	"github.com/hajimehoshi/ebiten/v2/internal/vmprotocol"
)

var (
	_ uiBackend            = (*remoteBackend)(nil)
	_ virtualMonitorSource = (*remoteBackend)(nil)
)

// remoteBackend implements uiBackend by forwarding rendering to and receiving input from a host
// process, instead of owning a window and a GPU.
type remoteBackend struct {
	*UserInterface

	endpoint string

	// graphics is the typed remote driver, kept for Flush. It is the same value stored in
	// u.graphicsDriver.
	graphics *remote.Graphics

	// monitor is the single virtual monitor exposed to the game.
	monitor *Monitor

	// vmHost queries host-owned values on demand. It is installed in serve.
	vmHost vmHostQuerier

	inputState InputState

	// gamepadStates is the latest set of gamepads forwarded by the host, passed to gamepad.Update each
	// tick. It is owned by the serve goroutine. It stays non-nil (possibly empty) so the guest's gamepad
	// subsystem is always virtual rather than polling real devices.
	gamepadStates []gamepad.VirtualGamepadState

	// vibrationsBuf is reused to drain the gamepad vibrations requested during a tick; wireVibrationsBuf
	// is reused to hold their wire-typed copy for the message. gob encodes the latter synchronously, so it
	// can be overwritten on the next flush. Both are owned by the serve goroutine.
	vibrationsBuf     []gamepad.VirtualGamepadVibration
	wireVibrationsBuf []vmprotocol.GamepadVibration

	// deviceVibrationsBuf is reused to drain the device vibration requested during a tick. It holds at
	// most one entry. It is owned by the serve goroutine.
	deviceVibrationsBuf []vibrate.Vibration

	// rawCursorX and rawCursorY are the injected cursor position in outside-screen device-independent
	// pixels. They are translated to logical coordinates per tick in updateInputStateForFrame.
	rawCursorX float64
	rawCursorY float64

	// rawTouches is the set of active touches, in outside-screen device-independent pixels. It is
	// mutated by the host's press/move/release events and translated to logical coordinates per tick in
	// updateInputStateForFrame, like the cursor.
	rawTouches []rawTouch

	// outsideWidth and outsideHeight are the host-supplied outside size, in device-independent
	// pixels. scale is the host's device scale factor, pulled per tick.
	outsideWidth  float64
	outsideHeight float64
	scale         float64

	ticked bool

	m sync.Mutex
}

func newRemoteBackend(u *UserInterface, endpoint string) *remoteBackend {
	r := &remoteBackend{
		UserInterface: u,
		endpoint:      endpoint,
	}
	r.monitor = &Monitor{virtual: r}
	r.scale = 1
	r.gamepadStates = []gamepad.VirtualGamepadState{}
	// A guest forwards the device vibration its game requests to the host, so start recording it. This is
	// reached only for an actual guest, so non-guest builds keep vibrate's recording off.
	vibrate.EnableRecording()
	return r
}

// run dials the host and serves it until the connection closes. A guest binary reaches here through
// the ordinary ebiten.RunGame entry point.
func (r *remoteBackend) run(game Game, options *RunOptions) (err error) {
	if options == nil {
		options = &RunOptions{}
	}

	conn, err := r.dialHost()
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, conn.Close())
	}()

	// A guest has no real render thread; graphics commands are flushed synchronously.
	r.mainThread = thread.NewNoopThread()
	r.context = newContext(game)

	r.graphics = remote.NewGraphics()
	r.graphicsDriver = r.graphics
	r.setGraphicsLibrary(GraphicsLibraryRemote)

	// Record the game's transparent-screen intent so the host can observe it. Recording only appends to
	// the command buffer; it reaches the host with the first frame's command batch. The glfw backend
	// forwards the same option to its own driver.
	r.graphics.SetTransparent(options.ScreenTransparent)

	r.setRunningBackend(r)
	defer r.setRunningBackend(nil)

	return r.serve(conn)
}

// dialHost connects to the configured host endpoint. Supported forms: unix:///path and tcp://host:port.
func (r *remoteBackend) dialHost() (net.Conn, error) {
	e, err := vmprotocol.ParseEndpoint(r.endpoint)
	if err != nil {
		return nil, err
	}
	return net.Dial(e.Network, e.Address)
}

func (r *remoteBackend) serve(conn net.Conn) error {
	// Assert a matching protocol version on the bare connection before wrapping it in gob codecs. The
	// host is the initiator, so the guest receives first.
	if err := vmprotocol.PerformHandshake(conn, false); err != nil {
		return err
	}

	dec := vmprotocol.NewDecoder(conn)
	enc := vmprotocol.NewEncoder(conn)

	// The remote driver forwards commands and read-backs to the host directly through this Host, and the
	// UI queries host-owned values (the device scale factor) through it too. Its methods run on this same
	// goroutine (the guest is single-threaded), so sharing enc/dec is safe.
	host := &vmHostClient{enc: enc, dec: dec}
	r.graphics.SetHost(host)
	r.vmHost = host

	if err := r.serveLoop(dec, enc); err != nil {
		// A guest is driven entirely by its host: once the connection to the host is gone — whether the
		// host closed it cleanly, exited, or crashed — there is nothing left to drive the guest, so end the
		// run without an error rather than surface the dropped connection as a failure the guest program
		// would report (typically log.Fatal). Any other error is a genuine failure and propagates.
		if isHostConnectionGone(err) {
			return nil
		}
		return err
	}
	return nil
}

// serveLoop runs the lockstep request/response loop until the host signals close, the connection ends,
// or an operation fails.
func (r *remoteBackend) serveLoop(dec *vmprotocol.Decoder, enc *vmprotocol.Encoder) error {
	// audioReadBuf is reused to answer host audio reads; the serve loop is single-threaded.
	var audioReadBuf []byte

	// lastSentTPS is the requested TPS last forwarded to the host; sentTPS guards the first send so that
	// the initial value is reported even when it equals the zero value.
	var lastSentTPS int
	var sentTPS bool

	for {
		var msg vmprotocol.HostMessage
		if err := dec.DecodeHostMessage(&msg); err != nil {
			return err
		}

		// The zero value's kind is GuestMessageKindDone: this is the operation's concluding message.
		var doneMsg vmprotocol.GuestMessage
		var closing bool
		switch msg.Kind {
		case vmprotocol.HostMessageKindSetOutsideSize:
			r.setOutsideSize(msg.Width, msg.Height)
		case vmprotocol.HostMessageKindAdvanceTick:
			// ebiten.Tick() reports this tick's number during the Update; capture it before advanceTick
			// runs the Update and increments the counter, so it can be attached to the tick's vibrations.
			tick := int(r.Tick())
			if err := r.advanceTick(); err != nil {
				// A regular termination is the guest's Update asking for a clean stop, not a failure;
				// forward it distinctly so the host can map it back to the ebiten.Termination sentinel.
				if errors.Is(err, RegularTermination) {
					doneMsg.Terminated = true
				} else {
					doneMsg.Err = err.Error()
				}
			} else {
				r.ticked = true
				// Let guest subsystems forward their per-tick messages (e.g. audio) over the connection,
				// stamping them with this tick.
				if err := vmguest.RunPostTickHooks(enc, tick); err != nil {
					return err
				}
				// Report the game's requested TPS whenever it changes, so a host pacing the guest itself
				// can honor its rate.
				if tps := clock.TPS(); !sentTPS || tps != lastSentTPS {
					if err := enc.EncodeGuestMessage(&vmprotocol.GuestMessage{
						Kind:         vmprotocol.GuestMessageKindRequestedTPS,
						RequestedTPS: tps,
					}); err != nil {
						return err
					}
					lastSentTPS = tps
					sentTPS = true
				}
				// Forward any gamepad vibrations the tick's Update requested.
				if err := r.flushGamepadVibrations(enc, tick); err != nil {
					return err
				}
				// Forward any device vibration the tick's Update requested.
				if err := r.flushVibration(enc, tick); err != nil {
					return err
				}
			}
		case vmprotocol.HostMessageKindAdvanceFrame:
			if !r.ticked {
				doneMsg.Err = "ui: a frame was requested before the first tick"
			} else if err := r.advanceFrame(); err != nil {
				doneMsg.Err = err.Error()
			}
		case vmprotocol.HostMessageKindPressKey:
			r.pressKey(Key(msg.Code))
		case vmprotocol.HostMessageKindReleaseKey:
			r.releaseKey(Key(msg.Code))
		case vmprotocol.HostMessageKindMoveCursor:
			r.moveCursor(msg.X, msg.Y)
		case vmprotocol.HostMessageKindPressMouseButton:
			r.pressMouseButton(MouseButton(msg.Code))
		case vmprotocol.HostMessageKindReleaseMouseButton:
			r.releaseMouseButton(MouseButton(msg.Code))
		case vmprotocol.HostMessageKindScrollWheel:
			r.scrollWheel(msg.X, msg.Y)
		case vmprotocol.HostMessageKindTypeRune:
			r.typeRune(msg.Rune)
		case vmprotocol.HostMessageKindReadAudio:
			// A guest subsystem (audio) decodes the samples into the buffer; this package does not
			// depend on it.
			audioReadBuf = slices.Grow(audioReadBuf[:0], msg.AudioMaxLenInBytes)[:msg.AudioMaxLenInBytes]
			n, eof := vmguest.RunAudioReadHandler(msg.AudioPlayerID, audioReadBuf)
			if err := enc.EncodeGuestMessage(&vmprotocol.GuestMessage{
				Kind:     vmprotocol.GuestMessageKindAudioData,
				AudioPCM: audioReadBuf[:n],
				AudioEOF: eof,
			}); err != nil {
				return err
			}
		case vmprotocol.HostMessageKindUpdateGamepads:
			r.updateGamepads(msg.GamepadStates)
		case vmprotocol.HostMessageKindPressTouch:
			r.setTouch(TouchID(msg.Code), msg.X, msg.Y)
		case vmprotocol.HostMessageKindMoveTouch:
			r.setTouch(TouchID(msg.Code), msg.X, msg.Y)
		case vmprotocol.HostMessageKindReleaseTouch:
			r.releaseTouch(TouchID(msg.Code))
		case vmprotocol.HostMessageKindTextInputState:
			vmguest.RunTextInputStateHandler(msg.TextInputID, msg.TextInputState)
		case vmprotocol.HostMessageKindEndTextInput:
			vmguest.RunTextInputEndHandler(msg.TextInputID)
		case vmprotocol.HostMessageKindClose:
			closing = true
		default:
			doneMsg.Err = fmt.Sprintf("ui: unknown host message kind: %d", msg.Kind)
		}

		if err := enc.EncodeGuestMessage(&doneMsg); err != nil {
			return err
		}
		if closing {
			return nil
		}
	}
}

// isHostConnectionGone reports whether err indicates the connection to the host has ended: a clean EOF
// or a truncated read at the boundary, a use of the now-closed connection, or a reset/aborted/broken
// connection (handled per OS by isConnectionReset). A guest is driven entirely by its host, so any of
// these means the host is gone and the guest should stop without error rather than treat it as a
// failure.
func isHostConnectionGone(err error) bool {
	return errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, net.ErrClosed) ||
		isConnectionReset(err)
}

// vmHostQuerier fetches host-owned values the guest needs on demand. The host owns the real window
// and GPU, so values like the device scale factor are queried from it rather than tracked here.
type vmHostQuerier interface {
	DeviceScaleFactor() (float64, error)
}

// setOutsideSize sets the outside (window-equivalent) size, in device-independent pixels. The given
// size is the size of the host-owned screen the guest renders into. The device scale factor is not
// set here; it is queried from the host per tick (it can change during a session).
func (r *remoteBackend) setOutsideSize(outsideWidth, outsideHeight float64) {
	r.m.Lock()
	defer r.m.Unlock()
	r.outsideWidth = outsideWidth
	r.outsideHeight = outsideHeight
}

func (r *remoteBackend) outsideSize() (outsideWidth, outsideHeight float64) {
	r.m.Lock()
	defer r.m.Unlock()
	return r.outsideWidth, r.outsideHeight
}

// pullDeviceScaleFactor fetches the host's current device scale factor and records it so a later
// Monitor.DeviceScaleFactor within the same tick returns the same value without another round-trip.
func (r *remoteBackend) pullDeviceScaleFactor() (float64, error) {
	s, err := r.vmHost.DeviceScaleFactor()
	if err != nil {
		return 0, err
	}
	r.m.Lock()
	defer r.m.Unlock()
	r.scale = s
	return s, nil
}

func (r *remoteBackend) deviceScaleFactor() float64 {
	r.m.Lock()
	defer r.m.Unlock()
	return r.scale
}

// advanceTick runs exactly one Update (one tick). The host supplies the clock by choosing when to
// call this, so the engine's wall-clock pacing is bypassed entirely.
func (r *remoteBackend) advanceTick() error {
	w, h := r.outsideSize()
	if w == 0 || h == 0 {
		return nil
	}
	s, err := r.pullDeviceScaleFactor()
	if err != nil {
		return err
	}
	// The guest has no devices of its own; drive the gamepad subsystem with the host-forwarded set (a
	// non-nil slice keeps it virtual). There is no native window.
	if err := gamepad.Update(0, r.gamepadStates); err != nil {
		return err
	}
	if err := r.context.updateTickForVMGuest(r.graphicsDriver, w, h, s, r.UserInterface); err != nil {
		return err
	}
	if err := atlas.SwapBuffers(r.graphicsDriver); err != nil {
		return err
	}
	return r.flushCommands()
}

// advanceFrame renders the current state into the host-owned screen without advancing game state.
func (r *remoteBackend) advanceFrame() error {
	w, h := r.outsideSize()
	if w == 0 || h == 0 {
		return nil
	}
	s, err := r.pullDeviceScaleFactor()
	if err != nil {
		return err
	}
	if err := r.context.drawFrameForVMGuest(r.graphicsDriver, w, h, s, r.UserInterface); err != nil {
		return err
	}
	if err := atlas.SwapBuffers(r.graphicsDriver); err != nil {
		return err
	}
	return r.flushCommands()
}

// flushCommands forwards the graphics commands recorded this frame to the host.
func (r *remoteBackend) flushCommands() error {
	return r.graphics.Flush()
}

func (r *remoteBackend) readInputState(inputState *InputState) {
	r.m.Lock()
	defer r.m.Unlock()
	r.inputState.copyAndReset(inputState)
}

func (r *remoteBackend) updateInputStateForFrame(deviceScaleFactor float64) error {
	r.m.Lock()
	defer r.m.Unlock()

	x, y := r.context.clientPositionToLogicalPosition(r.rawCursorX, r.rawCursorY, deviceScaleFactor)
	if !math.IsNaN(x) && !math.IsNaN(y) {
		r.inputState.CursorX, r.inputState.CursorY = x, y
	}

	// Rebuild the per-tick touch set from the event-applied raw set, translating each to logical
	// coordinates as the cursor is.
	r.inputState.Touches = r.inputState.Touches[:0]
	for i := range r.rawTouches {
		t := &r.rawTouches[i]
		tx, ty := r.context.clientPositionToLogicalPosition(t.x, t.y, deviceScaleFactor)
		if math.IsNaN(tx) || math.IsNaN(ty) {
			continue
		}
		r.inputState.Touches = append(r.inputState.Touches, Touch{
			ID: t.id,
			X:  int(tx),
			Y:  int(ty),
		})
	}
	return nil
}

// pressKey injects a key-press event.
func (r *remoteBackend) pressKey(key Key) {
	r.m.Lock()
	defer r.m.Unlock()
	r.inputState.setKeyPressed(key, r.InputTime())
}

// releaseKey injects a key-release event.
func (r *remoteBackend) releaseKey(key Key) {
	r.m.Lock()
	defer r.m.Unlock()
	r.inputState.setKeyReleased(key, r.InputTime())
}

// moveCursor sets the cursor position in outside-screen device-independent pixels.
func (r *remoteBackend) moveCursor(x, y float64) {
	r.m.Lock()
	defer r.m.Unlock()
	r.rawCursorX, r.rawCursorY = x, y
}

// pressMouseButton injects a mouse-button-press event.
func (r *remoteBackend) pressMouseButton(button MouseButton) {
	r.m.Lock()
	defer r.m.Unlock()
	r.inputState.setMouseButtonPressed(button, r.InputTime())
}

// releaseMouseButton injects a mouse-button-release event.
func (r *remoteBackend) releaseMouseButton(button MouseButton) {
	r.m.Lock()
	defer r.m.Unlock()
	r.inputState.setMouseButtonReleased(button, r.InputTime())
}

// scrollWheel injects a wheel movement (accumulated until the next tick reads it).
func (r *remoteBackend) scrollWheel(x, y float64) {
	r.m.Lock()
	defer r.m.Unlock()
	r.inputState.WheelX += x
	r.inputState.WheelY += y
}

// typeRune injects a typed character.
func (r *remoteBackend) typeRune(c rune) {
	r.m.Lock()
	defer r.m.Unlock()
	r.inputState.appendRune(c)
}

// rawTouch is one active touch's position, in outside-screen device-independent pixels, pending
// translation to logical coordinates in updateInputStateForFrame.
type rawTouch struct {
	id TouchID
	x  float64
	y  float64
}

// setTouch starts or moves the touch identified by id. A press and a move are applied identically: the
// touch set is membership-based (just-pressed and just-released are recovered by the guest's own
// inpututil diffing the set across ticks), so both upsert the position.
func (r *remoteBackend) setTouch(id TouchID, x, y float64) {
	r.m.Lock()
	defer r.m.Unlock()
	for i := range r.rawTouches {
		if r.rawTouches[i].id == id {
			r.rawTouches[i].x = x
			r.rawTouches[i].y = y
			return
		}
	}
	r.rawTouches = append(r.rawTouches, rawTouch{id: id, x: x, y: y})
}

// releaseTouch ends the touch identified by id. Releasing an unknown touch is a no-op.
func (r *remoteBackend) releaseTouch(id TouchID) {
	r.m.Lock()
	defer r.m.Unlock()
	for i := range r.rawTouches {
		if r.rawTouches[i].id == id {
			r.rawTouches = slices.Delete(r.rawTouches, i, i+1)
			return
		}
	}
}

// updateGamepads replaces the connected gamepads with the host's snapshot. Gamepads are a global
// subsystem rather than part of inputState, so this updates them directly; it needs no lock.
func (r *remoteBackend) updateGamepads(states []vmprotocol.GamepadState) {
	// gamepad.Update copies this snapshot into the gamepad's own storage, so the entries here may alias
	// states' slices and maps instead of copying them — states is a freshly decoded, read-only message,
	// so the aliases stay valid — and the buffer may be reused across ticks. It is kept non-nil so the
	// subsystem stays virtual.
	r.gamepadStates = r.gamepadStates[:0]
	for i := range states {
		s := &states[i]
		r.gamepadStates = append(r.gamepadStates, gamepad.VirtualGamepadState{
			ID:              gamepad.ID(s.ID),
			SDLID:           s.SDLID,
			Name:            s.Name,
			Axes:            s.Axes,
			Buttons:         s.Buttons,
			StandardAxes:    s.StandardAxes,
			StandardButtons: virtualStandardButtons(s.StandardButtons),
		})
	}
}

func virtualStandardButtons(buttons map[gamepaddb.StandardButton]vmprotocol.GamepadStandardButtonState) map[gamepaddb.StandardButton]gamepad.VirtualStandardGamepadButton {
	if buttons == nil {
		return nil
	}
	m := make(map[gamepaddb.StandardButton]gamepad.VirtualStandardGamepadButton, len(buttons))
	for b, s := range buttons {
		m[b] = gamepad.VirtualStandardGamepadButton{
			Pressed: s.Pressed,
			Value:   s.Value,
		}
	}
	return m
}

// flushGamepadVibrations forwards the vibrations the tick's Update requested to the host, tagged with
// the tick (the guest's ebiten.Tick() during that Update). It sends nothing when none were requested.
func (r *remoteBackend) flushGamepadVibrations(enc *vmprotocol.Encoder, tick int) error {
	r.vibrationsBuf = gamepad.AppendVirtualGamepadVibrations(r.vibrationsBuf[:0])
	if len(r.vibrationsBuf) == 0 {
		return nil
	}
	r.wireVibrationsBuf = r.wireVibrationsBuf[:0]
	for _, v := range r.vibrationsBuf {
		r.wireVibrationsBuf = append(r.wireVibrationsBuf, vmprotocol.GamepadVibration{
			ID:              int(v.ID),
			Duration:        v.Duration,
			StrongMagnitude: v.StrongMagnitude,
			WeakMagnitude:   v.WeakMagnitude,
		})
	}
	return enc.EncodeGuestMessage(&vmprotocol.GuestMessage{
		Kind:              vmprotocol.GuestMessageKindGamepadVibrations,
		GamepadVibrations: r.wireVibrationsBuf,
		StartTick:         tick,
	})
}

// flushVibration forwards the device vibration the tick's Update requested to the host, tagged with the
// tick (the guest's ebiten.Tick() during that Update). It sends nothing when none was requested.
func (r *remoteBackend) flushVibration(enc *vmprotocol.Encoder, tick int) error {
	r.deviceVibrationsBuf = vibrate.AppendPendingVibrations(r.deviceVibrationsBuf[:0])
	if len(r.deviceVibrationsBuf) == 0 {
		return nil
	}
	// AppendPendingVibrations yields at most one entry, since a device has a single vibration state.
	v := r.deviceVibrationsBuf[0]
	return enc.EncodeGuestMessage(&vmprotocol.GuestMessage{
		Kind: vmprotocol.GuestMessageKindVibration,
		Vibration: vmprotocol.Vibration{
			Duration:  v.Duration,
			Magnitude: v.Magnitude,
		},
		StartTick: tick,
	})
}

func (r *remoteBackend) KeyName(key Key) string {
	return ""
}

func (r *remoteBackend) updateIconIfNeeded() error {
	return nil
}

func (r *remoteBackend) IsFocused() bool {
	return true
}

func (r *remoteBackend) CursorMode() CursorMode {
	return CursorModeHidden
}

func (r *remoteBackend) SetCursorMode(mode CursorMode) {
}

func (r *remoteBackend) applyCursorShape() {
}

func (r *remoteBackend) IsFullscreen() bool {
	return false
}

func (r *remoteBackend) SetFullscreen(fullscreen bool) {
}

func (r *remoteBackend) applyFPSMode() {
}

func (r *remoteBackend) ScheduleFrame() {
}

func (r *remoteBackend) Window() backendWindow {
	return &nullWindow{}
}

func (r *remoteBackend) Monitor() *Monitor {
	return r.monitor
}

func (r *remoteBackend) appendMonitors(monitors []*Monitor) []*Monitor {
	return append(monitors, r.monitor)
}

func (r *remoteBackend) RunOnMainThread(f func()) {
	f()
}

// updateTickForVMGuest runs one Update step in a single atlas frame, without drawing.
func (c *context) updateTickForVMGuest(graphicsDriver graphicsdriver.Graphics, outsideWidth, outsideHeight, deviceScaleFactor float64, ui *UserInterface) (err error) {
	if outsideWidth == 0 || outsideHeight == 0 {
		return nil
	}

	if err := atlas.BeginFrame(graphicsDriver); err != nil {
		return err
	}
	defer func() {
		if atlasErr := atlas.EndFrame(graphicsDriver); atlasErr != nil {
			err = errors.Join(err, atlasErr)
		}
	}()

	if err := c.processFuncsInFrame(ui); err != nil {
		return err
	}

	if w, h, _ := c.layoutGame(outsideWidth, outsideHeight, deviceScaleFactor); w == 0 || h == 0 {
		return nil
	}

	if err := ui.updateInputStateForFrame(deviceScaleFactor); err != nil {
		return err
	}

	c.game.UpdateInputState(func(inputState *InputState) {
		ui.readInputState(inputState)
	})

	if err := hook.RunBeforeUpdateHooks(); err != nil {
		return err
	}
	// This process is a virtualization guest, so the pre-tick hooks get true and guest subsystems (such
	// as audio) route themselves to the host accordingly.
	if err := hook.RunBeforeUpdateHooksWithVMGuestInfo(true); err != nil {
		return err
	}
	if err := c.game.Update(); err != nil {
		return err
	}
	if err := ui.error(); err != nil {
		return err
	}
	ui.incrementTick()

	return ui.updateIconIfNeeded()
}

// drawFrameForVMGuest renders the current state in a single atlas frame.
func (c *context) drawFrameForVMGuest(graphicsDriver graphicsdriver.Graphics, outsideWidth, outsideHeight, deviceScaleFactor float64, ui *UserInterface) (err error) {
	if outsideWidth == 0 || outsideHeight == 0 {
		return nil
	}

	if err := atlas.BeginFrame(graphicsDriver); err != nil {
		return err
	}
	defer func() {
		if atlasErr := atlas.EndFrame(graphicsDriver); atlasErr != nil {
			err = errors.Join(err, atlasErr)
		}
	}()

	if err := c.processFuncsInFrame(ui); err != nil {
		return err
	}

	if w, h, _ := c.layoutGame(outsideWidth, outsideHeight, deviceScaleFactor); w == 0 || h == 0 {
		return nil
	}

	// The host decides when to draw, so a draw is never skipped here.
	_, err = c.drawGame(graphicsDriver, ui, true)
	return err
}

// vmHostClient forwards the guest's recorded graphics output to the host process over the connection.
type vmHostClient struct {
	enc *vmprotocol.Encoder
	dec *vmprotocol.Decoder
}

func (h *vmHostClient) SendGraphicsCommands(cmds []vmprotocol.GraphicsCommand) error {
	return h.enc.EncodeGuestMessage(&vmprotocol.GuestMessage{
		Kind:             vmprotocol.GuestMessageKindGraphicsCommands,
		GraphicsCommands: cmds,
	})
}

// query sends a query message and returns the host's answer, validating the answer's kind.
func (h *vmHostClient) query(msg *vmprotocol.GuestMessage, want vmprotocol.HostMessageKind) (*vmprotocol.HostMessage, error) {
	if err := h.enc.EncodeGuestMessage(msg); err != nil {
		return nil, err
	}
	var answer vmprotocol.HostMessage
	if err := h.dec.DecodeHostMessage(&answer); err != nil {
		return nil, err
	}
	if answer.Kind != want {
		return nil, fmt.Errorf("ui: unexpected host message kind %d as an answer: want %d", answer.Kind, want)
	}
	return &answer, nil
}

func (h *vmHostClient) ReadPixels(id graphicsdriver.ImageID, regions []image.Rectangle) ([][]byte, error) {
	answer, err := h.query(&vmprotocol.GuestMessage{
		Kind:        vmprotocol.GuestMessageKindQueryReadPixels,
		ReadImageID: id,
		ReadRegions: regions,
	}, vmprotocol.HostMessageKindAnswerReadPixels)
	if err != nil {
		return nil, err
	}
	if answer.Err != "" {
		return nil, errors.New(answer.Err)
	}
	return answer.Pixels, nil
}

func (h *vmHostClient) MaxImageSize() (int, error) {
	answer, err := h.query(&vmprotocol.GuestMessage{
		Kind: vmprotocol.GuestMessageKindQueryMaxImageSize,
	}, vmprotocol.HostMessageKindAnswerMaxImageSize)
	if err != nil {
		return 0, err
	}
	return answer.MaxImageSize, nil
}

func (h *vmHostClient) ColorSpace() (color.ColorSpace, error) {
	answer, err := h.query(&vmprotocol.GuestMessage{
		Kind: vmprotocol.GuestMessageKindQueryColorSpace,
	}, vmprotocol.HostMessageKindAnswerColorSpace)
	if err != nil {
		return 0, err
	}
	return answer.ColorSpace, nil
}

func (h *vmHostClient) DeviceScaleFactor() (float64, error) {
	answer, err := h.query(&vmprotocol.GuestMessage{
		Kind: vmprotocol.GuestMessageKindQueryDeviceScaleFactor,
	}, vmprotocol.HostMessageKindAnswerDeviceScaleFactor)
	if err != nil {
		return 0, err
	}
	return answer.ScaleFactor, nil
}
