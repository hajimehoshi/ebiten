// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Ebitengine Authors

package glfw_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	// Importing the package links its init into the test binary, so the
	// re-executed child below performs the resources-directory chdir.
	_ "github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

// chdirChildEnv, when set in the environment, makes the test binary report its
// working directory and exit, standing in for a bundled application's main.
const chdirChildEnv = "EBITENGINE_GLFW_CHDIR_TEST_CHILD"

func TestMain(m *testing.M) {
	if os.Getenv(chdirChildEnv) != "" {
		wd, err := os.Getwd()
		if err != nil {
			_, _ = os.Stderr.WriteString(err.Error())
			os.Exit(1)
		}
		_, _ = os.Stdout.WriteString(wd)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// TestChangeToResourcesDirectoryRunsBeforeMain checks that a bundled
// application's working directory is the bundle's Contents/Resources directory
// by the time main runs, even when launched from an unrelated directory as
// Launch Services does.
func TestChangeToResourcesDirectoryRunsBeforeMain(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	// Construct a minimal .app bundle around a copy of the test binary.
	const appName = "Test.app"
	appDir := filepath.Join(t.TempDir(), appName)
	macOSDir := filepath.Join(appDir, "Contents", "MacOS")
	resourcesDir := filepath.Join(appDir, "Contents", "Resources")
	if err := os.MkdirAll(macOSDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(resourcesDir, 0o755); err != nil {
		t.Fatal(err)
	}

	const exeName = "Test"
	bundledExe := filepath.Join(macOSDir, exeName)
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bundledExe, data, 0o755); err != nil {
		t.Fatal(err)
	}

	const infoPlist = `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key>
	<string>` + exeName + `</string>
	<key>CFBundleIdentifier</key>
	<string>com.ebitengine.glfw.chdirtest</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
</dict>
</plist>
`
	if err := os.WriteFile(filepath.Join(appDir, "Contents", "Info.plist"), []byte(infoPlist), 0o644); err != nil {
		t.Fatal(err)
	}

	// Launch the bundled binary from the root directory, mimicking how Launch
	// Services starts a .app.
	cmd := exec.Command(bundledExe)
	cmd.Dir = "/"
	cmd.Env = append(os.Environ(), chdirChildEnv+"=1")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("running bundled binary: %v", err)
	}

	got, err := filepath.EvalSymlinks(strings.TrimSpace(string(out)))
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(resourcesDir)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("working directory at main = %q, want %q", got, want)
	}
}
