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

//go:build (amd64 || arm64) && !android

package gbm_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/gbm"
)

func TestCRTCForEncoder(t *testing.T) {
	crtcs := []uint32{10, 20, 30}
	for _, test := range []struct {
		name             string
		crtcID, possible uint32
		want             uint32
	}{
		{name: "current CRTC", crtcID: 40, possible: 0, want: 40},
		{name: "first compatible CRTC", crtcID: 0, possible: 1 << 1, want: 20},
		{name: "no compatible CRTC", crtcID: 0, possible: 0, want: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := gbm.CRTCForEncoderForTesting(test.crtcID, test.possible, crtcs); got != test.want {
				t.Errorf("CRTCForEncoderForTesting(...) = %d, want %d", got, test.want)
			}
		})
	}
}

func TestPreferredMode(t *testing.T) {
	for _, test := range []struct {
		name  string
		types []uint32
		want  int
	}{
		{name: "preferred", types: []uint32{0, 1 << 3, 0}, want: 1},
		{name: "first when none preferred", types: []uint32{0, 0}, want: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := gbm.PreferredModeIndexForTesting(test.types); got != test.want {
				t.Errorf("PreferredModeIndexForTesting(...) = %d, want %d", got, test.want)
			}
		})
	}
}
