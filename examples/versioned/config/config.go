package config

import "os"

type Config struct {
	Addr string
	Foo  string
}

func ParseConfig() Config {
	require := func(name string) string {
		v := os.Getenv(name)
		if v == "" {
			panic(name + " env var is required")
		}
		return v
	}

	return Config{
		Addr: require("ADDR"),
		Foo:  require("FOO"),
	}
}
