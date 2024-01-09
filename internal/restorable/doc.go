// Copyright 2017 The Ebiten Authors
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

// Package restorable used to offer an Image struct that stores image commands
// and restores its pixel data from the commands when context lost happens.
//
// However, now Ebitengine doesn't handle context losts, and this package is
// just a thin wrapper.
//
// TODO: Integrate this package into internal/atlas and internal/graphicscommand (#805).
package restorable
