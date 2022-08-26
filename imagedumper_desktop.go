// Copyright 2022 The Ebitengine Authors
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

//go:build !android && !ios && !js
// +build !android,!ios,!js

package ebiten

import (
	"fmt"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// availableFilename returns a filename that is valid as a new file or directory.
func availableFilename(prefix, postfix string) (string, error) {
	const datetimeFormat = "20060102030405"

	now := time.Now()
	name := fmt.Sprintf("%s%s%s", prefix, now.Format(datetimeFormat), postfix)
	for i := 1; ; i++ {
		if _, err := os.Stat(name); err != nil {
			if os.IsNotExist(err) {
				break
			}
			if !os.IsNotExist(err) {
				return "", err
			}
		}
		name = fmt.Sprintf("%s%s_%d%s", prefix, now.Format(datetimeFormat), i, postfix)
	}
	return name, nil
}

func dumpInternalImages() error {
	dir, err := availableFilename("internalimages_", "")
	if err != nil {
		return err
	}

	if err := os.Mkdir(dir, 0755); err != nil {
		return err
	}

	if err := ui.DumpImages(dir); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(os.Stderr, "Dumped the internal images at: %s\n", dir); err != nil {
		return err
	}
	return nil
}
