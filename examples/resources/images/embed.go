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

package images

import (
	_ "embed"
)

var (
	//go:embed ebiten.png
	Ebiten_png []byte

	//go:embed fiveyears.jpg
	FiveYears_jpg []byte

	//go:embed gophers.jpg
	Gophers_jpg []byte

	//go:embed runner.png
	Runner_png []byte

	//go:embed smoke.png
	Smoke_png []byte

	//go:embed spritesheet.png
	Spritesheet_png []byte

	//go:embed tile.png
	Tile_png []byte

	//go:embed tiles.png
	Tiles_png []byte

	//go:embed ui.png
	UI_png []byte
)
