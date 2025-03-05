// Copyright 2014 Hajime Hoshi
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

package ebiten

import (
	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/builtinshader"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// Filter represents the type of texture filter to be used when an image is magnified or minified.
type Filter int

const (
	// FilterNearest represents nearest (crisp-edged) filter
	FilterNearest Filter = Filter(builtinshader.FilterNearest)

	// FilterLinear represents linear filter
	FilterLinear Filter = Filter(builtinshader.FilterLinear)

	// FilterPixelated represents a pixelated filter.
	// FilterPixelated is similar to FilterNearest, but it preserves the pixelated appearance even when scaled to non-integer sizes.
	FilterPixelated Filter = Filter(builtinshader.FilterPixelated)
)

// GraphicsLibrary represents graphics libraries supported by the engine.
type GraphicsLibrary int

const (
	// GraphicsLibraryAuto represents the automatic choose of graphics library by Ebitengine.
	GraphicsLibraryAuto GraphicsLibrary = GraphicsLibrary(ui.GraphicsLibraryAuto)

	// GraphicsLibraryUnknown represents the state at which graphics library cannot be determined,
	// e.g. hasn't loaded yet or failed to initialize.
	GraphicsLibraryUnknown GraphicsLibrary = GraphicsLibrary(ui.GraphicsLibraryUnknown)

	// GraphicsLibraryOpenGL represents the graphics library OpenGL.
	GraphicsLibraryOpenGL GraphicsLibrary = GraphicsLibrary(ui.GraphicsLibraryOpenGL)

	// GraphicsLibraryDirectX represents the graphics library Microsoft DirectX.
	GraphicsLibraryDirectX GraphicsLibrary = GraphicsLibrary(ui.GraphicsLibraryDirectX)

	// GraphicsLibraryMetal represents the graphics library Apple's Metal.
	GraphicsLibraryMetal GraphicsLibrary = GraphicsLibrary(ui.GraphicsLibraryMetal)

	// GraphicsLibraryMetal represents the graphics library PlayStation 5.
	GraphicsLibraryPlayStation5 GraphicsLibrary = GraphicsLibrary(ui.GraphicsLibraryPlayStation5)
)

// String returns a string representing the graphics library.
func (g GraphicsLibrary) String() string {
	return ui.GraphicsLibrary(g).String()
}

// Ensures GraphicsLibraryAuto is zero (the default value for RunOptions).
var _ [GraphicsLibraryAuto]int = [0]int{}

// DebugInfo is a struct to store debug info about the graphics.
type DebugInfo struct {
	// GraphicsLibrary represents the graphics library currently in use.
	GraphicsLibrary GraphicsLibrary

	// TotalGPUImageMemoryUsageInBytes is the total image memory usage for GPU in bytes.
	// TotalGPUImageMemoryUsageInBytes is approximately the total memory usage for GPU.
	TotalGPUImageMemoryUsageInBytes int64
}

// ReadDebugInfo writes debug info (e.g. current graphics library) into a provided struct.
//
// ReadDebugInfo is concurrent-safe.
func ReadDebugInfo(d *DebugInfo) {
	d.GraphicsLibrary = GraphicsLibrary(ui.Get().GraphicsLibrary())
	d.TotalGPUImageMemoryUsageInBytes = atlas.TotalGPUImageMemoryUsageInBytes()
}

// ColorSpace represents the color space of the screen.
type ColorSpace int

const (
	// ColorSpaceDefault represents the default color space.
	ColorSpaceDefault ColorSpace = iota

	// ColorSpaceSRGB represents the sRGB color space (https://en.wikipedia.org/wiki/SRGB).
	ColorSpaceSRGB

	// ColorSpaceDisplayP3 represents the Display P3 color space (https://en.wikipedia.org/wiki/DCI-P3).
	ColorSpaceDisplayP3
)
