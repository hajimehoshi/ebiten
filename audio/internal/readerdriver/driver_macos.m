// Copyright 2020 The Ebiten Authors
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

// +build darwin,!ios

#import <AppKit/AppKit.h>

#include "_cgo_export.h"

@interface EbitenReaderDriverNotificationObserver : NSObject {
}

@end

@implementation EbitenReaderDriverNotificationObserver {
}

- (void)receiveSleepNote:(NSNotification *)note {
  ebiten_readerdriver_setGlobalPause();
}

- (void)receiveWakeNote:(NSNotification *)note {
  ebiten_readerdriver_setGlobalResume();
}

@end

// ebiten_readerdriver_setNotificationHandler sets a handler for sleep/wake notifications.
void ebiten_readerdriver_setNotificationHandler() {
  EbitenReaderDriverNotificationObserver *observer = [[EbitenReaderDriverNotificationObserver alloc] init];

  [[[NSWorkspace sharedWorkspace] notificationCenter]
      addObserver:observer
         selector:@selector(receiveSleepNote:)
             name:NSWorkspaceWillSleepNotification
           object:NULL];
  [[[NSWorkspace sharedWorkspace] notificationCenter]
      addObserver:observer
         selector:@selector(receiveWakeNote:)
             name:NSWorkspaceDidWakeNotification
           object:NULL];
}
