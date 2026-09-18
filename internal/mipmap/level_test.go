// Copyright 2026 The Ebiten Authors
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

package mipmap_test

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/mipmap"
)

func TestLevelForExactSrcRegions(t *testing.T) {
	testCases := []struct {
		name       string
		level      int
		srcRegions [graphics.ShaderSrcImageCount]image.Rectangle
		want       int
	}{
		{
			name:  "level 0",
			level: 0,
			srcRegions: [graphics.ShaderSrcImageCount]image.Rectangle{
				image.Rect(1, 1, 3, 3),
			},
			want: 0,
		},
		{
			name:       "empty regions",
			level:      3,
			srcRegions: [graphics.ShaderSrcImageCount]image.Rectangle{},
			want:       3,
		},
		{
			name:  "aligned to the level",
			level: 3,
			srcRegions: [graphics.ShaderSrcImageCount]image.Rectangle{
				image.Rect(16, 8, 64, 32),
			},
			want: 3,
		},
		{
			name:  "aligned to a lower level",
			level: 3,
			srcRegions: [graphics.ShaderSrcImageCount]image.Rectangle{
				image.Rect(16, 8, 60, 32),
			},
			want: 2,
		},
		{
			name:  "odd bound",
			level: 3,
			srcRegions: [graphics.ShaderSrcImageCount]image.Rectangle{
				image.Rect(16, 8, 64, 33),
			},
			want: 0,
		},
		{
			name:  "thin region",
			level: 2,
			srcRegions: [graphics.ShaderSrcImageCount]image.Rectangle{
				image.Rect(32, 32, 33, 64),
			},
			want: 0,
		},
		{
			name:  "second source decides",
			level: 3,
			srcRegions: [graphics.ShaderSrcImageCount]image.Rectangle{
				image.Rect(0, 0, 64, 64),
				image.Rect(0, 0, 66, 64),
			},
			want: 1,
		},
		{
			name:  "empty region is ignored",
			level: 3,
			srcRegions: [graphics.ShaderSrcImageCount]image.Rectangle{
				image.Rect(0, 0, 64, 64),
				image.Rect(3, 3, 3, 3),
			},
			want: 3,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := mipmap.LevelForExactSrcRegions(tc.level, tc.srcRegions)
			if got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}
