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

// +build js !wasm

package stb

// stbvorbis.js
// URL:     https://github.com/hajimehoshi/stbvorbis.js
// License: Apache License 2.0
// Commit:  ac1c2ee9d24eb6085eb1e968f55e0fb32cacc03a

//go:generate file2byteslice -package=stb -input=stbvorbis.js -output=stbvorbis.js.go -var=stbvorbis_js -buildtags "js !wasm"
