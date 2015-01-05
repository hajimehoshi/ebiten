// Copyright 2015 Hajime Hoshi
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

package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

var port = flag.Int("port", 8000, "port number")

func init() {
	flag.Parse()
}

var rootPath = ""

func init() {
	_, path, _, _ := runtime.Caller(0)
	rootPath = filepath.Join(filepath.Dir(path), "..")
}

var jsDir = ""

func init() {
	var err error
	jsDir, err = ioutil.TempDir("", "ebiten")
	if err != nil {
		panic(err)
	}
}

func createJSIfNeeded(name string) (string, error) {
	out := filepath.Join(jsDir, name, "main.js")
	stat, err := os.Stat(out)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	if (err != nil && os.IsNotExist(err)) || time.Now().Sub(stat.ModTime()) > 10*time.Second {
		target := "github.com/hajimehoshi/ebiten/example/" + name
		if err := exec.Command("gopherjs", "build", "-o", out, target).Run(); err != nil {
			return "", err
		}
	}
	return out, nil
}

func serveFile(w http.ResponseWriter, path, mime string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w.Header().Set("Content-Type", mime)
	if _, err := io.Copy(w, f); err != nil {
		return err
	}
	return nil
}

func appName(r *http.Request) (string, error) {
	u, err := url.Parse(r.Referer())
	if err != nil {
		return "", err
	}
	q := u.RawQuery
	if q == "" {
		q = "blocks"
	}
	return q, nil
}

func serveMainJS(w http.ResponseWriter, r *http.Request) {
	name, err := appName(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}
	out, err := createJSIfNeeded(name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}
	if err := serveFile(w, out, "text/javascript"); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}
}

func serveMainJSMap(w http.ResponseWriter, r *http.Request) {
	name, err := appName(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err.Error())
		return
	}
	out, err := createJSIfNeeded(name)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}
	out += ".map"
	if err := serveFile(w, out, "application/octet-stream"); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}
}

func main() {
	http.HandleFunc("/main.js", serveMainJS)
	http.HandleFunc("/main.js.map", serveMainJSMap)
	http.Handle("/", http.FileServer(http.Dir(rootPath)))
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))
}
