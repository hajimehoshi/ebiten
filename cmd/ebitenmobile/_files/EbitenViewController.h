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

#import <UIKit/UIKit.h>

@interface {{.PrefixUpper}}EbitenViewController : UIViewController

// onErrorOnGameUpdate is called on the main thread when an error happens when updating a game.
// You can define your own error handler, e.g., using Crashlytics, by overwriting this method.
- (void)onErrorOnGameUpdate:(NSError*)err;

// suspendGame suspends the game.
// It is recommended to call this when the application is being suspended e.g.,
// UIApplicationDelegate's applicationWillResignActive is called.
- (void)suspendGame;

// resumeGame resumes the game.
// It is recommended to call this when the application is being resumed e.g.,
// UIApplicationDelegate's applicationDidBecomeActive is called.
- (void)resumeGame;

@end
