package flagconf

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

type testCase struct {
	config   interface{} // the (struct) initial configuration passed to ParseStrings
	toml     string      // text of TOML file
	args     []string    // user command-line arguments
	expected interface{} // the expected state of the configuration after application of toml and flag parsing
}

func checkCase(test *testCase) error {
	tempfile, err := ioutil.TempFile("", "flagconf-test")
	if err != nil {
		return err
	}
	if _, err := tempfile.WriteString(test.toml); err != nil {
		return err
	}
	name := tempfile.Name()
	defer func() { os.Remove(name) }()

	if err := tempfile.Close(); err != nil {
		return err
	}

	args := append([]string{"test"}, test.args...)
	err = ParseStrings(args, name, test.config, false)
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

type embeddedCase struct {
	S1 *simpleCase
}

type nonPointerEmbeddedCase struct {
	S1 simpleCase
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
		config: &embeddedCase{},
		toml: `[s1]
f1 = 3`,
		args:     nil,
		expected: &embeddedCase{&simpleCase{F1: 3}},
	},
	{
		config: &embeddedCase{&simpleCase{}},
		toml: `[s1]
f1 = 3`,
		args:     nil,
		expected: &embeddedCase{&simpleCase{F1: 3}},
	},
	{
		config: &embeddedCase{&simpleCase{}},
		toml: `[s1]
f1 = 3`,
		args:     []string{"-s1.f1=4"},
		expected: &embeddedCase{&simpleCase{F1: 4}},
	},
	{
		config: &nonPointerEmbeddedCase{},
		toml: `[s1]
f1 = 3`,
		args:     nil,
		expected: &nonPointerEmbeddedCase{simpleCase{F1: 3}},
	},
	{
		config: &nonPointerEmbeddedCase{simpleCase{}},
		toml: `[s1]
f1 = 3`,
		args:     nil,
		expected: &nonPointerEmbeddedCase{simpleCase{F1: 3}},
	},
	{
		config: &nonPointerEmbeddedCase{simpleCase{}},
		toml: `[s1]
f1 = 3`,
		args:     []string{"-s1.f1=4"},
		expected: &nonPointerEmbeddedCase{simpleCase{F1: 4}},
	},
}

func TestGoodConfigs(t *testing.T) {
	for _, test := range testCases {
		if err := checkCase(test); err != nil {
			t.Error(err)
		}
	}
}
