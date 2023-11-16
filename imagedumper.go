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

package ebiten

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/debug"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

func datetimeForFilename() string {
	const datetimeFormat = "20060102150405"
	now := time.Now()
	return now.Format(datetimeFormat)
}

func takeScreenshot(screen *Image, transparent bool) error {
	name := "screenshot_" + datetimeForFilename() + ".png"
	// Use the home directory for mobiles as a provisional implementation.
	if runtime.GOOS == "android" || runtime.GOOS == "ios" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		name = filepath.Join(home, name)
	}
	dumpedName, err := screen.image.DumpScreenshot(name, !transparent)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(os.Stderr, "Saved screenshot: %s\n", dumpedName); err != nil {
		return err
	}
	return nil
}

func dumpInternalImages() error {
	dumpedDir, err := ui.Get().DumpImages("internalimages_" + datetimeForFilename())
	if err != nil {
		return err
	}
	// Use the home directory for mobiles as a provisional implementation.
	if runtime.GOOS == "android" || runtime.GOOS == "ios" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		dumpedDir = filepath.Join(home, dumpedDir)
	}
	if _, err := fmt.Fprintf(os.Stderr, "Dumped the internal images at: %s\n", dumpedDir); err != nil {
		return err
	}
	return nil
}

type imageDumper struct {
	keyState map[Key]int

	hasScreenshotKey bool
	screenshotKey    Key
	toTakeScreenshot bool

	hasDumpInternalImagesKey bool
	dumpInternalImagesKey    Key
	toDumpInternalImages     bool

	err error
}

func envScreenshotKey() string {
	if env := os.Getenv("EBITENGINE_SCREENSHOT_KEY"); env != "" {
		return env
	}
	// For backward compatibility, read the EBITEN_ version.
	return os.Getenv("EBITEN_SCREENSHOT_KEY")
}

func envInternalImagesKey() string {
	if env := os.Getenv("EBITENGINE_INTERNAL_IMAGES_KEY"); env != "" {
		return env
	}
	// For backward compatibility, read the EBITEN_ version.
	return os.Getenv("EBITEN_INTERNAL_IMAGES_KEY")
}

func (i *imageDumper) update() error {
	if i.err != nil {
		return i.err
	}

	// If keyState is nil, all values are not initialized.
	if i.keyState == nil {
		i.keyState = map[Key]int{}

		if keyname := envScreenshotKey(); keyname != "" {
			if key, ok := keyNameToKeyCode(keyname); ok {
				i.hasScreenshotKey = true
				i.screenshotKey = key
			}
		}

		if keyname := envInternalImagesKey(); keyname != "" {
			if debug.IsDebug {
				if key, ok := keyNameToKeyCode(keyname); ok {
					i.hasDumpInternalImagesKey = true
					i.dumpInternalImagesKey = key
				}
			} else {
				fmt.Fprintf(os.Stderr, "EBITENGINE_INTERNAL_IMAGES_KEY is disabled. Specify a build tag 'ebitenginedebug' to enable it.\n")
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

func (i *imageDumper) dump(screen *Image, transparent bool) error {
	if i.toTakeScreenshot {
		i.toTakeScreenshot = false
		if err := takeScreenshot(screen, transparent); err != nil {
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
