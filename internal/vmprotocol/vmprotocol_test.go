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

package vmprotocol_test

import (
	"bytes"
	"encoding/binary"
	"image"
	"reflect"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/vmprotocol"
)

func TestGuestMessageRoundTrip(t *testing.T) {
	// The vertices are more than the codec converts at once.
	vertices := make([]float32, 100000)
	for i := range vertices {
		vertices[i] = float32(i) / 3
	}
	indices := make([]uint32, 30000)
	for i := range indices {
		indices[i] = uint32(i) * 0x10001
	}

	msgs := []vmprotocol.GuestMessage{
		{
			Kind: vmprotocol.GuestMessageKindGraphicsCommands,
			GraphicsCommands: []vmprotocol.GraphicsCommand{
				{
					Kind:     vmprotocol.GraphicsCommandKindSetVertices,
					Vertices: vertices,
					Indices:  indices,
				},
				{
					Kind:    vmprotocol.GraphicsCommandKindNewImage,
					ImageID: 1,
					Width:   2,
					Height:  1,
				},
				{
					Kind:    vmprotocol.GraphicsCommandKindWritePixels,
					ImageID: 1,
					Regions: []image.Rectangle{image.Rect(0, 0, 1, 1), image.Rect(1, 0, 2, 1)},
					Pixels:  [][]byte{{1, 2, 3, 4}, {5, 6, 7, 8}},
				},
			},
		},
		{
			Kind: vmprotocol.GuestMessageKindDone,
			Err:  "foo",
		},
	}

	var buf bytes.Buffer
	enc := vmprotocol.NewEncoder(&buf)
	for i := range msgs {
		if err := enc.EncodeGuestMessage(&msgs[i]); err != nil {
			t.Fatal(err)
		}
	}

	dec := vmprotocol.NewDecoder(&buf)
	for i, want := range msgs {
		var got vmprotocol.GuestMessage
		if err := dec.DecodeGuestMessage(&got); err != nil {
			t.Fatalf("message %d: %v", i, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("message %d was not decoded as it was encoded", i)
		}
	}
}

func TestHostMessageRoundTrip(t *testing.T) {
	msgs := []vmprotocol.HostMessage{
		{
			Kind:   vmprotocol.HostMessageKindAnswerReadPixelsAsync,
			Pixels: [][]byte{{1, 2, 3, 4}, {5, 6, 7, 8, 9, 10, 11, 12}},
		},
		{
			Kind: vmprotocol.HostMessageKindAdvanceFrame,
		},
	}

	var buf bytes.Buffer
	enc := vmprotocol.NewEncoder(&buf)
	for i := range msgs {
		if err := enc.EncodeHostMessage(&msgs[i]); err != nil {
			t.Fatal(err)
		}
	}

	dec := vmprotocol.NewDecoder(&buf)
	for i, want := range msgs {
		var got vmprotocol.HostMessage
		if err := dec.DecodeHostMessage(&got); err != nil {
			t.Fatalf("message %d: %v", i, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("message %d: got: %+v, want: %+v", i, got, want)
		}
	}
}

func TestDecodeTooLargeBulkPayload(t *testing.T) {
	var buf bytes.Buffer
	if err := vmprotocol.NewEncoder(&buf).EncodeGuestMessage(&vmprotocol.GuestMessage{
		Kind: vmprotocol.GuestMessageKindGraphicsCommands,
		GraphicsCommands: []vmprotocol.GraphicsCommand{
			{
				Kind: vmprotocol.GraphicsCommandKindBegin,
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	// The command has no bulk payloads, so the message ends with their three zero counts. Replace them
	// with a count that no stream can back.
	b := binary.AppendUvarint(buf.Bytes()[:buf.Len()-3], 1<<62)

	var msg vmprotocol.GuestMessage
	if err := vmprotocol.NewDecoder(bytes.NewReader(b)).DecodeGuestMessage(&msg); err == nil {
		t.Error("decoding a message with an oversized bulk payload succeeded")
	}
}
