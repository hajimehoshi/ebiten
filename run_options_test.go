// Copyright 2014 Hajime Hoshi
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

package ebiten

import "testing"

func TestRunOptionsDoNotChange(t *testing.T) {
	tests := []struct {
		className    string
		instanceName string
	}{
		{},
		{
			className: "CustomClass",
		},
		{
			instanceName: "custom-instance",
		},
		{
			className:    "CustomClass",
			instanceName: "custom-instance",
		},
	}
	for _, tt := range tests {
		options := RunGameOptions{
			GraphicsLibrary:          GraphicsLibraryOpenGL,
			InitUnfocused:            true,
			ScreenTransparent:        true,
			SkipTaskbar:              true,
			SingleThread:             true,
			DisableHiDPI:             true,
			ColorSpace:               ColorSpaceSRGB,
			ApplePressAndHoldEnabled: true,
			X11ClassName:             tt.className,
			X11InstanceName:          tt.instanceName,
			VMGuestEndpoint:          "unix:///unused.sock",
		}
		original := options
		for range 2 {
			got := toUIRunOptions(&options)
			if options != original {
				t.Errorf("class=%q instance=%q: caller options changed: got=%+v want=%+v", tt.className, tt.instanceName, options, original)
			}
			wantClass, wantInstance := tt.className, tt.instanceName
			if wantClass == "" {
				wantClass = "Ebitengine-Application"
			}
			if wantInstance == "" {
				wantInstance = "ebitengine-application"
			}
			if got.X11ClassName != wantClass || got.X11InstanceName != wantInstance {
				t.Errorf("resolved X11 names: %q %q", got.X11ClassName, got.X11InstanceName)
			}
			if !got.InitUnfocused || !got.ScreenTransparent || !got.SkipTaskbar || !got.SingleThread || !got.DisableHiDPI || !got.ApplePressAndHoldEnabled || got.VMGuestEndpoint != original.VMGuestEndpoint || int(got.GraphicsLibrary) != int(original.GraphicsLibrary) || int(got.ColorSpace) != int(original.ColorSpace) {
				t.Errorf("explicit options lost: %+v", got)
			}
		}
	}
}

func TestNilAndZeroRunOptions(t *testing.T) {
	unfocused, transparent := initUnfocused.Load(), screenTransparent.Load()
	t.Cleanup(func() { initUnfocused.Store(unfocused); screenTransparent.Store(transparent) })
	initUnfocused.Store(true)
	screenTransparent.Store(true)
	nilOptions := toUIRunOptions(nil)
	if !nilOptions.InitUnfocused || !nilOptions.ScreenTransparent {
		t.Error("nil options lost legacy settings")
	}
	options := RunGameOptions{}
	got := toUIRunOptions(&options)
	if got.InitUnfocused || got.ScreenTransparent {
		t.Error("explicit zero options inherited legacy settings")
	}
	if options != (RunGameOptions{}) {
		t.Error("zero options modified")
	}
}
