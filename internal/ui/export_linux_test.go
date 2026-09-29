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

//go:build !android && !nintendosdk && !playstation5

package ui

// NoWindowSystemSizeAfterCloseForTesting closes a display and returns its cached size.
func NoWindowSystemSizeAfterCloseForTesting(width, height int, closeDisplay func() error) (int, int, error) {
	b := newNoWindowSystemBackend(nil, width, height, nil, nil, closeDisplay)
	if err := b.closeOnMainThread(); err != nil {
		return 0, 0, err
	}
	w, h := b.screenSize()
	return w, h, nil
}
