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

//go:generate file2byteslice -package=main -input=gobind.go -output=gobind.go.go -var gobind_go

//go:generate file2byteslice -package=main -input=coffeecatch/coffeecatch.c -output=coffeecatch.c.go -var coffeecatch_c -buildtags=ebitenmobilegobind
//go:generate file2byteslice -package=main -input=coffeecatch.c.go -output=coffeecatch.c.go.go -var coffeecatch_c_go
//go:generate rm coffeecatch.c.go

//go:generate file2byteslice -package=main -input=coffeecatch/coffeecatch.h -output=coffeecatch.h.go -var coffeecatch_h -buildtags=ebitenmobilegobind
//go:generate file2byteslice -package=main -input=coffeecatch.h.go -output=coffeecatch.h.go.go -var coffeecatch_h_go
//go:generate rm coffeecatch.h.go

//go:generate file2byteslice -package=main -input=coffeecatch/coffeejni.c -output=coffeejni.c.go -var coffeejni_c -buildtags=ebitenmobilegobind
//go:generate file2byteslice -package=main -input=coffeejni.c.go -output=coffeejni.c.go.go -var coffeejni_c_go
//go:generate rm coffeejni.c.go

//go:generate file2byteslice -package=main -input=coffeecatch/coffeejni.h -output=coffeejni.h.go -var coffeejni_h -buildtags=ebitenmobilegobind
//go:generate file2byteslice -package=main -input=coffeejni.h.go -output=coffeejni.h.go.go -var coffeejni_h_go
//go:generate rm coffeejni.h.go

package main
