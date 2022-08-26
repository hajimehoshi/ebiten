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

//go:build !js
// +build !js

package ebiten

import (
	"fmt"
	"os"
	"time"
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


