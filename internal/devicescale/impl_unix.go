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
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
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
	cachedScale     uint64 // use atomic to read/write as multiple goroutines touch it.
	cacheUpdateWait = time.Millisecond * 100
)

func init() {
	go scaleUpdater()
}

func impl(x, y int) float64 {
	return math.Float64frombits(atomic.LoadUint64(&cachedScale))
}

// run as goroutine. Will keep the desktop scale up to date.
// This can be removed once the scale change event is implemented in GLFW 3.3
func scaleUpdater() {
	for {
		s := getscale(0, 0)
		atomic.StoreUint64(&cachedScale, math.Float64bits(s))
		time.Sleep(cacheUpdateWait)
	}
}

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

func getscale(x, y int) float64 {
	s := -1.0
	switch currentDesktop() {
	case desktopGnome:
		// TODO: Support wayland and per-monitor scaling https://wiki.gnome.org/HowDoI/HiDpi
		s = gnomeScale()
	case desktopCinnamon:
		s = cinnamonScale()
	case desktopUnity:
		// TODO: Implement, supports per-monitor scaling
	case desktopKDE:
		// TODO: Implement, appears to support per-monitor scaling
	case desktopXfce:
		// TODO: Implement
	}
	if s <= 0 {
		s = 1
	}

	return s
}
