package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

type buffer struct {
	bytes.Buffer
}

func (b *buffer) Close() error {
	return nil
}

func TestTransformToJSON(t *testing.T) {

	type testCase struct {
		testDescription string
		yaml            string
		expected        string
		shouldError     bool
	}
	testcases := []testCase{
		{
			"Valid YAML single document",
			`foo: bar`,
			`{"foo":"bar"}`,
			false,
		},
		{
			"No HTML escaping",
			`foo: <bar>`,
			`{"foo":"<bar>"}`,
			false,
		},
		{
			"Valid YAML multiple documents",
			`---
foo: bar
---
bar: baz`,
			`{"foo":"bar"}
{"bar":"baz"}`,
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
			`{"foo":"bar"}
{"bar":"baz"}`,
			false,
		},
		{
			"Invalid YAML tabs for indentation",
			`---
foo: bar
	baz: boo
`,
			"",
			true,
		},
		{
			"Invalid YAML duplicate keys",
			`---
foo: bar
foo: boo
`,
			"",
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
			"",
			true,
		},
	}
	for _, tCase := range testcases {
		t.Run(tCase.testDescription, func(t *testing.T) {
			var b buffer
			err := transformToJSON(
				bytes.NewReader([]byte(tCase.yaml)),
				&b,
			)

			actual := strings.Trim(b.String(), "\r\n")

			if !tCase.shouldError && err != nil {
				t.Errorf("Got: %s, running transformToJSON", err)
			}

			if !tCase.shouldError && !reflect.DeepEqual(tCase.expected, actual) {
				t.Errorf("Expected '%v' got '%v'", tCase.expected, actual)
			}

			if tCase.shouldError && err == nil {
				t.Error("Expected transformToJSON to return an error and it did not")
			}
		})
	}
}

func TestTransformToYAML(t *testing.T) {
	type testCase struct {
		testDescription string
		json            string
		expected        string
	}
	testcases := []testCase{
		{
			"Single JSON object",
			`{"foo": "bar"}`,
			`foo: bar`,
		},
		{
			"Array of JSON objects",
			`[
			   {"foo": "bar"},
			   {"bar": "baz"}
			 ]`,
			`- foo: bar
- bar: baz`,
		},
		{
			"Multiple JSON objects",
			` {"foo": "bar"}
			 {"bar": "baz"}`,
			`foo: bar
---
bar: baz`,
		},
	}
	for _, tCase := range testcases {
		var arr []string
		t.Run(tCase.testDescription, func(t *testing.T) {
			var b bytes.Buffer
			err := transformToYAML(
				bytes.NewReader([]byte(tCase.json)),
				&b,
			)

			actual := strings.Trim(b.String(), "\r\n")

			if err != nil {
				t.Errorf("Got: %s, running transformToJSON", err)
			}

			if !reflect.DeepEqual(tCase.expected, actual) {
				t.Errorf("Expected %v got %v", tCase.expected, arr)
			}
		})
	}
}
