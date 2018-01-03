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

// +build freebsd linux
// +build !js
// +build !android

package devicescale

import (
	"os/exec"
	"regexp"
	"strconv"
)

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

func impl() float64 {
	if s := gnomeScale(); s != 0 {
		return s
	}
	if s := cinnamonScale(); s != 0 {
		return s
	}
	return 1
}
