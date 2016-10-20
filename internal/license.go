// Copyright 2015 Hajime Hoshi
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

package internal

import (
	"regexp"
	"strconv"
	"strings"
)

const license = `Copyright 2013 The Ebiten Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
`

// LicenseComment returns the license string in Go comment style.
func LicenseComment() (string, error) {
	lines := strings.Split(license, "\n")
	return "// " + strings.Join(lines[:len(lines)-1], "\n// "), nil
}

func LicenseYear() (int, error) {
	license, err := LicenseComment()
	if err != nil {
		return 0, err
	}
	year, err := strconv.Atoi(regexp.MustCompile(`^// Copyright (\d+)`).FindStringSubmatch(license)[1])
	if err != nil {
		return 0, err
	}
	return year, nil
}
