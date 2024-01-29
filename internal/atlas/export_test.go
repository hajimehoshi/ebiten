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
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

const (
	BaseCountToPutOnSourceBackend = baseCountToPutOnSourceBackend
)

func PutImagesOnSourceBackendForTesting(graphicsDriver graphicsdriver.Graphics) {
	putImagesOnSourceBackend(graphicsDriver)
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

var FlushDeferredForTesting = flushDeferred

var FloorPowerOf2 = floorPowerOf2

func DeferredFuncCountForTesting() int {
	deferredM.Lock()
	defer deferredM.Unlock()
	return len(deferred)
}
