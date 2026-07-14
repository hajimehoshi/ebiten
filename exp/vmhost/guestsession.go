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
// The protocol is hidden behind the GuestSession API. A background goroutine owns the connection and
// the guest's mirror images, so a slow or wedged guest never blocks the host's own frame.
// [GuestSession.AdvanceTicks] drives the guest's Update and [GuestSession.AdvanceFrame] requests its
// next frame; [GuestSession.CompositeFrame] composites the guest's most recently completed frame into
// the host-owned screen set by [GuestSession.SetOutsideScreen] so the host can composite it into its
// own window. The audio the guest plays is exposed per player — never mixed — through the
// [NewGuestSessionOptions] OnAudioStream handler, which hands the host each new stream to observe and
// play separately. The gamepad and device vibrations the guest requests are delivered to the
// [NewGuestSessionOptions] OnGamepadVibration and OnVibration handlers. The handlers run during
// [GuestSession.AdvanceTicks] and [GuestSession.WaitTicks], on the goroutine calling them, so a host
// never deals with the session's own goroutine.
//
// This package is experimental and the API might be changed in the future.
package vmhost

import (
	"errors"
	"image"
	"io"
	"maps"
	"net"
	"slices"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/internal/clock"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
	"github.com/hajimehoshi/ebiten/v2/internal/vmprotocol"
)

// GuestSession is a session with a guest process, driven and observed from the host side. It holds
// no process handle: the guest's lifetime belongs to whoever spawned it.
//
// A background goroutine owns the connection and the guest's mirror images and performs all protocol
// I/O, so the host's calls do not block on the guest. [GuestSession.CompositeFrame] composites into the
// host's screen and [GuestSession.Close] releases the images it composites; these two must be called
// from within the host's frame (its Update or Draw), and not concurrently with one another. The other
// methods may be called from any goroutine.
type GuestSession struct {
	// conn is safe for concurrent use: the session goroutine reads and writes through the codecs while
	// Close pokes its deadline and closes it.
	conn net.Conn

	// The following fields are owned by the session goroutine; no lock guards them.
	enc *vmprotocol.Encoder
	dec *vmprotocol.Decoder
	// renderer mirrors the guest's images. Close disposes it after the goroutine has joined, and
	// finishFrame publishes its screen into compositableFrame under mu.
	renderer *frameRenderer
	// pixelsBuf and pixelsListBuf back the ReadPixels answers: one flat reused buffer, subsliced per
	// region.
	pixelsBuf     []byte
	pixelsListBuf [][]byte
	// audioReadPCM and audioReadEOF capture the answer to the in-flight HostMessageKindReadAudio while
	// sendAndReceive runs; runReadAudio hands them to the waiting reader.
	audioReadPCM []byte
	audioReadEOF bool
	// textInput is the guest's active text-input session, or nil.
	textInput *GuestTextInput

	// The following fields are owned by the host goroutine; no lock guards them.
	outsideScreen   *ebiten.Image
	sentWidth       float64
	sentHeight      float64
	compositeVtxBuf []float32

	mu   sync.Mutex
	cond *sync.Cond

	// The following fields are guarded by mu.
	//
	// ops is the ordered request queue (ticks coalesced into a count, plus input and size messages),
	// drained by the session goroutine in submission order.
	ops []op
	// submittedTicks and consumedTicks count ticks requested and processed; their difference is the
	// backlog. They only increase. submittedTicks is touched only by the host goroutine, but it lives
	// here because WaitTicks compares it against consumedTicks, which the session writes.
	submittedTicks int64
	consumedTicks  int64
	// requestedFrameSeq increments on each AdvanceFrame, so coalesced requests collapse to the latest
	// value. renderedFrameSeq is the request the most recently completed frame satisfies; a
	// frame is owed while requestedFrameSeq exceeds it, and it doubles as the consumed marker once the
	// frame is composited. WaitFrame compares a snapshot of requestedFrameSeq against renderedFrameSeq so
	// it resolves to a specific requested frame rather than any newer one.
	requestedFrameSeq int64
	renderedFrameSeq  int64
	// framePhase is the lifecycle of the single in-progress frame; the session does not render a new one
	// until the host has composited the last (back to framePhaseRenderable).
	framePhase framePhase
	// compositableFrame is the mirror screen handed to the host to composite while framePhase is
	// framePhaseCompositable.
	compositableFrame hostImage
	// requestedTPS is the guest game's most recently reported requested TPS. It starts at the standard
	// default and the guest updates it whenever its game changes the value.
	requestedTPS int
	// cursorShape is the guest game's most recently reported cursor shape. It starts at the default
	// shape and the guest updates it whenever its game changes the value.
	cursorShape ebiten.CursorShapeType
	// screenTransparent is the guest game's most recently reported transparent-screen intent. It starts
	// false (opaque, matching ebiten.RunGameOptions' default); the session publishes the renderer's value
	// here after each rendered batch so CompositeFrame can read it on the host goroutine.
	screenTransparent bool
	// pendingEvents holds the guest events (vibrations, new audio streams) awaiting handler delivery, in
	// arrival order. The session goroutine appends them and dispatchPendingEvents drains them on the
	// goroutine calling AdvanceTicks or WaitTicks. Events for a nil handler are never queued, so the queue
	// only grows while requested ticks produce events for a registered handler.
	pendingEvents []guestEvent
	// pendingEventsBuf is the spare buffer dispatchPendingEvents swaps in for pendingEvents, so
	// steady-state dispatch alternates between the two slices' storage instead of allocating.
	pendingEventsBuf []guestEvent
	// msgPool is a free list backing the queued host messages: a sender takes one through
	// takeMessageLocked and the session goroutine returns it once it is encoded, so steady-state input
	// forwarding allocates nothing. Pooled messages are reset, with their buffers' capacity kept for
	// reuse.
	msgPool []*vmprotocol.HostMessage
	// err holds the error(s) that ended the session, joined as they occur; closed is set by Close.
	err    error
	closed bool

	// done is closed when the session goroutine exits; Close waits on it. closeOnce and closeErr make
	// Close idempotent.
	done      chan struct{}
	closeOnce sync.Once
	closeErr  error

	// audioMu guards the audio player map and the reported sample rate. Each GuestAudioStream guards its
	// own fields with its own lock, so reads of one player neither block the map operations nor the other
	// players. These have a separate lock from mu because the host reads players from an arbitrary
	// goroutine (an audio player's source), independently of the host's frame-bound work under mu.
	audioMu         sync.Mutex
	audioStreams    map[int64]*GuestAudioStream
	audioSampleRate int

	// onGamepadVibration, if non-nil, is called for each vibration the guest requests. It is set at
	// construction and never modified, so it is read without a lock.
	onGamepadVibration func(GamepadVibration)

	// onVibration, if non-nil, is called for each device vibration the guest requests. Like
	// onGamepadVibration, it is set at construction and never modified.
	onVibration func(Vibration)

	// onAudioStream, if non-nil, is called for each new audio stream the guest starts. Like
	// onGamepadVibration it is set at construction and never modified, so it is read without a lock.
	onAudioStream func(*GuestAudioStream)

	// onTextInput, if non-nil, is called for each text-input session the guest's game starts. Like
	// onGamepadVibration it is set at construction and never modified, so it is read without a lock.
	onTextInput func(*GuestTextInput)
}

// opKind discriminates an operation the session goroutine performs.
type opKind int

const (
	// opTick is a run of tick requests, count in op.count.
	opTick opKind = iota
	// opMessage is a single host message (input or outside size), in op.msg.
	opMessage
	// opFrame renders a frame. It is never queued; nextOp derives it from a pending frame request.
	opFrame
	// opReadAudio reads one audio player's samples; the result goes to op.audioResp.
	opReadAudio
)

type op struct {
	kind  opKind
	count int
	msg   *vmprotocol.HostMessage
	// seq is the frame request an opFrame satisfies; it labels the rendered frame for WaitFrame.
	seq int64
	// audioID and audioMaxLenInBytes parameterize an opReadAudio; audioResp receives its result.
	audioID            int64
	audioMaxLenInBytes int
	audioResp          chan audioReadResult
}

// guestEventKind discriminates a guest event queued for handler delivery.
type guestEventKind int

const (
	guestEventGamepadVibration guestEventKind = iota
	guestEventVibration
	guestEventAudioStream
	guestEventTextInput
)

// guestEvent is one guest-originated event awaiting its handler; the field its kind selects carries the
// payload.
type guestEvent struct {
	kind             guestEventKind
	gamepadVibration GamepadVibration
	vibration        Vibration
	audioStream      *GuestAudioStream
	textInput        *GuestTextInput
}

// framePhase tracks which side may touch the screen mirror as the single in-progress frame moves from
// render to composite. The phases are exclusive, so the host never composites the mirror while the
// session is rendering into it.
type framePhase int

const (
	// framePhaseRenderable: the mirror is open for the session to render the next frame into; the host
	// must not composite it.
	framePhaseRenderable framePhase = iota
	// framePhaseRendering: the session is rendering into the mirror; the host must not composite it.
	framePhaseRendering
	// framePhaseCompositable: the mirror holds a completed frame for the host to composite; the session
	// must not render into it.
	framePhaseCompositable
)

// NewGuestSessionOptions represents options for [NewGuestSession].
type NewGuestSessionOptions struct {
	// IdleTimeout is the maximum duration the connection may make no progress while an operation
	// (including the handshake) is in progress: when it elapses, the operation fails with an error
	// matching [os.ErrDeadlineExceeded], the error surfaces from [GuestSession.Err], and the session is
	// unusable except for [GuestSession.Close]. It bounds silence, not an operation's total duration: a
	// guest that keeps sending never times out. The default (0) means no timeout.
	IdleTimeout time.Duration

	// OnGamepadVibration, if non-nil, is called for each gamepad vibration the guest's game requests. It
	// runs during [GuestSession.AdvanceTicks] and [GuestSession.WaitTicks], on the calling goroutine; a
	// host typically just calls [ebiten.VibrateGamepad]. A nil handler discards the guest's vibrations.
	OnGamepadVibration func(GamepadVibration)

	// OnVibration, if non-nil, is called for each device vibration the guest's game requests. It runs
	// during [GuestSession.AdvanceTicks] and [GuestSession.WaitTicks], on the calling goroutine; a host
	// typically just calls [ebiten.Vibrate]. A nil handler discards the guest's vibrations.
	OnVibration func(Vibration)

	// OnAudioStream, if non-nil, is called once for each new audio stream the guest starts, handed the
	// persistent [GuestAudioStream] to read and inspect. It runs during [GuestSession.AdvanceTicks] and
	// [GuestSession.WaitTicks], on the calling goroutine; a host typically uses the stream as the source
	// of an audio player. A nil handler discards the guest's audio.
	OnAudioStream func(*GuestAudioStream)

	// OnTextInput, if non-nil, is called once for each text-input session the guest's game starts,
	// handed the [GuestTextInput] to serve as the guest's IME. It runs during
	// [GuestSession.AdvanceTicks] and [GuestSession.WaitTicks], on the calling goroutine. With a nil
	// handler a session starts and never receives a state, like a platform without IME support.
	OnTextInput func(*GuestTextInput)
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
	g := &GuestSession{
		conn:         conn,
		enc:          vmprotocol.NewEncoder(rw),
		dec:          vmprotocol.NewDecoder(rw),
		renderer:     newFrameRenderer(),
		done:         make(chan struct{}),
		audioStreams: map[int64]*GuestAudioStream{},
		// The guest reports its requested TPS only when it changes; until then it runs at the standard
		// default, so report that rather than a meaningless zero.
		requestedTPS: clock.DefaultTPS,
	}
	if options != nil {
		// Set before the session goroutine starts, so they are read without a lock.
		g.onGamepadVibration = options.OnGamepadVibration
		g.onVibration = options.OnVibration
		g.onAudioStream = options.OnAudioStream
		g.onTextInput = options.OnTextInput
	}
	g.cond = sync.NewCond(&g.mu)
	go g.sessionLoop()
	return g, nil
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

// sessionLoop owns the connection: it repeatedly takes the next task, runs it without the lock, and
// records the result. All access to the shared state is confined to the helper methods it calls, each
// of which holds the lock for its own scope.
func (g *GuestSession) sessionLoop() {
	defer close(g.done)
	// Release audio reads still queued at shutdown; a read already in flight is released by runReadAudio.
	defer g.drainQueuedReads()
	// A guest text-input session does not outlive the guest session: closing it here lets its holders
	// observe the end through IsClosed.
	defer func() {
		if g.textInput != nil {
			g.textInput.markClosed()
		}
	}()
	for {
		o, ok := g.nextOp()
		if !ok {
			return
		}
		var err error
		switch o.kind {
		case opTick, opMessage:
			// A successful tick or message needs no completion step: consumeTick already wakes WaitTicks
			// per tick, and a message changes nothing a waiter observes.
			err = g.runOp(o)
		case opReadAudio:
			err = g.runReadAudio(o)
		case opFrame:
			if err = g.sendAndReceive(&vmprotocol.HostMessage{
				Kind: vmprotocol.HostMessageKindAdvanceFrame,
			}); err == nil {
				err = g.finishFrame(o.seq)
			}
		}
		if err != nil {
			g.fail(err)
			return
		}
	}
}

// nextOp blocks until there is work, then returns the next operation; ok is false when the session
// should stop. Queued operations are drained before a frame is rendered, so a frame always reflects
// every request submitted before it.
func (g *GuestSession) nextOp() (op, bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for {
		if g.closed || g.err != nil {
			return op{}, false
		}
		if len(g.ops) > 0 {
			o := g.ops[0]
			g.ops = g.ops[1:]
			return o, true
		}
		if g.requestedFrameSeq > g.renderedFrameSeq && g.consumedTicks > 0 && g.framePhase == framePhaseRenderable {
			// A frame is owed, the queue is drained, and the mirror is free. The frame satisfies every
			// request up to the latest, so it carries the current requestedFrameSeq.
			g.framePhase = framePhaseRendering
			return op{kind: opFrame, seq: g.requestedFrameSeq}, true
		}
		// Nothing is actionable yet; wait for the next state change.
		g.cond.Wait()
	}
}

// fail records an error that ends the session and wakes any waiters.
func (g *GuestSession) fail(err error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.setErrLocked(err)
	g.cond.Broadcast()
}

// finishFrame publishes the just-rendered mirror screen for the host to composite and wakes a waiting
// WaitFrame. It returns an error if the guest produced no screen.
func (g *GuestSession) finishFrame(seq int64) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.renderer.screen == nil {
		g.framePhase = framePhaseRenderable
		return errors.New("vmhost: no screen framebuffer was produced")
	}
	g.compositableFrame = *g.renderer.screen
	g.renderedFrameSeq = seq
	g.framePhase = framePhaseCompositable
	g.cond.Broadcast()
	return nil
}

// runOp performs one queued operation: a run of ticks or a single host message. It must be called
// without g.mu held.
func (g *GuestSession) runOp(o op) error {
	switch o.kind {
	case opTick:
		for range o.count {
			if err := g.sendAndReceive(&vmprotocol.HostMessage{
				Kind: vmprotocol.HostMessageKindAdvanceTick,
			}); err != nil {
				return err
			}
			g.consumeTick()
		}
		return nil
	case opMessage:
		err := g.sendAndReceive(o.msg)
		g.recycleMessage(o.msg)
		return err
	}
	return nil
}

// consumeTick records that one tick has been processed.
func (g *GuestSession) consumeTick() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.consumedTicks++
	g.cond.Broadcast()
}

// sendAndReceive sends one operation and reads the guest's messages until its concluding one,
// rendering command batches and answering queries in between. It must be called without g.mu held: the
// renderer is the session goroutine's alone here.
func (g *GuestSession) sendAndReceive(msg *vmprotocol.HostMessage) error {
	if err := g.enc.EncodeHostMessage(msg); err != nil {
		return err
	}
	for {
		var gm vmprotocol.GuestMessage
		if err := g.dec.DecodeGuestMessage(&gm); err != nil {
			return err
		}
		switch gm.Kind {
		case vmprotocol.GuestMessageKindGraphicsCommands:
			if err := g.renderer.render(gm.GraphicsCommands); err != nil {
				return err
			}
			// Publish the renderer's transparent-screen intent for cross-goroutine reads. The renderer is
			// this goroutine's alone here, so reading its field directly is safe.
			g.setScreenTransparent(g.renderer.screenTransparent)
			continue
		case vmprotocol.GuestMessageKindQueryReadPixels:
			if err := g.answerReadPixels(&gm); err != nil {
				return err
			}
			continue
		case vmprotocol.GuestMessageKindQueryMaxImageSize:
			if err := g.answerMaxImageSize(); err != nil {
				return err
			}
			continue
		case vmprotocol.GuestMessageKindQueryDeviceScaleFactor:
			if err := g.answerDeviceScaleFactor(); err != nil {
				return err
			}
			continue
		case vmprotocol.GuestMessageKindQueryColorSpace:
			if err := g.answerColorSpace(); err != nil {
				return err
			}
			continue
		case vmprotocol.GuestMessageKindAudioControl:
			g.applyAudioControl(&gm)
			continue
		case vmprotocol.GuestMessageKindAudioData:
			// gm is fresh each iteration, so AudioPCM is a newly allocated slice no later decode
			// overwrites, and the waiting Read copies it out before the next read; aliasing it directly
			// needs no copy.
			g.audioReadPCM = gm.AudioPCM
			g.audioReadEOF = gm.AudioEOF
			continue
		case vmprotocol.GuestMessageKindRequestedTPS:
			g.setRequestedTPS(gm.RequestedTPS)
			continue
		case vmprotocol.GuestMessageKindCursorShape:
			g.setCursorShape(ebiten.CursorShapeType(gm.CursorShape))
			continue
		case vmprotocol.GuestMessageKindGamepadVibrations:
			g.queueGamepadVibrations(&gm)
			continue
		case vmprotocol.GuestMessageKindVibration:
			g.queueVibration(&gm)
			continue
		case vmprotocol.GuestMessageKindTextInput:
			g.handleTextInput(&gm)
			continue
		case vmprotocol.GuestMessageKindTextInputEnd:
			g.handleTextInputEnd(&gm)
			continue
		}
		// GuestMessageKindDone.
		if gm.Terminated {
			return ebiten.Termination
		}
		if gm.Err != "" {
			return errors.New(gm.Err)
		}
		return nil
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

// answerMaxImageSize answers with the host graphics driver's maximum image size.
func (g *GuestSession) answerMaxImageSize() error {
	return g.enc.EncodeHostMessage(&vmprotocol.HostMessage{
		Kind:         vmprotocol.HostMessageKindAnswerMaxImageSize,
		MaxImageSize: ui.Get().GraphicsMaxImageSize(),
	})
}

// answerDeviceScaleFactor answers with the host's current device scale factor.
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

// answerColorSpace answers with the host graphics driver's color space.
func (g *GuestSession) answerColorSpace() error {
	return g.enc.EncodeHostMessage(&vmprotocol.HostMessage{
		Kind:       vmprotocol.HostMessageKindAnswerColorSpace,
		ColorSpace: ui.Get().GraphicsColorSpace(),
	})
}

// setErrLocked records an error that ends the session, joined with any already recorded. g.mu must be
// held.
func (g *GuestSession) setErrLocked(err error) {
	g.err = errors.Join(g.err, err)
}

// SetOutsideScreen sets the host-owned image the guest's frames composite into, and sizes the guest to
// it: the image is in device-dependent pixels, that is, the guest's outside size in
// device-independent pixels multiplied by the host's device scale factor
// ([ebiten.MonitorType.DeviceScaleFactor]). The image must not be nil. SetOutsideScreen must be
// called before the first [GuestSession.AdvanceTicks], and again when the host replaces its screen
// (e.g. on a resize).
func (g *GuestSession) SetOutsideScreen(screen *ebiten.Image) error {
	if screen == nil {
		return errors.New("vmhost: SetOutsideScreen requires a non-nil screen")
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed || g.err != nil {
		return nil
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
	g.sentWidth = w
	g.sentHeight = h
	msg := g.takeMessageLocked()
	msg.Kind = vmprotocol.HostMessageKindSetOutsideSize
	msg.Width = w
	msg.Height = h
	g.queueOpLocked(op{
		kind: opMessage,
		msg:  msg,
	})
	return nil
}

// AdvanceTicks requests n Updates on the guest. It does not block on the guest: the requests are queued
// and processed by the session goroutine. n must not be negative; a zero count requests nothing.
//
// Before returning it delivers the guest events processed ticks have produced (vibrations, new audio
// streams, and new text-input sessions) to the [NewGuestSessionOptions] handlers, on the calling
// goroutine.
//
// A regular termination, a timeout, and any deferred error surface from [GuestSession.Err]; the number
// of requested-but-unprocessed ticks is reported by [GuestSession.PendingTicks].
func (g *GuestSession) AdvanceTicks(n int) {
	if n < 0 {
		panic("vmhost: negative AdvanceTicks count")
	}
	g.queueTicks(n)
	g.dispatchPendingEvents()
}

// queueTicks queues n tick requests for the session goroutine.
func (g *GuestSession) queueTicks(n int) {
	if n == 0 {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed || g.err != nil {
		return
	}
	if g.outsideScreen == nil {
		g.setErrLocked(errors.New("vmhost: SetOutsideScreen must be called at least once before AdvanceTicks"))
		return
	}
	// Coalesce consecutive ticks into a single counted entry.
	if k := len(g.ops); k > 0 && g.ops[k-1].kind == opTick {
		g.ops[k-1].count += n
	} else {
		g.ops = append(g.ops, op{kind: opTick, count: n})
	}
	g.submittedTicks += int64(n)
	g.cond.Broadcast()
}

// AdvanceFrame requests the guest's next frame. It does not block: the frame is rendered by the session
// goroutine and presented later by [GuestSession.CompositeFrame], optionally after blocking for it with
// [GuestSession.WaitFrame]. Without that wait, CompositeFrame likely presents a previously requested
// frame, since the one just requested has yet to render. At most one frame is in flight: requests made
// before the last is composited by CompositeFrame coalesce into one. A regular termination, a timeout,
// and any deferred error surface from [GuestSession.Err].
func (g *GuestSession) AdvanceFrame() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed || g.err != nil {
		return
	}
	// A frame can only be produced after the first tick.
	if g.submittedTicks > 0 {
		g.requestedFrameSeq++
		g.cond.Broadcast()
	}
}

// WaitFrame blocks until the frame requested by a preceding [GuestSession.AdvanceFrame] has been
// rendered, leaving it for [GuestSession.CompositeFrame] to present. Inserted between them it makes
// capture deterministic: AdvanceTicks, AdvanceFrame, WaitFrame, CompositeFrame leaves the outside screen
// reflecting the ticks. The wait resolves to the most recent frame requested as of the call: an earlier
// completed frame still awaiting compositing is dropped rather than returned, while a frame requested
// concurrently afterward does not extend the wait. It reports whether a frame is ready; it returns false
// when no frame was requested (no preceding AdvanceFrame) or the session has ended (see
// [GuestSession.Err]). It must not be called concurrently with [GuestSession.CompositeFrame].
func (g *GuestSession) WaitFrame() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed || g.err != nil {
		return false
	}
	// Wait for the latest request as of now; requests arriving later do not move this target.
	want := g.requestedFrameSeq
	// Nothing is on the way: the wanted frame has already been rendered and composited, and no newer one
	// is owed.
	if want <= g.renderedFrameSeq && g.framePhase == framePhaseRenderable {
		return false
	}
	for {
		if g.closed || g.err != nil {
			return false
		}
		if g.framePhase == framePhaseCompositable {
			// The completed frame satisfies the request once it is at least as new as the wanted one.
			if g.renderedFrameSeq >= want {
				return true
			}
			// The completed frame predates the request and occupies the single mirror, blocking the session
			// from rendering the request. Drop it (it was never composited) so the request can render. The
			// next render carries a seq >= want, so the wait then resolves without chasing later requests.
			g.framePhase = framePhaseRenderable
			g.cond.Broadcast()
		}
		g.cond.Wait()
	}
}

// CompositeFrame composites the guest's most recently completed frame into the screen set by
// [GuestSession.SetOutsideScreen], freeing the session to render the next. It reports whether the
// outside screen advanced to a newly completed frame; it returns false when no newer frame is ready yet
// (the screen keeps its previous content) or when the session has ended (see [GuestSession.Err]). It
// must be called from within the host's frame.
//
// A guest that did not request a transparent screen ([ebiten.RunGameOptions].ScreenTransparent) is
// composited over opaque black, as a standalone non-transparent window presents its frame; a
// transparent guest's frame keeps its alpha.
func (g *GuestSession) CompositeFrame() bool {
	frame, transparent := g.takeFrame()
	if frame.img == nil {
		return false
	}
	// The frame is taken; free the mirror for the next render whether it is presented or dropped.
	defer g.markComposited()

	// Drop the frame if its size no longer matches the outside screen (a resize happened after it was
	// rendered).
	b := g.outsideScreen.Bounds()
	if frame.width != b.Dx() || frame.height != b.Dy() {
		return false
	}
	dst, dstRegion := ui.ImageFromEbitenImage(g.outsideScreen)
	if dst == nil {
		// A disposed outside screen draws nothing.
		return false
	}
	// Prefill opaque black and blend the frame over it; a transparent guest's frame is copied as-is.
	blend := graphicsdriver.BlendCopy
	if !transparent {
		dst.Fill(0, 0, 0, 1, dstRegion)
		blend = graphicsdriver.BlendSourceOver
	}
	n := 4 * graphics.VertexFloatCount
	g.compositeVtxBuf = slices.Grow(g.compositeVtxBuf[:0], n)[:n]
	graphics.QuadVerticesFromDstAndSrc(g.compositeVtxBuf,
		float32(dstRegion.Min.X), float32(dstRegion.Min.Y), float32(dstRegion.Max.X), float32(dstRegion.Max.Y),
		0, 0, float32(frame.width), float32(frame.height), 1, 1, 1, 1)
	srcs := [graphics.ShaderSrcImageCount]*ui.Image{frame.img}
	srcRegions := [graphics.ShaderSrcImageCount]image.Rectangle{image.Rect(0, 0, frame.width, frame.height)}
	dst.DrawTriangles(srcs, g.compositeVtxBuf, graphics.QuadIndices(), blend, dstRegion, srcRegions, ui.NearestFilterShader, nil, true)
	return true
}

// takeFrame returns the completed frame for the host to composite and the guest's transparent-screen
// intent, or the zero value (nil img) when none is ready.
func (g *GuestSession) takeFrame() (frame hostImage, transparent bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed || g.err != nil {
		return hostImage{}, false
	}
	if g.framePhase != framePhaseCompositable {
		return hostImage{}, false
	}
	return g.compositableFrame, g.screenTransparent
}

// markComposited records that the host has consumed the ready frame, freeing the mirror for the session
// to render the next.
func (g *GuestSession) markComposited() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.framePhase = framePhaseRenderable
	g.cond.Broadcast()
}

// WaitTicks blocks until the guest has processed every tick requested so far. It reports whether all
// were processed; it returns false when the session ended first (see [GuestSession.Err]).
//
// Before returning it delivers the guest events the ticks produced (vibrations, new audio streams,
// and new text-input sessions) to the [NewGuestSessionOptions] handlers, on the calling goroutine.
func (g *GuestSession) WaitTicks() bool {
	ok := g.waitTicks()
	g.dispatchPendingEvents()
	return ok
}

// waitTicks blocks until the guest has processed every tick requested so far, or the session ends
// first.
func (g *GuestSession) waitTicks() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	st := g.submittedTicks
	for g.consumedTicks < st {
		if g.closed || g.err != nil {
			return false
		}
		g.cond.Wait()
	}
	return true
}

// PendingTicks returns the number of ticks requested but not yet processed by the guest.
func (g *GuestSession) PendingTicks() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return int(g.submittedTicks - g.consumedTicks)
}

// ProcessedTicks returns the number of ticks the guest has processed so far.
func (g *GuestSession) ProcessedTicks() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return int(g.consumedTicks)
}

// Err returns the error that ended the session, or nil if it is still running. It reports
// [ebiten.Termination] (matchable with [errors.Is]) when the guest's Update signaled a regular
// termination, a timeout (matchable with [os.ErrDeadlineExceeded]), or any deferred error. Once it is
// non-nil the session is unusable except for [GuestSession.Close].
func (g *GuestSession) Err() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.err
}

// PressKey injects a key-press event. key is an ebiten.Key.
func (g *GuestSession) PressKey(key ebiten.Key) {
	msg := g.takeMessage()
	msg.Kind = vmprotocol.HostMessageKindPressKey
	msg.Code = int(key)
	g.postMessage(msg)
}

// ReleaseKey injects a key-release event.
func (g *GuestSession) ReleaseKey(key ebiten.Key) {
	msg := g.takeMessage()
	msg.Kind = vmprotocol.HostMessageKindReleaseKey
	msg.Code = int(key)
	g.postMessage(msg)
}

// MoveCursor sets the cursor position in outside-screen device-independent pixels.
func (g *GuestSession) MoveCursor(x, y float64) {
	msg := g.takeMessage()
	msg.Kind = vmprotocol.HostMessageKindMoveCursor
	msg.X = x
	msg.Y = y
	g.postMessage(msg)
}

// PressMouseButton injects a mouse-button-press event.
func (g *GuestSession) PressMouseButton(button ebiten.MouseButton) {
	msg := g.takeMessage()
	msg.Kind = vmprotocol.HostMessageKindPressMouseButton
	msg.Code = int(button)
	g.postMessage(msg)
}

// ReleaseMouseButton injects a mouse-button-release event.
func (g *GuestSession) ReleaseMouseButton(button ebiten.MouseButton) {
	msg := g.takeMessage()
	msg.Kind = vmprotocol.HostMessageKindReleaseMouseButton
	msg.Code = int(button)
	g.postMessage(msg)
}

// ScrollWheel injects a wheel movement.
func (g *GuestSession) ScrollWheel(x, y float64) {
	msg := g.takeMessage()
	msg.Kind = vmprotocol.HostMessageKindScrollWheel
	msg.X = x
	msg.Y = y
	g.postMessage(msg)
}

// TypeRune injects a typed character.
func (g *GuestSession) TypeRune(r rune) {
	msg := g.takeMessage()
	msg.Kind = vmprotocol.HostMessageKindTypeRune
	msg.Rune = r
	g.postMessage(msg)
}

// queueOpLocked appends o to the request queue and wakes the session goroutine to drain it. g.mu must
// be held.
func (g *GuestSession) queueOpLocked(o op) {
	g.ops = append(g.ops, o)
	g.cond.Broadcast()
}

// GamepadState is a snapshot of one gamepad to inject with [GuestSession.UpdateGamepads].
type GamepadState struct {
	ID    ebiten.GamepadID
	SDLID string
	Name  string

	// Axes and Buttons follow the ebiten gamepad view, where hats are folded into buttons (matching
	// [ebiten.GamepadAxisValue] and [ebiten.IsGamepadButtonPressed]).
	Axes    []float64
	Buttons []bool

	// StandardAxes and StandardButtons hold the standard-layout view; a present key means the standard
	// axis or button is available, which need not match the SDL ID's database entry — a host may present
	// any standard layout it likes.
	StandardAxes    map[ebiten.StandardGamepadAxis]float64
	StandardButtons map[ebiten.StandardGamepadButton]GamepadStandardButtonState
}

// GamepadStandardButtonState is one standard-layout button's pressed flag and its analog value in
// 0..1.
type GamepadStandardButtonState struct {
	Pressed bool
	Value   float64
}

// UpdateGamepads injects the complete set of connected gamepads; a gamepad absent from states is
// disconnected. Like the other input injectors it is fed independently of [GuestSession.AdvanceTicks]
// and observed by the guest at its next tick.
//
// The caller retains ownership of states and everything it references: the snapshot is copied out
// before UpdateGamepads returns, so the slice, its elements, and their slices and maps may be reused
// (e.g. refilled for the next tick).
func (g *GuestSession) UpdateGamepads(states []GamepadState) {
	// Gamepad state is polled per tick at the source — continuous axes plus a changing set of connected
	// devices — so each call resends the whole snapshot, keeping the guest's view authoritative and
	// self-correcting against a dropped or duplicated message.
	msg := g.takeMessage()
	msg.Kind = vmprotocol.HostMessageKindUpdateGamepads
	msg.GamepadStates = appendGamepadStatesToProtocol(msg.GamepadStates, states)
	g.postMessage(msg)
}

// appendGamepadStatesToProtocol copies states into dst in their protocol form and returns the extended
// slice, reusing dst's elements and their slices and maps within capacity. An emptied-but-kept buffer
// encodes like a nil one (gob omits zero-length fields).
func appendGamepadStatesToProtocol(dst []vmprotocol.GamepadState, states []GamepadState) []vmprotocol.GamepadState {
	// Reslice through the capacity so the elements there — and their inner buffers — survive both the
	// extension and a growth's copy.
	if n := len(states); n <= cap(dst) {
		dst = dst[:n]
	} else {
		dst = slices.Grow(dst[:cap(dst)], n-cap(dst))[:n]
	}
	for i := range states {
		s := &states[i]
		d := &dst[i]
		d.ID = int(s.ID)
		d.SDLID = s.SDLID
		d.Name = s.Name
		d.Axes = append(d.Axes[:0], s.Axes...)
		d.Buttons = append(d.Buttons[:0], s.Buttons...)
		d.StandardAxes = copyStandardAxesToProtocol(d.StandardAxes, s.StandardAxes)
		d.StandardButtons = copyStandardButtonsToProtocol(d.StandardButtons, s.StandardButtons)
	}
	return dst
}

// copyStandardAxesToProtocol copies src into dst, reusing dst's storage; it allocates only when dst is
// nil and src has entries.
func copyStandardAxesToProtocol(dst, src map[ebiten.StandardGamepadAxis]float64) map[ebiten.StandardGamepadAxis]float64 {
	if dst == nil {
		if len(src) == 0 {
			return nil
		}
		dst = make(map[ebiten.StandardGamepadAxis]float64, len(src))
	} else {
		clear(dst)
	}
	maps.Copy(dst, src)
	return dst
}

// copyStandardButtonsToProtocol copies src into dst in its protocol form, reusing dst's storage; it
// allocates only when dst is nil and src has entries.
func copyStandardButtonsToProtocol(dst map[ebiten.StandardGamepadButton]vmprotocol.GamepadStandardButtonState, src map[ebiten.StandardGamepadButton]GamepadStandardButtonState) map[ebiten.StandardGamepadButton]vmprotocol.GamepadStandardButtonState {
	if dst == nil {
		if len(src) == 0 {
			return nil
		}
		dst = make(map[ebiten.StandardGamepadButton]vmprotocol.GamepadStandardButtonState, len(src))
	} else {
		clear(dst)
	}
	for b, s := range src {
		dst[b] = vmprotocol.GamepadStandardButtonState{
			Pressed: s.Pressed,
			Value:   s.Value,
		}
	}
	return dst
}

// GamepadVibration is a vibration the guest's game requested for one gamepad, passed to the
// [NewGuestSessionOptions] OnGamepadVibration handler. GamepadID matches the
// [GuestSession.UpdateGamepads] ID, so a host applies it to the corresponding gamepad with
// [ebiten.VibrateGamepad].
type GamepadVibration struct {
	// StartTick is the guest's [ebiten.Tick] during the Update that requested the vibration.
	StartTick int

	GamepadID ebiten.GamepadID
	Duration  time.Duration

	// StrongMagnitude and WeakMagnitude are the low- and high-frequency rumble intensities, in 0..1.
	StrongMagnitude float64
	WeakMagnitude   float64
}

// queueGamepadVibrations queues one event per vibration the tick requested for the handler, stamping
// each with the tick that produced it. It runs on the session goroutine and does nothing when no handler
// is registered.
func (g *GuestSession) queueGamepadVibrations(msg *vmprotocol.GuestMessage) {
	if g.onGamepadVibration == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	for i := range msg.GamepadVibrations {
		v := &msg.GamepadVibrations[i]
		g.pendingEvents = append(g.pendingEvents, guestEvent{
			kind: guestEventGamepadVibration,
			gamepadVibration: GamepadVibration{
				StartTick:       msg.StartTick,
				GamepadID:       ebiten.GamepadID(v.ID),
				Duration:        v.Duration,
				StrongMagnitude: v.StrongMagnitude,
				WeakMagnitude:   v.WeakMagnitude,
			},
		})
	}
}

// Vibration is a device vibration the guest's game requested, passed to the [NewGuestSessionOptions]
// OnVibration handler. A host acts on it by vibrating its own device with [ebiten.Vibrate].
type Vibration struct {
	// StartTick is the guest's [ebiten.Tick] during the Update that requested the vibration.
	StartTick int

	Duration time.Duration

	// Magnitude is the vibration strength, in 0..1.
	Magnitude float64
}

// queueVibration queues the device vibration the tick requested for the handler, stamped with the tick
// that produced it. It runs on the session goroutine and does nothing when no handler is registered.
func (g *GuestSession) queueVibration(msg *vmprotocol.GuestMessage) {
	if g.onVibration == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.pendingEvents = append(g.pendingEvents, guestEvent{
		kind: guestEventVibration,
		vibration: Vibration{
			StartTick: msg.StartTick,
			Duration:  msg.Vibration.Duration,
			Magnitude: msg.Vibration.Magnitude,
		},
	})
}

// dispatchPendingEvents delivers the queued guest events to their handlers, in arrival order, on the
// calling goroutine. A queued event's handler is always registered: queueing checks it. Events queued
// while the handlers run are left for the next dispatch.
func (g *GuestSession) dispatchPendingEvents() {
	events := g.takePendingEvents()
	if len(events) == 0 {
		return
	}
	for i := range events {
		e := &events[i]
		switch e.kind {
		case guestEventGamepadVibration:
			g.onGamepadVibration(e.gamepadVibration)
		case guestEventVibration:
			g.onVibration(e.vibration)
		case guestEventAudioStream:
			g.onAudioStream(e.audioStream)
		case guestEventTextInput:
			g.onTextInput(e.textInput)
		}
	}
	g.recyclePendingEvents(events)
}

// takePendingEvents takes ownership of the queued events, handing the spare buffer to the queue so
// nothing appends to the returned slice. The spare is taken (not shared): a concurrent dispatch gets a
// fresh slice.
func (g *GuestSession) takePendingEvents() []guestEvent {
	g.mu.Lock()
	defer g.mu.Unlock()
	events := g.pendingEvents
	if len(events) == 0 {
		return nil
	}
	g.pendingEvents = g.pendingEventsBuf[:0]
	g.pendingEventsBuf = nil
	return events
}

// recyclePendingEvents returns the emptied events buffer as the spare, zeroing the elements so the
// payloads' pointers are not retained.
func (g *GuestSession) recyclePendingEvents(events []guestEvent) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.pendingEventsBuf = slices.Delete(events, 0, len(events))
}

// handleTextInput tracks the guest's new text-input session and queues it for the handler. It
// runs on the session goroutine.
func (g *GuestSession) handleTextInput(msg *vmprotocol.GuestMessage) {
	// The guest releases a session before starting the next one, but be robust to a missing end: a new
	// session supersedes the current one.
	if g.textInput != nil {
		g.textInput.markClosed()
	}
	t := &GuestTextInput{
		g:               g,
		id:              msg.TextInputID,
		bounds:          msg.TextInputBounds,
		textBeforeCaret: msg.TextInputTextBeforeCaret,
		textAfterCaret:  msg.TextInputTextAfterCaret,
		startTick:       msg.StartTick,
	}
	g.textInput = t
	if g.onTextInput == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.pendingEvents = append(g.pendingEvents, guestEvent{
		kind:      guestEventTextInput,
		textInput: t,
	})
}

// handleTextInputEnd marks the guest's text-input session as released. It runs on the session
// goroutine.
func (g *GuestSession) handleTextInputEnd(msg *vmprotocol.GuestMessage) {
	if g.textInput == nil || g.textInput.id != msg.TextInputID {
		return
	}
	g.textInput.markClosed()
	g.textInput = nil
}

// PressTouch injects a touch-press event at (x, y), in outside-screen device-independent pixels.
func (g *GuestSession) PressTouch(id ebiten.TouchID, x, y float64) {
	msg := g.takeMessage()
	msg.Kind = vmprotocol.HostMessageKindPressTouch
	msg.Code = int(id)
	msg.X = x
	msg.Y = y
	g.postMessage(msg)
}

// MoveTouch injects a touch-move event to (x, y), in outside-screen device-independent pixels.
func (g *GuestSession) MoveTouch(id ebiten.TouchID, x, y float64) {
	msg := g.takeMessage()
	msg.Kind = vmprotocol.HostMessageKindMoveTouch
	msg.Code = int(id)
	msg.X = x
	msg.Y = y
	g.postMessage(msg)
}

// ReleaseTouch injects a touch-release event.
func (g *GuestSession) ReleaseTouch(id ebiten.TouchID) {
	msg := g.takeMessage()
	msg.Kind = vmprotocol.HostMessageKindReleaseTouch
	msg.Code = int(id)
	g.postMessage(msg)
}

// postMessage queues a single host message in submission order. msg must come from takeMessage; the
// session goroutine recycles it once it is encoded (or it is recycled here when the session has
// already ended).
func (g *GuestSession) postMessage(msg *vmprotocol.HostMessage) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed || g.err != nil {
		g.recycleMessageLocked(msg)
		return
	}
	g.queueOpLocked(op{
		kind: opMessage,
		msg:  msg,
	})
}

// takeMessage returns a host message to queue, recycled from the pool when one is available. It is
// reset: the caller sets the fields its kind needs.
func (g *GuestSession) takeMessage() *vmprotocol.HostMessage {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.takeMessageLocked()
}

// takeMessageLocked is takeMessage for callers already holding g.mu.
func (g *GuestSession) takeMessageLocked() *vmprotocol.HostMessage {
	n := len(g.msgPool)
	if n == 0 {
		return &vmprotocol.HostMessage{}
	}
	msg := g.msgPool[n-1]
	g.msgPool[n-1] = nil
	g.msgPool = g.msgPool[:n-1]
	return msg
}

// recycleMessage resets msg and returns it to the pool for the next takeMessage.
func (g *GuestSession) recycleMessage(msg *vmprotocol.HostMessage) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.recycleMessageLocked(msg)
}

// recycleMessageLocked is recycleMessage for callers already holding g.mu.
func (g *GuestSession) recycleMessageLocked(msg *vmprotocol.HostMessage) {
	// The reset keeps the GamepadStates buffer — and, within it, each element's slices and maps — so the
	// next UpdateGamepads copies into the same storage.
	*msg = vmprotocol.HostMessage{GamepadStates: msg.GamepadStates[:0]}
	g.msgPool = append(g.msgPool, msg)
}

// Close stops the session, releases the host images mirrored for it, and closes the connection. It
// ends the session, not the guest's process: the guest's [ebiten.RunGame] returns (with a nil error,
// as when its window is closed), and the process exits on its own. Close releases the mirror images
// that [GuestSession.CompositeFrame] composites, so it must be called from within the host's frame and
// not concurrently with it.
func (g *GuestSession) Close() error {
	g.closeOnce.Do(func() {
		g.requestClose()
		// Unblock the session goroutine if it is mid-read on a wedged guest. When it is idle this is a
		// harmless no-op: it wakes from the broadcast and exits without touching the connection again.
		_ = g.conn.SetDeadline(time.Now())
		<-g.done

		g.renderer.dispose()
		g.closeErr = g.conn.Close()
	})
	return g.closeErr
}

// requestClose marks the session closed and wakes the session goroutine.
func (g *GuestSession) requestClose() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.closed = true
	g.cond.Broadcast()
}

// audioReadResult is the outcome of an opReadAudio, delivered to the waiting GuestAudioStream.Read.
type audioReadResult struct {
	pcm []byte
	eof bool
}

// runReadAudio reads one audio player's samples from the guest and delivers them to the waiting
// GuestAudioStream.Read. It must be called without g.mu held. It always closes o.audioResp before
// returning: the result is sent first on success, while a connection error closes it without sending,
// so the reader reports end-of-stream.
func (g *GuestSession) runReadAudio(o op) error {
	defer close(o.audioResp)
	g.audioReadPCM = nil
	g.audioReadEOF = false
	if err := g.sendAndReceive(&vmprotocol.HostMessage{
		Kind:               vmprotocol.HostMessageKindReadAudio,
		AudioPlayerID:      o.audioID,
		AudioMaxLenInBytes: o.audioMaxLenInBytes,
	}); err != nil {
		return err
	}
	o.audioResp <- audioReadResult{
		pcm: g.audioReadPCM,
		eof: g.audioReadEOF,
	}
	return nil
}

// drainQueuedReads closes the response channel of every audio read still queued when the session ends,
// so its waiting GuestAudioStream.Read reports end-of-stream. It runs once the session loop has stopped,
// so g.closed or g.err is set and no further read can be queued; a read already in flight is closed by
// runReadAudio instead, so the two never close the same channel.
func (g *GuestSession) drainQueuedReads() {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, o := range g.ops {
		if o.kind == opReadAudio {
			close(o.audioResp)
		}
	}
	g.ops = nil
}

// applyAudioControl applies a tick's control changes: it records the sample rate and each player's
// playing flag and volume, creating a GuestAudioStream for an unseen ID. It runs on the session
// goroutine; the players' Read fetch their samples concurrently, each under its own lock.
func (g *GuestSession) applyAudioControl(msg *vmprotocol.GuestMessage) {
	// The new streams are collected under audioMu and queued for the handler under mu afterward, so the
	// two locks never nest.
	newStreams := g.updateAudioStreams(msg)
	if len(newStreams) == 0 {
		return
	}
	// Queue each new stream for the handler in creation order (AudioControls arrive sorted by ID). By the
	// time the handler runs, the stream's control is applied, so its Playing and Volume reflect at least
	// this tick.
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, p := range newStreams {
		g.pendingEvents = append(g.pendingEvents, guestEvent{
			kind:        guestEventAudioStream,
			audioStream: p,
		})
	}
}

// updateAudioStreams records the sample rate and each player's playing flag and volume, creating a
// GuestAudioStream for an unseen ID. It returns the streams created for the handler, none when no
// handler is registered.
func (g *GuestSession) updateAudioStreams(msg *vmprotocol.GuestMessage) []*GuestAudioStream {
	g.audioMu.Lock()
	defer g.audioMu.Unlock()
	var newStreams []*GuestAudioStream
	g.audioSampleRate = msg.AudioSampleRate
	for i := range msg.AudioControls {
		c := &msg.AudioControls[i]
		if c.Closed {
			// The guest closed the player: mark the stream not playing and drop it. An unknown ID (never
			// observed, or already gone) is ignored.
			if p := g.audioStreams[c.ID]; p != nil {
				p.markClosed()
				delete(g.audioStreams, c.ID)
			}
			continue
		}
		p := g.audioStreams[c.ID]
		if p == nil {
			p = &GuestAudioStream{
				session:   g,
				id:        c.ID,
				rate:      msg.AudioSampleRate,
				startTick: msg.StartTick,
			}
			g.audioStreams[c.ID] = p
			if g.onAudioStream != nil {
				newStreams = append(newStreams, p)
			}
		}
		p.setControl(c.Playing, c.Volume)
	}
	return newStreams
}

// AudioSampleRate returns the sample rate of the guest's audio, in per-channel samples per second, as
// reported by the guest. It is 0 until the guest has played audio. The guest uses the rate its game
// chose; the host need not match it (or play the audio at all), but a host that does play it should
// use this rate.
func (g *GuestSession) AudioSampleRate() int {
	g.audioMu.Lock()
	defer g.audioMu.Unlock()
	return g.audioSampleRate
}

// setRequestedTPS records the guest's reported requested TPS.
func (g *GuestSession) setRequestedTPS(tps int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.requestedTPS = tps
}

// RequestedTPS returns the ticks-per-second the guest's game requests via [ebiten.SetTPS], which may be
// [ebiten.SyncWithFPS]. It is the standard default until the guest's game changes it. The host drives
// the guest's ticks itself, so this is advisory; a host pacing the guest in real time should honor it.
func (g *GuestSession) RequestedTPS() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.requestedTPS
}

// setCursorShape records the guest's reported cursor shape.
func (g *GuestSession) setCursorShape(shape ebiten.CursorShapeType) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.cursorShape = shape
}

// CursorShape returns the cursor shape the guest's game requests via [ebiten.SetCursorShape]. It is
// [ebiten.CursorShapeDefault] until the guest's game changes it. The guest has no cursor of its own,
// so a host honors it by applying the shape to its own cursor while the cursor is over the guest's
// screen.
func (g *GuestSession) CursorShape() ebiten.CursorShapeType {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.cursorShape
}

// setScreenTransparent records the guest's reported transparent-screen intent for CompositeFrame.
func (g *GuestSession) setScreenTransparent(transparent bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.screenTransparent = transparent
}

// readGuestAudio fills b with player id's samples from the guest, truncated to whole stereo frames,
// and reports the number of bytes written and whether the source has ended. It is called from a
// GuestAudioStream.Read on an arbitrary goroutine: it queues a read for the session goroutine (which
// owns the connection) and waits. A closed or failed session reads as end-of-stream.
func (g *GuestSession) readGuestAudio(id int64, b []byte) (n int, eof bool) {
	// Round len(b) down to a multiple of 8 — one float32 per channel, two channels — so a sample is
	// never split across reads.
	maxLenInBytes := len(b) - len(b)%8
	// Query even when maxLenInBytes is 0, so the guest's true end-of-stream state is reported regardless of
	// the buffer size. The guest returns at most maxLenInBytes, so the result always fits in b.
	resp, ok := g.queueReadAudio(id, maxLenInBytes)
	if !ok {
		return 0, true
	}

	// The session goroutine sends the result, or closes resp without sending when the session ends before
	// the read completes; a closed resp reports end-of-stream.
	res, ok := <-resp
	if !ok {
		return 0, true
	}
	return copy(b, res.pcm), res.eof
}

// queueReadAudio queues a read of player id's samples for the session goroutine and wakes it,
// returning the channel its result will arrive on. ok is false when the session has closed or failed,
// so no read is queued.
func (g *GuestSession) queueReadAudio(id int64, maxLenInBytes int) (resp chan audioReadResult, ok bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.closed || g.err != nil {
		return nil, false
	}
	resp = make(chan audioReadResult, 1)
	g.queueOpLocked(op{
		kind:               opReadAudio,
		audioID:            id,
		audioMaxLenInBytes: maxLenInBytes,
		audioResp:          resp,
	})
	return resp, true
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
