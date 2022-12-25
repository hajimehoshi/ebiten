// Copyright 2023 The Ebitengine Authors
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

//go:build !ios

#import <Cocoa/Cocoa.h>

void ebitengine_textinput_update(const char* text, int start, int end, BOOL committed);
void ebitengine_textinput_end();

@interface TextInputClient : NSView<NSTextInputClient>
{
  NSString* markedText_;
  NSRange markedRange_;
  NSRange selectedRange_;
}
@end

@implementation TextInputClient

- (BOOL)hasMarkedText {
  return markedText_ != nil;
}

- (NSRange)markedRange {
  return markedRange_;
}

- (NSRange)selectedRange {
  return selectedRange_;
}

- (void)setMarkedText:(id)string 
        selectedRange:(NSRange)selectedRange 
     replacementRange:(NSRange)replacementRange {
  if ([string isKindOfClass:[NSAttributedString class]]) {
    string = [string string];
  }
  markedText_ = string;
  selectedRange_ = selectedRange;
  markedRange_ = NSMakeRange(0, [string length]);
  ebitengine_textinput_update([string UTF8String], selectedRange.location, selectedRange.location + selectedRange.length, NO);
}

- (void)unmarkText {
  markedText_ = nil;
}

- (NSArray<NSAttributedStringKey> *)validAttributesForMarkedText {
  return @[];
}

- (NSAttributedString *)attributedSubstringForProposedRange:(NSRange)range 
                                                actualRange:(NSRangePointer)actualRange {
  return nil;
}

- (void)insertText:(id)string 
  replacementRange:(NSRange)replacementRange {
  if ([string isKindOfClass:[NSAttributedString class]]) {
    string = [string string];
  }
  if ([string length] == 1 && [string characterAtIndex:0] < 0x20) {
    return;
  }
  ebitengine_textinput_update([string UTF8String], 0, [string length], YES);
}

- (NSUInteger)characterIndexForPoint:(NSPoint)point {
  return 0;
}


- (NSRect)firstRectForCharacterRange:(NSRange)range 
                         actualRange:(NSRangePointer)actualRange {
  NSWindow* window = [self window];
  return [window convertRectToScreen:[self frame]];
}

- (void)doCommandBySelector:(SEL)selector {
  // Do nothing.
}

- (BOOL)resignFirstResponder {
  ebitengine_textinput_end();
  return [super resignFirstResponder];
}

@end
