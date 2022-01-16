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

//go:build !android && !js && !ios
// +build !android,!js,!ios

package ebiten

import (
	"fmt"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
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

func takeScreenshot(screen *Image) error {
	newname, err := availableFilename("screenshot_", ".png")
	if err != nil {
		return err
	}

	blackbg := !IsScreenTransparent()
	if err := screen.mipmap.DumpScreenshot(newname, blackbg); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(os.Stderr, "Saved screenshot: %s\n", newname); err != nil {
		return err
	}
	return nil
}

func dumpInternalImages() error {
	dir, err := availableFilename("internalimages_", "")
	if err != nil {
		return err
	}

	if err := os.Mkdir(dir, 0755); err != nil {
		return err
	}

	if err := atlas.DumpImages(dir); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(os.Stderr, "Dumped the internal images at: %s\n", dir); err != nil {
		return err
	}
	return nil
}

type imageDumper struct {
	g Game

	keyState map[Key]int

	hasScreenshotKey bool
	screenshotKey    Key
	toTakeScreenshot bool

	hasDumpInternalImagesKey bool
	dumpInternalImagesKey    Key
	toDumpInternalImages     bool

	err error
}

func (i *imageDumper) update() error {
	if i.err != nil {
		return i.err
	}

	const (
		envScreenshotKey     = "EBITEN_SCREENSHOT_KEY"
		envInternalImagesKey = "EBITEN_INTERNAL_IMAGES_KEY"
	)

	if err := i.g.Update(); err != nil {
		return err
	}

	// If keyState is nil, all values are not initialized.
	if i.keyState == nil {
		i.keyState = map[Key]int{}

		if keyname := os.Getenv(envScreenshotKey); keyname != "" {
			if key, ok := keyNameToKeyCode(keyname); ok {
				i.hasScreenshotKey = true
				i.screenshotKey = key
			}
		}

		if keyname := os.Getenv(envInternalImagesKey); keyname != "" {
			if isDebug() {
				if key, ok := keyNameToKeyCode(keyname); ok {
					i.hasDumpInternalImagesKey = true
					i.dumpInternalImagesKey = key
				}
			} else {
				fmt.Fprintf(os.Stderr, "%s is disabled. Specify a build tag 'ebitendebug' to enable it.\n", envInternalImagesKey)
			}
		}
	}

	keys := map[Key]struct{}{}
	if i.hasScreenshotKey {
		keys[i.screenshotKey] = struct{}{}
	}
	if i.hasDumpInternalImagesKey {
		keys[i.dumpInternalImagesKey] = struct{}{}
	}

	for key := range keys {
		if IsKeyPressed(key) {
			i.keyState[key]++
			if i.keyState[key] == 1 {
				if i.hasScreenshotKey && key == i.screenshotKey {
					i.toTakeScreenshot = true
				}
				if i.hasDumpInternalImagesKey && key == i.dumpInternalImagesKey {
					i.toDumpInternalImages = true
				}
			}
		} else {
			i.keyState[key] = 0
		}
	}
	return nil
}

func (i *imageDumper) dump(screen *Image) error {
	if i.toTakeScreenshot {
		i.toTakeScreenshot = false
		if err := takeScreenshot(screen); err != nil {
			return err
		}
	}

	if i.toDumpInternalImages {
		i.toDumpInternalImages = false
		if err := dumpInternalImages(); err != nil {
			return err
		}
	}

	return nil
}
