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

// +build js

package web

import (
	"strings"
	"syscall/js"
)

func IsBrowser() bool {
	return true
}

var (
	userAgent = js.Global().Get("navigator").Get("userAgent").String()

	isIOSSafari     bool
	isAndroidChrome bool
)

func init() {
	isIOSSafari = strings.Contains(userAgent, "iPhone")

	if !strings.Contains(userAgent, "Android") {
		isAndroidChrome = false
		return
	}
	if !strings.Contains(userAgent, "Chrome") {
		isAndroidChrome = false
		return
	}
	isAndroidChrome = true
}

func IsIOSSafari() bool {
	return isIOSSafari
}

func IsAndroidChrome() bool {
	return isAndroidChrome
}

func IsMobileBrowser() bool {
	return IsIOSSafari() || IsAndroidChrome()
}
