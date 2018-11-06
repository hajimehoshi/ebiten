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

// +build dragonfly freebsd linux netbsd openbsd solaris
// +build !js
// +build !android

package devicescale

import (
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type desktop int

const (
	desktopUnknown desktop = iota
	desktopGnome
	desktopCinnamon
	desktopUnity
	desktopKDE
	desktopXfce
)

var (
	cachedScale    float64
	cachedAt       int64
	scaleExpiresMS = int64(16)
)

func currentDesktop() desktop {
	tokens := strings.Split(os.Getenv("XDG_CURRENT_DESKTOP"), ":")
	switch tokens[len(tokens)-1] {
	case "GNOME":
		return desktopGnome
	case "X-Cinnamon":
		return desktopCinnamon
	case "Unity":
		return desktopUnity
	case "KDE":
		return desktopKDE
	case "XFCE":
		return desktopXfce
	default:
		return desktopUnknown
	}
}

var gsettingsRe = regexp.MustCompile(`\Auint32 (\d+)\s*\z`)

func gnomeScale() float64 {
	out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "scaling-factor").Output()
	if err != nil {
		if err == exec.ErrNotFound {
			return 0
		}
		if _, ok := err.(*exec.ExitError); ok {
			return 0
		}
		panic(err)
	}
	m := gsettingsRe.FindStringSubmatch(string(out))
	s, err := strconv.Atoi(m[1])
	if err != nil {
		return 0
	}
	return float64(s)
}

func cinnamonScale() float64 {
	out, err := exec.Command("gsettings", "get", "org.cinnamon.desktop.interface", "scaling-factor").Output()
	if err != nil {
		if err == exec.ErrNotFound {
			return 0
		}
		if _, ok := err.(*exec.ExitError); ok {
			return 0
		}
		panic(err)
	}
	m := gsettingsRe.FindStringSubmatch(string(out))
	s, err := strconv.Atoi(m[1])
	if err != nil {
		return 0
	}
	return float64(s)
}

func impl(x, y int) float64 {
	now := time.Now().UnixNano() / int64(time.Millisecond)
	if now-cachedAt < scaleExpiresMS {
		return cachedScale
	}

	// TODO: Can Linux has different scales for multiple monitors?
	//  Gnome supports fractional and per-monitor scaling in wayland.
	s := 1.0
	switch currentDesktop() {
	case desktopGnome:
		s = gnomeScale()
	case desktopCinnamon:
		s = cinnamonScale()
	case desktopUnity:
		// TODO: Implement
	case desktopKDE:
		// TODO: Implement
	case desktopXfce:
		// TODO: Implement
	}
	if s <= 0 {
		s = 1
	}

	// Cache the scale for later.
	now = time.Now().UnixNano() / int64(time.Millisecond)
	cachedScale = s
	cachedAt = now
	return 1
}
