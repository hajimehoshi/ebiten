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

package devicescale

import (
	"sync"
)

var (
	scale = 0.0
	m     sync.Mutex
)

func DeviceScale() float64 {
	s := 0.0
	m.Lock()
	if scale == 0.0 {
		scale = impl()
	}
	s = scale
	m.Unlock()
	return s
}
