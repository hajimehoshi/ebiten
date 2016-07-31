// Copyright 2016 The Ebiten Authors
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

package twenty48

import (
	"image/color"
)

var (
	backgroundColor = color.RGBA{0xfa, 0xf8, 0xef, 0xff}
	frameColor      = color.RGBA{0xbb, 0xad, 0xa0, 0xff}
)

func tileColor(value int) color.Color {
	switch value {
	case 2, 4:
		return color.RGBA{0x77, 0x6e, 0x65, 0xff}
	case 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536:
		return color.RGBA{0xf9, 0xf6, 0xf2, 0xff}
	}
	panic("not reach")
}

func tileBackgroundColor(value int) color.Color {
	switch value {
	case 0:
		return color.NRGBA{0xee, 0xe4, 0xda, 0x59}
	case 2:
		return color.RGBA{0xee, 0xe4, 0xda, 0xff}
	case 4:
		return color.RGBA{0xed, 0xe0, 0xc8, 0xff}
	case 8:
		return color.RGBA{0xf2, 0xb1, 0x79, 0xff}
	case 16:
		return color.RGBA{0xf5, 0x95, 0x63, 0xff}
	case 32:
		return color.RGBA{0xf6, 0x7c, 0x5f, 0xff}
	case 64:
		return color.RGBA{0xf6, 0x5e, 0x3b, 0xff}
	case 128:
		return color.RGBA{0xed, 0xcf, 0x72, 0xff}
	case 256:
		return color.RGBA{0xed, 0xcc, 0x61, 0xff}
	case 512:
		return color.RGBA{0xed, 0xc8, 0x50, 0xff}
	case 1024:
		return color.RGBA{0xed, 0xc5, 0x3f, 0xff}
	case 2048:
		return color.RGBA{0xed, 0xc2, 0x2e, 0xff}
	case 4096:
		return color.NRGBA{0xa3, 0x49, 0xa4, 0x7f}
	case 8192:
		return color.NRGBA{0xa3, 0x49, 0xa4, 0xb2}
	case 16384:
		return color.NRGBA{0xa3, 0x49, 0xa4, 0xcc}
	case 32768:
		return color.NRGBA{0xa3, 0x49, 0xa4, 0xe5}
	case 65536:
		return color.NRGBA{0xa3, 0x49, 0xa4, 0xff}
	}
	panic("not reach")
}
