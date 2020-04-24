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

//go:generate file2byteslice -package=main -input=gogame.cs -output=gogame.cs.go -var=gogame_cs
//go:generate file2byteslice -package=main -input=program.cs -output=program.cs.go -var=program_cs
//go:generate file2byteslice -package=main -input=project.csproj -output=project.csproj.go -var=project_csproj
//go:generate file2byteslice -package=main -input=content.mgcb -output=content.mgcb.go -var=content_mgcb
//go:generate file2byteslice -package=main -input=shader.fx -output=shader.fx.go -var=shader_fx
//go:generate gofmt -s -w .

package main
