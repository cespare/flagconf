package main

import (
	"fmt"
	"log"
	"time"

	"github.com/cespare/flagconf"
)

type Config struct {
	N time.Time `toml:"m" flag:"n"`
	Foo struct { Bar string `flag:"bar"` } `flag:"foo"`
}

func main() {
	c := Config{}
	err := flagconf.Parse("example/simple.toml", &c)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("\033[01;34m>>>> c: %v\x1B[m\n", c)
}
