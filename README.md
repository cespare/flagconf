# flagconf

This is a Go library that unifies the standard library's [flag package](http://golang.org/pkg/flag) with
@BurntSushi's excellent [TOML parsing library](https://github.com/BurntSushi/toml). It reduces boilerplate for
a common use case for me: configuring my program (usually a server of some kind) using a TOML configuration
file, but allowing for overriding specific settings with flags if I wish.

**Note:** This is a non-functional WIP but it should be ready for consumption soon.

## Installation

    $ go get -u github.com/cespare/flagconf

Notice that this has a dependency: `github.com/BurntSushi/toml`. I recommend vendoring your libraries to lock
in the versions in case they change or disappear in the future.

## Usage

Doesn't quite work yet, but this is an example of how it will look:

``` go
import (
  "github.com/cespare/flagconf"
)

type Config struct {
  MaxProcs int    `flag:"maxprocs"`
  Addr     string `flag:"addr"`
}

func main() {
  // Set the defaults
  config := Config{
    MaxProcs: 4,
  }
  flagconf.Parse("config.toml", &config)
}
```

Now if your toml looks like this:

```
maxprocs = 8
addr = "localhost:7755"
```

and you run your program with

    $ ./prog -addr ":8888"

then `conf` will be:

    MaxProcs: 8
    Addr: ":8888"

## License

MIT
