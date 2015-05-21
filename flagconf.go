// Package flagconf combines the standard library's flag package
// with Andrew Gallant's excellent TOML parsing library:
// https://github.com/BurntSushi/toml.
//
// This package sets program options from a TOML configuration file
// while allowing the settings to be overridden with command-line flags as well.
package flagconf

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// ParseStrings reads a TOML configuration file at path as well as
// command-line arguments in args and sets matching options in config,
// which must be a non-nil pointer to a struct.
//
// ParseStrings is similar to Parse except that it provides the caller
// with more fine-grained control.
//
// The argument array is passed into ParseStrings the args parameter;
// note that this is treated like os.Args in that args[0] is interpreted
// as the executable name and args[1:] are flags.
//
// The allowNoConfig parameter controls whether ParseStrings returns an error
// if no file is found at path.
func ParseStrings(args []string, path string, config interface{}, allowNoConfigFile bool) error {
	if len(args) < 1 {
		return fmt.Errorf("flagconf: ParseStrings called with empty args")
	}
	v := reflect.ValueOf(config)
	if v.Kind() != reflect.Ptr || reflect.Indirect(v).Kind() != reflect.Struct {
		return fmt.Errorf("flagconf: config must be a pointer to a struct")
	}

	flagset := flag.NewFlagSet(args[0], flag.ContinueOnError)
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

	// Prevent flagset.Parse from printing error and usage to stderr if parsing
	// fails
	flagset.Usage = func() {}
	buf := &bytes.Buffer{}
	flagset.SetOutput(buf)

	// Override any settings with configured flags
	if err = flagset.Parse(args[1:]); err != nil {
		// In case flag parsing fails, return a custom error containing usage
		// info if user wants to print it.
		fmt.Fprintf(buf, "Usage of %s:\n", args[0])
		flagset.PrintDefaults()
		err = FlagError{Err: err, Usage: strings.TrimSpace(buf.String())}
	}
	return err
}

// FlagError combines error received from flag parsing with default usage info.
type FlagError struct {
	Err   error
	Usage string
}

func (e FlagError) Error() string {
	return e.Err.Error()
}

// IsHelp checks whether err was caused by user requesting help output
// (setting -h or --help flags).
func IsHelp(err error) bool {
	if err, ok := err.(FlagError); ok {
		return err.Err == flag.ErrHelp
	}
	return err == flag.ErrHelp
}

/*
Parse reads a TOML configuration file at path as well as user-supplied options
from os.Args and sets matching options in config, which must be a non-nil pointer to a struct.

Typical usage is that the user represents configuration options with a struct type
and then populates a value of that type with the default configuration values.

Then the user calls flagconf.Parse, passing in the path to the configuration file
and a pointer to the configuration value. This function will read settings
from the TOML file and then read the user-supplied arguments from os.Args.

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

Now if the TOML file looks like this:

		# config.toml
    maxprocs = 8
    addr     = "localhost:7755"

and the user runs the program with

    ./prog -addr ":8888"

then conf will be:

    MaxProcs: 8
    Addr:     ":8888"

(That is, TOML settings override the defaults and flags given override those.)

Descriptions for the flags are taken from the "desc" struct tag.
A default description is created based on the field type if a tag is not provided.

TOML matches are attempted for every exported field in the configuration struct.
Flag names are constructed for every exported field. Unexported fields,
as well as exported fields tagged with `flag:"-"`, are ignored by flagconf.
(If a field is ignored by using this tag, it is typically best to also use
`toml:"-"` so that the field is not picked up by the TOML parser.)

Parse returns an error if no file can be found at path.

Types

The basic types flagconf supports are those which are directly supported by both
package flag and TOML:

    bool
    string
    int
    int64
    uint
    uint64
    float64

Flagconf also supports any type implementing flag.Value, as long as TOML also supports it.

Finally, flagconf supports nesting by recursively inspecting structs
and creating them as necessary when the config value contains a nil struct pointer.
In TOML, a struct corresponds to a nested section; in flags the name will be dot-separated:

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

Embedded structs are handled like in encoding/json: their exported fields are
treated as if they were fields of the outer struct.

Naming

Matching names from TOML values to struct field names is much like encoding/json:
exact matches are preferred and then case-insensitive matching will be accepted.
(TOML names are typically lowercase, but the struct fields must be exported.)
The struct tag "toml" can be used to set a different name.

The flag names are constructed by lowercasing the struct field name.
The "flag" struct tag controls the flag name.

    type Conf struct {
      Foo string `toml:"bar" flag:"baz"`
    }
*/
func Parse(path string, config interface{}) error {
	return ParseStrings(os.Args, path, config, false)
}

// MustParse is like Parse except if parsing fails it prints the error and
// exits the program.
func MustParse(path string, config interface{}) {
	if err := ParseStrings(os.Args, path, config, false); err != nil {
		if ferr, ok := err.(FlagError); ok {
			fmt.Fprintln(os.Stderr, ferr.Usage)
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(2)
	}
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
				if tag == "-" {
					continue
				}
				name = tag
			}
			desc := typ.Tag.Get("desc")
			newNS := joinNS(namespace, name)
			// For embedded fields don't create an extra nested namespace.
			if v.Type().Field(i).Anonymous {
				newNS = namespace
			}
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

	name := reflect.ValueOf(namespace)
	usage := reflect.ValueOf(description)
	if description == "" {
		usage = reflect.ValueOf(fmt.Sprintf("(%s flag, no description given)", v.Type()))
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
		// reflect.Type of flag.Value
		fvt := reflect.TypeOf((*flag.Value)(nil)).Elem()
		if v.Type().Implements(fvt) {
			flagset.Var(v.Interface().(flag.Value), namespace, usage.String())
			return nil
		}
		// If value is addressable, its pointer may implement flag.Value
		if v.CanAddr() && v.Addr().Type().Implements(fvt) {
			flagset.Var(v.Addr().Interface().(flag.Value), namespace, usage.String())
			return nil
		}

		return fmt.Errorf("flagconf: unhandled type: %s", v.Type())
	}

	p := v.Addr()
	args := []reflect.Value{p, name, v, usage}
	flagFunc.Call(args)
	return nil
}

// Strings is a convenience wrapper around a string slice that implements
// flag.Value and handles the value as comma-separated list.
//
// For example flag -peers=127.0.0.1,127.0.0.2 will result in a slice
// {"127.0.0.1", "127.0.0.2"}.
type Strings []string

func (ss Strings) String() string {
	return strings.Join(ss, ",")
}

func (ss *Strings) Set(s string) error {
	*ss = Strings(strings.Split(s, ","))
	return nil
}

// Ints is a convenience wrapper around an int slice that implements flag.Value
// and handles the value as comma-separated list.
//
// For example flag -cpu=1,2,4 will result in a slice {1, 2, 4}.
type Ints []int

func (is Ints) String() string {
	ss := make([]string, 0, len(is))
	for _, i := range is {
		ss = append(ss, strconv.Itoa(i))
	}
	return strings.Join(ss, ",")
}

func (is *Ints) Set(ss string) error {
	for _, s := range strings.Split(ss, ",") {
		i, err := strconv.Atoi(s)
		if err != nil {
			return err
		}
		*is = append(*is, i)
	}
	return nil
}
