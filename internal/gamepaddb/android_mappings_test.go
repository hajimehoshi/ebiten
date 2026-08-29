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

package gamepaddb

import "testing"

// TestAndroidDefaultMappingsIDLength confirms that an id whose decoded length is not 16 bytes is rejected.
func TestAndroidDefaultMappingsIDLength(t *testing.T) {
	cases := []struct {
		Name string
		ID   string
	}{
		{
			Name: "ShorterThan16Bytes",
			// 12 bytes
			ID: "030000004c05000068020000",
		},
		{
			Name: "LongerThan16Bytes",
			// 17 bytes with a valid button mask
			ID: "030000004c050000680200000f00000000",
		},
	}

	for _, c := range cases {
		if addAndroidDefaultMappings(c.ID) {
			t.Errorf("%s: an id whose decoded length is not 16 bytes must not be mapped", c.Name)
		}
	}
}
