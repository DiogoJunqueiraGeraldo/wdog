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
	ErrToleranceExceeded       = errors.New("watchdog tolerance exceeded")
	ErrTaskPanicked            = errors.New("task under watch panicked")
	ErrTaskNotContextCompliant = errors.New("task not context compliant")
)

type NoiseType string

const (
	Growl NoiseType = "growl"
	Bark  NoiseType = "bark"
	Cry   NoiseType = "cry"
)

type WDog struct {
	owner Owner

	errCount        int32
	hall            chan Noise
	hallTimeout     time.Duration
	teardownTimeout time.Duration
	toleranceWindow time.Duration
	toleranceCap    int32
	isDebug         bool
}

type Noise struct {
	Type NoiseType

	ErrCount int32
	Error    error

	Payload any
}

type Owner interface {
	Hear(Noise)
}

func New(cfg *Configuration) *WDog {
	hall := make(chan Noise, cfg.hallSize)

	return &WDog{
		errCount:        0,
		hall:            hall,
		hallTimeout:     cfg.hallTimeout,
		owner:           cfg.owner,
		teardownTimeout: cfg.teardownTimeout,
		toleranceWindow: cfg.toleranceWindow,
		toleranceCap:    cfg.toleranceCap,
		isDebug:         cfg.isDebug,
	}
}

func (w *WDog) Watch() {
	go w.monitorTolerance()
	go w.listenToHall()
}

func (w *WDog) monitorTolerance() {
	ticker := time.NewTicker(w.toleranceWindow)

	for {
		select {
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

func (w *WDog) emitNoise(noise Noise) {
	select {
	case w.hall <- noise:
		w.log("emitted noise to hall")
	case <-time.After(w.teardownTimeout):
		w.log("timeout emitting noise to hall")
	}
}

func (w *WDog) listenToHall() {
	for {
		select {
		case noise := <-w.hall:
			w.log("listening to hall noise")
			w.owner.Hear(noise)
		}
	}
}

func (w *WDog) log(msg string) {
	if w.isDebug {
		log.Println("DEBUG (wdog): " + msg)
	}
}

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
