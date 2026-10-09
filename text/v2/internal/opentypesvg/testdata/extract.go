// Copyright 2026 The Ebitengine Authors
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

//go:build ignore

// Extract the font records described in README.md and inventory into the current directory:
//
// go run extract.go /path/NotoColorEmoji-Regular.ttf /path/TwitterColorEmoji-SVGinOT.ttf
package main

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type fontSpec struct {
	name     string
	selected []uint16
}

var fonts = []fontSpec{
	{
		name:     "noto",
		selected: []uint16{2, 3330, 3629, 3682, 3737},
	},
	{
		name:     "twemoji",
		selected: []uint16{692, 839, 1039, 1338, 1355},
	},
}

type document struct {
	File       string    `json:"file"`
	GlyphRange [2]uint16 `json:"glyph_range"`
	Bytes      int       `json:"bytes"`
}

type inventory struct {
	Font            string         `json:"font"`
	UnitsPerEm      uint16         `json:"units_per_em"`
	Records         uint16         `json:"records"`
	GzipRecords     int            `json:"gzip_records"`
	UniqueDocuments int            `json:"unique_documents"`
	Elements        map[string]int `json:"elements"`
	Attributes      map[string]int `json:"attributes"`
	PathCommands    map[string]int `json:"path_commands"`
	Selected        []document     `json:"selected"`
}

func countConstructs(data []byte, elements, attributes, commands map[string]int) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		element, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		elements[element.Name.Local]++
		for _, attr := range element.Attr {
			if attr.Name.Space == "xmlns" || (attr.Name.Space == "" && attr.Name.Local == "xmlns") {
				continue
			}
			attributes[attr.Name.Local]++
			if attr.Name.Space == "" && attr.Name.Local == "d" {
				for _, c := range attr.Value {
					if strings.ContainsRune("MmZzLlHhVvCcSsQqTtAa", c) {
						commands[string(c)]++
					}
				}
			}
		}
	}
}

func decompress(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func extract(path string, spec fontSpec) (inventory, map[string][]byte, error) {
	var result inventory
	data, err := os.ReadFile(path)
	if err != nil {
		return result, nil, err
	}
	tables := map[string][]byte{}
	be := binary.BigEndian
	if len(data) < 12 || 12+16*int(be.Uint16(data[4:])) > len(data) {
		return result, nil, fmt.Errorf("%s: truncated font directory", path)
	}
	for i := range int(be.Uint16(data[4:])) {
		record := data[12+16*i:]
		offset, length := be.Uint32(record[8:]), be.Uint32(record[12:])
		if uint64(offset)+uint64(length) > uint64(len(data)) {
			return result, nil, fmt.Errorf("%s: table outside font data", path)
		}
		tables[string(record[:4])] = data[int(offset) : int(offset)+int(length)]
	}
	svg := tables["SVG "]
	if len(svg) < 10 || len(tables["head"]) < 20 {
		return result, nil, fmt.Errorf("%s: missing or truncated SVG/head table", path)
	}
	listOffset := uint64(be.Uint32(svg[2:]))
	if listOffset+2 > uint64(len(svg)) {
		return result, nil, fmt.Errorf("%s: SVG document list outside table", path)
	}
	list := svg[listOffset:]
	if 2+12*int(be.Uint16(list)) > len(list) {
		return result, nil, fmt.Errorf("%s: truncated SVG document list", path)
	}
	result = inventory{
		Font:         spec.name,
		UnitsPerEm:   be.Uint16(tables["head"][18:]),
		Records:      be.Uint16(list),
		Elements:     map[string]int{},
		Attributes:   map[string]int{},
		PathCommands: map[string]int{},
	}
	seen := map[[2]uint32]bool{}
	documents := map[string][]byte{}
	for i := range int(result.Records) {
		record := list[2+12*i:]
		lo, hi := be.Uint16(record), be.Uint16(record[2:])
		offset, length := be.Uint32(record[4:]), be.Uint32(record[8:])
		if uint64(offset)+uint64(length) > uint64(len(list)) {
			return result, nil, fmt.Errorf("%s: SVG document outside table", path)
		}
		raw := list[int(offset) : int(offset)+int(length)]
		if bytes.HasPrefix(raw, []byte{0x1f, 0x8b}) {
			result.GzipRecords++
			raw, err = decompress(raw)
			if err != nil {
				return result, nil, err
			}
		}
		location := [2]uint32{offset, length}
		if !seen[location] {
			seen[location] = true
			if err := countConstructs(raw, result.Elements, result.Attributes, result.PathCommands); err != nil {
				return result, nil, err
			}
		}
		if !slices.Contains(spec.selected, lo) {
			continue
		}
		filename := fmt.Sprintf("real/%s-%d-%d.svg", spec.name, lo, hi)
		documents[filename] = raw
		result.Selected = append(result.Selected, document{
			File:       filename,
			GlyphRange: [2]uint16{lo, hi},
			Bytes:      len(raw),
		})
	}
	if len(result.Selected) != len(spec.selected) {
		return result, nil, fmt.Errorf("%s: missing selected records", spec.name)
	}
	result.UniqueDocuments = len(seen)
	return result, documents, nil
}

func run() error {
	if len(os.Args) != len(fonts)+1 {
		return fmt.Errorf("usage: go run extract.go /path/NotoColorEmoji-Regular.ttf /path/TwitterColorEmoji-SVGinOT.ttf (run from testdata)")
	}
	var inventories []inventory
	documents := map[string][]byte{}
	for i, spec := range fonts {
		result, extracted, err := extract(os.Args[i+1], spec)
		if err != nil {
			return err
		}
		inventories = append(inventories, result)
		for name, data := range extracted {
			documents[name] = data
		}
	}
	data, err := json.MarshalIndent(inventories, "", "  ")
	if err != nil {
		return err
	}
	// Read both fonts and their selected records before writing files.
	for name, data := range documents {
		if err := os.WriteFile(filepath.FromSlash(name), data, 0o644); err != nil {
			return err
		}
	}
	return os.WriteFile("inventory.json", append(data, '\n'), 0o644)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
