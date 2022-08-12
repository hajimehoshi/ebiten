// Copyright 2019 The Ebiten Authors
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

//go:generate go run github.com/hajimehoshi/file2byteslice/cmd/file2byteslice -package=vorbis_test -input=./test_mono.ogg -output=./testmonoogg_test.go -var=test_mono_ogg
//go:generate go run github.com/hajimehoshi/file2byteslice/cmd/file2byteslice -package=vorbis_test -input=./test_tooshort.ogg -output=./testtooshortogg_test.go -var=test_tooshort_ogg
//go:generate gofmt -s -w .

package vorbis
