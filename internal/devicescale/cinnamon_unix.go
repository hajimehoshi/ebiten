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

func cinnamonScaleFromXML() (float64, error) {
	type cinnamonMonitors struct {
		XMLName       xml.Name `xml:"monitors"`
		Version       string   `xml:"version,attr"`
		Configuration struct {
			BaseScale float64 `xml:"base_scale"`
			Output    []struct {
				Scale   float64 `xml:"scale"`
				Primary xmlBool `xml:"primary"`
			} `xml:"output"`
		} `xml:"configuration"`
	}

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

	scale := monitors.Configuration.BaseScale
	for _, v := range monitors.Configuration.Output {
		// TODO: Get the monitor at the specified position.
		if v.Primary {
			if v.Scale != 0.0 {
				scale = v.Scale
			}
			break
		}
	}
	return scale, nil
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
