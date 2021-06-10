// Copyright 2019 The Ebiten Authors
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

//go:build ((ios && arm) || (ios && arm64)) && !ebitengl
// +build ios,arm ios,arm64
// +build !ebitengl

package mobile

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/mtl"
)

func (*UserInterface) Graphics() driver.Graphics {
	if _, err := mtl.CreateSystemDefaultDevice(); err != nil {
		panic(fmt.Sprintf("ebiten: mtl.CreateSystemDefaultDevice failed on iOS: %v", err))
	}
	return metal.Get()
}
