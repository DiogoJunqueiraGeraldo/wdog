package wdog

import (
	"os"
	"time"
)

const (
	DefaultHallSize    = 1024
	DefaultHallTimeout = time.Millisecond * 10

	DefaultTeardownTimeout = time.Millisecond * 50
	DefaultToleranceWindow = time.Millisecond * 100
	DefaultToleranceCap    = 2
)

type Option func(*Configuration)
type Configuration struct {
	owner Owner

	hallSize    int
	hallTimeout time.Duration

	teardownTimeout time.Duration
	toleranceWindow time.Duration
	toleranceCap    int32

	isDebug bool
}

func NewConfiguration(owner Owner, opts ...Option) *Configuration {
	c := &Configuration{
		owner: owner,

		hallSize:    1,
		hallTimeout: DefaultHallTimeout,

		teardownTimeout: DefaultTeardownTimeout,
		toleranceWindow: DefaultToleranceWindow,
		toleranceCap:    DefaultToleranceCap,

		isDebug: os.Getenv("WDG_DEBUG") == "true",
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func WithOwner(owner Owner) Option {
	return func(cfg *Configuration) {
		cfg.owner = owner
	}
}

func WithHallSize(size int) Option {
	return func(cfg *Configuration) {
		cfg.hallSize = size
	}
}

func WithHallTimeout(timeout time.Duration) Option {
	return func(cfg *Configuration) {
		cfg.hallTimeout = timeout
	}
}

func WithToleranceWindow(window time.Duration) Option {
	return func(cfg *Configuration) {
		cfg.toleranceWindow = window
	}
}

func WithToleranceCap(cap int32) Option {
	return func(cfg *Configuration) {
		cfg.toleranceCap = cap
	}
}

func WithDebug(debug bool) Option {
	return func(cfg *Configuration) {
		cfg.isDebug = debug
	}
}
