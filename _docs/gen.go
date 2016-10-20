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

// +build ignore

package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/internal"
)

func execute(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	msg, err := ioutil.ReadAll(stderr)
	if err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("%v: %s", err, string(msg))
	}
	return nil
}

var (
	copyright     = ""
	stableVersion = ""
	devVersion    = ""
)

func init() {
	year, err := internal.LicenseYear()
	if err != nil {
		panic(err)
	}
	copyright = fmt.Sprintf("© %d Hajime Hoshi", year)
}

func init() {
	b, err := exec.Command("git", "tag").Output()
	if err != nil {
		panic(err)
	}
	lastStableVersion := ""
	lastCommitTime := 0
	for _, tag := range strings.Split(string(b), "\n") {
		m := regexp.MustCompile(`^v(\d.+)$`).FindStringSubmatch(tag)
		if m == nil {
			continue
		}
		t, err := exec.Command("git", "log", tag, "-1", "--format=%ct").Output()
		if err != nil {
			panic(err)
		}
		tt, err := strconv.Atoi(strings.TrimSpace(string(t)))
		if err != nil {
			panic(err)
		}
		if lastCommitTime >= tt {
			continue
		}
		lastCommitTime = tt
		lastStableVersion = m[1]
	}
	// See the HEAD commit time
	stableVersion = lastStableVersion
}

func init() {
	b, err := exec.Command("git", "show", "master:version.txt").Output()
	if err != nil {
		panic(err)
	}
	devVersion = strings.TrimSpace(string(b))
}

func comment(text string) template.HTML {
	// http://www.w3.org/TR/html-markup/syntax.html#comments
	// The text part of comments has the following restrictions:
	// * must not start with a ">" character
	// * must not start with the string "->"
	// * must not contain the string "--"
	// * must not end with a "-" character

	for strings.HasPrefix(text, ">") {
		text = text[1:]
	}
	for strings.HasPrefix(text, "->") {
		text = text[2:]
	}
	text = strings.Replace(text, "--", "", -1)
	for strings.HasSuffix(text, "-") {
		text = text[:len(text)-1]
	}
	return template.HTML("<!--\n" + text + "\n-->")
}

func safeHTML(text string) template.HTML {
	return template.HTML(text)
}

type example struct {
	Name        string
	ThumbWidth  int
	ThumbHeight int
}

func (e *example) Width() int {
	return e.ThumbWidth * 2
}

func (e *example) Height() int {
	return e.ThumbHeight * 2
}

const commentForBlocks = `// Please read examples/blocks/main.go and examples/blocks/blocks/*.go
// NOTE: If Gamepad API is available in your browswer, you can use gamepads. Try it out!`

func (e *example) Source() string {
	if e.Name == "blocks" {
		return commentForBlocks
	}

	path := filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "hajimehoshi", "ebiten", "examples", e.Name, "main.go")
	b, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	str := regexp.MustCompile("(?s)^.*?\n\n").ReplaceAllString(string(b), "")
	str = strings.Replace(str, "\t", "        ", -1)
	return str
}

func versions() string {
	return fmt.Sprintf("v%s (dev: v%s)", stableVersion, devVersion)
}

var examples = []example{
	{"alphablending", 320, 240},
	{"audio", 320, 240},
	{"font", 320, 240},
	{"hsv", 320, 240},
	{"hue", 320, 240},
	{"gamepad", 320, 240},
	{"infinitescroll", 320, 240},
	{"keyboard", 320, 240},
	{"life", 320, 240},
	{"masking", 320, 240},
	{"mosaic", 320, 240},
	{"noise", 320, 240},
	{"paint", 320, 240},
	{"perspective", 320, 240},
	{"piano", 320, 240},
	{"rotate", 320, 240},
	{"sprites", 320, 240},
	{"blocks", 256, 240},
}

func clear() error {
	if err := filepath.Walk("public", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if m, _ := regexp.MatchString("~$", path); m {
			return nil
		}
		// Remove auto-generated html files.
		m, err := regexp.MatchString(".html$", path)
		if err != nil {
			return err
		}
		if m {
			return os.Remove(path)
		}
		// Remove example resources that are copied.
		m, err = regexp.MatchString("^public/examples/_resources/images$", path)
		if err != nil {
			return err
		}
		if m {
			if err := os.RemoveAll(path); err != nil {
				return err
			}
			return filepath.SkipDir
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func outputMain() error {
	f, err := os.Create("public/index.html")
	if err != nil {
		return err
	}
	defer f.Close()

	funcs := template.FuncMap{
		"comment":  comment,
		"safeHTML": safeHTML,
	}
	const templatePath = "index.tmpl.html"
	name := filepath.Base(templatePath)
	t, err := template.New(name).Funcs(funcs).ParseFiles(templatePath)
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"Copyright":     copyright,
		"StableVersion": stableVersion,
		"DevVersion":    devVersion,
		"Examples":      examples,
	}
	return t.Funcs(funcs).Execute(f, data)
}

func outputExampleImages() error {
	// TODO: Using cp command might not be portable.
	// Use io.Copy instead.
	const dir = "public/examples"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return execute("cp", "-R", "../examples/_resources/images", "public/examples/_resources/images")
}

func outputExampleContent(e *example) error {
	const dir = "public/examples"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dir, e.Name+".content.html"))
	if err != nil {
		return err
	}
	defer f.Close()

	funcs := template.FuncMap{
		"comment":  comment,
		"safeHTML": safeHTML,
	}
	const templatePath = "examplecontent.tmpl.html"
	name := filepath.Base(templatePath)
	t, err := template.New(name).Funcs(funcs).ParseFiles(templatePath)
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"Copyright": copyright,
		"Example":   e,
	}
	if err := t.Funcs(funcs).Execute(f, data); err != nil {
		return err
	}

	out := filepath.Join(dir, e.Name+".js")
	path := "github.com/hajimehoshi/ebiten/examples/" + e.Name
	if err := execute("gopherjs", "build", "--tags", "example", "-m", "-o", out, path); err != nil {
		return err
	}

	return nil
}

func outputExample(e *example) error {
	const dir = "public/examples"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	f, err := os.Create(filepath.Join(dir, e.Name+".html"))
	if err != nil {
		return err
	}
	defer f.Close()

	funcs := template.FuncMap{
		"comment":  comment,
		"safeHTML": safeHTML,
	}
	const templatePath = "example.tmpl.html"
	name := filepath.Base(templatePath)
	t, err := template.New(name).Funcs(funcs).ParseFiles(templatePath)
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"Copyright": copyright,
		"Example":   e,
	}
	return t.Funcs(funcs).Execute(f, data)
}

func main() {
	// Do not call temporarily.
	// TODO: Uncomment this out after 1.4 stable is released.
	// docs/examples/_resource/images/arcade.png should also be remove.
	/*if err := clear(); err != nil {
		log.Fatal(err)
	}*/
	if err := outputMain(); err != nil {
		log.Fatal(err)
	}
	if err := outputExampleImages(); err != nil {
		log.Fatal(err)
	}
	for _, e := range examples {
		if err := outputExampleContent(&e); err != nil {
			log.Fatal(err)
		}
		if err := outputExample(&e); err != nil {
			log.Fatal(err)
		}
	}
}
