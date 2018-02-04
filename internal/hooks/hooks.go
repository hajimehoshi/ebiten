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

package hooks

var onUpdateHooks = []func() error{}

// AppendHookOnUpdate appends a hook function that is run before the main update function
// every frame.
func AppendHookOnUpdate(f func() error) {
	onUpdateHooks = append(onUpdateHooks, f)
}

func Run() error {
	for _, f := range onUpdateHooks {
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}
