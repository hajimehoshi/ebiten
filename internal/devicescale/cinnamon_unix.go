// Copyright 2020 The Ebiten Authors
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
	"encoding/xml"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/hajimehoshi/ebiten/internal/glfw"
)

type xmlBool bool

func (b *xmlBool) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	if err := d.DecodeElement(&s, &start); err != nil {
		return err
	}
	*b = xmlBool(s == "yes")
	return nil
}

type cinnamonMonitors struct {
	XMLName       xml.Name                        `xml:"monitors"`
	Version       string                          `xml:"version,attr"`
	Configuration []cinnamonMonitorsConfiguration `xml:"configuration"`
}

type cinnamonMonitorsConfiguration struct {
	BaseScale float64 `xml:"base_scale"`
	Output    []struct {
		X       int     `xml:"x"`
		Y       int     `xml:"y"`
		Width   int     `xml:"width"`
		Height  int     `xml:"height"`
		Scale   float64 `xml:"scale"`
		Primary xmlBool `xml:"primary"`
	} `xml:"output"`
}

func (c *cinnamonMonitorsConfiguration) matchesWithGLFWMonitors(monitors []*glfw.Monitor) bool {
	type area struct {
		X, Y, Width, Height int
	}
	areas := map[area]struct{}{}

	for _, o := range c.Output {
		if o.Width == 0 || o.Height == 0 {
			continue
		}
		areas[area{
			X:      o.X,
			Y:      o.Y,
			Width:  o.Width,
			Height: o.Height,
		}] = struct{}{}
	}

	if len(areas) != len(monitors) {
		return false
	}

	for _, m := range monitors {
		x, y := m.GetPos()
		v := m.GetVideoMode()
		a := area{
			X:      x,
			Y:      y,
			Width:  v.Width,
			Height: v.Height,
		}
		if _, ok := areas[a]; !ok {
			return false
		}
	}
	return true
}

func cinnamonScaleFromXML() (float64, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return 0, err
	}
	f, err := os.Open(filepath.Join(home, ".config", "cinnamon-monitors.xml"))
	if err != nil {
		return 0, err
	}
	defer f.Close()

	d := xml.NewDecoder(f)

	var monitors cinnamonMonitors
	if err = d.Decode(&monitors); err != nil {
		return 0, err
	}

	for _, c := range monitors.Configuration {
		if !c.matchesWithGLFWMonitors(glfw.GetMonitors()) {
			continue
		}
		for _, v := range c.Output {
			// TODO: Get the monitor at the specified position.
			// TODO: Consider the base scale?
			if v.Primary && v.Scale != 0.0 {
				return v.Scale, nil
			}
		}
	}
	return 0, nil
}

func cinnamonScale() float64 {
	if s, err := cinnamonScaleFromXML(); err == nil && s > 0 {
		return s
	}

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
