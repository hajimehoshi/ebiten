// Copyright 2026 The Ebitengine Authors
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

//go:build ebitenginevmguest

package ui

import "os"

// vmGuestEndpointEnv names the host endpoint a guest dials when no endpoint is given in RunOptions.
const vmGuestEndpointEnv = "EBITENGINE_VM_ENDPOINT"

// vmGuestEndpoint is the host endpoint configured in the environment.
var vmGuestEndpoint = func() string {
	// The variable is removed from the environment as it is read: the endpoint addresses one guest
	// session, so a process started by this one must not inherit it. A value set afterwards reaches
	// children as usual.
	ep := os.Getenv(vmGuestEndpointEnv)
	_ = os.Unsetenv(vmGuestEndpointEnv)
	return ep
}()

// vmGuestEndpointFromEnv returns the host endpoint configured in the environment. Reading the
// environment is enabled by the ebitenginevmguest build tag.
func vmGuestEndpointFromEnv() string {
	return vmGuestEndpoint
}
