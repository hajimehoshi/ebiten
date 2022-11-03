// Copyright 2021 The Ebiten Authors
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

//go:build microsoftgdk || nintendosdk

// This file is for some special environments using 'microsoftgdk' or 'nintendosdk'.
// You usually don't have to care about this file.
// Actually this example works without this file in usual cases.

package main

import "C"

//export GoMain
func GoMain() {
	main()
}
