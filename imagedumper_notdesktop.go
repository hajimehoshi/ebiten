// Copyright 2018 The Ebiten Authors
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

// +build android js ios

package ebiten

type imageDumper struct {
	f func(screen *Image) error
}

func (i *imageDumper) update(screen *Image) error {
	return i.f(screen)
}

func (i *imageDumper) dump(screen *Image) error {
	// Do nothing
	return nil
}
