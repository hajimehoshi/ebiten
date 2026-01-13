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

package atlas

import (
	"fmt"
	"log/slog"
	"sync"
)

type gpuResourcesStatePhase int

const (
	gpuResourcesStatePhaseNone gpuResourcesStatePhase = iota
	gpuResourcesStatePhaseSaveRequested
	gpuResourcesStatePhaseSaved
	gpuResourcesStatePhaseRestoreRequested
)

func (g gpuResourcesStatePhase) String() string {
	switch g {
	case gpuResourcesStatePhaseNone:
		return "None"
	case gpuResourcesStatePhaseSaveRequested:
		return "SaveRequested"
	case gpuResourcesStatePhaseSaved:
		return "Saved"
	case gpuResourcesStatePhaseRestoreRequested:
		return "RestoreRequested"
	default:
		return fmt.Sprintf("gpuResourcesStatePhase(%d)", int(g))
	}
}

// gpuResourcesState represents the state of GPU resources especially for Android.
//
// Any operations on gpuResourcesState are not blocked by other atlas operations.
// This matters for mobile platforms where the main thread must not be blocked for a long time.
type gpuResourcesState struct {
	phase gpuResourcesStatePhase

	m sync.Mutex
}

var theGPUResourcesState gpuResourcesState

// requestToSaveGPUResources requests to save GPU resources.
//
// requestToSaveGPUResources is invoked from a different thread from the rendering thread.
func (a *gpuResourcesState) requestToSaveGPUResources() bool {
	a.m.Lock()
	defer a.m.Unlock()

	slog.Debug("atlas: requestToSaveGPUResources was called", "phase", a.phase)

	origPhase := a.phase
	switch a.phase {
	case gpuResourcesStatePhaseNone:
		a.phase = gpuResourcesStatePhaseSaveRequested
		return true
	case gpuResourcesStatePhaseSaveRequested:
		return true
	case gpuResourcesStatePhaseSaved:
	case gpuResourcesStatePhaseRestoreRequested:
		// When restoring is requested, GPU state is no longer reliable so saving again is meaningless.
	default:
		panic(fmt.Sprintf("atlas: invalid gpuResourcesStatePhase: %d", a.phase))
	}

	slog.Error("atlas: requestToSaveGPUResources was called unexpectedly", "phase", origPhase)
	return false

}

// isSavingGPUResourcesRequested reports whether saving GPU resources is requested.
//
// isSavingGPUResourcesRequested is invoked from the rendering thread.
func (a *gpuResourcesState) isSavingGPUResourcesRequested() bool {
	a.m.Lock()
	defer a.m.Unlock()
	return a.phase == gpuResourcesStatePhaseSaveRequested
}

// finishSavingGPUResources finishes saving GPU resources.
//
// finishSavingGPUResources is invoked from the rendering thread.
func (a *gpuResourcesState) finishSavingGPUResources() bool {
	a.m.Lock()
	defer a.m.Unlock()

	slog.Debug("atlas: finishSavingGPUResources was called", "phase", a.phase)

	origPhase := a.phase
	switch a.phase {
	case gpuResourcesStatePhaseNone:
	case gpuResourcesStatePhaseSaveRequested:
		a.phase = gpuResourcesStatePhaseSaved
		return true
	case gpuResourcesStatePhaseSaved:
	case gpuResourcesStatePhaseRestoreRequested:
		// Restoring would happen immediately.
	default:
		panic(fmt.Sprintf("atlas: invalid gpuResourcesStatePhase: %d", a.phase))
	}

	slog.Error("atlas: finishSavingGPUResources was called unexpectedly", "phase", origPhase)
	return false
}

// areGPUResourcesSaved reports whether the GPU resources are saved.
//
// areGPUResourcesSaved is invoked from a different thread from the rendering thread.
func (a *gpuResourcesState) areGPUResourcesSaved() bool {
	a.m.Lock()
	defer a.m.Unlock()
	return a.phase == gpuResourcesStatePhaseSaved
}

// requestToRestoreGPUResources requests to restore GPU resources.
//
// requestToRestoreGPUResources is invoked from a different thread from the rendering thread.
func (a *gpuResourcesState) requestToRestoreGPUResources() bool {
	a.m.Lock()
	defer a.m.Unlock()

	slog.Debug("atlas: requestToRestoreGPUResources was called", "phase", a.phase)

	origPhase := a.phase
	switch a.phase {
	case gpuResourcesStatePhaseNone:
		// GPU resources are not saved, so there is nothing to restore.
	case gpuResourcesStatePhaseSaveRequested:
		// GPU resources are not saved, so there is nothing to restore.
		a.phase = gpuResourcesStatePhaseNone
	case gpuResourcesStatePhaseSaved:
		a.phase = gpuResourcesStatePhaseRestoreRequested
		return true
	case gpuResourcesStatePhaseRestoreRequested:
		return true
	default:
		panic(fmt.Sprintf("atlas: invalid gpuResourcesStatePhase: %d", a.phase))
	}

	slog.Error("atlas: requestToRestoreGPUResources was called unexpectedly", "phase", origPhase)
	return false
}

// startRestoringGPUResourcesIfNeeded starts restoring GPU resources if requested.
//
// startRestoringGPUResourcesIfNeeded is invoked from the rendering thread.
func (a *gpuResourcesState) startRestoringGPUResourcesIfNeeded() bool {
	a.m.Lock()
	defer a.m.Unlock()

	// As this case is too often, avoid logging unless needed.
	if a.phase != gpuResourcesStatePhaseNone {
		slog.Debug("atlas: startRestoringGPUResourcesIfNeeded was called", "phase", a.phase)
	}

	switch a.phase {
	case gpuResourcesStatePhaseNone:
		return false
	case gpuResourcesStatePhaseSaveRequested:
		// The saving process is not finished yet, and should not be reset.
		return false
	case gpuResourcesStatePhaseSaved:
		// The app restarted, but the restoring process is not requested.
		a.phase = gpuResourcesStatePhaseNone
		return false
	case gpuResourcesStatePhaseRestoreRequested:
		a.phase = gpuResourcesStatePhaseNone
		return true
	default:
		panic(fmt.Sprintf("atlas: invalid gpuResourcesStatePhase: %d", a.phase))
	}
}
