// Copyright 2016 Hajime Hoshi
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
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"strconv"
)

var port = flag.Int("port", 8000, "port number")

func init() {
	flag.Parse()
}

var rootPath = ""

func init() {
	_, path, _, _ := runtime.Caller(0)
	rootPath = filepath.Join(filepath.Dir(path), "..", "..", "docs")
}

func main() {
	http.Handle("/", http.FileServer(http.Dir(rootPath)))
	fmt.Printf("http://localhost:%d/\n", *port)
	log.Fatal(http.ListenAndServe("localhost:"+strconv.Itoa(*port), nil))
}
