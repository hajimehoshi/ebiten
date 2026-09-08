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

package graphicscommand

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2/internal/debug"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

type WritePixelsCommandArgs = writePixelsCommandArgs

func (i *Image) BufferedWritePixelsArgsForTesting() []WritePixelsCommandArgs {
	return i.bufferedWritePixelsArgs
}

func PrependPreservedUniforms(uniforms []uint32, shader *Shader, dst *Image, srcs [graphics.ShaderSrcImageCount]*Image, dstRegion image.Rectangle, srcRegions [graphics.ShaderSrcImageCount]image.Rectangle) []uint32 {
	return prependPreservedUniforms(uniforms, shader, dst, srcs, dstRegion, srcRegions)
}

type CommandQueueForTesting = commandQueue

func (q *commandQueue) FlushForTesting(driver graphicsdriver.Graphics, mode graphicsdriver.FlushMode) error {
	return q.flush(driver, mode, debug.SwitchFrameLogger())
}

func (q *commandQueue) AddFinalizerForTesting(f func()) {
	q.addFinalizer(f)
}

func (q *commandQueue) AllocUniformsForTesting(n int) {
	q.uint32sBuffer.alloc(n)
}

func (q *commandQueue) PendingResourcesForTesting() (uniforms, finalizers int) {
	return len(q.uint32sBuffer.buf), len(q.finalizers)
}
