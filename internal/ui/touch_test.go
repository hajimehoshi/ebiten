// Copyright 2026 Hajime Hoshi
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

package ui_test

import (
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// touchIDsForSets feeds each set's platform IDs to a fresh allocator and returns the IDs issued for
// them, set by set.
func touchIDsForSets(sets [][]int) [][]ui.TouchID {
	var a ui.TouchIDAllocator
	var got [][]ui.TouchID
	for _, platformIDs := range sets {
		a.NextTouchSet()
		var ids []ui.TouchID
		for _, platformID := range platformIDs {
			ids = append(ids, a.ID(platformID))
		}
		got = append(got, ids)
	}
	return got
}

func TestTouchIDAllocator(t *testing.T) {
	testCases := []struct {
		name string
		sets [][]int
		want [][]ui.TouchID
	}{
		{
			name: "one touch keeps its ID",
			sets: [][]int{
				{0},
				{0},
				{0},
			},
			want: [][]ui.TouchID{
				{0},
				{0},
				{0},
			},
		},
		{
			name: "consecutive touches under one platform ID",
			sets: [][]int{
				{0},
				{},
				{0},
				{0},
				{},
				{0},
			},
			want: [][]ui.TouchID{
				{0},
				{},
				{1},
				{1},
				{},
				{2},
			},
		},
		{
			name: "a lifted finger's platform ID given to a new finger",
			sets: [][]int{
				{0, 1},
				{1},
				{1, 0},
				{1, 0},
			},
			want: [][]ui.TouchID{
				{0, 1},
				{1},
				{1, 2},
				{1, 2},
			},
		},
		{
			name: "platform IDs that start high",
			sets: [][]int{
				{1000, 7},
				{7},
				{7, 1000},
			},
			want: [][]ui.TouchID{
				{0, 1},
				{1},
				{1, 2},
			},
		},
		{
			name: "a platform ID repeated within a set",
			sets: [][]int{
				{3, 3},
				{3},
			},
			want: [][]ui.TouchID{
				{0, 0},
				{0},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := touchIDsForSets(tc.sets)
			for i := range tc.sets {
				if !slices.Equal(got[i], tc.want[i]) {
					t.Errorf("set %d (%v): got %v, want %v", i, tc.sets[i], got[i], tc.want[i])
				}
			}
		})
	}
}
