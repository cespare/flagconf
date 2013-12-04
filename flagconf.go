// Package flagconf combine's the standard library's flag package with Andrew Gallant's excellent TOML parsing
// library: https://github.com/BurntSushi/toml.
//
// This package sets program options from a TOML configuration file while allowing the settings to be
// overridden with command-line flags as well.
package flagconf

import (
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/cespare/flagconf/toml"
)

// ParseStrings reads a TOML configuration file at path as well as command-line arguments in args and sets
// matching options in config, which must be a non-nil pointer to a struct.
//
// ParseStrings is similar to Parse except that it provides the caller with more fine-grained control.
//
// The argument array is passed into ParseStrings the args parameter; note that this is treated like os.Args
// in that args[0] is interpreted as the executable name and args[1:] are flags.
//
// The allowNoConfig parameter controls whether ParseStrings returns an error if no file is found at path.
func ParseStrings(args []string, path string, config interface{}, allowNoConfigFile bool) error {
	if len(args) < 1 {
		return fmt.Errorf("flagconf: ParseStrings called with empty args")
	}
	v := reflect.ValueOf(config)
	if v.Kind() != reflect.Ptr || reflect.Indirect(v).Kind() != reflect.Struct {
		return fmt.Errorf("flagconf: config must be a pointer to a struct")
	}

	flagset := flag.NewFlagSet(args[0], flag.ExitOnError)
	// Create flags
	if err := registerFlags(flagset, reflect.Indirect(v), "", ""); err != nil {
		return err
	}

	// Load TOML
	_, err := toml.DecodeFile(path, config)
	if err != nil {
		if !(os.IsNotExist(err) && allowNoConfigFile) {
			return err
		}
	}

	// Override any settings with configured flags
	return flagset.Parse(args[1:])
}

/*
Parse reads a TOML configuration file at path as well as user-supplied options from os.Args and sets matching
options in config, which must be a non-nil pointer to a struct.

The idea is that you will put your configuration settings into a struct and populate an instance of that
struct with the default values.

Then you will call flagconf.Parse, passing in the path to the configuration file and a pointer to your
configuration. This function will read settings in from the TOML file and then read the user-supplied
arguments from os.Args.

Example

Here is a small example:

		import (
		  "github.com/cespare/flagconf"
		)

		type Config struct {
		  MaxProcs int    `desc:"maximum OS threads"`
		  Addr     string `desc:"listen address (with port)"`
		}

		func main() {
		  // Set the defaults
		  config := Config{
		    MaxProcs: 4,
		  }
		  flagconf.Parse("config.toml", &config)
		}

Now if your toml looks like this:

		maxprocs = 8
		addr = "localhost:7755"

and you run your program with

    $ ./prog -addr ":8888"

then conf will be:

    MaxProcs: 8
    Addr: ":8888"

(That is, TOML settings override the defaults and flags given override those.)

Descriptions for the flags are taken from the "desc" struct tag. A default description is created based on the
field type if a tag is not provided.

TOML matches are attempted for every exported field in the configuration struct. Flag names are constructed
for every exported field. (Unexported fields are ignored by flagconf.)

Parse returns an error if no file can be found at path.

Types

For simplicity, flagconf supports a fairly limited set of types: the intersection of types supported by TOML
and the types supported by `flag`. The basic types are therefore:

		bool
		string
		int
		int64
		uint
		uint64
		float64

The only exception is that flagconf has special handling for structs. Flagconf recursively inspects structs
and and creates them as necessary when your config contains a nil struct pointer. In TOML, a struct
corresponds to a nested section; in flags a struct will be dot-separated:

		type Conf {
			S *struct {
				N int
			}
		}

		// corresponds to this TOML
		[s]
		n = 3

		// and this flag
    -s.n=3

Naming

Matching names from TOML values to struct field names is much like encoding/json: exact matches are preferred
and then case-insensitive matching will be accepted. (Typically you'll use all lowercase for your TOML names,
but the struct fields must be exported.) You can use the struct tag "toml" to set a different name if you
wish.

The flag names are constructed by lowercasing the struct field name. You can use the "flag" struct tag to set
that name if you wish, as well.

		type Conf struct {
			Foo string `toml:"bar" flag:"baz"`
		}
*/
func Parse(path string, config interface{}) error {
	return ParseStrings(os.Args, path, config, false)
}

func joinNS(ns, name string) string {
	if ns == "" {
		return name
	}
	return ns + "." + name
}

func registerFlags(flagset *flag.FlagSet, v reflect.Value, namespace, description string) error {
	if v.Kind() == reflect.Struct {
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if !field.CanSet() {
				continue
			}
			typ := v.Type().Field(i)
			name := strings.ToLower(typ.Name)
			if tag := typ.Tag.Get("flag"); tag != "" {
				name = tag
			}
			desc := typ.Tag.Get("desc")
			newNS := joinNS(namespace, name)
			// Create uninitialized pointers to structs
			if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct {
				if field.IsNil() && field.IsValid() && field.CanSet() {
					newField := reflect.New(field.Type().Elem())
					field.Set(newField)
				}
				field = field.Elem()
			}
			if err := registerFlags(flagset, field, newNS, desc); err != nil {
				return err
			}
		}
		return nil
	}

	var flagFunc reflect.Value

	switch v.Type() {
	case reflect.TypeOf(false):
		flagFunc = reflect.ValueOf(flagset.BoolVar)
	case reflect.TypeOf(float64(0)):
		flagFunc = reflect.ValueOf(flagset.Float64Var)
	case reflect.TypeOf(int(0)):
		flagFunc = reflect.ValueOf(flagset.IntVar)
	case reflect.TypeOf(int64(0)):
		flagFunc = reflect.ValueOf(flagset.Int64Var)
	case reflect.TypeOf(""):
		flagFunc = reflect.ValueOf(flagset.StringVar)
	case reflect.TypeOf(uint(0)):
		flagFunc = reflect.ValueOf(flagset.UintVar)
	case reflect.TypeOf(uint64(0)):
		flagFunc = reflect.ValueOf(flagset.Uint64Var)
	default:
		return fmt.Errorf("flagconf: unhandled type: %s", v.Type())
	}

	p := v.Addr()
	name := reflect.ValueOf(namespace)
	usage := reflect.ValueOf(description)
	if description == "" {
		usage = reflect.ValueOf(fmt.Sprintf("(%s flag, no description given)", v.Type()))
	}
	args := []reflect.Value{p, name, v, usage}
	flagFunc.Call(args)
	return nil
}
