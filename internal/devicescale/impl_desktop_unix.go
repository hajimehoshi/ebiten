// Copyright 2018 The Ebiten Authors
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

//go:build (dragonfly || freebsd || linux || netbsd || openbsd || solaris) && !android
// +build dragonfly freebsd linux netbsd openbsd solaris
// +build !android

package devicescale

import (
	"log"
	"math"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/randr"
	"github.com/jezek/xgb/xproto"
)

func impl(x, y int) float64 {
	// BEWARE: if https://github.com/glfw/glfw/issues/1961 gets fixed, this function may need revising.
	// In case GLFW decides to switch to returning logical pixels, we can just return 1.0.

	// Note: GLFW currently returns physical pixel sizes,
	// but we need to predict the window system-side size of the fullscreen window
	// for our `ScreenSizeInFullscreen` public API.
	// Also at the moment we need this prior to switching to fullscreen, but that might be replacable.
	// So this function computes the ratio of physical per logical pixels.
	m := monitorAt(x, y)
	sx, _ := m.GetContentScale()
	monitorX, monitorY := m.GetPos()
	xconn, err := xgb.NewConn()
	defer xconn.Close()
	if err != nil {
		// No X11 connection?
		// Assume we're on pure Wayland then.
		// GLFW/Wayland shouldn't be having this issue.
		// log.Print("No X11.")
		return float64(sx)
	}
	err = randr.Init(xconn)
	if err != nil {
		// No RANDR extension?
		// No problem.
		// log.Print("No RANDR.")
		return float64(sx)
	}
	root := xproto.Setup(xconn).DefaultScreen(xconn).Root
	res, err := randr.GetScreenResourcesCurrent(xconn, root).Reply()
	if err != nil {
		// Likely means RANDR is not working.
		// No problem.
		// log.Print("RANDR not working.")
		return float64(sx)
	}
	for _, crtc := range res.Crtcs[0:res.NumCrtcs] {
		info, err := randr.GetCrtcInfo(xconn, crtc, res.ConfigTimestamp).Reply()
		if err != nil {
			// This Crtc is bad. Maybe just got disconnected?
			continue
		}
		if info.NumOutputs == 0 {
			// This Crtc is not connected to any output.
			// In other words, a disabled monitor.
			continue
		}
		if int(info.X) == monitorX && int(info.Y) == monitorY {
			// log.Printf("Monitor found: %+v.", info)
			xWidth, xHeight := info.Width, info.Height
			vm := m.GetVideoMode()
			physWidth, physHeight := vm.Width, vm.Height
			// log.Printf("Virtual: %vx%v <- physical: %vx%v.", xWidth, xHeight, physWidth, physHeight)
			// We must return one scale, even though there may be two.
			// Return the _larger_ scale, as this would yield a letterboxed display on mismatch, rather than a cut-off one.
			scale := math.Max(float64(physWidth)/float64(xWidth), float64(physHeight)/float64(xHeight))
			// log.Printf("Screen scale: %v.", scale)
			log.Printf("sx=%v scale=%v -> %v", sx, scale, float64(sx)*scale)
			return float64(sx) * scale
		}
	}
	// Monitor not known to XRandR. Weird.
	// log.Print("Monitor not found.")
	return float64(sx)
}
