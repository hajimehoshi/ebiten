// Copyright 2022 The Ebitengine Authors
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

package gamepaddb

// See https://github.com/libsdl-org/SDL/blob/120c76c84bbce4c1bfed4e9eb74e10678bd83120/include/SDL_gamecontroller.h#L655-L680
const (
	SDLControllerButtonA             = 0
	SDLControllerButtonB             = 1
	SDLControllerButtonX             = 2
	SDLControllerButtonY             = 3
	SDLControllerButtonBack          = 4
	SDLControllerButtonGuide         = 5
	SDLControllerButtonStart         = 6
	SDLControllerButtonLeftStick     = 7
	SDLControllerButtonRightStick    = 8
	SDLControllerButtonLeftShoulder  = 9
	SDLControllerButtonRightShoulder = 10
	SDLControllerButtonDpadUp        = 11
	SDLControllerButtonDpadDown      = 12
	SDLControllerButtonDpadLeft      = 13
	SDLControllerButtonDpadRight     = 14
	SDLControllerButtonMisc1         = 15
	SDLControllerButtonPaddle1       = 16
	SDLControllerButtonPaddle2       = 17
	SDLControllerButtonPaddle3       = 18
	SDLControllerButtonPaddle4       = 19
	SDLControllerButtonTouchpad      = 20
	SDLControllerButtonMax           = SDLControllerButtonTouchpad // This is different from the original SDL_CONTROLLER_BUTTON_MAX.
)

// See https://github.com/libsdl-org/SDL/blob/120c76c84bbce4c1bfed4e9eb74e10678bd83120/include/SDL_gamecontroller.h#L550-L560
const (
	SDLControllerAxisLeftX        = 0
	SDLControllerAxisLeftY        = 1
	SDLControllerAxisRightX       = 2
	SDLControllerAxisRightY       = 3
	SDLControllerAxisTriggerLeft  = 4
	SDLControllerAxisTriggerRight = 5
)
