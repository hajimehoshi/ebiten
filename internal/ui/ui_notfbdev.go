// Copyright 2026 The Ebitengine Authors
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

//go:build !linux || android || nintendosdk || playstation5

package ui

import (
	"errors"
)

// fbdevBackend is never instantiated here. It exists so that Run can name the
// same type on every system.
type fbdevBackend struct{}

// maybeNewFbdevBackend never returns a backend: a framebuffer device is a Linux
// concept.
func maybeNewFbdevBackend(u *UserInterface) (*fbdevBackend, error) {
	return nil, errors.New("a framebuffer device is a Linux feature")
}

func (b *fbdevBackend) run(game Game, options *RunOptions) error {
	return errors.New("ui: a framebuffer device is not available in this environment")
}
