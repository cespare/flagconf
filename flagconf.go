package flagconf

import (
	"flag"
	"fmt"
	"os"
	"reflect"

	"github.com/BurntSushi/toml"
)

var (
	AllowNoConfigFile = false
)

func Parse(path string, config interface{}) error {
	// Create flags
	if err := registerFlags(reflect.ValueOf(config), ""); err != nil {
		return err
	}

	// Load TOML
	_, err := toml.DecodeFile(path, config)
	if err != nil {
		if !(os.IsNotExist(err) && AllowNoConfigFile) {
			return err
		}
	}

	// Override any settings with configured flags
	flag.Parse()

	return nil
}

func joinNS(ns, name string) string {

	if ns == "" {
		return name
	}
	return ns + "." + name
}

func registerFlags(s reflect.Value, namespace string) error {
	kind := s.Kind()
	fmt.Printf("\033[01;34m>>>> kind: %v\x1B[m\n", kind)
	var flagFunc reflect.Value

	switch kind {
	case reflect.Ptr:
		if s.IsNil() {
			return fmt.Errorf("Nil interfaces are unhandled.")
		}
		return registerFlags(reflect.Indirect(s), namespace)
	case reflect.Struct:
		for i := 0; i < s.NumField(); i++ {
			typ := s.Type().Field(i)
			name := typ.Name
			if tag := typ.Tag.Get("flag"); tag != "" {
				name = tag
			}
			newNS := joinNS(namespace, name)
			if err := registerFlags(s.Field(i), newNS); err != nil {
				return err
			}
		}
		return nil

	case reflect.Bool:
		flagFunc = reflect.ValueOf(flag.BoolVar)
	case reflect.Float64:
		flagFunc = reflect.ValueOf(flag.Float64Var)
	case reflect.Int:
		flagFunc = reflect.ValueOf(flag.IntVar)
	case reflect.Int64:
		flagFunc = reflect.ValueOf(flag.Int64Var)
	case reflect.String:
		flagFunc = reflect.ValueOf(flag.StringVar)
	case reflect.Uint:
		flagFunc = reflect.ValueOf(flag.UintVar)
	case reflect.Uint64:
		flagFunc = reflect.ValueOf(flag.Uint64Var)

	default:
		return fmt.Errorf("Unhandled type: %s", kind)
	}

	p := s.Addr()
	name := reflect.ValueOf(namespace)
	zero := reflect.Zero(s.Type())
	usage := reflect.ValueOf("usage description here")
	args := []reflect.Value{p, name, zero, usage}
	flagFunc.Call(args)
	return nil
}
