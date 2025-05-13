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

// TODO: Remove Cgo with PureGo (#1162).

#import <Cocoa/Cocoa.h>

int ebitengine_textinput_hasMarkedText();
void ebitengine_textinput_markedRange(int64_t* start, int64_t* length);
void ebitengine_textinput_selectedRange(int64_t* start, int64_t* length);
void ebitengine_textinput_unmarkText();
void ebitengine_textinput_setMarkedText(const char* text, int64_t selectionStart, int64_t selectionLen, int64_t replaceStart, int64_t replaceLen);
void ebitengine_textinput_insertText(const char* text, int64_t replaceStart, int64_t replaceLen);
NSRect ebitengine_textinput_firstRectForCharacterRange(uintptr_t self, NSRange range, NSRangePointer actualRange);
void ebitengine_textinput_end();

@interface TextInputClient : NSView<NSTextInputClient>
{
}
@end

@implementation TextInputClient

- (BOOL)hasMarkedText {
  return ebitengine_textinput_hasMarkedText() != 0;
}

- (NSRange)markedRange {
  int64_t start = 0;
  int64_t length = 0;
  ebitengine_textinput_markedRange(&start, &length);
  return NSMakeRange(start, length);
}

- (NSRange)selectedRange {
  int64_t start = 0;
  int64_t length = 0;
  ebitengine_textinput_selectedRange(&start, &length);
  return NSMakeRange(start, length);
}

- (void)setMarkedText:(id)string 
        selectedRange:(NSRange)selectedRange 
     replacementRange:(NSRange)replacementRange {
  if ([string isKindOfClass:[NSAttributedString class]]) {
    string = [string string];
  }
  ebitengine_textinput_setMarkedText([string UTF8String], selectedRange.location, selectedRange.length, replacementRange.location, replacementRange.length);
}

- (void)unmarkText {
  ebitengine_textinput_unmarkText();
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
  ebitengine_textinput_insertText([string UTF8String], replacementRange.location, replacementRange.length);
}

- (NSUInteger)characterIndexForPoint:(NSPoint)point {
  return 0;
}


- (NSRect)firstRectForCharacterRange:(NSRange)range 
                         actualRange:(NSRangePointer)actualRange {
  return ebitengine_textinput_firstRectForCharacterRange((uintptr_t)(self), range, actualRange);
}

- (void)doCommandBySelector:(SEL)selector {
  // Do nothing.
}

- (BOOL)resignFirstResponder {
  ebitengine_textinput_end();
  return [super resignFirstResponder];
}

@end
