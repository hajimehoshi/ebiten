// Copyright 2014 Hajime Hoshi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build ignore

package main

import (
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	outputPath   = "public/index.html"
	templatePath = "index.tmpl.html"
)

var license = ""

func init() {
	b, err := ioutil.ReadFile("../license.txt")
	if err != nil {
		panic(err)
	}
	license = string(b)

	// TODO: Year check
}

var stableVersion = ""

var devVersion = ""

func init() {
	b, err := ioutil.ReadFile("../version.txt")
	if err != nil {
		panic(err)
	}
	devVersion = string(b)
}

func comment(text string) template.HTML {
	// TODO: text should be escaped
	return template.HTML("<!--" + text + "-->")
}

func safeHTML(text string) template.HTML {
	return template.HTML(text)
}

type example struct {
	Name string
}

func (e *example) Width() int {
	if e.Name == "blocks" {
		return 256
	}
	return 320
}

func (e *example) Height() int {
	if e.Name == "blocks" {
		return 240
	}
	return 240
}

func (e *example) Source() string {
	if e.Name == "blocks" {
		return "// Please read example/blocks/main.go and example/blocks/blocks/*.go"
	}

	path := filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "hajimehoshi", "ebiten", "example", e.Name, "main.go")
	b, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	str := regexp.MustCompile("(?s)^.*?\n\n").ReplaceAllString(string(b), "")
	return str
}

func versions() string {
	vers := []string{}
	if stableVersion != "" {
		vers = append(vers, "Stable: "+stableVersion)
	}
	if devVersion != "" {
		vers = append(vers, "Development: "+devVersion)
	}
	return strings.Join(vers, ", ")
}

func main() {
	f, err := os.Create(outputPath)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	funcs := template.FuncMap{
		"comment":  comment,
		"safeHTML": safeHTML,
	}
	name := filepath.Base(templatePath)
	t, err := template.New(name).Funcs(funcs).ParseFiles(templatePath)
	if err != nil {
		log.Fatal(err)
	}

	examples := []example{
		{Name: "blocks"},
		{Name: "hue"},
		{Name: "mosaic"},
		{Name: "perspective"},
		{Name: "rotate"},
	}
	data := map[string]interface{}{
		"License":  license,
		"Versions": versions(),
		"Examples": examples,
	}
	if err := t.Funcs(funcs).Execute(f, data); err != nil {
		log.Fatal(err)
	}
}
