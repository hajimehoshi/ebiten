// Copyright 2024 The Ebitengine Authors
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
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/thread"
)

func BenchmarkPrependPreservedUniforms(b *testing.B) {
	var uniforms [graphics.PreservedUniformDwordCount]uint32
	dst := graphicscommand.NewImage(16, 16, false, "")
	src := graphicscommand.NewImage(16, 16, false, "")
	dr := image.Rect(0, 0, 16, 16)
	sr := image.Rect(0, 0, 16, 16)
	for i := 0; i < b.N; i++ {
		graphicscommand.PrependPreservedUniforms(uniforms[:], nearestFilterShader, dst, [graphics.ShaderSrcImageCount]*graphicscommand.Image{src}, dr, [graphics.ShaderSrcImageCount]image.Rectangle{sr})
	}
}

func BenchmarkFlushThread(b *testing.B) {
	defer graphicscommand.SetVsyncEnabled(true)
	for _, name := range []string{"NoopThread", "OSThread"} {
		b.Run(name, func(b *testing.B) {
			var renderThread thread.Thread = thread.NewNoopThread()
			if name == "OSThread" {
				renderThread = thread.NewOSThread()
				ctx, cancel := context.WithCancel(context.Background())
				done := make(chan struct{})
				go func() {
					defer close(done)
					_ = renderThread.LoopAndStop(ctx)
				}()
				defer func() { cancel(); <-done }()
			}
			defer graphicscommand.SetRenderThreadForTesting(renderThread)()
			for _, vsync := range []bool{false, true} {
				name := "Async"
				if vsync {
					name = "Sync"
				}
				b.Run(name, func(b *testing.B) {
					graphicscommand.SetVsyncEnabled(vsync)
					var manager graphicscommand.CommandQueueManagerForTesting
					var driver asyncFlushDriver
					// Warm the queue pool and apply the initial vsync state.
					for range 3 {
						if err := manager.FlushForTesting(&driver, graphicsdriver.FlushModePresent); err != nil {
							b.Fatal(err)
						}
					}
					thread.Call(renderThread, func() {})
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						if err := manager.FlushForTesting(&driver, graphicsdriver.FlushModePresent); err != nil {
							b.Fatal(err)
						}
					}
					b.StopTimer()
					thread.Call(renderThread, func() {})
				})
			}
		})
	}
}
