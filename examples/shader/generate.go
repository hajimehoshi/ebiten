// Copyright 2020 The Ebiten Authors
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

//go:build example
// +build example

package main

//go:generate file2byteslice -package=main -input=default.go -output=default_go.go -var=default_go -buildtags=example
//go:generate file2byteslice -package=main -input=texel.go -output=texel_go.go -var=texel_go -buildtags=example
//go:generate file2byteslice -package=main -input=lighting.go -output=lighting_go.go -var=lighting_go -buildtags=example
//go:generate file2byteslice -package=main -input=radialblur.go -output=radialblur_go.go -var=radialblur_go -buildtags=example
//go:generate file2byteslice -package=main -input=chromaticaberration.go -output=chromaticaberration_go.go -var=chromaticaberration_go -buildtags=example
//go:generate file2byteslice -package=main -input=dissolve.go -output=dissolve_go.go -var=dissolve_go -buildtags=example
//go:generate file2byteslice -package=main -input=water.go -output=water_go.go -var=water_go -buildtags=example
