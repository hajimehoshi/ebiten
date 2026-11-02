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
	"context"
	"errors"
	"fmt"
	"image"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/thread"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
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

var errFlushForTesting = errors.New("graphicscommand: an error for testing")

// failingCommand is a command that fails and needs no synchronization.
type failingCommand struct{}

func (*failingCommand) Exec(commandQueue *graphicscommand.CommandQueueForTesting, graphicsDriver graphicsdriver.Graphics, indexOffset int) error {
	return errFlushForTesting
}

func (*failingCommand) NeedsSync() bool {
	return false
}

func (*failingCommand) String() string {
	return "failing"
}

type asyncFlushDriver struct {
	graphicsdriver.Graphics
	// ends is incremented on the render thread.
	ends atomic.Int32
}

func (*asyncFlushDriver) Begin() error {
	return nil
}

func (g *asyncFlushDriver) End(mode graphicsdriver.FlushMode) error {
	g.ends.Add(1)
	return nil
}

func (*asyncFlushDriver) SetVsyncEnabled(enabled bool) {
}

func TestAsynchronousFlushError(t *testing.T) {
	// A flush is asynchronous only when vsync is disabled.
	graphicscommand.SetVsyncEnabled(false)
	defer graphicscommand.SetVsyncEnabled(true)

	var manager graphicscommand.CommandQueueManagerForTesting
	var driver asyncFlushDriver

	manager.EnqueueCommandForTesting(&failingCommand{})
	err := manager.FlushForTesting(&driver, graphicsdriver.FlushModePresent)
	if err == nil {
		// An asynchronous flush reports its error at the next flush at the latest.
		err = manager.FlushForTesting(&driver, graphicsdriver.FlushModePresent)
	}
	if !errors.Is(err, errFlushForTesting) {
		t.Errorf("the error of an asynchronous flush was not reported: got %v, want %v", err, errFlushForTesting)
	}
	if driver.ends.Load() == 0 {
		t.Errorf("End was not called for a failing flush")
	}
}

func TestFlushStoppedNoopThread(t *testing.T) {
	renderThread := thread.NewNoopThread()
	done := make(chan struct{})
	close(done)
	renderThread.LoopAndStop(done)
	defer graphicscommand.SetRenderThreadForTesting(renderThread)()
	defer graphicscommand.SetVsyncEnabled(true)
	for _, vsync := range []bool{false, true} {
		graphicscommand.SetVsyncEnabled(vsync)
		var q graphicscommand.CommandQueueForTesting
		var manager graphicscommand.CommandQueueManagerForTesting
		q.Enqueue(&failingCommand{})
		q.AddFinalizerForTesting(func() { t.Error("finalizer executed after stopping") })
		// A stopped thread must not access the nil graphics driver.
		if err := q.Flush(&manager, nil, graphicsdriver.FlushModePresent); err != nil {
			t.Error(err)
		}
	}
}

type resourceCommand struct {
	finalize func()
}

func (c *resourceCommand) Exec(q *graphicscommand.CommandQueueForTesting, _ graphicsdriver.Graphics, _ int) error {
	q.AddFinalizerForTesting(c.finalize)
	return nil
}

func (*resourceCommand) NeedsSync() bool { return false }
func (*resourceCommand) String() string  { return "resource" }

func (*frameDriver) SetVsyncEnabled(bool) {}

// Issue #3814
func TestCompleteFramesAfterIntermediateFlushes(t *testing.T) {
	for _, threaded := range []bool{false, true} {
		t.Run(fmt.Sprintf("threaded=%t", threaded), func(t *testing.T) {
			var renderThread thread.Thread
			if threaded {
				renderThread = thread.NewOSThread()
				ctx, cancel := context.WithCancel(context.Background())
				done := make(chan struct{})
				go func() {
					defer close(done)
					renderThread.LoopAndStop(ctx.Done())
				}()
				defer func() { cancel(); <-done }()
			} else {
				renderThread = thread.NewNoopThread()
			}
			defer graphicscommand.SetRenderThreadForTesting(renderThread)()
			graphicscommand.SetVsyncEnabled(false)
			defer graphicscommand.SetVsyncEnabled(true)
			for _, mode := range []graphicsdriver.FlushMode{graphicsdriver.FlushModeEndFrame, graphicsdriver.FlushModePresent} {
				for _, intermediate := range []int{0, 1, 2, 3} {
					t.Run(fmt.Sprintf("mode=%d/intermediate=%d", mode, intermediate), func(t *testing.T) {
						var manager graphicscommand.CommandQueueManagerForTesting
						var driver frameDriver
						var finalized int
						for frame := range 5 {
							for range intermediate {
								manager.EnqueueCommandForTesting(&resourceCommand{
									finalize: func() {
										if driver.frames != frame+1 {
											t.Errorf("finalizer ran before frame completion: completed frames = %d, want %d", driver.frames, frame+1)
										}
										finalized++
									},
								})
								if err := manager.FlushForTesting(&driver, graphicsdriver.FlushModeIntermediate); err != nil {
									t.Fatal(err)
								}
							}
							// Synchronize with the render thread before inspecting callback results.
							thread.Call(renderThread, func() {})
							if want := frame * intermediate; finalized != want {
								t.Errorf("after intermediate flushes: finalized = %d, want %d", finalized, want)
							}
							// No commands remain for the frame-completion flush.
							if err := manager.FlushForTesting(&driver, mode); err != nil {
								t.Fatal(err)
							}
							thread.Call(renderThread, func() {})
							if want := (frame + 1) * intermediate; finalized != want {
								t.Errorf("after frame completion: finalized = %d, want %d", finalized, want)
							}
						}
					})
				}
			}
		})
	}
}

type capturedDraw struct {
	indexOffset int
	indexCount  int
	blend       graphicsdriver.Blend
}

type vertexCaptureDriver struct {
	frameDriver
	vertices []float32
	indices  []uint32
	uploads  int
	draws    []capturedDraw
}

func (g *vertexCaptureDriver) SetVertices(vertices []float32, indices []uint32) error {
	g.uploads++
	g.vertices = append(g.vertices, vertices...)
	g.indices = append(g.indices, indices...)
	return nil
}

func (g *vertexCaptureDriver) DrawTriangles(_ graphicsdriver.ImageID, _ [graphics.ShaderSrcImageCount]graphicsdriver.ImageID, _ graphicsdriver.ShaderID, regions []graphicsdriver.DstRegion, indexOffset int, blend graphicsdriver.Blend, _ []uint32) error {
	var indexCount int
	for _, region := range regions {
		indexCount += region.IndexCount
	}
	g.draws = append(g.draws, capturedDraw{
		indexOffset: indexOffset,
		indexCount:  indexCount,
		blend:       blend,
	})
	return nil
}

func TestDrawBatchVertexData(t *testing.T) {
	// Presentation flushes must finish before the captured data is inspected.
	graphicscommand.SetVsyncEnabled(true)
	dst := graphicscommand.NewImage(16, 16, false, "")
	defer dst.Dispose()
	src := graphicscommand.NewImage(16, 16, false, "")
	defer src.Dispose()
	if err := graphicscommand.FlushCommands(ui.Get().GraphicsDriverForTesting(), graphicsdriver.FlushModeIntermediate); err != nil {
		t.Fatal(err)
	}

	var manager graphicscommand.CommandQueueManagerForTesting
	var driver vertexCaptureDriver
	enqueue := func(vertices []float32, indices []uint32, blend graphicsdriver.Blend) {
		graphicscommand.EnqueueDrawTrianglesForTesting(&manager, dst, [graphics.ShaderSrcImageCount]*graphicscommand.Image{src}, vertices, indices, blend, image.Rect(0, 0, 16, 16), [graphics.ShaderSrcImageCount]image.Rectangle{image.Rect(0, 0, 16, 16)}, nearestFilterShader, nil)
	}
	// Return a large and a small queue to the pool through normal flushes.
	warmVertices := make([]float32, 8192*graphics.VertexFloatCount)
	for i := range warmVertices {
		warmVertices[i] = -float32(i + 1)
	}
	enqueue(warmVertices, []uint32{0, 1, 2}, graphicsdriver.BlendSourceOver)
	if err := manager.FlushForTesting(&driver, graphicsdriver.FlushModeIntermediate); err != nil {
		t.Fatal(err)
	}
	enqueue(warmVertices[:12*graphics.VertexFloatCount], []uint32{0, 1, 2}, graphicsdriver.BlendSourceOver)
	if err := manager.FlushForTesting(&driver, graphicsdriver.FlushModePresent); err != nil {
		t.Fatal(err)
	}
	driver.vertices = driver.vertices[:0]
	driver.indices = driver.indices[:0]
	driver.draws = driver.draws[:0]
	driver.uploads = 0
	var wantVertices []float32
	for i, vertexCount := range []int{3, 6, 4096} {
		vertices := make([]float32, vertexCount*graphics.VertexFloatCount)
		for j := range vertices {
			vertices[j] = float32(len(wantVertices) + j)
		}
		wantVertices = append(wantVertices, vertices...)
		indices := []uint32{0, 1, 2}
		if i == 1 {
			indices = []uint32{0, 1, 2, 3, 4, 5}
		}
		blend := graphicsdriver.BlendSourceOver
		if i == 2 {
			// Grow the vertex buffer after a merged command, then start a new batch.
			indices = []uint32{0, 2047, 4095}
			blend = graphicsdriver.BlendCopy
		}
		enqueue(vertices, indices, blend)
	}
	if err := manager.FlushForTesting(&driver, graphicsdriver.FlushModePresent); err != nil {
		t.Fatal(err)
	}
	if driver.uploads != 1 {
		t.Errorf("vertex uploads = %d, want 1", driver.uploads)
	}
	if !slices.Equal(driver.vertices, wantVertices) {
		t.Error("uploaded vertices differ from queued data")
	}
	if want := []uint32{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 2056, 4104}; !slices.Equal(driver.indices, want) {
		t.Errorf("uploaded indices = %v, want %v", driver.indices, want)
	}
	wantDraws := []capturedDraw{
		{
			indexCount: 9,
			blend:      graphicsdriver.BlendSourceOver,
		},
		{
			indexOffset: 9,
			indexCount:  3,
			blend:       graphicsdriver.BlendCopy,
		},
	}
	if !slices.Equal(driver.draws, wantDraws) {
		t.Errorf("draws = %v, want %v", driver.draws, wantDraws)
	}
}
