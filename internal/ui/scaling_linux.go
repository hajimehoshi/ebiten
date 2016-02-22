// Copyright 2016 Hajime Hoshi
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

package ui

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
)

var scalingFactorSyntax = regexp.MustCompile(`\Auint32\s+(\d+)\s*\z`)
var deviceScaleFactor = 0

// adjustScaleForGLFW adjusts the given scale which is passed to the GLFW API.
func adjustScaleForGLFW(scale int) int {
	if 0 < deviceScaleFactor {
		return scale * deviceScaleFactor
	}
	// Execute gsettings command instead of calling gtk functions so as not to depend on
	// gobject-2.0 library.
	c := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "scaling-factor")
	o, err := c.Output()
	if err != nil {
		panic(fmt.Sprintf("graphics: executing gsettings error %v", err))
	}
	m := scalingFactorSyntax.FindStringSubmatch(string(o))
	if m == nil {
		panic("graphics: gsettings result syntax is not expected")
	}
	s, err := strconv.Atoi(m[1])
	if err != nil {
		panic(fmt.Sprintf("graphics: %v", err))
	}
	deviceScaleFactor = s
	return scale * deviceScaleFactor
}
