# flagconf

[![GoDoc](https://godoc.org/github.com/cespare/flagconf?status.svg)](https://godoc.org/github.com/cespare/flagconf)

Package flagconf combines the standard library's [flag package](http://golang.org/pkg/flag) with Andrew
Gallant's excellent [TOML parsing library](https://github.com/BurntSushi/toml).

This package sets program options from a TOML configuration file while allowing the settings to be overridden
with command-line flags as well.

## Installation

    $ go get -u github.com/cespare/flagconf

## Usage

Documentation is on [godoc.org](http://godoc.org/github.com/cespare/flagconf).

Here is a small example:

``` go
import (
  "github.com/cespare/flagconf"
)

type Config struct {
  MaxProcs int    `desc:"maximum OS threads"`
  Addr     string `desc:"listen address (with port)"`
}

func main() {
  // Set the defaults.
  config := Config{
    MaxProcs: 4,
  }
  flagconf.Parse("config.toml", &config)
}
```

Now if the TOML file looks like this:

``` toml
# config.toml
maxprocs = 8
addr     = "localhost:7755"
```

and the program is run like this:

    $ ./prog -addr ":8888"

then `conf` will be:

    MaxProcs: 8
    Addr:     ":8888"

(That is, TOML settings override the defaults and flags given override those.)

## License

MIT
