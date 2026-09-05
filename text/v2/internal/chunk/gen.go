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

// gen.go emits aterm.go and sterm.go from the UCD
// SentenceBreakProperty.txt file. Bump ucdVersion to upgrade to a
// newer Unicode revision; the file structure is stable across
// revisions.
package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	ucdVersion = "18.0.0"
	ucdURL     = "https://www.unicode.org/Public/" + ucdVersion + "/ucd/auxiliary/SentenceBreakProperty.txt"
)

// emit configures one generated file: which property to extract, the
// Go function name to emit, and the output filename.
type emit struct {
	prop     string // UAX #29 property name as it appears in the UCD file
	funcName string // Go identifier for the predicate
	filename string // output file relative to this directory
}

var emits = []emit{
	{prop: "ATerm", funcName: "isATerm", filename: "aterm.go"},
	{prop: "STerm", funcName: "isSTerm", filename: "sterm.go"},
}

// entry is one property assignment in the UCD file: a single
// codepoint (start == end) or an inclusive range, plus the
// human-readable name extracted from the trailing comment.
type entry struct {
	start, end rune
	name       string
}

func main() {
	if err := xmain(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func xmain() error {
	byProp, err := fetchProps()
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	for _, e := range emits {
		entries := byProp[e.prop]
		if len(entries) == 0 {
			return fmt.Errorf("no entries for property %q", e.prop)
		}
		if err := writeFile(e.filename, e.funcName, e.prop, entries); err != nil {
			return fmt.Errorf("write %s: %w", e.filename, err)
		}
	}
	return nil
}

// fetchProps downloads the UCD file and groups entries by property
// name (the second column).
func fetchProps() (map[string][]entry, error) {
	resp, err := http.Get(ucdURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", ucdURL, resp.Status)
	}

	byProp := map[string][]entry{}
	s := bufio.NewScanner(resp.Body)
	for s.Scan() {
		line := s.Text()
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		semi := strings.Index(line, ";")
		if semi < 0 {
			continue
		}
		cpField := strings.TrimSpace(line[:semi])
		rest := line[semi+1:]
		var prop, name string
		if hash := strings.Index(rest, "#"); hash >= 0 {
			prop = strings.TrimSpace(rest[:hash])
			name = strings.TrimSpace(rest[hash+1:])
		} else {
			prop = strings.TrimSpace(rest)
		}
		start, end, err := parseCodepoints(cpField)
		if err != nil {
			return nil, fmt.Errorf("parse %q: %w", cpField, err)
		}
		byProp[prop] = append(byProp[prop], entry{
			start: start,
			end:   end,
			name:  cleanName(start == end, name),
		})
	}
	return byProp, s.Err()
}

// parseCodepoints accepts "NNNN" or "NNNN..MMMM" and returns the
// inclusive range; for a single codepoint, start == end.
func parseCodepoints(s string) (rune, rune, error) {
	if i := strings.Index(s, ".."); i >= 0 {
		a, err := strconv.ParseInt(s[:i], 16, 32)
		if err != nil {
			return 0, 0, err
		}
		b, err := strconv.ParseInt(s[i+2:], 16, 32)
		if err != nil {
			return 0, 0, err
		}
		return rune(a), rune(b), nil
	}
	v, err := strconv.ParseInt(s, 16, 32)
	if err != nil {
		return 0, 0, err
	}
	return rune(v), rune(v), nil
}

// cleanName strips the leading General_Category abbreviation (and
// the count column for ranges) from a UCD trailing comment, leaving
// just the character name (or "NAME..NAME2" for ranges).
func cleanName(single bool, s string) string {
	if single {
		fields := strings.Fields(s)
		if len(fields) <= 1 {
			return s
		}
		return strings.Join(fields[1:], " ")
	}
	if i := strings.Index(s, "] "); i >= 0 {
		return s[i+2:]
	}
	return s
}

func writeFile(filename, funcName, propName string, entries []entry) error {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].start < entries[j].start
	})
	var ranges, singles []entry
	for _, e := range entries {
		if e.start == e.end {
			singles = append(singles, e)
		} else {
			ranges = append(ranges, e)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, `// Copyright 2026 The Ebitengine Authors
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

// Code generated by gen.go; DO NOT EDIT.

package chunk

// %s reports whether r is in UAX #29's Sentence_Break = %s class.
func %s(r rune) bool {
`, funcName, propName, funcName)

	if len(ranges) > 0 {
		b.WriteString("\tswitch {\n")
		for _, e := range ranges {
			fmt.Fprintf(&b, "\tcase r >= 0x%04X && r <= 0x%04X: // %s\n\t\treturn true\n", e.start, e.end, e.name)
		}
		b.WriteString("\t}\n")
	}
	if len(singles) > 0 {
		b.WriteString("\tswitch r {\n")
		for _, e := range singles {
			fmt.Fprintf(&b, "\tcase 0x%04X: // %s\n\t\treturn true\n", e.start, e.name)
		}
		b.WriteString("\t}\n")
	}
	b.WriteString("\treturn false\n}\n")

	return os.WriteFile(filename, []byte(b.String()), 0644)
}
