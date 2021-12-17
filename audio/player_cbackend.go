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

//go:build ebitencbackend
// +build ebitencbackend

package audio

import (
	"io"

	"github.com/hajimehoshi/ebiten/v2/audio/internal/cbackend"
)

func newContext(sampleRate, channelNum, bitDepthInBytes int) (context, chan struct{}, error) {
	ctx, ready, err := cbackend.NewContext(sampleRate, channelNum, bitDepthInBytes)
	return &contextProxy{ctx}, ready, err
}

// contextProxy is a proxy between cbackend.Context and context.
type contextProxy struct {
	*cbackend.Context
}

// NewPlayer implements context.
func (c *contextProxy) NewPlayer(r io.Reader) player {
	return c.Context.NewPlayer(r)
}
