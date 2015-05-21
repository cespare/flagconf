package flagconf

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"
)

type testCase struct {
	config    interface{} // the (struct) initial configuration passed to ParseStrings
	toml      string      // text of TOML file
	args      []string    // user command-line arguments
	expected  interface{} // the expected state of the configuration after application of toml and flag parsing
	expectErr bool
}

func checkCase(test *testCase) error {
	tempfile, err := ioutil.TempFile("", "flagconf-test")
	if err != nil {
		return err
	}
	name := tempfile.Name()
	defer os.Remove(name)
	if _, err := tempfile.WriteString(test.toml); err != nil {
		return err
	}

	if err := tempfile.Close(); err != nil {
		return err
	}

	args := append([]string{"test"}, test.args...)
	err = ParseStrings(args, name, test.config, false)
	if test.expectErr {
		if err == nil {
			return fmt.Errorf("parsing succeeded when it was expected to fail")
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("parsing failed when it was expected to succeed: %s", err)
	}
	if !reflect.DeepEqual(test.config, test.expected) {
		return fmt.Errorf("Expected %#v, but got %#v.", test.expected, test.config)
	}
	return nil
}

type simpleCase struct {
	F1 int
}

type flagTagCase struct {
	F1 int `flag:"f2"`
}

type nestedCase struct {
	S1 *simpleCase
}

type nonPointerNestedCase struct {
	S1 simpleCase
}

type ignoreCase struct {
	F int
	D time.Duration `flag:"-"`
}

type sliceCase struct {
	S Strings
	F Ints
}

type embeddedCase struct {
	EmbeddedInner
}

type embeddedCasePtr struct {
	*EmbeddedInner
}

type EmbeddedInner struct {
	F int
}

var testCases = []*testCase{
	{
		config:   &simpleCase{5},
		toml:     "",
		args:     nil,
		expected: &simpleCase{F1: 5},
	},
	{
		config:   &simpleCase{},
		toml:     "f1 = 3",
		args:     []string{"-f1=4"},
		expected: &simpleCase{F1: 4},
	},
	{
		config:   &simpleCase{},
		toml:     "f1 = 3",
		args:     nil,
		expected: &simpleCase{F1: 3},
	},
	{
		config:   &flagTagCase{},
		toml:     "f1 = 3",
		args:     []string{"-f2=4"},
		expected: &flagTagCase{F1: 4},
	},
	{
		config: &nestedCase{},
		toml: `[s1]
f1 = 3`,
		args:     nil,
		expected: &nestedCase{&simpleCase{F1: 3}},
	},
	{
		config: &nestedCase{&simpleCase{}},
		toml: `[s1]
f1 = 3`,
		args:     nil,
		expected: &nestedCase{&simpleCase{F1: 3}},
	},
	{
		config: &nestedCase{&simpleCase{}},
		toml: `[s1]
f1 = 3`,
		args:     []string{"-s1.f1=4"},
		expected: &nestedCase{&simpleCase{F1: 4}},
	},
	{
		config: &nonPointerNestedCase{},
		toml: `[s1]
f1 = 3`,
		args:     nil,
		expected: &nonPointerNestedCase{simpleCase{F1: 3}},
	},
	{
		config: &nonPointerNestedCase{simpleCase{}},
		toml: `[s1]
f1 = 3`,
		args:     nil,
		expected: &nonPointerNestedCase{simpleCase{F1: 3}},
	},
	{
		config: &nonPointerNestedCase{simpleCase{}},
		toml: `[s1]
f1 = 3`,
		args:     []string{"-s1.f1=4"},
		expected: &nonPointerNestedCase{simpleCase{F1: 4}},
	},
	{
		config:   &ignoreCase{},
		toml:     "f = 3",
		args:     nil,
		expected: &ignoreCase{F: 3},
	},
	{
		config: &sliceCase{},
		toml: `s = ["a", "b"]
f = [1, 2]`,
		args:     nil,
		expected: &sliceCase{S: Strings{"a", "b"}, F: Ints{1, 2}},
	},
	{
		config:   &sliceCase{},
		toml:     "",
		args:     []string{"-s=a,b", "-f=1,2"},
		expected: &sliceCase{S: Strings{"a", "b"}, F: Ints{1, 2}},
	},
	{
		config:   &sliceCase{},
		toml:     `s = ["a", "b"]`,
		args:     []string{"-f=1,2"},
		expected: &sliceCase{S: Strings{"a", "b"}, F: Ints{1, 2}},
	},
	{
		config:    &simpleCase{},
		args:      []string{"-f1=NaN"},
		expectErr: true,
	},
	{
		config:    &simpleCase{},
		toml:      `f1 = "a"`,
		expectErr: true,
	},
	{
		config:    &simpleCase{},
		toml:      `f1 = 1`,
		args:      []string{"-f1=NaN"},
		expectErr: true,
	},
	{
		config:    &simpleCase{},
		toml:      `f1 = "a"`,
		args:      []string{"-f1=1"},
		expectErr: true,
	},
	{
		config:   &embeddedCase{},
		toml:     "f = 3",
		args:     nil,
		expected: &embeddedCase{EmbeddedInner{F: 3}},
	},
	{
		config:   &embeddedCase{},
		toml:     "",
		args:     []string{"-f=3"},
		expected: &embeddedCase{EmbeddedInner{F: 3}},
	},
	{
		config:   &embeddedCasePtr{},
		toml:     "f = 3",
		args:     nil,
		expected: &embeddedCasePtr{&EmbeddedInner{F: 3}},
	},
	{
		config:   &embeddedCasePtr{},
		toml:     "",
		args:     []string{"-f=3"},
		expected: &embeddedCasePtr{&EmbeddedInner{F: 3}},
	},
}

func TestGoodConfigs(t *testing.T) {
	for _, test := range testCases {
		if err := checkCase(test); err != nil {
			t.Log(test)
			t.Error(err)
		}
	}
}

func TestIsHelp(t *testing.T) {
	if !IsHelp(flag.ErrHelp) {
		t.Fatal("IsHelp(flag.ErrHelp) != true")
	}
	if !IsHelp(FlagError{Err: flag.ErrHelp}) {
		t.Fatal("IsHelp(FlagError{Err: flag.ErrHelp}) != true")
	}
	if IsHelp(errors.New("not help")) {
		t.Fatal("IsHelp(not help) == true")
	}
}
