// Copyright 2018 The Ebiten Authors
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

package ui

import (
	"github.com/hajimehoshi/ebiten/internal/devicescale"
)

type deviceScale struct {
	val         float64
	frame       int64
	lastUpdated int64
}

func (d *deviceScale) Update() {
	d.frame++
}

func (d *deviceScale) Get() float64 {
	// As devicescale.DeviceScale accesses OS API, not call this too often.
	if d.val == 0 || d.frame-d.lastUpdated > 30 {
		d.val = devicescale.Get()
		d.lastUpdated = d.frame
	}
	return d.val
}
