// Copyright 2019 The Ebiten Authors
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

package atlas

import (
	"weak"

	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

const (
	BaseCountToPutOnSourceBackend = baseCountToPutOnSourceBackend
)

func PutImagesOnSourceBackendForTesting() {
	putImagesOnSourceBackend()
}

var (
	oldMinSourceSize      int
	oldMinDestinationSize int
	oldMaxSize            int
)

func SetImageSizeForTesting(minSource, minDestination, max int) {
	oldMinSourceSize = minSourceSize
	oldMinDestinationSize = minDestinationSize
	oldMaxSize = maxSize

	minSourceSize = minSource
	minDestinationSize = minDestination
	maxSize = max
}

func ResetImageSizeForTesting() {
	minSourceSize = oldMinSourceSize
	minDestinationSize = oldMinDestinationSize
	maxSize = oldMaxSize
}

func (i *Image) PaddingSizeForTesting() int {
	return i.paddingSize()
}

func (i *Image) BackendSizeForTesting() (int, int) {
	backendsM.Lock()
	defer backendsM.Unlock()
	if i.backend == nil {
		return 0, 0
	}
	return i.backend.width, i.backend.height
}

func (i *Image) IsOnSourceBackendForTesting() bool {
	backendsM.Lock()
	defer backendsM.Unlock()
	return i.isOnSourceBackend()
}

func (i *Image) EnsureIsolatedFromSourceForTesting(backends []*backend) {
	backendsM.Lock()
	defer backendsM.Unlock()
	i.ensureIsolatedFromSource(backends)
}

func FlushDeferredForTesting() {
	backendsM.Lock()
	defer backendsM.Unlock()
	flushDeferred()
}

func (i *Image) IsDeallocatedFuncForTesting() func() bool {
	impl := i.imageImpl
	return func() bool {
		backendsM.Lock()
		defer backendsM.Unlock()
		return impl.backend == nil
	}
}

var FloorPowerOf2 = floorPowerOf2

func BackendCountForTesting() int {
	backendsM.Lock()
	defer backendsM.Unlock()
	return len(theBackends)
}

func ClearShaderForTesting() *Shader {
	return clearShader
}

func (s *Shader) SourceIDForTesting() shaderir.SourceID {
	return s.ir.SourceID
}

func (s *Shader) IsRegisteredFuncForTesting() func() bool {
	shader := weak.Make(s)
	return func() bool {
		theShadersWithInternalShader.m.Lock()
		defer theShadersWithInternalShader.m.Unlock()
		_, ok := theShadersWithInternalShader.shaders[shader]
		return ok
	}
}

type GPUResourcesState = gpuResourcesState

func (a *gpuResourcesState) RequestToSaveGPUResources() bool {
	return a.requestToSaveGPUResources()
}

func (a *gpuResourcesState) FinishSavingGPUResources(succeeded bool) bool {
	return a.finishSavingGPUResources(succeeded)
}

func (a *gpuResourcesState) AreGPUResourcesSaved() bool {
	return a.areGPUResourcesSaved()
}

func (a *gpuResourcesState) RequestToRestoreGPUResources() bool {
	return a.requestToRestoreGPUResources()
}

func (a *gpuResourcesState) StartRestoringGPUResourcesIfNeeded() bool {
	return a.startRestoringGPUResourcesIfNeeded()
}
