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

// gbmBackend is never instantiated here. It exists so that Run can name the
// same type on every system.
type gbmBackend struct{}

// maybeNewGBMBackend never returns a backend: DRM/KMS with GBM is a Linux
// concept.
func maybeNewGBMBackend(u *UserInterface) (*gbmBackend, error) {
	return nil, errors.New("a DRM/KMS GBM display is a Linux feature")
}

func (b *gbmBackend) run(game Game, options *RunOptions) error {
	return errors.New("ui: a DRM/KMS GBM display is not available in this environment")
}
