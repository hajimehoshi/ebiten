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

package audio_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

func TestPositionBeforeContextReady(t *testing.T) {
	// A device can exist before it is ready, e.g. while Web Audio waits for a user gesture.
	p, err := audio.NewPlayerF32WithUnreadyContextForTesting(bytes.NewReader(make([]byte, 44100*8)), 44100)
	if err != nil {
		t.Fatal(err)
	}
	const position = 500 * time.Millisecond
	if err := p.SetPosition(position); err != nil {
		t.Fatal(err)
	}
	if err := audio.EnsurePlayerForTesting(p); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		audio.UpdatePositionForTesting(p)
		if got := p.Position(); got != position {
			t.Errorf("Position before readiness: got %v, want %v", got, position)
		}
	}
	audio.SetContextReadyForTesting(p)
	audio.UpdatePositionForTesting(p)
	if got := p.Position(); got != position {
		t.Errorf("Position after readiness: got %v, want %v", got, position)
	}
}
