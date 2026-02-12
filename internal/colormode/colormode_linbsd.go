// Copyright 2026 The Ebitengine Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build (freebsd || (linux && !android) || netbsd || openbsd) && !nintendosdk && !playstation5

package colormode

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func systemColorMode() ColorMode {
	if mode := checkGSettings(); mode != Unknown {
		return mode
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Unknown
	}
	if mode := checkGTKSettingsFile(filepath.Join(homeDir+".config", "gtk-4.0", "settings.ini")); mode != Unknown {
		return mode
	}

	if mode := checkGTKSettingsFile(filepath.Join(homeDir+".config", "gtk-3.0", "settings.ini")); mode != Unknown {
		return mode
	}

	return Unknown
}

func checkGSettings() ColorMode {
	out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "color-scheme").Output()
	if err != nil {
		return Unknown
	}

	value := strings.TrimSpace(string(out))
	value = strings.Trim(value, "'")

	switch value {
	case "prefer-dark":
		return Dark
	case "default", "prefer-light":
		return Light
	default:
		return Unknown
	}
}

func checkGTKSettingsFile(path string) ColorMode {
	data, err := os.ReadFile(path)
	if err != nil {
		return Unknown
	}

	content := strings.ToLower(string(data))
	if strings.Contains(content, "gtk-application-prefer-dark-theme=true") {
		return Dark
	}
	return Light
}
