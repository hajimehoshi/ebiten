// Copyright 2022 The Ebiten Authors
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

package directx

import (
	"errors"
	"fmt"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/microsoftgdk"
	"github.com/hajimehoshi/ebiten/v2/internal/winver"
)

type stencilMode int

const (
	noStencil stencilMode = iota
	incrementStencil
	invertStencil
	drawWithStencil
)

const frameCount = 2

func pow2(x uint32) uint32 {
	if x > (math.MaxUint32+1)/2 {
		return math.MaxUint32
	}

	var p2 uint32 = 1
	for p2 < x {
		p2 *= 2
	}
	return p2
}

func parseFeatureLevel(str string) (_D3D_FEATURE_LEVEL, bool) {
	switch str {
	case "11_0":
		return _D3D_FEATURE_LEVEL_11_0, true
	case "11_1":
		return _D3D_FEATURE_LEVEL_11_1, true
	case "12_0":
		return _D3D_FEATURE_LEVEL_12_0, true
	case "12_1":
		return _D3D_FEATURE_LEVEL_12_1, true
	case "12_2":
		return _D3D_FEATURE_LEVEL_12_2, true
	default:
		return 0, false
	}
}

// NewGraphics creates an implementation of graphicsdriver.Graphics for DirectX.
// The returned graphics value is nil iff the error is not nil.
func NewGraphics() (graphicsdriver.Graphics, error) {
	if !isD3DCompilerDLLAvailable() {
		return nil, fmt.Errorf("directx: d3dcompiler_*.dll is missing in this environment")
	}

	var useWARP bool
	var useDebugLayer bool
	version := 11

	// Specify the feature level 11 by default.
	// Some old cards don't work well with the default feature level (#2447, #2486).
	featureLevel := _D3D_FEATURE_LEVEL_11_0

	// Parse a special environment variable for backward compatibility.
	if env := os.Getenv("EBITENGINE_DIRECTX_FEATURE_LEVEL"); env != "" {
		if fl, ok := parseFeatureLevel(env); ok {
			featureLevel = fl
		}
	}

	env := os.Getenv("EBITENGINE_DIRECTX")
	if env == "" {
		// For backward compatibility, read the EBITEN_ version.
		env = os.Getenv("EBITEN_DIRECTX")
	}

	for _, t := range strings.Split(env, ",") {
		t := strings.TrimSpace(t)
		switch {
		case t == "warp":
			// TODO: Is WARP available on Xbox?
			useWARP = true
		case t == "debug":
			useDebugLayer = true
		case strings.HasPrefix(t, "version="):
			v, err := strconv.Atoi(t[len("version="):])
			if err != nil {
				continue
			}
			version = v
		case strings.HasPrefix(t, "featurelevel="):
			fl, ok := parseFeatureLevel(t[len("featurelevel="):])
			if !ok {
				continue
			}
			featureLevel = fl
		}
	}

	// On Xbox, only DirectX 12 is available.
	if microsoftgdk.IsXbox() {
		version = 12
	}

	switch version {
	case 11:
		g, err := newGraphics11(useWARP, useDebugLayer)
		if err != nil {
			return nil, err
		}
		return g, nil
	case 12:
		g, err := newGraphics12(useWARP, useDebugLayer, featureLevel)
		if err != nil {
			return nil, err
		}
		return g, nil
	default:
		panic(fmt.Sprintf("directx: unexpected DirectX version: %d", version))
	}
}

type graphicsInfra struct {
	*graphicsInfraResources

	allowTearing bool

	// occluded reports whether the screen is invisible or not.
	occluded bool

	// lastTime is the last time for rendering.
	lastTime time.Time

	bufferCount int

	cleanup runtime.Cleanup
}

type graphicsInfraResources struct {
	factory    *_IDXGIFactory
	swapChain  *_IDXGISwapChain
	swapChain4 *_IDXGISwapChain4
}

// newGraphicsInfra takes the ownership of the given factory.
func newGraphicsInfra(factory *_IDXGIFactory) (*graphicsInfra, error) {
	g := &graphicsInfra{
		graphicsInfraResources: &graphicsInfraResources{
			factory: factory,
		},
	}
	g.cleanup = runtime.AddCleanup(g, (*graphicsInfraResources).releaseResources, g.graphicsInfraResources)

	if f, err := g.factory.QueryInterface(&_IID_IDXGIFactory5); err == nil && f != nil {
		factory := (*_IDXGIFactory5)(f)
		defer factory.Release()

		var allowTearing int32
		if err := factory.CheckFeatureSupport(_DXGI_FEATURE_PRESENT_ALLOW_TEARING, unsafe.Pointer(&allowTearing), uint32(unsafe.Sizeof(allowTearing))); err == nil && allowTearing != 0 {
			g.allowTearing = true
		}
	}

	return g, nil
}

func (g *graphicsInfra) release() {
	g.releaseResources()
	g.cleanup.Stop()
}

func (g *graphicsInfraResources) releaseResources() {
	if g.factory != nil {
		g.factory.Release()
		g.factory = nil
	}
	if g.swapChain != nil {
		g.swapChain.Release()
		g.swapChain = nil
	}
	if g.swapChain4 != nil {
		g.swapChain4.Release()
		g.swapChain4 = nil
	}
}

// appendAdapters appends found adapters to the given adapters.
// Releasing them is the caller's responsibility.
//
// warpForDX12 is valid only for DirectX 12.
func (g *graphicsInfra) appendAdapters(adapters []*_IDXGIAdapter1, warpForDX12 bool) ([]*_IDXGIAdapter1, error) {
	f, err := g.factory.QueryInterface(&_IID_IDXGIFactory4)
	if err != nil {
		return nil, err
	}
	if f == nil {
		return nil, fmt.Errorf("directx: IID_IDXGIFactory4 was not available")
	}
	factory4 := (*_IDXGIFactory4)(f)
	defer factory4.Release()

	if warpForDX12 {
		a, err := factory4.EnumWarpAdapter()
		if err != nil {
			return nil, err
		}
		adapters = append(adapters, a)
		return adapters, nil
	}

	for i := uint32(0); ; i++ {
		a, err := factory4.EnumAdapters1(i)
		if errors.Is(err, _DXGI_ERROR_NOT_FOUND) {
			break
		}
		if err != nil {
			return nil, err
		}

		adapters = append(adapters, a)
	}

	return adapters, nil
}

func (g *graphicsInfra) isSwapChainInited() bool {
	return g.swapChain != nil
}

func (g *graphicsInfra) initSwapChain(width, height int, device unsafe.Pointer, window windows.HWND) (ferr error) {
	if g.swapChain != nil {
		return fmt.Errorf("directx: swap chain must not be initialized at initSwapChain, but is already done")
	}

	// Create a swap chain.
	//
	// DXGI_ALPHA_MODE_PREMULTIPLIED doesn't work with a HWND well.
	//
	//     IDXGIFactory::CreateSwapChain: Alpha blended swapchains must be created with CreateSwapChainForComposition,
	//     or CreateSwapChainForCoreWindow with the DXGI_SWAP_CHAIN_FLAG_FOREGROUND_LAYER flag
	//
	// Use *_SEQUENTIAL swap effects to follow the Mozilla way:
	// https://github.com/mozilla/gecko-dev/blob/0907529ff72c456ddb47839f5f7ba16291f28dce/gfx/layers/d3d11/CompositorD3D11.cpp#L167-L254
	desc := &_DXGI_SWAP_CHAIN_DESC{
		BufferDesc: _DXGI_MODE_DESC{
			Width:  uint32(width),
			Height: uint32(height),
			Format: _DXGI_FORMAT_B8G8R8A8_UNORM,
		},
		SampleDesc: _DXGI_SAMPLE_DESC{
			Count:   1,
			Quality: 0,
		},
		BufferUsage:  _DXGI_USAGE_RENDER_TARGET_OUTPUT,
		BufferCount:  frameCount,
		OutputWindow: window,
		Windowed:     1,
		SwapEffect:   _DXGI_SWAP_EFFECT_FLIP_SEQUENTIAL,
	}

	// DXGI_SWAP_EFFECT_FLIP_SEQUENTIAL/DISCARD are not supported for older Windows than 10 or DirectX 12.
	// https://learn.microsoft.com/en-us/windows/win32/api/dxgi/ne-dxgi-dxgi_swap_effect
	if !winver.IsWindows10OrGreater() {
		desc.SwapEffect = _DXGI_SWAP_EFFECT_SEQUENTIAL
		// With the non-flip (bitblt) mode, the buffer count should be 1. See also:
		// * https://bugzilla.mozilla.org/show_bug.cgi?id=1419293#c18
		// * https://learn.microsoft.com/en-us/windows/win32/direct3ddxgi/dxgi-flip-model
		desc.BufferCount = 1
	}

	g.bufferCount = int(desc.BufferCount)

	if g.allowTearing {
		desc.Flags |= uint32(_DXGI_SWAP_CHAIN_FLAG_ALLOW_TEARING)
	}
	s, err := g.factory.CreateSwapChain(device, desc)
	if err != nil {
		return err
	}
	g.swapChain = s
	defer func() {
		if ferr != nil {
			g.release()
		}
	}()

	if s4, err := g.swapChain.QueryInterface(&_IID_IDXGISwapChain4); err == nil && s4 != nil {
		g.swapChain4 = (*_IDXGISwapChain4)(s4)
	}

	// MakeWindowAssociation should be called after swap chain creation.
	// https://docs.microsoft.com/en-us/windows/win32/api/dxgi/nf-dxgi-idxgifactory-makewindowassociation
	if err := g.factory.MakeWindowAssociation(window, _DXGI_MWA_NO_WINDOW_CHANGES|_DXGI_MWA_NO_ALT_ENTER); err != nil {
		return err
	}

	return nil
}

func (g *graphicsInfra) resizeSwapChain(width, height int) error {
	if g.swapChain == nil {
		return fmt.Errorf("directx: swap chain must be initialized at resizeSwapChain, but is not")
	}

	var flag uint32
	if g.allowTearing {
		flag |= uint32(_DXGI_SWAP_CHAIN_FLAG_ALLOW_TEARING)
	}
	if err := g.swapChain.ResizeBuffers(uint32(g.bufferCount), uint32(width), uint32(height), _DXGI_FORMAT_B8G8R8A8_UNORM, flag); err != nil {
		return err
	}
	return nil
}

func (g *graphicsInfra) currentBackBufferIndex() (int, error) {
	if g.swapChain4 == nil {
		return 0, fmt.Errorf("directx: IDXGISwapChain4 is not available")
	}
	return int(g.swapChain4.GetCurrentBackBufferIndex()), nil
}

func (g *graphicsInfra) present(vsyncEnabled bool) error {
	if g.swapChain == nil {
		return fmt.Errorf("directx: swap chain must be initialized at present, but is not")
	}

	var syncInterval uint32
	var flags _DXGI_PRESENT
	if g.occluded {
		// The screen is not visible. Test whether we can resume.
		flags |= _DXGI_PRESENT_TEST
	} else {
		// Do actual rendering only when the screen is visible.
		if vsyncEnabled {
			syncInterval = 1
		} else if g.allowTearing {
			flags |= _DXGI_PRESENT_ALLOW_TEARING
		}
	}

	occluded, err := g.swapChain.Present(syncInterval, uint32(flags))
	if err != nil {
		return err
	}
	g.occluded = occluded

	// Reduce FPS when the screen is invisible.
	now := time.Now()
	if g.occluded {
		if delta := 100*time.Millisecond - now.Sub(g.lastTime); delta > 0 {
			time.Sleep(delta)
		}
	}
	g.lastTime = now

	return nil
}

func (g *graphicsInfra) getBuffer(buffer uint32, riid *windows.GUID) (unsafe.Pointer, error) {
	return g.swapChain.GetBuffer(buffer, riid)
}
