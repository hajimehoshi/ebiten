// Copyright 2023 The Ebitengine Authors
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

//go:build !js && !playstation5

package gl

import (
	"fmt"
)

type procAddressGetter struct {
	ctx *defaultContext
	err error
}

func (p *procAddressGetter) get(name string) uintptr {
	proc, err := p.ctx.getProcAddress(name)
	if err != nil {
		p.err = fmt.Errorf("gl: %s is missing: %w", name, err)
		return 0
	}
	if proc == 0 {
		p.err = fmt.Errorf("gl: %s is missing", name)
		return 0
	}
	return proc
}

func (p *procAddressGetter) error() error {
	return p.err
}
