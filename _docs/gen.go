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
)

const (
	outputPath   = "public/index.html"
	templatePath = "index_tmpl.html"
)

// TODO: License should be on another file
const license = `Copyright 2014 Hajime Hoshi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.`

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
		"Examples": examples,
	}
	if err := t.Funcs(funcs).Execute(f, data); err != nil {
		log.Fatal(err)
	}
}
