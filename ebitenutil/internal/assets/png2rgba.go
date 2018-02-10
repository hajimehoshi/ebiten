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

// +build ignore

package main

import (
	"compress/gzip"
	"flag"
	"image/png"
	"io"
	"os"
)

var (
	inputFilename  = flag.String("input", "", "input file name")
	outputFilename = flag.String("output", "", "output file name")
)

func run() error {
	var in io.Reader
	if *inputFilename != "" {
		f, err := os.Open(*inputFilename)
		if err != nil {
			return err
		}
		defer f.Close()
		in = f
	} else {
		in = os.Stdin
	}

	var out io.Writer
	if *outputFilename != "" {
		f, err := os.Create(*outputFilename)
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
	} else {
		out = os.Stdout
	}

	img, err := png.Decode(in)
	if err != nil {
		return err
	}

	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	rgba := make([]byte, w*h*4)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			r, g, b, a := img.At(i, j).RGBA()
			rgba[4*(w*j+i)] = byte(r >> 8)
			rgba[4*(w*j+i)+1] = byte(g >> 8)
			rgba[4*(w*j+i)+2] = byte(b >> 8)
			rgba[4*(w*j+i)+3] = byte(a >> 8)
		}
	}

	cw := gzip.NewWriter(out)
	defer cw.Close()

	if _, err := cw.Write(rgba); err != nil {
		return err
	}
	return nil
}

func main() {
	flag.Parse()
	if err := run(); err != nil {
		panic(err)
	}
}
