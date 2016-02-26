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

// Since js.Object (Program) can't be keys of a map, use integers (programID) instead.

type locationCache struct {
	uniformLocationCache map[programID]map[string]UniformLocation
	attribLocationCache  map[programID]map[string]AttribLocation
}

func newLocationCache() *locationCache {
	return &locationCache{
		uniformLocationCache: map[programID]map[string]UniformLocation{},
		attribLocationCache:  map[programID]map[string]AttribLocation{},
	}
}

type uniformLocationGetter interface {
	getUniformLocation(p Program, location string) UniformLocation
}

// TODO: Rename these functions not to be confusing

func (c *locationCache) GetUniformLocation(g uniformLocationGetter, p Program, location string) UniformLocation {
	id := p.id()
	if _, ok := c.uniformLocationCache[id]; !ok {
		c.uniformLocationCache[id] = map[string]UniformLocation{}
	}
	l, ok := c.uniformLocationCache[id][location]
	if !ok {
		l = g.getUniformLocation(p, location)
		c.uniformLocationCache[id][location] = l
	}
	return l
}

type attribLocationGetter interface {
	getAttribLocation(p Program, location string) AttribLocation
}

func (c *locationCache) GetAttribLocation(g attribLocationGetter, p Program, location string) AttribLocation {
	id := p.id()
	if _, ok := c.attribLocationCache[id]; !ok {
		c.attribLocationCache[id] = map[string]AttribLocation{}
	}
	l, ok := c.attribLocationCache[id][location]
	if !ok {
		l = g.getAttribLocation(p, location)
		c.attribLocationCache[id][location] = l
	}
	return l
}
