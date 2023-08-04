// Copyright 2023 The Ebitengine Authors
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

package graphicscommand

type temporaryBytes struct {
	pixels           []byte
	pos              int
	notFullyUsedTime int
}

func temporaryBytesSize(size int) int {
	l := 16
	for l < size {
		l *= 2
	}
	return l
}

// alloc allocates the pixels and returns it.
//
// Be careful that the returned pixels might not be zero-cleared.
func (t *temporaryBytes) alloc(size int) []byte {
	if len(t.pixels) < t.pos+size {
		t.pixels = make([]byte, max(len(t.pixels)*2, temporaryBytesSize(size)))
		t.pos = 0
	}
	pix := t.pixels[t.pos : t.pos+size]
	t.pos += size
	return pix
}

func (t *temporaryBytes) reset() {
	// reset is called in a render thread.
	// When reset is called, a queue is being flushed in a render thread, and the queue is never used in the game thread.
	// Thus, a mutex lock is not needed in alloc and reset.

	const maxNotFullyUsedTime = 60

	if temporaryBytesSize(t.pos) < len(t.pixels) {
		if t.notFullyUsedTime < maxNotFullyUsedTime {
			t.notFullyUsedTime++
		}
	} else {
		t.notFullyUsedTime = 0
	}

	// Let the pixels GCed if this is not used for a while.
	if t.notFullyUsedTime == maxNotFullyUsedTime && len(t.pixels) > 0 {
		t.pixels = nil
		t.notFullyUsedTime = 0
	}

	// Reset the position and reuse the allocated bytes.
	// t.pixels should already be sent to GPU, then this can be reused.
	t.pos = 0
}

// AllocBytes allocates bytes from the cache.
//
// Be careful that the returned pixels might not be zero-cleared.
func AllocBytes(size int) []byte {
	return currentCommandQueue().temporaryBytes.alloc(size)
}
