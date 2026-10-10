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

// Package svgcss parses the CSS syntax subset used by OpenType SVG documents.
package svgcss

import (
	"errors"
	"strings"
)

const (
	// MaxBytes bounds stylesheet, declaration list and splitting input sizes in bytes.
	MaxBytes = 1 << 20
	// MaxValueBytes bounds individual CSS values in bytes.
	MaxValueBytes = 1 << 14
	// MaxRules bounds the number of rules in one stylesheet.
	MaxRules = 4096
	// MaxDeclarations bounds the number of fields in a declaration list.
	MaxDeclarations = 1 << 16
	// MaxSelectorParts bounds selector parts, list entries and ID/class selectors per part.
	MaxSelectorParts = 16
	maxNesting       = 32
)

var (
	ErrSyntax      = errors.New("svgcss: invalid syntax")
	ErrUnsupported = errors.New("svgcss: unsupported syntax")
	ErrLimit       = errors.New("svgcss: parsing limit exceeded")
)

// Declaration contains a property name and its unvalidated value, including priority.
type Declaration struct {
	Name  string
	Value string
}

// Value contains declaration text without its optional important priority.
type Value struct {
	Text      string
	Important bool
}

// SelectorPart describes one compound selector.
type SelectorPart struct {
	Tag     string
	ID      string
	Classes []string
	// Child requires a direct parent relationship with the preceding part.
	Child bool
}

// Selector describes a selector from ancestors to descendants.
type Selector struct {
	Parts []SelectorPart
	// Specificity is comparable across selectors; higher values win.
	Specificity int
}

// Rule contains selectors and declarations in source order.
type Rule struct {
	Selectors    []Selector
	Declarations []Declaration
}

// Variable contains a CSS var() reference and any text following it.
type Variable struct {
	Name        string
	Fallback    string
	HasFallback bool
	Trailing    string
}

// ParseDeclarations parses a declaration list without interpreting property values.
func ParseDeclarations(text string) ([]Declaration, error) {
	text, err := stripComments(text)
	if err != nil {
		return nil, err
	}
	return parseDeclarations(text)
}

func parseDeclarations(text string) ([]Declaration, error) {
	fields, err := split(text, ';', MaxDeclarations)
	if err != nil {
		return nil, err
	}
	var result []Declaration
	for _, field := range fields {
		name, value, ok := strings.Cut(strings.TrimSpace(field), ":")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if !strings.HasPrefix(name, "--") {
			name = strings.ToLower(name)
		}
		result = append(result, Declaration{
			Name:  name,
			Value: strings.TrimSpace(value),
		})
	}
	return result, nil
}

// ParseValue separates a declaration's value and optional important priority.
// The trimmed value text must not exceed MaxValueBytes.
func ParseValue(text string) (Value, error) {
	if len(text) > MaxBytes {
		return Value{}, ErrLimit
	}
	parts, err := split(text, '!', MaxDeclarations)
	if err != nil {
		return Value{}, err
	}
	if len(parts) > 2 || (len(parts) == 2 && !strings.EqualFold(strings.TrimSpace(parts[1]), "important")) {
		return Value{}, ErrSyntax
	}
	value := strings.TrimSpace(parts[0])
	if len(value) > MaxValueBytes {
		return Value{}, ErrLimit
	}
	return Value{
		Text:      value,
		Important: len(parts) == 2,
	}, nil
}

// HasFunction reports whether text starts with the named CSS function.
func HasFunction(text, name string) bool {
	return len(text) > len(name) && text[len(name)] == '(' && strings.EqualFold(text[:len(name)], name)
}

func isAlpha(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// stripComments removes comments between supported tokens.
func stripComments(text string) (string, error) {
	if len(text) > MaxBytes {
		return "", ErrLimit
	}
	var out strings.Builder
	var quote, previous byte
	var comment bool
	var commentEnd int
	for i := range len(text) {
		if i < commentEnd {
			continue
		}
		c := text[i]
		if c == '\\' {
			return "", ErrUnsupported
		}
		if quote == 0 && c == '/' && i+1 < len(text) && text[i+1] == '*' {
			end := strings.Index(text[i+2:], "*/")
			if end < 0 {
				return "", ErrSyntax
			}
			commentEnd = i + end + 4
			comment = true
			continue
		}
		if comment {
			// Splitting a name, number or function token is outside this subset.
			word := func(c byte) bool {
				return isAlpha(c) || isDigit(c) || c == '_' || c == '-'
			}
			if (word(previous) && (word(c) || c == '(' || c == '%')) || (isDigit(previous) && c == '.') {
				return "", ErrUnsupported
			}
			comment = false
		}
		if quote != 0 {
			if c == quote {
				quote = 0
			}
		} else if c == '\'' || c == '"' {
			quote = c
		}
		out.WriteByte(c)
		previous = c
	}
	if quote != 0 {
		return "", ErrSyntax
	}
	return out.String(), nil
}

// ValidateValue checks the size and balance of quotes and parentheses in value text.
func ValidateValue(text string) error {
	if len(text) > MaxValueBytes {
		return ErrLimit
	}
	_, err := split(text, ',', MaxValueBytes)
	return err
}

// split separates text at delimiters outside quotes and parentheses, up to limit fields.
func split(text string, delimiter byte, limit int) ([]string, error) {
	if len(text) > MaxBytes {
		return nil, ErrLimit
	}
	var result []string
	var quote byte
	var depth, start int
	for i := range len(text) {
		c := text[i]
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '\'', '"':
			quote = c
		case '(':
			depth++
			if depth > maxNesting {
				return nil, ErrLimit
			}
		case ')':
			depth--
			if depth < 0 {
				return nil, ErrSyntax
			}
		default:
			if c == delimiter && depth == 0 {
				if len(result) >= limit {
					return nil, ErrLimit
				}
				result = append(result, text[start:i])
				start = i + 1
			}
		}
	}
	if quote != 0 || depth != 0 {
		return nil, ErrSyntax
	}
	if len(result) >= limit {
		return nil, ErrLimit
	}
	return append(result, text[start:]), nil
}

func identifier(s string) bool {
	if s == "" || !(isAlpha(s[0]) || s[0] == '_' || s[0] == '-') {
		return false
	}
	for i := range len(s) {
		if !isAlpha(s[i]) && !isDigit(s[i]) && s[i] != '_' && s[i] != '-' {
			return false
		}
	}
	return true
}

func parseSelector(text string) (Selector, error) {
	var result Selector
	var child bool
	text = strings.TrimSpace(text)
	for text != "" {
		if len(result.Parts) >= MaxSelectorParts {
			return Selector{}, ErrLimit
		}
		end := strings.IndexAny(text, " >\t\r\n\f")
		if end < 0 {
			end = len(text)
		}
		token := text[:end]
		if token == "" {
			return Selector{}, ErrUnsupported
		}
		part := SelectorPart{
			Child: child,
		}
		n := strings.IndexAny(token, ".#")
		if n < 0 {
			n = len(token)
		}
		if tag := token[:n]; tag != "" {
			if tag != "*" && !identifier(tag) {
				return Selector{}, ErrUnsupported
			}
			part.Tag = tag
			if tag != "*" {
				result.Specificity++
			}
		}
		token = token[n:]
		var tests int
		for token != "" {
			tests++
			if tests > MaxSelectorParts {
				return Selector{}, ErrLimit
			}
			prefix := token[0]
			token = token[1:]
			end := strings.IndexAny(token, ".#")
			if end < 0 {
				end = len(token)
			}
			name := token[:end]
			token = token[end:]
			if !identifier(name) {
				return Selector{}, ErrUnsupported
			}
			if prefix == '#' {
				if part.ID != "" {
					return Selector{}, ErrUnsupported
				}
				part.ID = name
				result.Specificity += 1 << 20
			} else {
				part.Classes = append(part.Classes, name)
				result.Specificity += 1 << 10
			}
		}
		result.Parts = append(result.Parts, part)
		rest := text[end:]
		text = strings.TrimSpace(rest)
		child = false
		if strings.HasPrefix(text, ">") {
			child = true
			text = strings.TrimSpace(text[1:])
			if text == "" {
				return Selector{}, ErrSyntax
			}
		}
	}
	if len(result.Parts) == 0 {
		return Selector{}, ErrSyntax
	}
	return result, nil
}

// ParseStylesheet parses SVG CSS rules in source order.
func ParseStylesheet(text string) ([]Rule, error) {
	text, err := stripComments(text)
	if err != nil {
		return nil, err
	}
	var rules []Rule
	for strings.TrimSpace(text) != "" {
		if strings.HasPrefix(strings.TrimSpace(text), "@") {
			return nil, ErrUnsupported
		}
		if len(rules) >= MaxRules {
			return nil, ErrLimit
		}
		before, rest, ok := strings.Cut(text, "{")
		if !ok {
			return nil, ErrSyntax
		}
		body, rest, ok := strings.Cut(rest, "}")
		if !ok || strings.ContainsAny(body, "{}") {
			return nil, ErrSyntax
		}
		text = rest
		selectors, err := split(before, ',', MaxSelectorParts)
		if err != nil {
			return nil, err
		}
		var rule Rule
		for _, s := range selectors {
			sel, err := parseSelector(s)
			if err != nil {
				return nil, err
			}
			rule.Selectors = append(rule.Selectors, sel)
		}
		rule.Declarations, err = parseDeclarations(body)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

// ParseVariable parses a CSS var() reference with optional fallback and trailing text.
func ParseVariable(text string) (Variable, error) {
	if len(text) > MaxValueBytes {
		return Variable{}, ErrLimit
	}
	if !HasFunction(text, "var") {
		return Variable{}, ErrSyntax
	}
	if _, err := split(text, ',', MaxValueBytes); err != nil {
		return Variable{}, err
	}
	var depth int
	var quote byte
	end := -1
	for offset := range len(text) - 4 {
		i := offset + 4
		c := text[i]
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '\'' || c == '"' {
			quote = c
			continue
		}
		if c == '(' {
			depth++
		}
		if c == ')' {
			if depth == 0 {
				end = i
				break
			}
			depth--
		}
	}
	if end < 0 {
		return Variable{}, ErrSyntax
	}
	name, fallback, ok := strings.Cut(text[4:end], ",")
	name = strings.TrimSpace(name)
	if !strings.HasPrefix(name, "--") || !identifier(name) || len(name) == 2 {
		return Variable{}, ErrSyntax
	}
	return Variable{
		Name:        name,
		Fallback:    strings.TrimSpace(fallback),
		HasFallback: ok,
		Trailing:    strings.TrimSpace(text[end+1:]),
	}, nil
}

// Components splits CSS values at outer whitespace and function boundaries.
func Components(text string) ([]string, error) {
	if len(text) > MaxValueBytes {
		return nil, ErrLimit
	}
	var components []string
	var depth int
	var quote byte
	start := -1
	for i := range len(text) {
		c := text[i]
		if quote == 0 && depth == 0 && strings.ContainsRune(" \t\r\n\f", rune(c)) {
			if start >= 0 {
				components = append(components, text[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '\'', '"':
			quote = c
		case '(':
			depth++
			if depth > maxNesting {
				return nil, ErrLimit
			}
		case ')':
			depth--
			if depth < 0 {
				return nil, ErrSyntax
			}
			if depth == 0 {
				components = append(components, text[start:i+1])
				start = -1
			}
		}
	}
	if quote != 0 || depth != 0 {
		return nil, ErrSyntax
	}
	if start >= 0 {
		components = append(components, text[start:])
	}
	if len(components) == 0 {
		return nil, ErrSyntax
	}
	return components, nil
}
