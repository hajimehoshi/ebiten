// Copyright 2022 The Ebitengine Authors
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

package audio

import (
	_ "embed"
)

var (
	//go:embed jab.wav
	Jab_wav []byte

	//go:embed jump.ogg
	Jump_ogg []byte

	//go:embed ragtime.mp3
	Ragtime_mp3 []byte

	//go:embed ragtime.ogg
	Ragtime_ogg []byte
)
