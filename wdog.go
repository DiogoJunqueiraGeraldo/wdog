// Package wdog implements a lightweight, concurrency-safe watchdog mechanism
// for monitoring the behavior of goroutines in Go programs.
//
// It emits signals ("noises") when a task panics, fails to respect its context,
// or when too many errors occur within a configured time window.
// The main use case is monitoring critical background tasks, especially in
// systems where resilience and error observability are key.
//
// A WDog can be integrated with custom event loggers, metrics systems,
// or incident responders via the Owner interface.
package wdog

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync/atomic"
	"time"
)

var (
	// ErrToleranceExceeded is emitted when the number of accumulated errors exceeds
	// the configured tolerance cap within a given time window.
	ErrToleranceExceeded = errors.New("watchdog tolerance exceeded")

	// ErrTaskPanicked is emitted when a monitored task panics.
	ErrTaskPanicked = errors.New("task under watch panicked")

	// ErrTaskNotContextCompliant is emitted when a task fails to terminate
	// within the configured teardownTimeout after its context is cancelled.
	ErrTaskNotContextCompliant = errors.New("task not context compliant")
)

// NoiseType represents the category of the alert emitted by the watchdog.
type NoiseType string

const (
	// Growl signals a task did not respect its context deadline (context non-compliant).
	Growl NoiseType = "growl"

	// Bark signals the watchdog's tolerance for errors has been exceeded.
	Bark NoiseType = "bark"

	// Cry signals a task panicked during execution.
	Cry NoiseType = "cry"
)

// Noise represents an alert emitted by the watchdog system.
//
// Consumers should treat each Noise as a self-contained report about a single
// observed issue, such as a panic, a context violation, or an error accumulation.
type Noise struct {
	// Type indicates the category of the emitted noise.
	Type NoiseType

	// ErrCount reflects the error count at the time of emission (used with Bark).
	ErrCount int32

	// Error describes the root cause or context of the alert.
	Error error

	// Payload optionally carries any extra data (e.g., recovered panic object).
	Payload any
}

// Silence is a sentinel zero-value Noise, representing "no alert".
var Silence = Noise{}

// Owner is the consumer of emitted noises from a Watchdog instance.
//
// Implementations of Owner should handle alerts safely and asynchronously.
// For example, they might send them to a logger, metrics backend, or alerting system.
type Owner interface {
	// Hear is called with each emitted Noise.
	Hear(Noise)
}

// WDog monitors tasks for panics, timeout violations, and error bursts.
//
// It emits structured alerts (Noises) to a registered Owner and supports
// soft failure semantics via context cancellation and configurable tolerance.
//
// This implementation favors liveness over strict delivery guarantees—some
// alerts may be dropped under load.
type WDog struct {
	owner Owner

	errCount int32         // Accumulated error counter.
	hall     chan Noise    // Internal buffered channel for emitted noises.
	close    chan struct{} // Signals watchdog shutdown.

	hallTimeout time.Duration // Max duration to wait when emitting a Noise.

	teardownTimeout time.Duration // Grace period after context cancellation before a task is considered faulty.
	toleranceWindow time.Duration // Time window to evaluate the tolerance threshold.
	toleranceCap    int32         // Maximum errors tolerated within the window.

	isDebug bool // If true, emits debug logs.
}

// New creates a new watchdog instance using the provided configuration.
func New(cfg *Configuration) *WDog {
	return &WDog{
		errCount:        0,
		hall:            make(chan Noise, cfg.hallSize),
		close:           make(chan struct{}),
		hallTimeout:     cfg.hallTimeout,
		owner:           cfg.owner,
		teardownTimeout: cfg.teardownTimeout,
		toleranceWindow: cfg.toleranceWindow,
		toleranceCap:    cfg.toleranceCap,
		isDebug:         cfg.isDebug,
	}
}

// Watch starts internal goroutines for monitoring tolerance and delivering alerts.
// Must be called before any tasks are submitted.
func (w *WDog) Watch() {
	go w.monitorTolerance()
	go w.listenToHall()
}

// Close stops the watchdog by signalling all internal goroutines to shut down.
//
// After Close is called, no further alerts will be emitted, and calling Go()
// becomes undefined behavior (may panic or silently fail).
func (w *WDog) Close() {
	close(w.close)
}

// Go runs the given task under watchdog supervision.
//
// It monitors for:
//   - panics (emitting Cry)
//   - context deadline violations (emitting Growl)
//   - cumulative errors (emitting Bark via tolerance monitor)
//
// Use this instead of `go func()` when spawning monitored tasks.
func (w *WDog) Go(ctx context.Context, t func(ctx context.Context)) {
	w.log("going up")
	done := make(chan struct{})

	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				w.emitNoise(Noise{
					Type:    Cry,
					Error:   ErrTaskPanicked,
					Payload: r,
				})
				w.log("recovered from panic")
			}
		}()

		t(ctx)
		w.log("task completed")
	}()

	go func() {
		select {
		case <-done:
			w.log("task completed before ctx cancellation")
		case <-ctx.Done():
			select {
			case <-done:
				w.log("task completed before teardown timeout")
			case <-time.After(w.teardownTimeout):
				w.log("task teardown timeout")
				atomic.AddInt32(&w.errCount, 1)
				w.emitNoise(Noise{
					Type:  Growl,
					Error: ErrTaskNotContextCompliant,
				})
			}
		}
	}()
}

// emitNoise attempts to deliver a Noise into the hall channel.
//
// If the watchdog is closed or the hall is blocked for longer than
// hallTimeout, the Noise is dropped silently.
func (w *WDog) emitNoise(noise Noise) {
	select {
	case <-w.close:
		w.log("watchdog is closed: no more events can be emitted")
	case w.hall <- noise:
		w.log("emitted noise to hall")
	case <-time.After(w.hallTimeout):
		w.log("timeout emitting noise to hall")
	}
}

// monitorTolerance checks periodically whether the error count exceeds the tolerance cap.
// Emits a Bark noise when the threshold is breached and resets the counter.
func (w *WDog) monitorTolerance() {
	ticker := time.NewTicker(w.toleranceWindow)

	for {
		select {
		case <-w.close:
			w.log("closing watchdog: stop monitoring tolerance")
			ticker.Stop()
			return
		case <-ticker.C:
			errCountSnapshot := atomic.LoadInt32(&w.errCount)
			w.log(fmt.Sprintf("monitoring tolerance err count: %d", errCountSnapshot))

			if errCountSnapshot >= w.toleranceCap {
				w.log("tolerance exceeded")
				w.emitNoise(Noise{
					Type:     Bark,
					ErrCount: errCountSnapshot,
					Error:    ErrToleranceExceeded,
				})
				atomic.StoreInt32(&w.errCount, 0)
			}
		}
	}
}

// listenToHall receives emitted Noises and forwards them to the Owner.
//
// This goroutine exits gracefully when the watchdog is closed.
func (w *WDog) listenToHall() {
	for {
		select {
		case <-w.close:
			w.log("closing watchdog: stop listening to hall")
			return
		case noise := <-w.hall:
			w.log("listening to hall noise")
			w.owner.Hear(noise)
		}
	}
}

// log prints debug messages to the standard logger if isDebug is true.
func (w *WDog) log(msg string) {
	if w.isDebug {
		log.Printf("[DEBUG] %s\n", msg)
	}
}
