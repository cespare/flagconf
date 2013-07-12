package flagconf

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

type testCase struct {
	config   interface{} // the (struct) initial configuration passed to Parse
	toml     string      // text of TOML file
	args     []string      // user command-line arguments
	expected interface{} // the expected state of the configuration after application of toml and flag parsing
}

func checkCase(test *testCase) (err error) {
	tempfile, err := ioutil.TempFile("", "flagconf-test")
	if err != nil {
		return err
	}
	if _, err := tempfile.WriteString(test.toml); err != nil {
		return err
	}
	name := tempfile.Name()
	defer func() {
		fmt.Println("first")
		fmt.Println(name)
		err = os.Remove(name)
	}()
	if err := tempfile.Close(); err != nil {
		return err
	}

	oldArgs := make([]string, len(os.Args))
	copy(oldArgs, os.Args)
	newArgs := make([]string, len(test.args) + 1)
	newArgs[0] = oldArgs[0]
	copy(newArgs[1:], test.args)
	os.Args = newArgs
	defer func() {
		fmt.Println("Second")
		copy(os.Args, oldArgs)
	}()

	fmt.Printf("\033[01;34m>>>> os.Args: %v\x1B[m\n", os.Args)
	err = Parse(name, test.config)
	if err != nil {
		return fmt.Errorf("parsing failed when it was expected to succeed: %s", err)
	}
	if !reflect.DeepEqual(test.config, test.expected) {
		return fmt.Errorf("Expected %#v, but got %#v.", test.expected, test.config)
	}
	return nil
}

type simple1 struct {
	F1 int `toml:"f1"`
}

var testCases = []*testCase{
	{
		config: &simple1{},
		toml: "f1 = 3",
		args: []string{"-f1=4"},
		expected: &simple1{F1: 4},
	},
}

func TestGoodConfigs(t *testing.T) {
	for _, test := range testCases {
		if err := checkCase(test); err != nil {
			t.Error(err)
		}
	}
}
