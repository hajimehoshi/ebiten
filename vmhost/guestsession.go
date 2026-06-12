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

// Package vmhost is the host side of virtualization: it drives an Ebitengine guest running in a
// separate process over a connection, and renders the guest's frames into host-owned images on the
// host's own GPU.
//
// The protocol is hidden behind the GuestSession API. AdvanceTick drives the guest's Update;
// AdvanceFrame renders the guest's current state into the host-owned screen set by SetOutsideScreen
// so the host can composite it into its own window.
package vmhost

import (
	"errors"
	"fmt"
	"io"
	"net"
	"slices"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
	"github.com/hajimehoshi/ebiten/v2/internal/vmprotocol"
)

// GuestSession is a session with a guest process, driven and observed from the host side. It holds
// no process handle: the guest's lifetime belongs to whoever spawned it.
//
// [GuestSession.AdvanceTick] and [GuestSession.AdvanceFrame] issue draws on the host's GPU through the
// ordinary ebiten stack, so they must be called from within the host's frame (its Update or Draw).
type GuestSession struct {
	conn net.Conn
	enc  *vmprotocol.Encoder
	dec  *vmprotocol.Decoder

	renderer *frameRenderer

	// outsideScreen is the host-owned image the guest's frames render into, set by SetOutsideScreen.
	outsideScreen *ebiten.Image

	// sentWidth/sentHeight are the last outside size sent, in device-independent pixels, so the size
	// is resent only when it changes.
	sentWidth  float64
	sentHeight float64

	// ticked reports whether an Update has completed on the guest, gating AdvanceFrame: a frame must
	// not be drawn before the first tick.
	ticked bool

	// err holds an error deferred from AdvanceFrame; it surfaces at the next AdvanceTick.
	err error

	// pixelsBuf and pixelsListBuf back the ReadPixels answers: one flat reused buffer, subsliced per
	// region. The encode writes them out before the next answer reuses them.
	pixelsBuf     []byte
	pixelsListBuf [][]byte
}

// NewGuestSessionOptions represents options for [NewGuestSession].
type NewGuestSessionOptions struct {
	// IdleTimeout is the maximum duration the connection may make no progress while an operation
	// (including the handshake) is in progress: when it elapses, the operation fails with an error
	// matching [os.ErrDeadlineExceeded], and the session is unusable except for
	// [GuestSession.Close]. It bounds silence, not an operation's total duration: a guest that keeps
	// sending never times out. The default (0) means no timeout.
	IdleTimeout time.Duration
}

// NewGuestSession opens a session with a guest process over an established connection. It performs the
// protocol handshake and returns an error if the guest's protocol version does not match the host's.
// options can be nil, which means the default options.
func NewGuestSession(conn net.Conn, options *NewGuestSessionOptions) (*GuestSession, error) {
	var rw io.ReadWriter = conn
	if options != nil && options.IdleTimeout > 0 {
		rw = &idleTimeoutConn{
			conn:        conn,
			idleTimeout: options.IdleTimeout,
		}
	}
	// The handshake runs before the connection is wrapped in gob codecs (the host is the initiator).
	if err := vmprotocol.PerformHandshake(rw, true); err != nil {
		return nil, err
	}
	return &GuestSession{
		conn:     conn,
		enc:      vmprotocol.NewEncoder(rw),
		dec:      vmprotocol.NewDecoder(rw),
		renderer: newFrameRenderer(),
	}, nil
}

// idleTimeoutConn bounds each Read and Write with a deadline refreshed at every call, so the timeout
// limits silence on the connection rather than an operation's total duration.
type idleTimeoutConn struct {
	conn        net.Conn
	idleTimeout time.Duration
}

func (c *idleTimeoutConn) Read(p []byte) (int, error) {
	if err := c.conn.SetReadDeadline(time.Now().Add(c.idleTimeout)); err != nil {
		return 0, err
	}
	return c.conn.Read(p)
}

func (c *idleTimeoutConn) Write(p []byte) (int, error) {
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.idleTimeout)); err != nil {
		return 0, err
	}
	return c.conn.Write(p)
}

// sendMessage sends one operation and reads the guest's messages until its concluding one, answering any
// mid-operation queries in between.
func (g *GuestSession) sendMessage(msg *vmprotocol.HostMessage) (*vmprotocol.GuestMessage, error) {
	if err := g.enc.EncodeHostMessage(msg); err != nil {
		return nil, err
	}
	for {
		var gm vmprotocol.GuestMessage
		if err := g.dec.DecodeGuestMessage(&gm); err != nil {
			return nil, err
		}
		switch gm.Kind {
		case vmprotocol.GuestMessageKindGraphicsCommands:
			if err := g.renderer.render(gm.GraphicsCommands); err != nil {
				return nil, err
			}
			continue
		case vmprotocol.GuestMessageKindQueryReadPixels:
			if err := g.answerReadPixels(&gm); err != nil {
				return nil, err
			}
			continue
		case vmprotocol.GuestMessageKindQueryMaxImageSize:
			if err := g.answerMaxImageSize(); err != nil {
				return nil, err
			}
			continue
		case vmprotocol.GuestMessageKindQueryDeviceScaleFactor:
			if err := g.answerDeviceScaleFactor(); err != nil {
				return nil, err
			}
			continue
		case vmprotocol.GuestMessageKindQueryColorSpace:
			if err := g.answerColorSpace(); err != nil {
				return nil, err
			}
			continue
		}
		// GuestMessageKindDone.
		if gm.Terminated {
			return &gm, ebiten.Termination
		}
		if gm.Err != "" {
			return &gm, errors.New(gm.Err)
		}
		return &gm, nil
	}
}

// answerReadPixels reads the requested regions back from the renderer and answers with the pixels.
// The commands reproducing the image have already arrived and been rendered.
func (g *GuestSession) answerReadPixels(query *vmprotocol.GuestMessage) error {
	answer := vmprotocol.HostMessage{
		Kind: vmprotocol.HostMessageKindAnswerReadPixels,
	}
	// One flat reused buffer backs all the regions; the total is computed first so that growing the
	// buffer cannot move the per-region subslices.
	var total int
	for _, r := range query.ReadRegions {
		total += 4 * r.Dx() * r.Dy()
	}
	g.pixelsBuf = slices.Grow(g.pixelsBuf[:0], total)[:total]
	g.pixelsListBuf = g.pixelsListBuf[:0]
	var off int
	for _, r := range query.ReadRegions {
		n := 4 * r.Dx() * r.Dy()
		g.pixelsListBuf = append(g.pixelsListBuf, g.pixelsBuf[off:off+n])
		off += n
	}
	if err := g.renderer.readPixels(g.pixelsListBuf, query.ReadImageID, query.ReadRegions); err != nil {
		answer.Err = err.Error()
	} else {
		answer.Pixels = g.pixelsListBuf
	}
	return g.enc.EncodeHostMessage(&answer)
}

// answerMaxImageSize answers with the host graphics driver's maximum image size. The query is served
// from within the host's frame, so the driver is initialized.
func (g *GuestSession) answerMaxImageSize() error {
	return g.enc.EncodeHostMessage(&vmprotocol.HostMessage{
		Kind:         vmprotocol.HostMessageKindAnswerMaxImageSize,
		MaxImageSize: ui.Get().GraphicsMaxImageSize(),
	})
}

// answerDeviceScaleFactor answers with the host's current device scale factor. The query is served
// from within the host's frame, so the host's monitor is available.
func (g *GuestSession) answerDeviceScaleFactor() error {
	return g.enc.EncodeHostMessage(&vmprotocol.HostMessage{
		Kind:        vmprotocol.HostMessageKindAnswerDeviceScaleFactor,
		ScaleFactor: hostDeviceScaleFactor(),
	})
}

// hostDeviceScaleFactor returns the host's current device scale factor, defaulting to 1 when no monitor
// is available (e.g. before the game starts).
func hostDeviceScaleFactor() float64 {
	if m := ebiten.Monitor(); m != nil {
		return m.DeviceScaleFactor()
	}
	return 1
}

// answerColorSpace answers with the host graphics driver's color space. The query is served from
// within the host's frame, so the driver is initialized.
func (g *GuestSession) answerColorSpace() error {
	return g.enc.EncodeHostMessage(&vmprotocol.HostMessage{
		Kind:       vmprotocol.HostMessageKindAnswerColorSpace,
		ColorSpace: ui.Get().GraphicsColorSpace(),
	})
}

// deferError records an error to surface at the next AdvanceTick, joined with any error already
// deferred.
func (g *GuestSession) deferError(err error) {
	g.err = errors.Join(g.err, err)
}

// SetOutsideScreen sets the host-owned image the guest's frames render into, and sizes the guest to
// it: the image is in device-dependent pixels, that is, the guest's outside size in
// device-independent pixels multiplied by the host's device scale factor
// ([ebiten.MonitorType.DeviceScaleFactor]). The image must not be nil. SetOutsideScreen must be
// called before the first [GuestSession.AdvanceTick], and again when the host replaces its screen
// (e.g. on a resize).
func (g *GuestSession) SetOutsideScreen(screen *ebiten.Image) error {
	if screen == nil {
		return errors.New("vmhost: SetOutsideScreen requires a non-nil screen")
	}
	g.outsideScreen = screen

	// The outside size is the screen's size divided by the host's device scale factor. The scale
	// itself is not sent: the guest pulls it from the host per tick and renders at the screen's full
	// device-dependent resolution.
	scale := hostDeviceScaleFactor()
	b := screen.Bounds()
	w := float64(b.Dx()) / scale
	h := float64(b.Dy()) / scale
	if w == g.sentWidth && h == g.sentHeight {
		return nil
	}
	if _, err := g.sendMessage(&vmprotocol.HostMessage{
		Kind:   vmprotocol.HostMessageKindSetOutsideSize,
		Width:  w,
		Height: h,
	}); err != nil {
		return err
	}
	g.sentWidth = w
	g.sentHeight = h
	return nil
}

// AdvanceTick runs one Update on the guest. It returns [ebiten.Termination] (matchable with
// [errors.Is]) when the guest's Update signals a regular termination, and it also carries any error
// deferred from a preceding [GuestSession.AdvanceFrame].
func (g *GuestSession) AdvanceTick() error {
	if g.err != nil {
		err := g.err
		g.err = nil
		return err
	}
	if g.outsideScreen == nil {
		return errors.New("vmhost: SetOutsideScreen must be called at least once before AdvanceTick")
	}
	// The guest streams its tick commands back as graphics-command batches, which sendMessage() renders to keep the
	// renderer's image and shader state current for later read-backs and frames.
	if _, err := g.sendMessage(&vmprotocol.HostMessage{
		Kind: vmprotocol.HostMessageKindAdvanceTick,
	}); err != nil {
		return err
	}
	g.ticked = true
	return nil
}

// AdvanceFrame has the guest draw its current state and composites the frame into the screen set by
// [GuestSession.SetOutsideScreen], without advancing the game state. It must not be called before the
// first [GuestSession.AdvanceTick]: a frame is never drawn before the first tick. Errors are deferred
// to [GuestSession.AdvanceTick].
func (g *GuestSession) AdvanceFrame() {
	if g.outsideScreen == nil {
		g.deferError(errors.New("vmhost: SetOutsideScreen must be called at least once before AdvanceFrame"))
		return
	}
	if !g.ticked {
		g.deferError(errors.New("vmhost: AdvanceTick must be called at least once before AdvanceFrame"))
		return
	}

	// The guest streams its draw commands back as graphics-command batches, which sendMessage() renders
	// into the screen.
	if _, err := g.sendMessage(&vmprotocol.HostMessage{
		Kind: vmprotocol.HostMessageKindAdvanceFrame,
	}); err != nil {
		g.deferError(err)
		return
	}

	sw, sh, err := g.renderer.screenSize()
	if err != nil {
		g.deferError(err)
		return
	}
	b := g.outsideScreen.Bounds()
	if sw != b.Dx() || sh != b.Dy() {
		g.deferError(fmt.Errorf("vmhost: rendered screen %dx%d does not match the outside screen %dx%d",
			sw, sh, b.Dx(), b.Dy()))
		return
	}
	dst, dstRegion := ui.ImageFromEbitenImage(g.outsideScreen)
	if dst == nil {
		// A disposed outside screen draws nothing.
		return
	}
	if err := g.renderer.compositeScreen(dst, dstRegion); err != nil {
		g.deferError(err)
	}
}

// PressKey injects a key-press event. key is an ebiten.Key.
func (g *GuestSession) PressKey(key ebiten.Key) error {
	_, err := g.sendMessage(&vmprotocol.HostMessage{
		Kind: vmprotocol.HostMessageKindPressKey,
		Code: int(key),
	})
	return err
}

// ReleaseKey injects a key-release event.
func (g *GuestSession) ReleaseKey(key ebiten.Key) error {
	_, err := g.sendMessage(&vmprotocol.HostMessage{
		Kind: vmprotocol.HostMessageKindReleaseKey,
		Code: int(key),
	})
	return err
}

// MoveCursor sets the cursor position in outside-screen device-independent pixels.
func (g *GuestSession) MoveCursor(x, y float64) error {
	_, err := g.sendMessage(&vmprotocol.HostMessage{
		Kind: vmprotocol.HostMessageKindMoveCursor,
		X:    x,
		Y:    y,
	})
	return err
}

// PressMouseButton injects a mouse-button-press event.
func (g *GuestSession) PressMouseButton(button ebiten.MouseButton) error {
	_, err := g.sendMessage(&vmprotocol.HostMessage{
		Kind: vmprotocol.HostMessageKindPressMouseButton,
		Code: int(button),
	})
	return err
}

// ReleaseMouseButton injects a mouse-button-release event.
func (g *GuestSession) ReleaseMouseButton(button ebiten.MouseButton) error {
	_, err := g.sendMessage(&vmprotocol.HostMessage{
		Kind: vmprotocol.HostMessageKindReleaseMouseButton,
		Code: int(button),
	})
	return err
}

// ScrollWheel injects a wheel movement.
func (g *GuestSession) ScrollWheel(x, y float64) error {
	_, err := g.sendMessage(&vmprotocol.HostMessage{
		Kind: vmprotocol.HostMessageKindScrollWheel,
		X:    x,
		Y:    y,
	})
	return err
}

// TypeRune injects a typed character.
func (g *GuestSession) TypeRune(r rune) error {
	_, err := g.sendMessage(&vmprotocol.HostMessage{
		Kind: vmprotocol.HostMessageKindTypeRune,
		Rune: r,
	})
	return err
}

// Close tells the guest to stop, releases the host images mirrored for it, and closes the connection.
// It ends the session, not the guest's process: the guest's [ebiten.RunGame] returns (with a nil
// error, as when its window is closed), and the process exits on its own. Close must be called from
// within the host's frame (it deallocates ebiten images).
func (g *GuestSession) Close() error {
	_, err := g.sendMessage(&vmprotocol.HostMessage{
		Kind: vmprotocol.HostMessageKindClose,
	})
	g.renderer.dispose()
	return errors.Join(err, g.conn.Close())
}

// EndpointURLFromAddr formats a host endpoint URL for the given address, typically a [net.Listener]'s,
// so it can be passed to a guest through EBITENGINE_VM_ENDPOINT or [ebiten.RunGameOptions].VMGuestEndpoint.
func EndpointURLFromAddr(addr net.Addr) (string, error) {
	e := vmprotocol.Endpoint{
		Network: addr.Network(),
		Address: addr.String(),
	}
	return e.URL()
}
