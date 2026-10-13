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

package main

import (
	"slices"
	"testing"
)

func TestObjectPoints(t *testing.T) {
	for _, tc := range []struct {
		name  string
		walls []line
		want  [][2]float64
	}{
		{
			name: "horizontal",
			walls: []line{
				{
					X1: 10,
					Y1: 20,
					X2: 30,
					Y2: 20,
				},
			},
			want: [][2]float64{{30, 20}, {10, 20}},
		},
		{
			name: "vertical",
			walls: []line{
				{
					X1: 10,
					Y1: 20,
					X2: 10,
					Y2: 40,
				},
			},
			want: [][2]float64{{10, 40}, {10, 20}},
		},
		{
			name: "diagonal",
			walls: []line{
				{
					X1: 10,
					Y1: 20,
					X2: 30,
					Y2: 40,
				},
			},
			want: [][2]float64{{30, 40}, {10, 20}},
		},
		{
			name:  "closed",
			walls: rect(10, 20, 20, 20),
			want:  [][2]float64{{10, 40}, {30, 40}, {30, 20}, {10, 20}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := (object{walls: tc.walls}).points(); !slices.Equal(got, tc.want) {
				t.Errorf("points() = %v, want %v", got, tc.want)
			}
		})
	}
}
