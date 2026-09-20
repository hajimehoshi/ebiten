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

package mipmap_test

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/mipmap"
)

func TestMipmapLevelFromDistance(t *testing.T) {
	testCases := []struct {
		name string
		dx0  float32
		dy0  float32
		dx1  float32
		dy1  float32
		sx0  float32
		sy0  float32
		sx1  float32
		sy1  float32
		want int
	}{
		{
			name: "same size",
			dx0:  0, dy0: 0, dx1: 100, dy1: 100,
			sx0: 0, sy0: 0, sx1: 100, sy1: 100,
			want: 0,
		},
		{
			name: "scale 0.5",
			dx0:  0, dy0: 0, dx1: 50, dy1: 50,
			sx0: 0, sy0: 0, sx1: 100, sy1: 100,
			want: 0,
		},
		{
			name: "scale 0.01",
			dx0:  0, dy0: 0, dx1: 10, dy1: 10,
			sx0: 0, sy0: 0, sx1: 100, sy1: 100,
			want: 3,
		},
		{
			name: "extremely small scale",
			dx0:  0, dy0: 0, dx1: 1e-7, dy1: 1e-7,
			sx0: 0, sy0: 0, sx1: 4096, sy1: 4096,
			want: 6,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := mipmap.MipmapLevelFromDistance(tc.dx0, tc.dy0, tc.dx1, tc.dy1, tc.sx0, tc.sy0, tc.sx1, tc.sy1)
			if got != tc.want {
				t.Errorf("mipmapLevelFromDistance: got: %d, want: %d", got, tc.want)
			}
		})
	}
}

func TestRegionForLevel(t *testing.T) {
	testCases := []struct {
		name   string
		width  int
		height int
		region image.Rectangle
		level  int
		want   image.Rectangle
	}{
		{
			name:   "empty",
			width:  64,
			height: 64,
			region: image.Rectangle{},
			level:  1,
			want:   image.Rectangle{},
		},
		{
			name:   "whole image",
			width:  64,
			height: 64,
			region: image.Rect(0, 0, 64, 64),
			level:  1,
			want:   image.Rect(0, 0, 32, 32),
		},
		{
			name:   "aligned sub-image",
			width:  64,
			height: 64,
			region: image.Rect(16, 16, 48, 48),
			level:  2,
			want:   image.Rect(4, 4, 12, 12),
		},
		{
			name:   "unaligned sub-image",
			width:  64,
			height: 64,
			region: image.Rect(1, 3, 5, 9),
			level:  1,
			want:   image.Rect(0, 1, 3, 5),
		},
		{
			name:   "odd image size",
			width:  7,
			height: 7,
			region: image.Rect(0, 0, 7, 7),
			level:  1,
			want:   image.Rect(0, 0, 3, 3),
		},
		{
			name:   "sub-image outside the level image",
			width:  7,
			height: 7,
			region: image.Rect(6, 0, 7, 7),
			level:  1,
			want:   image.Rectangle{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := mipmap.New(tc.width, tc.height, atlas.ImageTypeRegular)
			got := m.RegionForLevel(tc.region, tc.level)
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
