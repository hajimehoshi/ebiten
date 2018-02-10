// Copyright 2014 Hajime Hoshi
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

//go:generate file2byteslice -input text.png -output bindata.go -package assets -var textPng
//go:generate gofmt -s -w .

package assets

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
)

const (
	CharWidth  = 6
	CharHeight = 16
)

func CreateTextImage() image.Image {
	img, _, err := image.Decode(bytes.NewBuffer(textPng))
	if err != nil {
		panic(fmt.Sprintf("assets: image.Decode failed, %v", err))
	}
	return img
}
