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

//go:build (freebsd || (linux && !android) || netbsd) && !nintendosdk && !playstation5

package colormode_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/colormode"
)

func TestCheckGTKSettingsFile(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		want    colormode.ColorMode
	}{
		{
			name:    "dark",
			content: "[Settings]\ngtk-application-prefer-dark-theme=true\n",
			want:    colormode.Dark,
		},
		{
			name:    "dark by 1",
			content: "[Settings]\ngtk-application-prefer-dark-theme=1\n",
			want:    colormode.Dark,
		},
		{
			name:    "light",
			content: "[Settings]\ngtk-application-prefer-dark-theme=false\n",
			want:    colormode.Light,
		},
		{
			name:    "light by 0",
			content: "[Settings]\ngtk-application-prefer-dark-theme=0\n",
			want:    colormode.Light,
		},
		{
			name:    "spaces around the equals sign",
			content: "[Settings]\ngtk-application-prefer-dark-theme = true\n",
			want:    colormode.Dark,
		},
		{
			name:    "indented",
			content: "[Settings]\n\tgtk-application-prefer-dark-theme=true\n",
			want:    colormode.Dark,
		},
		{
			name:    "CRLF",
			content: "[Settings]\r\ngtk-application-prefer-dark-theme=true\r\n",
			want:    colormode.Dark,
		},
		{
			name:    "no trailing new line",
			content: "[Settings]\ngtk-application-prefer-dark-theme=true",
			want:    colormode.Dark,
		},
		{
			name:    "empty",
			content: "",
			want:    colormode.Unknown,
		},
		{
			name:    "other keys only",
			content: "[Settings]\ngtk-font-name=Cantarell 11\n",
			want:    colormode.Unknown,
		},
		{
			name:    "commented out",
			content: "[Settings]\n#gtk-application-prefer-dark-theme=true\n",
			want:    colormode.Unknown,
		},
		{
			name:    "commented out with a space",
			content: "[Settings]\n # gtk-application-prefer-dark-theme=true\n",
			want:    colormode.Unknown,
		},
		{
			name:    "no group",
			content: "gtk-application-prefer-dark-theme=true\n",
			want:    colormode.Unknown,
		},
		{
			name:    "other group",
			content: "[Other]\ngtk-application-prefer-dark-theme=true\n",
			want:    colormode.Unknown,
		},
		{
			name:    "group restored",
			content: "[Other]\ngtk-application-prefer-dark-theme=false\n[Settings]\ngtk-application-prefer-dark-theme=true\n",
			want:    colormode.Dark,
		},
		{
			name:    "invalid value",
			content: "[Settings]\ngtk-application-prefer-dark-theme=TRUE\n",
			want:    colormode.Unknown,
		},
		{
			name:    "no value",
			content: "[Settings]\ngtk-application-prefer-dark-theme\n",
			want:    colormode.Unknown,
		},
		{
			name:    "other key with a similar name",
			content: "[Settings]\nx-gtk-application-prefer-dark-theme=true\n",
			want:    colormode.Unknown,
		},
		{
			name:    "the last key wins",
			content: "[Settings]\ngtk-application-prefer-dark-theme=true\ngtk-application-prefer-dark-theme=false\n",
			want:    colormode.Light,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.ini")
			if err := os.WriteFile(path, []byte(tc.content), 0644); err != nil {
				t.Fatal(err)
			}
			if got, want := colormode.CheckGTKSettingsFile(path), tc.want; got != want {
				t.Errorf("colormode.CheckGTKSettingsFile(%q): got: %d, want: %d", tc.content, got, want)
			}
		})
	}
}

func TestCheckGTKSettingsFileNotExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.ini")
	if got, want := colormode.CheckGTKSettingsFile(path), colormode.Unknown; got != want {
		t.Errorf("colormode.CheckGTKSettingsFile(%q): got: %d, want: %d", path, got, want)
	}
}
