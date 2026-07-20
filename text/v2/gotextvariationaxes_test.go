// Copyright 2026 The Ebitengine Authors
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

package text_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func TestGoTextFaceSourceAppendVariationAxes(t *testing.T) {
	staticFont, err := os.Open(filepath.Join("testdata", "Roboto-Regular.ttf"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = staticFont.Close()
	}()
	staticSrc, err := text.NewGoTextFaceSource(staticFont)
	if err != nil {
		t.Fatal(err)
	}
	if got := staticSrc.AppendVariationAxes(nil); len(got) != 0 {
		t.Errorf("AppendVariationAxes(nil) for a static font: got %v, want empty", got)
	}

	variableFont, err := os.Open(filepath.Join("testdata", "RobotoFlex.ttf"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = variableFont.Close()
	}()
	variableSrc, err := text.NewGoTextFaceSource(variableFont)
	if err != nil {
		t.Fatal(err)
	}
	wantAxes := []text.VariationAxis{
		{Tag: text.MustParseTag("opsz"), Min: 8, Default: 14, Max: 144},
		{Tag: text.MustParseTag("wght"), Min: 100, Default: 400, Max: 1000},
		{Tag: text.MustParseTag("GRAD"), Min: -200, Default: 0, Max: 150},
		{Tag: text.MustParseTag("wdth"), Min: 25, Default: 100, Max: 151},
		{Tag: text.MustParseTag("slnt"), Min: -10, Default: 0, Max: 0},
		{Tag: text.MustParseTag("XOPQ"), Min: 27, Default: 96, Max: 175},
		{Tag: text.MustParseTag("YOPQ"), Min: 25, Default: 79, Max: 135},
		{Tag: text.MustParseTag("XTRA"), Min: 323, Default: 468, Max: 603},
		{Tag: text.MustParseTag("YTUC"), Min: 528, Default: 712, Max: 760},
		{Tag: text.MustParseTag("YTLC"), Min: 416, Default: 514, Max: 570},
		{Tag: text.MustParseTag("YTAS"), Min: 649, Default: 750, Max: 854},
		{Tag: text.MustParseTag("YTDE"), Min: -305, Default: -203, Max: -98},
		{Tag: text.MustParseTag("YTFI"), Min: 560, Default: 738, Max: 788},
	}
	if got := variableSrc.AppendVariationAxes(nil); !slices.Equal(got, wantAxes) {
		t.Errorf("AppendVariationAxes(nil): got %v, want %v", got, wantAxes)
	}

	prefix := []text.VariationAxis{
		{Tag: text.MustParseTag("ital"), Min: 0, Default: 0, Max: 1},
	}
	if got, want := variableSrc.AppendVariationAxes(prefix), append(prefix, wantAxes...); !slices.Equal(got, want) {
		t.Errorf("AppendVariationAxes(%v): got %v, want %v", prefix, got, want)
	}
}
