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

// Package vmprotocol is the message protocol connecting a host process to a guest process.
//
// On connect, both ends exchange a handshake to confirm a matching ProtocolVersion before
// interpreting the stream. Messages are named by their sender. The host sends one HostMessage
// operation at a time; the guest sends back a sequence of GuestMessages belonging to that operation
// — graphics-command batches, audio control and data, and queries, concluded by a done message — in
// lockstep. A query suspends the operation until the host answers it with the corresponding
// HostMessage. The encoding is gob.
package vmprotocol

import (
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"image"
	"io"
	"net"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/color"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

// HostMessageKind discriminates the messages a host sends to a guest: the HostMessageKindAnswer*
// kinds answer a guest query; the rest are operations for the guest to perform.
type HostMessageKind int

// The values follow iota: adding, inserting, or reordering kinds is a wire-affecting change (see
// ProtocolVersion).
const (
	HostMessageKindSetOutsideSize HostMessageKind = iota
	HostMessageKindAdvanceTick
	HostMessageKindAdvanceFrame
	HostMessageKindPressKey
	HostMessageKindReleaseKey
	HostMessageKindMoveCursor
	HostMessageKindPressMouseButton
	HostMessageKindReleaseMouseButton
	HostMessageKindScrollWheel
	HostMessageKindTypeRune
	HostMessageKindUpdateGamepads
	HostMessageKindClose

	// HostMessageKindReadAudio asks the guest to decode and return up to AudioMaxLenInBytes of one audio
	// player's samples (identified by AudioPlayerID). The guest answers with GuestMessageKindAudioData.
	HostMessageKindReadAudio

	// HostMessageKindAnswerReadPixels answers a GuestMessageKindQueryReadPixels.
	HostMessageKindAnswerReadPixels
	// HostMessageKindAnswerMaxImageSize answers a GuestMessageKindQueryMaxImageSize.
	HostMessageKindAnswerMaxImageSize
	// HostMessageKindAnswerDeviceScaleFactor answers a GuestMessageKindQueryDeviceScaleFactor.
	HostMessageKindAnswerDeviceScaleFactor
	// HostMessageKindAnswerColorSpace answers a GuestMessageKindQueryColorSpace.
	HostMessageKindAnswerColorSpace
)

// HostMessage is a message the host sends to a guest: an operation to perform, or the answer to a
// guest query. Only the fields relevant to Kind are populated.
type HostMessage struct {
	Kind HostMessageKind

	// SetOutsideSize.
	Width  float64
	Height float64

	// PressKey/ReleaseKey carry a ui.Key; PressMouseButton/ReleaseMouseButton carry a ui.MouseButton.
	Code int

	// MoveCursor and ScrollWheel.
	X float64
	Y float64

	// TypeRune.
	Rune rune

	// GamepadStates is the complete set of connected gamepads. Set on HostMessageKindUpdateGamepads; a
	// gamepad absent from it is disconnected.
	GamepadStates []GamepadState

	// Err is the query failure, if any, as a string (gob cannot carry an error value). Set on
	// HostMessageKindAnswerReadPixels.
	Err string

	// Pixels is set on HostMessageKindAnswerReadPixels: one buffer per requested region.
	Pixels [][]byte

	// MaxImageSize is set on HostMessageKindAnswerMaxImageSize.
	MaxImageSize int

	// ScaleFactor is set on HostMessageKindAnswerDeviceScaleFactor.
	ScaleFactor float64

	// ColorSpace is set on HostMessageKindAnswerColorSpace.
	ColorSpace color.ColorSpace

	// AudioPlayerID and AudioMaxLenInBytes are set on HostMessageKindReadAudio: the player to read from and
	// the maximum number of bytes to return.
	AudioPlayerID      int64
	AudioMaxLenInBytes int
}

// GamepadState is the forwarded state of one gamepad. The raw axes and buttons mirror the host's
// public gamepad view, where hats are folded into buttons; the standard maps hold the standard-layout
// view, with a present key meaning the standard button or axis is available.
type GamepadState struct {
	ID    int
	SDLID string
	Name  string

	// Axes holds each raw axis value, in -1..1.
	Axes []float64
	// Buttons holds each raw button's pressed state.
	Buttons []bool

	// StandardAxes and StandardButtons hold the standard-layout view; a key is present exactly when the
	// standard axis or button is available.
	StandardAxes    map[gamepaddb.StandardAxis]float64
	StandardButtons map[gamepaddb.StandardButton]GamepadStandardButtonState
}

// GamepadStandardButtonState is one standard-layout button's state: its pressed flag and its analog
// value in 0..1.
type GamepadStandardButtonState struct {
	Pressed bool
	Value   float64
}

// GuestMessageKind discriminates the messages a guest sends to the host.
type GuestMessageKind int

// The values follow iota: adding, inserting, or reordering kinds is a wire-affecting change (see
// ProtocolVersion).
const (
	// GuestMessageKindDone concludes the guest's message sequence for an operation and carries the
	// operation's outcome (Err and Terminated). Exactly one concludes each operation, after any
	// number of other messages.
	GuestMessageKindDone GuestMessageKind = iota
	// GuestMessageKindGraphicsCommands carries a batch of graphics commands for the host to replay.
	// Zero or more precede the concluding GuestMessageKindDone within one operation.
	GuestMessageKindGraphicsCommands
	// GuestMessageKindQueryReadPixels asks the host to read pixels back (from the commands already
	// sent) before the guest can finish the operation. The host answers with
	// HostMessageKindAnswerReadPixels.
	GuestMessageKindQueryReadPixels
	// GuestMessageKindQueryMaxImageSize asks the host for its graphics driver's maximum image size
	// before the guest can finish the operation. The host answers with
	// HostMessageKindAnswerMaxImageSize.
	GuestMessageKindQueryMaxImageSize
	// GuestMessageKindQueryDeviceScaleFactor asks the host for its current device scale factor before
	// the guest can finish the operation. The host answers with HostMessageKindAnswerDeviceScaleFactor.
	// Unlike the maximum image size it is not cached: the host's scale can change during a session
	// (e.g. the window moving to a monitor with a different DPI).
	GuestMessageKindQueryDeviceScaleFactor
	// GuestMessageKindQueryColorSpace asks the host for its graphics driver's color space before the
	// guest can finish the operation. The host answers with HostMessageKindAnswerColorSpace.
	GuestMessageKindQueryColorSpace
	// GuestMessageKindAudioControl carries the audio control changes since the last tick: the context's
	// sample rate and, for each player whose state changed, its playing flag and volume, or its removal
	// when closed. Zero or one precedes the concluding GuestMessageKindDone of an advance-tick operation.
	// The samples themselves are pulled separately (HostMessageKindReadAudio), so each guest player stays
	// its own stream, never mixed.
	GuestMessageKindAudioControl
	// GuestMessageKindAudioData answers a HostMessageKindReadAudio with one player's decoded samples and
	// whether its source has ended. It precedes the concluding GuestMessageKindDone of that operation.
	GuestMessageKindAudioData
)

// GuestMessage is a message a guest sends to the host while handling an operation: recorded graphics
// commands, a query, or the operation's conclusion. Only the fields relevant to Kind are populated.
type GuestMessage struct {
	Kind GuestMessageKind

	// Err is the deferred game error, if any, as a string (gob cannot carry an error value). Set on
	// GuestMessageKindDone.
	Err string

	// Terminated reports that the guest's Update signalled a regular termination (rather than failing).
	// It is kept distinct from Err so the host can map it back to the ebiten.Termination sentinel instead
	// of matching an error string. Set on GuestMessageKindDone.
	Terminated bool

	// GraphicsCommands is the batch of graphics commands the host must replay. Set on
	// GuestMessageKindGraphicsCommands.
	GraphicsCommands []GraphicsCommand

	// ReadImageID and ReadRegions identify the read-back request on GuestMessageKindQueryReadPixels.
	ReadImageID graphicsdriver.ImageID
	ReadRegions []image.Rectangle

	// AudioSampleRate is the guest audio context's sample rate, in per-channel samples per second. The
	// guest reports its own rate (it is not asked to match the host's); the host plays or inspects the
	// streams at it. Set on GuestMessageKindAudioControl.
	AudioSampleRate int

	// AudioControls carries the audio players whose control state changed since the last tick. Set on
	// GuestMessageKindAudioControl.
	AudioControls []AudioControl

	// AudioPCM is one player's decoded samples: 32-bit little-endian floats with two interleaved
	// channels, at AudioSampleRate, with the volume NOT applied. AudioEOF reports that the player's
	// source has ended (any final samples accompany it). Both set on GuestMessageKindAudioData.
	AudioPCM []byte
	AudioEOF bool
}

// AudioControl is one guest audio player's control state. The samples are pulled separately
// (HostMessageKindReadAudio); this carries only what the game changes from its Update.
type AudioControl struct {
	// ID identifies the guest player; it is stable for the player's lifetime and unique within a
	// session.
	ID int64

	// Playing reports whether the player is currently playing.
	Playing bool

	// Volume is the player's volume in [0,1]. It is reported, not applied to the samples, so the host
	// can observe the raw stream and apply the volume itself.
	Volume float64

	// Closed reports that the guest player was removed — closed by the game, or reclaimed once it
	// finished and was abandoned; the host drops the stream. When set, the other control fields carry no
	// state.
	Closed bool
}

// Encoder writes messages to a connection.
type Encoder struct {
	enc *gob.Encoder
}

// Decoder reads messages from a connection.
type Decoder struct {
	dec *gob.Decoder
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{enc: gob.NewEncoder(w)}
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{dec: gob.NewDecoder(r)}
}

func (e *Encoder) EncodeHostMessage(msg *HostMessage) error {
	return e.enc.Encode(msg)
}

func (d *Decoder) DecodeHostMessage(msg *HostMessage) error {
	return d.dec.Decode(msg)
}

func (e *Encoder) EncodeGuestMessage(msg *GuestMessage) error {
	return e.enc.Encode(msg)
}

func (d *Decoder) DecodeGuestMessage(msg *GuestMessage) error {
	return d.dec.Decode(msg)
}

// GuestMessageEncoder encodes a guest message to the host. A post-tick hook receives one so a guest
// subsystem can forward its own messages without depending on the UI backend that owns the connection.
// *Encoder implements it.
type GuestMessageEncoder interface {
	EncodeGuestMessage(*GuestMessage) error
}

// ProtocolVersion identifies the wire protocol. PerformHandshake asserts on connect that both ends
// hold the same value; a mismatched pair refuses to talk (there is no negotiation or fallback).
//
// The wire format is frozen within a patch series: builds sharing the same Ebitengine minor version
// (x.y) always agree on this value and so are always compatible. A change that affects the wire — the
// HostMessage/GuestMessage kinds or fields, or the GraphicsCommand schema or its semantics — bumps this
// value and may only land in a minor or major release.
const ProtocolVersion = 1

// handshakeMagic prefixes the handshake so a peer that isn't a vmguest (or speaks an incompatible
// preamble) is rejected with a clear error instead of misreading the stream.
const handshakeMagic = "ebvm"

// PerformHandshake exchanges and validates the protocol version over the connection, returning an
// error if the peer's magic or version does not match. Both ends call it once, before any other byte
// on the connection. Exactly one end must pass initiator=true.
func PerformHandshake(conn io.ReadWriter, initiator bool) error {
	// The preamble is raw fixed-size bytes rather than gob, so a mismatch is reported even when the
	// body encoding itself is what differs. The initiator sends first and the responder receives
	// first, so the exchange does not rely on the transport buffering a send ahead of a receive.
	if initiator {
		if err := writeHandshake(conn); err != nil {
			return err
		}
		buf, err := readHandshake(conn)
		if err != nil {
			return err
		}
		return validateHandshake(buf)
	}
	// Answer with the local preamble before validating the received one: if validation fails, the
	// initiator still receives the version it is mismatched against (instead of a bare connection
	// close).
	buf, err := readHandshake(conn)
	if err != nil {
		return err
	}
	if err := writeHandshake(conn); err != nil {
		return err
	}
	return validateHandshake(buf)
}

func writeHandshake(w io.Writer) error {
	return writeHandshakeVersion(w, ProtocolVersion)
}

func writeHandshakeVersion(w io.Writer, version uint32) error {
	var buf [8]byte
	copy(buf[0:4], handshakeMagic)
	binary.BigEndian.PutUint32(buf[4:8], version)
	_, err := w.Write(buf[:])
	return err
}

func readHandshake(r io.Reader) ([8]byte, error) {
	var buf [8]byte
	_, err := io.ReadFull(r, buf[:])
	return buf, err
}

func validateHandshake(buf [8]byte) error {
	if string(buf[0:4]) != handshakeMagic {
		return fmt.Errorf("vmprotocol: not a vmguest connection (magic %q)", buf[0:4])
	}
	if v := binary.BigEndian.Uint32(buf[4:8]); v != ProtocolVersion {
		return fmt.Errorf("vmprotocol: protocol version mismatch: local %d, peer %d", ProtocolVersion, v)
	}
	return nil
}

// Endpoint is a host's listening address: the (network, address) pair a guest passes to net.Dial.
// Its URL form is what the host advertises via EBITENGINE_VM_ENDPOINT.
type Endpoint struct {
	// Network is "unix" or "tcp".
	Network string

	// Address is the absolute OS-native socket path, or host:port. The host may be a domain name or
	// an IP literal.
	Address string
}

// ParseEndpoint parses an endpoint URL, recovering the OS-native socket path. It inverts
// Endpoint.URL.
func ParseEndpoint(endpoint string) (Endpoint, error) {
	if p, ok := strings.CutPrefix(endpoint, "unix://"); ok {
		// The path keeps a leading slash. On Unix it is part of the absolute socket path; on Windows the
		// path is drive-lettered (unix:///C:/foo), so the leading slash is dropped to recover C:/foo.
		if runtime.GOOS == "windows" {
			p = strings.TrimPrefix(p, "/")
		}
		addr := filepath.FromSlash(p)
		if !filepath.IsAbs(addr) {
			return Endpoint{}, fmt.Errorf("vmprotocol: a unix endpoint address must be an absolute path: %s", endpoint)
		}
		return Endpoint{Network: "unix", Address: addr}, nil
	}
	if p, ok := strings.CutPrefix(endpoint, "tcp://"); ok {
		// The host may be an IP literal, a bracketed IPv6 literal, or a domain name; net.Dial resolves
		// names at dial time.
		if _, _, err := net.SplitHostPort(p); err != nil {
			return Endpoint{}, fmt.Errorf("vmprotocol: invalid tcp endpoint: %w", err)
		}
		return Endpoint{Network: "tcp", Address: p}, nil
	}
	return Endpoint{}, fmt.Errorf("vmprotocol: unsupported endpoint: %s", endpoint)
}

// URL builds the endpoint URL the host advertises. Unix paths use the file://-style form:
// unix:///path, and unix:///C:/path on Windows. ParseEndpoint inverts it.
func (e Endpoint) URL() (string, error) {
	switch e.Network {
	case "unix":
		// A relative path would be resolved against the dialing process's own working directory, not
		// the listener's, so it cannot name the listener's socket reliably.
		if !filepath.IsAbs(e.Address) {
			return "", fmt.Errorf("vmprotocol: a unix endpoint address must be an absolute path: %s", e.Address)
		}
		// Normalize to forward slashes and ensure a leading slash so the result is a file://-style URL
		// on every OS (a Windows drive-lettered path is absolute yet has no leading slash).
		p := filepath.ToSlash(e.Address)
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		return "unix://" + p, nil
	case "tcp":
		if _, _, err := net.SplitHostPort(e.Address); err != nil {
			return "", fmt.Errorf("vmprotocol: invalid tcp endpoint: %w", err)
		}
		return "tcp://" + e.Address, nil
	default:
		return "", fmt.Errorf("vmprotocol: unsupported endpoint network: %s", e.Network)
	}
}
