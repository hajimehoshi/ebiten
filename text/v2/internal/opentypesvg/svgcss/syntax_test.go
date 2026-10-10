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

package svgcss_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/opentypesvg/svgcss"
)

func TestStylesheetSyntax(t *testing.T) {
	rules, err := svgcss.ParseStylesheet(`.a, g > rect.tile#x { FILL:var(--color0, rgb(255,0,0)) !important; unknown:value; --Brand:blue } .a/**/.b {fill:red;fill:blue}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 || len(rules[0].Selectors) != 2 || len(rules[0].Declarations) != 3 {
		t.Fatalf("unexpected rule structure: %+v", rules)
	}
	selector := rules[0].Selectors[1]
	if len(selector.Parts) != 2 {
		t.Fatalf("selector parts=%v", selector.Parts)
	}
	if selector.Parts[0].Tag != "g" || selector.Parts[1].Tag != "rect" || selector.Parts[1].ID != "x" || !selector.Parts[1].Child || !slices.Equal(selector.Parts[1].Classes, []string{"tile"}) {
		t.Errorf("selector=%+v", selector)
	}
	if selector.Specificity <= rules[0].Selectors[0].Specificity {
		t.Errorf("compound selector specificity=%d, simple class=%d", selector.Specificity, rules[0].Selectors[0].Specificity)
	}
	declaration := rules[0].Declarations[0]
	if declaration.Name != "fill" {
		t.Errorf("property name=%q", declaration.Name)
	}
	value, err := svgcss.ParseValue(declaration.Value)
	if err != nil {
		t.Fatal(err)
	}
	if !value.Important || value.Text != "var(--color0, rgb(255,0,0))" {
		t.Errorf("value=%+v", value)
	}
	if rules[0].Declarations[1].Name != "unknown" || rules[0].Declarations[2].Name != "--Brand" {
		t.Errorf("parser discarded or reinterpreted declarations: %+v", rules[0].Declarations)
	}
	if len(rules[1].Selectors[0].Parts) != 1 || !slices.Equal(rules[1].Selectors[0].Parts[0].Classes, []string{"a", "b"}) {
		t.Errorf("comment changed compound selector: %+v", rules[1].Selectors)
	}
	if rules[1].Declarations[0].Value != "red" || rules[1].Declarations[1].Value != "blue" {
		t.Errorf("declaration order=%+v", rules[1].Declarations)
	}
}

func TestInlineDeclarationsAndPriority(t *testing.T) {
	declarations, err := svgcss.ParseDeclarations(`/*start*/ fill:red; missing-colon; fill:url('#bang!'); opacity:.5 ! IMPORTANT`)
	if err != nil {
		t.Fatal(err)
	}
	if len(declarations) != 3 {
		t.Fatalf("declarations=%+v", declarations)
	}
	for i, d := range declarations {
		value, err := svgcss.ParseValue(d.Value)
		if err != nil {
			t.Fatal(err)
		}
		if value.Important != (i == 2) {
			t.Errorf("declaration %d importance=%t", i, value.Important)
		}
	}
	for _, value := range []string{"blue !unknown", "blue!important!important"} {
		if _, err := svgcss.ParseValue(value); !errors.Is(err, svgcss.ErrSyntax) {
			t.Errorf("invalid priority %q: %v", value, err)
		}
	}
}

func TestVariableSyntax(t *testing.T) {
	for _, tc := range []struct {
		text string
		want svgcss.Variable
	}{
		{
			text: "var(--color0)",
			want: svgcss.Variable{
				Name: "--color0",
			},
		},
		{
			text: "var(--color0, red, blue)",
			want: svgcss.Variable{
				Name:        "--color0",
				Fallback:    "red, blue",
				HasFallback: true,
			},
		},
		{
			text: "VAR(--Brand, var(--color1, red)) blue",
			want: svgcss.Variable{
				Name:        "--Brand",
				Fallback:    "var(--color1, red)",
				HasFallback: true,
				Trailing:    "blue",
			},
		},
		{
			text: "var(--color0, nonsense)",
			want: svgcss.Variable{
				Name:        "--color0",
				Fallback:    "nonsense",
				HasFallback: true,
			},
		},
	} {
		got, err := svgcss.ParseVariable(tc.text)
		if err != nil {
			t.Errorf("ParseVariable(%q): %v", tc.text, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseVariable(%q)=%+v, want %+v", tc.text, got, tc.want)
		}
	}
	for _, text := range []string{"var(color0)", "var(--)", "var(--color0,red", "rgb(1,2,3)", "var(--color0,'red)"} {
		if _, err := svgcss.ParseVariable(text); !errors.Is(err, svgcss.ErrSyntax) {
			t.Errorf("invalid variable %q: %v", text, err)
		}
	}
}

func TestComponentsAndSplit(t *testing.T) {
	components, err := svgcss.Components(`rotate(45deg)scale(2) url('a)b') circle(50%)fill-box`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"rotate(45deg)", "scale(2)", "url('a)b')", "circle(50%)", "fill-box"}
	if !slices.Equal(components, want) {
		t.Errorf("components=%q, want %q", components, want)
	}
	fields, err := svgcss.Split(`a,var(--color0,rgb(1,2,3)),"b,c"`, ',', 3)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(fields, []string{"a", "var(--color0,rgb(1,2,3))", `"b,c"`}) {
		t.Errorf("fields=%q", fields)
	}
	if _, err := svgcss.Split("a,b", ',', 1); !errors.Is(err, svgcss.ErrLimit) {
		t.Errorf("field limit: %v", err)
	}
	for _, text := range []string{"", "a(", "a)", "'a"} {
		if _, err := svgcss.Components(text); !errors.Is(err, svgcss.ErrSyntax) {
			t.Errorf("broken components %q: %v", text, err)
		}
	}
	if !svgcss.HasFunction("VAR(--x)", "var") || svgcss.HasFunction("var (--x)", "var") {
		t.Error("incorrect function-name recognition")
	}
}

func TestUnsupportedSyntax(t *testing.T) {
	for _, text := range []string{`@media all{rect{fill:red}}`, `@import "a.css";`, `rect:hover{fill:red}`, `[id=x]{fill:red}`, `rect+rect{fill:red}`, `.a\\:b{fill:red}`, `.a{fill:r/**/ed}`} {
		if _, err := svgcss.ParseStylesheet(text); !errors.Is(err, svgcss.ErrUnsupported) {
			t.Errorf("unsupported syntax %q: %v", text, err)
		}
	}
	for _, text := range []string{"rect{", "rect{fill:red", "/*", "rect{fill:'red}"} {
		if _, err := svgcss.ParseStylesheet(text); !errors.Is(err, svgcss.ErrSyntax) {
			t.Errorf("invalid syntax %q: %v", text, err)
		}
	}
}

func TestParserLimits(t *testing.T) {
	if _, err := svgcss.ParseValue(strings.Repeat("x", svgcss.MaxValueBytes+1)); !errors.Is(err, svgcss.ErrLimit) {
		t.Errorf("single value limit: %v", err)
	}
	if _, err := svgcss.ParseStylesheet(strings.Repeat("a{}", svgcss.MaxRules+1)); !errors.Is(err, svgcss.ErrLimit) {
		t.Errorf("rule limit: %v", err)
	}
	if _, err := svgcss.ParseStylesheet(strings.Repeat("g ", svgcss.MaxSelectorParts) + "rect{}"); !errors.Is(err, svgcss.ErrLimit) {
		t.Errorf("selector limit: %v", err)
	}
	if _, err := svgcss.ParseDeclarations(strings.Repeat("a:b;", svgcss.MaxDeclarations)); !errors.Is(err, svgcss.ErrLimit) {
		t.Errorf("declaration limit: %v", err)
	}
	if _, err := svgcss.ParseDeclarations(strings.Repeat(" ", svgcss.MaxBytes+1)); !errors.Is(err, svgcss.ErrLimit) {
		t.Errorf("input limit: %v", err)
	}
	if _, err := svgcss.ParseVariable("var(--x," + strings.Repeat(" ", svgcss.MaxValueBytes) + ")"); !errors.Is(err, svgcss.ErrLimit) {
		t.Errorf("variable limit: %v", err)
	}
	if _, err := svgcss.Components(strings.Repeat("f(", svgcss.MaxNesting+1) + strings.Repeat(")", svgcss.MaxNesting+1)); !errors.Is(err, svgcss.ErrLimit) {
		t.Errorf("nesting limit: %v", err)
	}
}

func FuzzSyntax(f *testing.F) {
	for _, source := range []string{".a{fill:var(--color0,red)!important}", "g>.a rect{fill:blue}", "var(--color0,red) blue", "url('#x')blur(2px)", "@media all{}", "/*"} {
		f.Add(source)
	}
	f.Fuzz(func(t *testing.T, source string) {
		if len(source) > 1<<16 {
			t.Skip()
		}
		rules, err := svgcss.ParseStylesheet(source)
		if err == nil {
			for _, rule := range rules {
				if len(rule.Selectors) == 0 {
					t.Error("successful rule has no selectors")
				}
				for _, selector := range rule.Selectors {
					if len(selector.Parts) == 0 || len(selector.Parts) > svgcss.MaxSelectorParts {
						t.Errorf("selector part count=%d", len(selector.Parts))
					}
				}
			}
		}
		declarations, err := svgcss.ParseDeclarations(source)
		if err == nil {
			for _, d := range declarations {
				_, _ = svgcss.ParseValue(d.Value)
			}
		}
		_ = svgcss.ValidateValue(source)
		_, _ = svgcss.ParseValue(source)
		_, _ = svgcss.ParseVariable(source)
		_, _ = svgcss.Components(source)
		_, _ = svgcss.Split(source, ',', 32)
	})
}

func TestValidateValue(t *testing.T) {
	for _, value := range []string{"", "red", `var(--color0,rgb(1,2,3))`, `"a)b"`, "青"} {
		if err := svgcss.ValidateValue(value); err != nil {
			t.Errorf("valid value %q: %v", value, err)
		}
	}
	for _, value := range []string{"(", ")", "'unfinished"} {
		if err := svgcss.ValidateValue(value); !errors.Is(err, svgcss.ErrSyntax) {
			t.Errorf("invalid value %q: %v", value, err)
		}
	}
	if err := svgcss.ValidateValue(strings.Repeat(" ", svgcss.MaxValueBytes+1)); !errors.Is(err, svgcss.ErrLimit) {
		t.Errorf("value size limit: %v", err)
	}
}

func TestCommentSkippingPreservesBytes(t *testing.T) {
	declarations, err := svgcss.ParseDeclarations(`/* \ ' ( 青 */fill:'青';/**//**/stroke:blue`)
	if err != nil {
		t.Fatal(err)
	}
	if len(declarations) != 2 {
		t.Fatalf("declarations=%+v", declarations)
	}
	if declarations[0].Name != "fill" || declarations[0].Value != "'青'" || declarations[1].Name != "stroke" || declarations[1].Value != "blue" {
		t.Errorf("comment skipping changed declarations: %+v", declarations)
	}
}

func TestValueLimitExcludesPriority(t *testing.T) {
	for _, size := range []int{svgcss.MaxValueBytes, svgcss.MaxValueBytes + 1} {
		value := "rgb(" + strings.Repeat(" ", size-len("rgb(255,0,0)")) + "255,0,0)"
		for _, suffix := range []string{"", " !important", " \t ! IMPORTANT"} {
			got, err := svgcss.ParseValue(" \t " + value + suffix + " \t ")
			if size > svgcss.MaxValueBytes {
				if !errors.Is(err, svgcss.ErrLimit) {
					t.Errorf("size %d, suffix %q: %v", size, suffix, err)
				}
				continue
			}
			if err != nil {
				t.Errorf("size %d, suffix %q: %v", size, suffix, err)
				continue
			}
			if got.Text != value || got.Important != (suffix != "") {
				t.Errorf("size %d, suffix %q: value or priority changed", size, suffix)
			}
		}
	}
	if _, err := svgcss.ParseValue("red" + strings.Repeat(" ", svgcss.MaxBytes)); !errors.Is(err, svgcss.ErrLimit) {
		t.Errorf("raw input limit: %v", err)
	}
}
