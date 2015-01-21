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

package opengl

// Since js.Object (Program) can't be keys of a map, use integers (ProgramID) instead.

var uniformLocationCache = map[ProgramID]map[string]UniformLocation{}
var attribLocationCache = map[ProgramID]map[string]AttribLocation{}

type UniformLocationGetter interface {
	GetUniformLocation(p Program, location string) UniformLocation
}

func GetUniformLocation(g UniformLocationGetter, p Program, location string) UniformLocation {
	id := GetProgramID(p)
	if _, ok := uniformLocationCache[id]; !ok {
		uniformLocationCache[id] = map[string]UniformLocation{}
	}
	l, ok := uniformLocationCache[id][location]
	if !ok {
		l = g.GetUniformLocation(p, location)
		uniformLocationCache[id][location] = l
	}
	return l
}

type AttribLocationGetter interface {
	GetAttribLocation(p Program, location string) AttribLocation
}

func GetAttribLocation(g AttribLocationGetter, p Program, location string) AttribLocation {
	id := GetProgramID(p)
	if _, ok := attribLocationCache[id]; !ok {
		attribLocationCache[id] = map[string]AttribLocation{}
	}
	l, ok := attribLocationCache[id][location]
	if !ok {
		l = g.GetAttribLocation(p, location)
		attribLocationCache[id][location] = l
	}
	return l
}
