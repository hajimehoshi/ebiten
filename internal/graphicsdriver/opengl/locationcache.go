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

//go:build !playstation5

package opengl

type locationCache struct {
	uniformLocationCache map[program]map[string]uniformLocation
}

func newLocationCache() *locationCache {
	return &locationCache{
		uniformLocationCache: map[program]map[string]uniformLocation{},
	}
}

func (c *locationCache) GetUniformLocation(context *context, p program, location string) uniformLocation {
	if _, ok := c.uniformLocationCache[p]; !ok {
		c.uniformLocationCache[p] = map[string]uniformLocation{}
	}
	l, ok := c.uniformLocationCache[p][location]
	if !ok {
		l = uniformLocation(context.ctx.GetUniformLocation(uint32(p), location))
		c.uniformLocationCache[p][location] = l
	}
	return l
}

func (c *locationCache) deleteProgram(p program) {
	delete(c.uniformLocationCache, p)
}
