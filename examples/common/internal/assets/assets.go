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

//go:generate go-bindata -nocompress -pkg=assets arcadefont.png
//go:generate gofmt -s -w .

package assets

import (
	"bytes"
	"image"
	_ "image/png"
)

func ArcadeFontImage() (image.Image, error) {
	b, err := Asset("arcadefont.png")
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	return img, nil
}
