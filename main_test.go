package main

import (
	"bytes"
	"reflect"
	"testing"
)

func TestTransformToJSON(t *testing.T) {
	type testCase struct {
		testDescription string
		actual          string
		expected        []string
		shouldError     bool
	}
	testcases := []testCase{
		{
			"Valid YAML single document",
			`foo: bar`,
			[]string{`{"foo":"bar"}`},
			false,
		},
		{
			"Valid YAML multiple documents",
			`---
foo: bar
---
bar: baz`,
			[]string{`{"foo":"bar"}`, `{"bar":"baz"}`},
			false,
		},
		{
			"Valid YAML multiple empty documents",
			`---
foo: bar
---
---
bar: baz
---`,
			[]string{`{"foo":"bar"}`, `{"bar":"baz"}`},
			false,
		},
		{
			"Invalid YAML tabs for indentation",
			`---
foo: bar
	baz: boo
`,
			[]string{},
			true,
		},
		{
			"Invalid YAML duplicate keys",
			`---
foo: bar
foo: boo
`,
			[]string{},
			true,
		},
		{
			"Invalid YAML mixed types",
			`---
foo: bar
baz:
  - boo
  bob: alice
`,
			[]string{},
			true,
		},
	}
	for _, tCase := range testcases {
		var arr []string
		t.Run(tCase.testDescription, func(t *testing.T) {
			err := transformToJSON(
				bytes.NewReader([]byte(tCase.actual)),
				func(b []byte) error {
					arr = append(arr, string(b))
					return nil
				})

			if !tCase.shouldError && err != nil {
				t.Errorf("Got: %s, running transformToJSON", err)
			}

			if !tCase.shouldError && !reflect.DeepEqual(tCase.expected, arr) {
				t.Errorf("Expected %v got %v", tCase.expected, arr)
			}

			if tCase.shouldError && err == nil {
				t.Error("Expected transformToJSON to return an error and it did not")
			}
		})
	}
}
