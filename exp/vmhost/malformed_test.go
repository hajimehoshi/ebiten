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

// Ill-formed guest messages. A real guest never sends these, so the tests here speak the protocol
// directly instead of launching one. Each case once took the whole host process down with a panic,
// so a regression fails the test binary rather than the test.

package vmhost_test

import (
	"image"
	"net"
	"os/exec"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/vmprotocol"
)

// fakeGuest is a guest speaking the protocol by hand, so a test can send what a real guest cannot.
type fakeGuest struct {
	enc *vmprotocol.Encoder
	dec *vmprotocol.Decoder
}

// sendCommands sends one graphics-command batch.
func (fg *fakeGuest) sendCommands(cmds ...vmprotocol.GraphicsCommand) {
	fg.send(&vmprotocol.GuestMessage{
		Kind:             vmprotocol.GuestMessageKindGraphicsCommands,
		GraphicsCommands: cmds,
	})
}

// send sends one guest message.
func (fg *fakeGuest) send(msg *vmprotocol.GuestMessage) {
	_ = fg.enc.EncodeGuestMessage(msg)
}

// receive blocks for the host's next message, reporting an error once the connection is gone.
func (fg *fakeGuest) receive() (vmprotocol.HostMessage, error) {
	var msg vmprotocol.HostMessage
	if err := fg.dec.DecodeHostMessage(&msg); err != nil {
		return vmprotocol.HostMessage{}, err
	}
	return msg, nil
}

// newFakeGuestSession opens a session against a fake guest that runs script for each tick the host
// requests and concludes the operation once it returns. All resources are released via t.Cleanup.
func newFakeGuestSession(t *testing.T, script func(fg *fakeGuest)) *vmhost.GuestSession {
	t.Helper()
	skipIfVMUnsupported(t)

	hostConn, guestConn := net.Pipe()
	go func() {
		if err := vmprotocol.PerformHandshake(guestConn, false); err != nil {
			return
		}
		fg := &fakeGuest{
			enc: vmprotocol.NewEncoder(guestConn),
			dec: vmprotocol.NewDecoder(guestConn),
		}
		for {
			msg, err := fg.receive()
			if err != nil {
				return
			}
			if msg.Kind == vmprotocol.HostMessageKindAdvanceTick {
				script(fg)
			}
			fg.send(&vmprotocol.GuestMessage{Kind: vmprotocol.GuestMessageKindDone})
		}
	}()

	guest, err := vmhost.NewGuestSession(hostConn, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// Closing the session's end first releases the fake guest, which may be blocked writing to a host
		// that stopped reading.
		if err := guest.Close(); err != nil {
			t.Errorf("closing the guest session failed: %v", err)
		}
		if err := guestConn.Close(); err != nil {
			t.Errorf("closing the fake guest's connection failed: %v", err)
		}
	})
	return guest
}

// setOutsideScreen gives the session the host-owned screen every session needs before its first tick.
func setOutsideScreen(t *testing.T, guest *vmhost.GuestSession) {
	t.Helper()
	// The guest renders at the host's device scale factor, so the screen is physical-sized.
	scale := ebiten.Monitor().DeviceScaleFactor()
	if err := guest.SetOutsideScreen(ebiten.NewImage(int(320*scale), int(240*scale))); err != nil {
		t.Fatal(err)
	}
}

// fillShaderSource is a shader taking no uniforms of its own, for draws whose ill-formed part is
// elsewhere.
var fillShaderSource = []byte(`//kage:unit pixels
package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(1)
}
`)

// drawPrelude records the image, shader, and vertex buffer a DrawTriangles needs, so a case can
// follow it with only the command it is about.
func drawPrelude(vertexCount int, indices []uint32) []vmprotocol.GraphicsCommand {
	return []vmprotocol.GraphicsCommand{
		{
			Kind:    vmprotocol.GraphicsCommandKindNewImage,
			ImageID: 1,
			Width:   16,
			Height:  16,
		},
		{
			Kind:         vmprotocol.GraphicsCommandKindNewShader,
			ShaderID:     1,
			ShaderSource: fillShaderSource,
		},
		{
			Kind:     vmprotocol.GraphicsCommandKindSetVertices,
			Vertices: make([]float32, vertexCount*graphics.VertexFloatCount),
			Indices:  indices,
		},
	}
}

// drawTriangles records a draw of the prelude's image and shader.
func drawTriangles(indexCount, indexOffset int) vmprotocol.GraphicsCommand {
	return vmprotocol.GraphicsCommand{
		Kind:        vmprotocol.GraphicsCommandKindDrawTriangles,
		Dst:         1,
		ShaderID:    1,
		DstRegions:  []graphicsdriver.DstRegion{{Region: image.Rect(0, 0, 16, 16), IndexCount: indexCount}},
		IndexOffset: indexOffset,
		Uniforms:    make([]uint32, graphics.PreservedUniformDwordCount),
	}
}

func TestIllFormedGuestMessageFailsTheSession(t *testing.T) {
	cases := []struct {
		name   string
		script func(fg *fakeGuest)
	}{
		{
			name: "NewImage with a non-positive size",
			script: func(fg *fakeGuest) {
				fg.sendCommands(vmprotocol.GraphicsCommand{
					Kind:    vmprotocol.GraphicsCommandKindNewImage,
					ImageID: 1,
					Width:   -4,
					Height:  -4,
				})
			},
		},
		{
			name: "NewImage larger than the host allows",
			script: func(fg *fakeGuest) {
				fg.sendCommands(vmprotocol.GraphicsCommand{
					Kind:    vmprotocol.GraphicsCommandKindNewImage,
					ImageID: 1,
					Width:   1 << 20,
					Height:  1 << 20,
				})
			},
		},
		{
			name: "NewImage reusing a live image ID",
			script: func(fg *fakeGuest) {
				fg.sendCommands(
					vmprotocol.GraphicsCommand{
						Kind:    vmprotocol.GraphicsCommandKindNewImage,
						ImageID: 1,
						Width:   16,
						Height:  16,
					},
					vmprotocol.GraphicsCommand{
						Kind:    vmprotocol.GraphicsCommandKindNewImage,
						ImageID: 1,
						Width:   32,
						Height:  32,
					},
				)
			},
		},
		{
			name: "SetVertices with a ragged vertex buffer",
			script: func(fg *fakeGuest) {
				fg.sendCommands(vmprotocol.GraphicsCommand{
					Kind:     vmprotocol.GraphicsCommandKindSetVertices,
					Vertices: make([]float32, graphics.VertexFloatCount+1),
					Indices:  []uint32{0, 0, 1},
				})
			},
		},
		{
			name: "WritePixels with fewer pixel buffers than regions",
			script: func(fg *fakeGuest) {
				fg.sendCommands(
					vmprotocol.GraphicsCommand{
						Kind:    vmprotocol.GraphicsCommandKindNewImage,
						ImageID: 1,
						Width:   16,
						Height:  16,
					},
					vmprotocol.GraphicsCommand{
						Kind:    vmprotocol.GraphicsCommandKindWritePixels,
						ImageID: 1,
						Regions: []image.Rectangle{image.Rect(0, 0, 4, 4)},
					})
			},
		},
		{
			name: "WritePixels with a pixel buffer the region does not fit",
			script: func(fg *fakeGuest) {
				fg.sendCommands(
					vmprotocol.GraphicsCommand{
						Kind:    vmprotocol.GraphicsCommandKindNewImage,
						ImageID: 1,
						Width:   16,
						Height:  16,
					},
					vmprotocol.GraphicsCommand{
						Kind:    vmprotocol.GraphicsCommandKindWritePixels,
						ImageID: 1,
						Regions: []image.Rectangle{image.Rect(0, 0, 4, 4)},
						Pixels:  [][]byte{make([]byte, 3)},
					})
			},
		},
		{
			name: "WritePixels outside the image",
			script: func(fg *fakeGuest) {
				fg.sendCommands(
					vmprotocol.GraphicsCommand{
						Kind:    vmprotocol.GraphicsCommandKindNewImage,
						ImageID: 1,
						Width:   16,
						Height:  16,
					},
					vmprotocol.GraphicsCommand{
						Kind:    vmprotocol.GraphicsCommandKindWritePixels,
						ImageID: 1,
						Regions: []image.Rectangle{image.Rect(100, 100, 104, 104)},
						Pixels:  [][]byte{make([]byte, 4*4*4)},
					})
			},
		},
		{
			name: "DrawTriangles before any SetVertices",
			script: func(fg *fakeGuest) {
				fg.sendCommands(
					vmprotocol.GraphicsCommand{
						Kind:    vmprotocol.GraphicsCommandKindNewImage,
						ImageID: 1,
						Width:   16,
						Height:  16,
					},
					vmprotocol.GraphicsCommand{
						Kind:         vmprotocol.GraphicsCommandKindNewShader,
						ShaderID:     1,
						ShaderSource: fillShaderSource,
					},
					drawTriangles(3, 0))
			},
		},
		{
			name: "DrawTriangles starting outside the indices",
			script: func(fg *fakeGuest) {
				fg.sendCommands(append(drawPrelude(4, []uint32{0, 1, 2}), drawTriangles(3, 100))...)
			},
		},
		{
			name: "DrawTriangles taking a negative number of indices",
			script: func(fg *fakeGuest) {
				fg.sendCommands(append(drawPrelude(4, []uint32{0, 1, 2}), drawTriangles(-1, 0))...)
			},
		},
		{
			name: "DrawTriangles taking more indices than were set",
			script: func(fg *fakeGuest) {
				fg.sendCommands(append(drawPrelude(4, []uint32{0, 1, 2}), drawTriangles(9, 0))...)
			},
		},
		{
			name: "DrawTriangles referencing a vertex that was not set",
			script: func(fg *fakeGuest) {
				fg.sendCommands(append(drawPrelude(4, []uint32{0, 1, 9999}), drawTriangles(3, 0))...)
			},
		},
		{
			name: "DrawTriangles using its dst as a src",
			script: func(fg *fakeGuest) {
				cmd := drawTriangles(3, 0)
				cmd.Srcs[0] = 1
				fg.sendCommands(append(drawPrelude(4, []uint32{0, 1, 2}), cmd)...)
			},
		},
		{
			name: "DrawTriangles with uniforms the shader does not take",
			script: func(fg *fakeGuest) {
				cmd := drawTriangles(3, 0)
				cmd.Uniforms = append(cmd.Uniforms, 1, 2, 3)
				fg.sendCommands(append(drawPrelude(4, []uint32{0, 1, 2}), cmd)...)
			},
		},
		{
			name: "an unknown guest message kind",
			script: func(fg *fakeGuest) {
				fg.send(&vmprotocol.GuestMessage{Kind: vmprotocol.GuestMessageKind(9999)})
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			guest := newFakeGuestSession(t, tc.script)
			setOutsideScreen(t, guest)
			guest.AdvanceTicks(1)
			guest.WaitTicks()
			if guest.Err() == nil {
				t.Error("the session did not fail; want an error")
			}
		})
	}
}

func TestIllFormedReadPixelsQueryIsAnsweredWithAnError(t *testing.T) {
	cases := []struct {
		name    string
		regions []image.Rectangle
	}{
		{
			// image.Rect canonicalizes, but the wire does not, so a guest can send Max before Min.
			name:    "a region running backwards",
			regions: []image.Rectangle{{Min: image.Pt(0, 0), Max: image.Pt(-4, 4)}},
		},
		{
			name:    "an empty region",
			regions: []image.Rectangle{image.Rect(4, 4, 4, 4)},
		},
		{
			name:    "a region outside the image",
			regions: []image.Rectangle{image.Rect(100, 100, 104, 104)},
		},
		{
			name: "regions asking for more pixels than the image holds",
			regions: []image.Rectangle{
				image.Rect(0, 0, 16, 16),
				image.Rect(0, 0, 16, 16),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			answers := make(chan string, 1)
			guest := newFakeGuestSession(t, func(fg *fakeGuest) {
				fg.sendCommands(vmprotocol.GraphicsCommand{
					Kind:    vmprotocol.GraphicsCommandKindNewImage,
					ImageID: 1,
					Width:   16,
					Height:  16,
				})
				fg.send(&vmprotocol.GuestMessage{
					Kind:        vmprotocol.GuestMessageKindQueryReadPixels,
					ReadImageID: 1,
					ReadRegions: tc.regions,
				})
				answer, err := fg.receive()
				if err != nil {
					return
				}
				answers <- answer.Err
			})
			setOutsideScreen(t, guest)
			guest.AdvanceTicks(1)
			// A query the host cannot serve is reported to the guest, leaving the session usable.
			if !guest.WaitTicks() {
				t.Fatalf("advancing the guest failed: %v", guest.Err())
			}
			if err := <-answers; err == "" {
				t.Error("the host answered the query without an error; want a failure")
			}
		})
	}
}

// TestDisposingTheScreenMirrorDropsTheFrame asserts that a guest disposing its screen framebuffer
// leaves the host with no frame, rather than compositing the mirror it just released.
func TestDisposingTheScreenMirrorDropsTheFrame(t *testing.T) {
	guest := newFakeGuestSession(t, func(fg *fakeGuest) {
		fg.sendCommands(
			vmprotocol.GraphicsCommand{
				Kind:    vmprotocol.GraphicsCommandKindNewScreenFramebufferImage,
				ImageID: 1,
				Width:   320,
				Height:  240,
				Screen:  true,
			},
			vmprotocol.GraphicsCommand{
				Kind:    vmprotocol.GraphicsCommandKindDisposeImage,
				ImageID: 1,
			},
		)
	})
	setOutsideScreen(t, guest)
	guest.AdvanceTicks(1)
	guest.WaitTicks()
	guest.AdvanceFrame()
	if guest.WaitFrame() {
		t.Error("a frame was ready after the guest disposed its screen mirror; want none")
	}
	if guest.Err() == nil {
		t.Error("the session did not fail; want an error")
	}
}

// TestGuestQueryBeforeTheHostGameStarts asserts that a query arriving before the host's game runs
// fails the session rather than panicking the host, whose graphics driver does not exist yet. The
// host under test is a separate process, as this test's own host is already running.
func TestGuestQueryBeforeTheHostGameStarts(t *testing.T) {
	skipIfVMUnsupported(t)

	// The fixture is a host rather than a guest, so it is built without the guest's build tag.
	hostBin := buildGuest(t, "./testdata/prerungame", activateByOptions)
	if out, err := exec.Command(hostBin).CombinedOutput(); err != nil {
		t.Errorf("the host process failed: %v\n%s", err, out)
	}
}
