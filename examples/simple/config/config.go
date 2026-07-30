package config

import (
	"os"
	"strings"
)

type LogLevel int8

const (
	LogLevelInfo  LogLevel = iota
	LogLevelWarn           = iota
	LogLevelError          = iota
)

type Config struct {
	Addr      string
	Foo       string
	LogLevel  LogLevel
	SentryDSN string // optional; empty disables the SDK
}

func ParseConfig() Config {
	require := func(name string) string {
		v := os.Getenv(name)
		if v == "" {
			panic(name + " env var is required")
		}
		return v
	}

	parseLogLevel := func() LogLevel {
		switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
		case "info":
			return LogLevelInfo
		case "warn":
			return LogLevelWarn
		case "error":
			return LogLevelError
		}
		return LogLevelError
	}

	return Config{
		Addr:      require("ADDR"),
		Foo:       require("FOO"),
		LogLevel:  parseLogLevel(),
		SentryDSN: os.Getenv("SENTRY_DSN"),
	}
}
