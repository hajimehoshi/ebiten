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

package vmhost_test

import (
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
)

// TestGuestAudioStreamPositionLong tests that the position of a long-running stream is correct, where
// converting the samples to a duration would overflow int64 if the multiplication came first.
func TestGuestAudioStreamPositionLong(t *testing.T) {
	const (
		rate           = 48000
		bytesPerSample = 8
	)
	for _, hours := range []int{1, 53, 54, 1000} {
		want := time.Duration(hours) * time.Hour
		readBytes := int64(hours) * 3600 * rate * bytesPerSample
		s := vmhost.NewGuestAudioStreamForTesting(rate, readBytes)
		if got := s.Position(); got != want {
			t.Errorf("Position() after %d hours: got: %v, want: %v", hours, got, want)
		}
	}
}
