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

// TestParseLineIDLength confirms that a line whose id is not a hex-encoded 16-byte GUID
// is ignored on any platform.
func TestParseLineIDLength(t *testing.T) {
	const valid = "030000004c0500006802000000007200"

	cases := []struct {
		Name   string
		Line   string
		WantID string
	}{
		{
			Name:   "Valid",
			Line:   valid + ",DualShock 4,a:b0,",
			WantID: valid,
		},
		{
			Name: "ShorterThan16Bytes",
			Line: "030000004c05000068020000,DualShock 4,a:b0,",
		},
		{
			Name: "LongerThan16Bytes",
			Line: valid + "00,DualShock 4,a:b0,",
		},
		{
			Name: "NotHex",
			Line: "xinput,XInput Controller,a:b0,",
		},
	}

	for _, c := range cases {
		for _, p := range []platform{platformWindows, platformDarwin, platformUnix, platformAndroid} {
			id, _, _, _, err := parseLine(c.Line, p)
			if err != nil {
				t.Errorf("%s/%v: parseLine returned an unexpected error: %v", c.Name, p, err)
				continue
			}
			if id != c.WantID {
				t.Errorf("%s/%v: got id %q, want %q", c.Name, p, id, c.WantID)
			}
		}
	}
}
