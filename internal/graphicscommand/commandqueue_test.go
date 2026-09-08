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

package graphicscommand_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

type frameDriver struct {
	graphicsdriver.Graphics
	frames   int
	presents int
}

func (*frameDriver) Begin() error {
	return nil
}

func (g *frameDriver) End(mode graphicsdriver.FlushMode) error {
	if mode != graphicsdriver.FlushModeIntermediate {
		g.frames++
	}
	if mode == graphicsdriver.FlushModePresent {
		g.presents++
	}
	return nil
}

func TestCompleteFramesWithoutPresenting(t *testing.T) {
	var q graphicscommand.CommandQueueForTesting
	var driver frameDriver
	var finalized int
	const frames = 10
	for frame := range frames {
		q.AllocUniformsForTesting(16)
		q.AddFinalizerForTesting(func() { finalized++ })
		if err := q.FlushForTesting(&driver, graphicsdriver.FlushModeIntermediate); err != nil {
			t.Fatal(err)
		}
		if finalized != frame {
			t.Errorf("frame %d after intermediate flush: finalized = %d, want %d", frame, finalized, frame)
		}
		if err := q.FlushForTesting(&driver, graphicsdriver.FlushModeEndFrame); err != nil {
			t.Fatal(err)
		}
		if finalized != frame+1 {
			t.Errorf("frame %d after completion: finalized = %d, want %d", frame, finalized, frame+1)
		}
		if uniforms, finalizers := q.PendingResourcesForTesting(); uniforms != 0 || finalizers != 0 {
			t.Errorf("frame %d after completion: pending uniforms = %d, finalizers = %d; want 0, 0", frame, uniforms, finalizers)
		}
	}
	if driver.frames != frames || driver.presents != 0 {
		t.Errorf("after non-presented frames: frames = %d, presents = %d; want %d, 0", driver.frames, driver.presents, frames)
	}
	if err := q.FlushForTesting(&driver, graphicsdriver.FlushModePresent); err != nil {
		t.Fatal(err)
	}
	if driver.frames != frames+1 || driver.presents != 1 {
		t.Errorf("after presentation resumes: frames = %d, presents = %d; want %d, 1", driver.frames, driver.presents, frames+1)
	}
}
