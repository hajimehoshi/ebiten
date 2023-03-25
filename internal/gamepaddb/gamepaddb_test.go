// Copyright 2022 The Ebitengine Authors
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

package gamepaddb_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
)

func TestUpdate(t *testing.T) {
	cases := []struct {
		Input string
		Err   bool
	}{
		{
			Input: "",
			Err:   false,
		},
		{
			Input: "{}",
			Err:   true,
		},
		{
			Input: "00000000000000000000000000000000",
			Err:   true,
		},
		{
			Input: "00000000000000000000000000000000,foo",
			Err:   false,
		},
		{
			Input: "00000000000000000000000000000000,foo,platform",
			Err:   true,
		},
		{
			Input: "00000000000000000000000000000000,foo,platform:Foo",
			Err:   true,
		},
		{
			Input: "00000000000000000000000000000000,foo,platform:Windows",
			Err:   false,
		},
	}

	for _, c := range cases {
		err := gamepaddb.Update([]byte(c.Input))
		if err == nil && c.Err {
			t.Errorf("Update(%q) should return an error but not", c.Input)
		}
		if err != nil && !c.Err {
			t.Errorf("Update(%q) should not return an error but returned %v", c.Input, err)
		}
	}
}
