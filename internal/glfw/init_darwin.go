// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Ebitengine Authors

package glfw

import (
	"bytes"
	"os"
	"path/filepath"
)

func init() {
	// Change the working directory to the application bundle's resources
	// directory at package initialization, so that it is already in effect when
	// main runs. GLFW is otherwise initialized lazily in RunGame, which would
	// defer the chdir until after main has started.
	//
	// TODO: Remove this implicit chdir in the future (#2919).
	changeToResourcesDirectory()
}

// changeToResourcesDirectory changes the current working directory to the
// application bundle's resources directory, if present.
//
// This mirrors GLFW's GLFW_COCOA_CHDIR_RESOURCES init hint, which defaults to
// true. An unbundled binary has no such directory and is left untouched.
func changeToResourcesDirectory() {
	bundle := cfBundleGetMainBundle()
	if bundle == 0 {
		return
	}

	resourcesURL := cfBundleCopyResourcesDirectoryURL(bundle)
	if resourcesURL == 0 {
		return
	}
	defer cfRelease(resourcesURL)

	// MAXPATHLEN on macOS.
	var buf [1024]byte
	if !cfURLGetFileSystemRepresentation(resourcesURL, true, &buf[0], len(buf)) {
		return
	}

	n := bytes.IndexByte(buf[:], 0)
	if n < 0 {
		return
	}
	path := string(buf[:n])

	if filepath.Base(path) != "Resources" {
		return
	}

	_ = os.Chdir(path)
}
