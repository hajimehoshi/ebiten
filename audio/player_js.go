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

//go:build !ebitencbackend
// +build !ebitencbackend

package audio

import (
	"syscall/js"

	"github.com/hajimehoshi/oto/v2"

	"github.com/hajimehoshi/ebiten/v2/audio/internal/go2cpp"
)

func newContext(sampleRate, channelNum, bitDepthInBytes int) (context, chan struct{}, error) {
	if js.Global().Get("go2cpp").Truthy() {
		ready := make(chan struct{})
		close(ready)
		return otoContextToContext(go2cpp.NewContext(sampleRate, channelNum, bitDepthInBytes)), ready, nil
	}

	ctx, ready, err := oto.NewContext(sampleRate, channelNum, bitDepthInBytes)
	return otoContextToContext(ctx), ready, err
}
