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

//go:generate file2byteslice -package=audio -input=./audio/classic.mp3 -output=./audio/classic.go -var=Classic_mp3
//go:generate file2byteslice -package=audio -input=./audio/jab.wav -output=./audio/jab.go -var=Jab_wav
//go:generate file2byteslice -package=audio -input=./audio/jump.ogg -output=./audio/jump.go -var=Jump_ogg
//go:generate file2byteslice -package=audio -input=./audio/ragtime.ogg -output=./audio/ragtime.go -var=Ragtime_ogg
//go:generate file2byteslice -package=fonts -input=./fonts/arcade_n.ttf -output=./fonts/arcaden.go -var=ArcadeN_ttf
//go:generate file2byteslice -package=fonts -input=./fonts/mplus-1p-regular.ttf -output=./fonts/mplus1pregular.go -var=MPlus1pRegular_ttf
//go:generate file2byteslice -package=images -input=./images/ebiten.png -output=./images/ebiten.go -var=Ebiten_png
//go:generate file2byteslice -package=images -input=./images/fiveyears.jpg -output=./images/fiveyears.go -var=FiveYears_jpg
//go:generate file2byteslice -package=images -input=./images/gophers.jpg -output=./images/gophers.go -var=Gophers_jpg
//go:generate file2byteslice -package=images -input=./images/smoke.png -output=./images/smoke.go -var=Smoke_png
//go:generate file2byteslice -package=images -input=./images/tile.png -output=./images/tile.go -var=Tile_png
//go:generate file2byteslice -package=images -input=./images/runner.png -output=./images/runner.go -var=Runner_png
//go:generate file2byteslice -package=images -input=./images/tiles.png -output=./images/tiles.go -var=Tiles_png
//go:generate file2byteslice -package=images -input=./images/ui.png -output=./images/ui.go -var=UI_png
//go:generate file2byteslice -package=blocks -input=./images/blocks/background.png -output=./images/blocks/background.go -var=Background_png
//go:generate file2byteslice -package=blocks -input=./images/blocks/blocks.png -output=./images/blocks/blocks.go -var=Blocks_png
//go:generate file2byteslice -package=flappy -input=./images/flappy/gopher.png -output=./images/flappy/gopher.go -var=Gopher_png
//go:generate file2byteslice -package=flappy -input=./images/flappy/tiles.png -output=./images/flappy/tiles.go -var=Tiles_png
//go:generate file2byteslice -package=mascot -input=./images/mascot/out01.png -output=./images/mascot/out01.go -var=Out01_png
//go:generate file2byteslice -package=mascot -input=./images/mascot/out02.png -output=./images/mascot/out02.go -var=Out02_png
//go:generate file2byteslice -package=mascot -input=./images/mascot/out03.png -output=./images/mascot/out03.go -var=Out03_png
//go:generate file2byteslice -package=shader -input=./images/shader/gopher.png -output=./images/shader/gopher.go -var=Gopher_png
//go:generate file2byteslice -package=shader -input=./images/shader/gopherbg.png -output=./images/shader/gopherbg.go -var=GopherBg_png
//go:generate file2byteslice -package=shader -input=./images/shader/normal.png -output=./images/shader/normal.go -var=Normal_png
//go:generate file2byteslice -package=shader -input=./images/shader/noise.png -output=./images/shader/noise.go -var=Noise_png
//go:generate file2byteslice -package=platformer -input=./images/platformer/background.png -output=./images/platformer/background.go -var=Background_png
//go:generate file2byteslice -package=platformer -input=./images/platformer/left.png -output=./images/platformer/left.go -var=Left_png
//go:generate file2byteslice -package=platformer -input=./images/platformer/mainchar.png -output=./images/platformer/mainchar.go -var=MainChar_png
//go:generate file2byteslice -package=platformer -input=./images/platformer/right.png -output=./images/platformer/right.go -var=Right_png
//go:generate gofmt -s -w .

package resources

import (
	// Dummy imports for go.mod for some Go files with 'ignore' tags. For example, `go mod tidy` does not
	// recognize Go files with 'ignore' build tag.
	//
	// Note that this affects only importing this package, but not 'file2byteslice' commands in //go:generate.
	_ "github.com/hajimehoshi/file2byteslice"
)
