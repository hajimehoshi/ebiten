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

//go:generate png2compressedrgba -input text.png -output /tmp/compressedTextRGBA
//go:generate file2byteslice -input /tmp/compressedTextRGBA -output textrgba.go -package assets -var compressedTextRGBA
//go:generate gofmt -s -w .

package assets

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"image"
	"io/ioutil"
)

const (
	imgWidth  = 192
	imgHeight = 128

	CharWidth  = 6
	CharHeight = 16
)

func CreateTextImage() *image.RGBA {
	s, err := gzip.NewReader(bytes.NewReader(compressedTextRGBA))
	if err != nil {
		panic(fmt.Sprintf("assets: gzip.NewReader failed: %v", err))
	}
	defer s.Close()

	pix, err := ioutil.ReadAll(s)
	if err != nil {
		panic(fmt.Sprintf("assets: ioutil.ReadAll failed: %v", err))
	}
	return &image.RGBA{
		Pix:    pix,
		Stride: 4 * imgWidth,
		Rect:   image.Rect(0, 0, imgWidth, imgHeight),
	}
}
