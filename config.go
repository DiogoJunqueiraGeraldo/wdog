package wdog

import (
	"fmt"
	"os"
	"time"
)

const (
	// DefaultHallSize defines the buffer size of the internal alert channel.
	DefaultHallSize = 1024

	// DefaultHallTimeout is the maximum duration to wait when trying to
	// emit a noise to the alert channel.
	DefaultHallTimeout = time.Millisecond * 10

	// DefaultTeardownTimeout defines the grace period a task has to shut
	// down after its context is cancelled before being considered faulty.
	DefaultTeardownTimeout = time.Millisecond * 50

	// DefaultToleranceWindow is the interval over which accumulated errors
	// are evaluated against the tolerance cap.
	DefaultToleranceWindow = time.Millisecond * 100

	// DefaultToleranceCap is the maximum number of errors tolerated within
	// a single tolerance window before a Bark noise is emitted.
	DefaultToleranceCap = 2
)

const (
	// Reasonable bounds to prevent misconfiguration and ensure stable operation.

	MinHallSize = 1
	MaxHallSize = 1024 * 1024

	MinHallTimeout = 5 * time.Millisecond
	MaxHallTimeout = 500 * time.Millisecond

	MinToleranceWindow = 5 * time.Millisecond
	MaxToleranceWindow = 60 * time.Second

	MinTeardownTimeout = 5 * time.Millisecond
	MaxTeardownTimeout = 200 * time.Millisecond

	MinToleranceCap = 1
	MaxToleranceCap = 1_000_000
)

// Configuration holds all customizable parameters used to instantiate a WDog.
//
// It follows a functional options pattern via Option to allow partial overrides.
type Configuration struct {
	owner Owner

	hallSize    int           // Buffered channel size for emitted noises.
	hallTimeout time.Duration // Max time to wait to emit an alert to the hall.

	teardownTimeout time.Duration // Max wait time for a task to exit after context cancel.
	toleranceWindow time.Duration // Window used to check accumulated errors.
	toleranceCap    int32         // Max tolerated errors in the window before Bark is emitted.

	isDebug bool // Enables internal debug logging to stdout.
}

// Option is a functional setter used to modify Configuration fields.
type Option func(*Configuration)

// NewConfiguration builds a Configuration instance with sensible defaults,
// and applies any user-provided options.
//
// The debug mode will automatically activate if the WDG_DEBUG environment variable is set to "true".
func NewConfiguration(owner Owner, opts ...Option) *Configuration {
	if owner == nil {
		panic("owner cannot be nil")
	}

	c := &Configuration{
		owner: owner,

		hallSize:    DefaultHallSize,
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

// WithHallSize sets the buffer size of the internal alert channel.
func WithHallSize(size int) Option {
	if size < MinHallSize || size > MaxHallSize {
		panic(fmt.Sprintf("size out of range min %d, max %d", MinHallSize, MaxHallSize))
	}

	return func(cfg *Configuration) {
		cfg.hallSize = size
	}
}

// WithHallTimeout sets the timeout for alert delivery via emitNoise.
func WithHallTimeout(timeout time.Duration) Option {
	if timeout < MinHallTimeout || timeout > MaxHallTimeout {
		panic(fmt.Sprintf("timeout out of range min %d, max %d", MinHallTimeout, MaxHallTimeout))
	}

	return func(cfg *Configuration) {
		cfg.hallTimeout = timeout
	}
}

// WithToleranceWindow configures the interval for checking accumulated error counts.
func WithToleranceWindow(window time.Duration) Option {
	if window < MinToleranceWindow || window > MaxToleranceWindow {
		panic(fmt.Sprintf("window out of range min %d, max %d", MinToleranceWindow, MaxToleranceWindow))
	}

	return func(cfg *Configuration) {
		cfg.toleranceWindow = window
	}
}

// WithTeardownTimeout configures the grace period allowed for goroutines to comply with context cancellation.
func WithTeardownTimeout(timeout time.Duration) Option {
	if timeout < MinTeardownTimeout || timeout > MaxTeardownTimeout {
		panic(fmt.Sprintf("timeout out of range min %d, max %d", MinTeardownTimeout, MaxTeardownTimeout))
	}

	return func(cfg *Configuration) {
		cfg.teardownTimeout = timeout
	}
}

// WithToleranceCap sets the maximum number of tolerated errors within a tolerance window.
func WithToleranceCap(cap int32) Option {
	if cap < MinToleranceCap || cap > MaxToleranceCap {
		panic(fmt.Sprintf("cap out of range min %d, max %d", MinToleranceCap, MaxToleranceCap))
	}

	return func(cfg *Configuration) {
		cfg.toleranceCap = cap
	}
}

// WithDebug explicitly enables or disables debug logging.
func WithDebug(debug bool) Option {
	return func(cfg *Configuration) {
		cfg.isDebug = debug
	}
}
