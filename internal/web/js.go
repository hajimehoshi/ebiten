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
	"runtime"
	"strings"

	"github.com/gopherjs/gopherwasm/js"
)

func IsGopherJS() bool {
	return runtime.GOOS != "js"
}

func IsBrowser() bool {
	return true
}

func IsIOSSafari() bool {
	ua := js.Global().Get("navigator").Get("userAgent").String()
	if !strings.Contains(ua, "iPhone") {
		return false
	}
	return true
}

func IsAndroidChrome() bool {
	ua := js.Global().Get("navigator").Get("userAgent").String()
	if !strings.Contains(ua, "Android") {
		return false
	}
	if !strings.Contains(ua, "Chrome") {
		return false
	}
	return true
}

func IsMobileBrowser() bool {
	return IsIOSSafari() || IsAndroidChrome()
}
