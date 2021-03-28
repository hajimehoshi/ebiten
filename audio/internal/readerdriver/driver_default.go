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

// +build !android
// +build !js

package readerdriver

import (
	"fmt"
	"runtime"
)

type context struct {
	sampleRate      int
	channelNum      int
	bitDepthInBytes int
}

func IsAvailable() bool {
	return false
}

func NewContext(sampleRate int, channelNum int, bitDepthInBytes int) (Context, chan struct{}, error) {
	panic(fmt.Sprintf("readerdriver: NewContext is not available on this environment: GOOS=%s", runtime.GOOS))
}
