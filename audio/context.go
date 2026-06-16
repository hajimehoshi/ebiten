// Copyright 2021 The Ebiten Authors
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

package audio

import (
	"io"

	"github.com/ebitengine/oto/v3"

	"github.com/hajimehoshi/ebiten/v2/audio/internal/vmaudio"
)

func newContext(sampleRate int, vmGuest bool) (context, chan struct{}, error) {
	// A virtualization guest plays no audio of its own: its audio goes to a virtual device the host
	// reads, instead of a real device.
	if vmGuest {
		c := vmaudio.NewContext(sampleRate)
		// The virtual device needs no asynchronous initialization and is ready from the start.
		ready := make(chan struct{})
		close(ready)
		return &vmaudioContext{c}, ready, nil
	}

	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   sampleRate,
		ChannelCount: channelCount,
		Format:       oto.FormatFloat32LE,
	})
	err = addErrorInfo(err)
	return &otoContext{ctx}, ready, err
}

// otoContext is a proxy between oto.Context and context.
type otoContext struct {
	*oto.Context
}

// NewPlayer implements context.
func (c *otoContext) NewPlayer(r io.Reader) player {
	return c.Context.NewPlayer(r)
}

// vmaudioContext is a proxy between vmaudio.Context and context.
type vmaudioContext struct {
	*vmaudio.Context
}

// NewPlayer implements context.
func (c *vmaudioContext) NewPlayer(r io.Reader) player {
	return c.Context.NewPlayer(r)
}
