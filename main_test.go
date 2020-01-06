package main

import (
	"bytes"
	"io/ioutil"
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

func TestCompileCommand(t *testing.T) {
	type testCase struct {
		testDescription string
		osArgs          []string
		expectedCmd     []string
		expectedFiles   []string
		shouldError     bool
	}
	testcases := []testCase{
		{
			"Works when there is no file passed",
			[]string{"yq", "-r", "."},
			[]string{"jq", "-r", "."},
			[]string{},
			false,
		},
		{
			"Errors when there is no flag passed",
			[]string{"yq"},
			[]string{},
			[]string{},
			true,
		},
		{
			"Errors when there is help flag passed",
			[]string{"yq", "-h"},
			[]string{},
			[]string{},
			true,
		},
		{
			"Works with single arg",
			[]string{"yq", "-r", ".", "test_resources/foo.yaml"},
			[]string{"jq", "-r", "."},
			[]string{"test_resources/foo.yaml"},
			false,
		},
		{
			"Works with multiple args",
			[]string{"yq", "-r", "-s", ".", "test_resources/foo.yaml"},
			[]string{"jq", "-r", "-s", "."},
			[]string{"test_resources/foo.yaml"},
			false,
		},
		{
			"Works with jq slurpfile flag",
			[]string{"yq", "--slurpfile", "a", "b", ".", "test_resources/foo.yaml"},
			[]string{"jq", "--slurpfile", "a", "b", "."},
			[]string{"test_resources/foo.yaml"},
			false,
		},
		{
			"Works with jq rawfile flag",
			[]string{"yq", "--rawfile", "a", "b", ".", "test_resources/foo.yaml"},
			[]string{"jq", "--rawfile", "a", "b", "."},
			[]string{"test_resources/foo.yaml"},
			false,
		},
		{
			"Complex jq args",
			[]string{"yq", "-y", "-s", ".[0] * .[1]", "test_resources/foo.yaml", "test_resources/foo.yaml"},
			[]string{"jq", "-s", ".[0] * .[1]"},
			[]string{"test_resources/foo.yaml", "test_resources/foo.yaml"},
			false,
		},
	}
	for _, tCase := range testcases {
		var y yq
		t.Run(tCase.testDescription, func(t *testing.T) {
			y.jqCmd.Path = "jq"
			err := y.compileJqCmd(tCase.osArgs, ioutil.Discard)
			actualCmd := y.jqCmd.Args
			actualFiles := y.files

			if !tCase.shouldError && err != nil {
				t.Error("Did not expect an error got: ", err)
			}

			if !tCase.shouldError && !reflect.DeepEqual(tCase.expectedCmd, actualCmd) {
				t.Errorf("Expected: '%v', got: '%v'", tCase.expectedCmd, actualCmd)
			}

			if !tCase.shouldError &&
				(!(len(tCase.expectedFiles) == 0 && len(actualFiles) == 0) &&
					!reflect.DeepEqual(tCase.expectedFiles, actualFiles)) {
				t.Errorf("Expected: '%v', got: '%v'", tCase.expectedFiles, actualFiles)
			}
		})
	}
}
