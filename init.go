// Copyright 2020 The Ebiten Authors
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

package ebiten

import (
	"github.com/hajimehoshi/ebiten/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/internal/mipmap"
	"github.com/hajimehoshi/ebiten/internal/shareable"
)

var _ = __EBITEN_REQUIRES_GO_VERSION_1_12_OR_LATER__

func init() {
	mipmap.SetGraphicsDriver(uiDriver().Graphics())
	shareable.SetGraphicsDriver(uiDriver().Graphics())
	graphicscommand.SetGraphicsDriver(uiDriver().Graphics())
}
